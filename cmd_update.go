package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download and install the latest release via cem.pw",
	Run: func(cmd *cobra.Command, args []string) {
		if err := selfUpdate(); err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func selfUpdate() error {
	osName := runtime.GOOS
	archName := runtime.GOARCH
	ext := ""
	if osName == "windows" {
		ext = ".exe"
	}

	myPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf(L("kendi yolu alınamadı: %w", "cannot resolve own path: %w"), err)
	}
	installDir := filepath.Dir(myPath)

	// Önce GitHub'dan son sürümü öğren — kullanıcıya neyin geleceğini söyle.
	latest, err := fetchLatestVersion()
	if err == nil && latest != "" {
		current := version // main paketindeki LDFLAGS değişkeni
		if !semverLess(current, latest) {
			// current >= latest → indirme gereksiz (devel build veya gecikmiş tag)
			fmt.Println(styleSuccess.Render(fmt.Sprintf(
				L("  ✓ zaten güncel: %s  (uzaktaki son: %s)", "  ✓ already up to date: %s  (latest remote: %s)"), current, latest)))
			// Cache'i de güncelle ki update notice tetiklenmesin
			saveUpdateCheckCache(updateCheckCache{
				LastCheck: time.Now(), LatestVersion: latest,
			})
			return nil
		}
		fmt.Println(styleDim.Render(fmt.Sprintf(L("  ⓘ son sürüm: %s  (mevcut: %s)", "  ⓘ latest: %s  (current: %s)"), latest, current)))
	}

	// Pre-flight: kurulum dizinine yazabiliyor muyuz?
	if !canWriteDir(installDir) {
		if osName == "windows" {
			fmt.Println(styleError.Render("  ✗ " + installDir + L(" yazılabilir değil", " is not writable")))
			fmt.Println(styleDim.Render(L("  Yönetici PowerShell'inde çalıştır:  cem update", "  Run in an elevated PowerShell:  cem update")))
			return fmt.Errorf("yetkisiz")
		}
		// Unix: sudo ile yeniden başlat
		fmt.Println(styleDim.Render("  ⚠ " + installDir + L(" yazılabilir değil, sudo ile devam ediliyor...", " is not writable, continuing with sudo...")))
		c := exec.Command("sudo", myPath, "update")
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}

	fmt.Println(styleDim.Render(fmt.Sprintf(L("  ⏳ son sürüm indiriliyor (%s/%s)...", "  ⏳ downloading the latest release (%s/%s)..."), osName, archName)))

	for _, name := range []string{"cem", "cemi", "cemir"} {
		asset := fmt.Sprintf("%s-%s-%s%s", name, osName, archName, ext)
		url := "https://cem.pw/r/" + asset
		dst := filepath.Join(installDir, name+ext)

		tmp, err := downloadToTemp(url)
		if err != nil {
			return fmt.Errorf(L("%s indirme hatası: %w", "%s download error: %w"), name, err)
		}
		if err := os.Chmod(tmp, 0o755); err != nil {
			os.Remove(tmp)
			return err
		}
		if err := replaceBinary(tmp, dst); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("%s → %s: %w", name, dst, err)
		}
		fmt.Println(styleSuccess.Render(fmt.Sprintf(L("  ✓ %s güncellendi → %s", "  ✓ %s updated → %s"), name, dst)))
	}

	// Update başarılı: cache'i yenile ki bir sonraki çağrıda eskimiş bildirim
	// gösterilmesin.
	if latest != "" {
		saveUpdateCheckCache(updateCheckCache{
			LastCheck:     time.Now(),
			LatestVersion: latest,
		})
	}

	if v, err := exec.Command(myPath, "--version").Output(); err == nil {
		fmt.Printf("\n  %s", string(v))
	}
	return nil
}

// fetchLatestVersion — GitHub Releases API'sinden son tag adını çeker.
func fetchLatestVersion() (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/muslu/cem/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var r struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	return r.TagName, nil
}

// canWriteDir — geçici dosya açıp silerek dizinin yazılabilirliğini test eder.
func canWriteDir(dir string) bool {
	f, err := os.CreateTemp(dir, ".cem-update-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

func downloadToTemp(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d (%s)", resp.StatusCode, url)
	}
	f, err := os.CreateTemp("", "cem-update-*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

// replaceBinary — running exe'yi de yenileyebilmek için Windows'ta önce
// hedefi .old'a taşır, sonra yenisini yerine koyar.
func replaceBinary(src, dst string) error {
	if runtime.GOOS == "windows" {
		if _, err := os.Stat(dst); err == nil {
			_ = os.Remove(dst + ".old")
			if err := os.Rename(dst, dst+".old"); err != nil {
				return err
			}
		}
		return os.Rename(src, dst)
	}
	// Linux/macOS: rename atomik, çalışan exe için de güvenli
	if err := os.Rename(src, dst); err != nil {
		// Cross-device rename hatası → copy+rename
		return copyThenReplace(src, dst)
	}
	return nil
}

func copyThenReplace(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	out.Close()
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return err
	}
	_ = os.Remove(src)
	return nil
}
