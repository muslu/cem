package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove CEM from the system",
	Long: `  cem uninstall   → delete cem, cemi, cemir binaries
                    asks whether to delete the config directory`,
	Run: func(cmd *cobra.Command, args []string) {
		PrintBanner(BannerCem)
		runUninstall()
	},
}

func init_uninstall() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall() {
	fmt.Println(styleBold.Render(L("  CEM kaldırılacak.", "  CEM will be removed.")))
	fmt.Println(styleDim.Render(L("  Bu işlem cem, cemi ve cemir komutlarını siler.", "  This removes the cem, cemi and cemir commands.")))
	fmt.Println()

	if !askYN("  Devam edilsin mi?") {
		fmt.Println(styleDim.Render(L("  İptal.", "  Cancelled.")))
		return
	}

	// ── Binary'leri bul ve sil ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println(styleBold.Render(L("  Binary'ler aranıyor...", "  Looking for binaries...")))

	names := []string{"cem", "cemi", "cemir"}
	if runtime.GOOS == "windows" {
		names = []string{"cem.exe", "cemi.exe", "cemir.exe"}
	}

	selfPath, _ := os.Executable()
	scheduledSelfDelete := ""

	removed := 0
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			fmt.Printf(L("  %s %-8s bulunamadı, atlandı\n", "  %s %-8s not found, skipped\n"), styleDim.Render("○"), name)
			continue
		}
		if err := os.Remove(path); err != nil {
			// Windows: çalışan exe'yi silemiyoruz — detached cmd ile gecikmeli sil
			if runtime.GOOS == "windows" && samePath(path, selfPath) {
				if scheduleWindowsSelfDelete(path) {
					fmt.Printf(L("  %s %-8s çıkışta silinecek %s\n", "  %s %-8s will be deleted on exit %s\n"),
						styleSuccess.Render("✓"), name, styleDim.Render(path))
					scheduledSelfDelete = path
					removed++
					continue
				}
			}
			// Unix: sudo ile dene
			if sudoRemove(path) {
				fmt.Printf("  %s %-8s %s\n",
					styleSuccess.Render("✓"), name, styleDim.Render(path))
				removed++
			} else {
				fmt.Printf("  %s %-8s silinemedi: %v\n",
					styleError.Render("✗"), name, err)
				hint := "sudo rm " + path
				if runtime.GOOS == "windows" {
					hint = `del /f "` + path + `"  (yönetici cmd)`
				}
				fmt.Printf("    %s\n", styleDim.Render("Manuel: "+hint))
			}
		} else {
			fmt.Printf("  %s %-8s %s\n",
				styleSuccess.Render("✓"), name, styleDim.Render(path))
			removed++
		}
	}
	_ = scheduledSelfDelete

	// ── Config klasörü ────────────────────────────────────────────────────────
	fmt.Println()
	home, _ := os.UserHomeDir()
	cemDir := filepath.Join(home, ".cem")

	if _, err := os.Stat(cemDir); err == nil {
		fmt.Printf(L("  Config klasörü: %s\n", "  Config directory: %s\n"), styleDim.Render(cemDir))
		if askYN("  Config ve ayarlar da silinsin mi?") {
			if err := os.RemoveAll(cemDir); err != nil {
				fmt.Printf("  %s Config silinemedi: %v\n", styleError.Render("✗"), err)
			} else {
				fmt.Printf("  %s Config silindi\n", styleSuccess.Render("✓"))
			}
		} else {
			fmt.Printf("  %s Config korundu → %s\n",
				styleDim.Render("○"), cemDir)
		}
	}

	// ── Proje .cem.yaml ────────────────────────────────────────────────────────
	if _, err := os.Stat(".cem.yaml"); err == nil {
		fmt.Println()
		if askYN("  Bu dizindeki .cem.yaml da silinsin mi?") {
			os.Remove(".cem.yaml")
			fmt.Printf("  %s .cem.yaml silindi\n", styleSuccess.Render("✓"))
		}
	}

	// ── Sonuç ─────────────────────────────────────────────────────────────────
	fmt.Println()
	if removed > 0 {
		fmt.Println(styleSuccess.Render(L("  ✓ CEM kaldırıldı.", "  ✓ CEM removed.")))
		fmt.Println()
		fmt.Println(styleDim.Render(L("  Yeniden kurmak için:", "  To reinstall:")))
		if runtime.GOOS == "windows" {
			fmt.Println(styleDim.Render("  irm cem.pw/install.ps1 | iex"))
		} else {
			fmt.Println(styleDim.Render("  curl -fsSL cem.pw/install | sh"))
		}
	} else {
		fmt.Println(styleWarn.Render(L("  ⚠ Hiçbir binary silinemedi.", "  ⚠ No binaries could be deleted.")))
		fmt.Println(styleDim.Render("  sudo ile dene veya manuel sil."))
	}
	fmt.Println()
}

// sudoRemove — sudo ile silmeyi dene
func sudoRemove(path string) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	cmd := exec.Command("sudo", "rm", "-f", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// stdin'i bağla (sudo şifre sorabilir)
	cmd.Stdin = os.Stdin
	return cmd.Run() == nil
}

// samePath — iki yolu OS-uygun karşılaştırır (Windows büyük/küçük harf duyarsız).
func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	aa, _ := filepath.Abs(a)
	bb, _ := filepath.Abs(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(aa, bb)
	}
	return aa == bb
}

// scheduleWindowsSelfDelete — biz çıktıktan sonra cem.exe'yi silmesi için
// detached bir cmd.exe başlatır. ping ile ~1s bekler, sonra del.
func scheduleWindowsSelfDelete(path string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	// /c "ping -n 2 127.0.0.1 > nul & del /f /q <path>"
	cmd := exec.Command("cmd", "/c",
		"ping -n 2 127.0.0.1 > nul & del /f /q "+`"`+path+`"`)
	// stdin/out/err'i bağlama → süreç ana process'e tutunmadan devam etsin
	if err := cmd.Start(); err != nil {
		return false
	}
	_ = cmd.Process.Release()
	return true
}

// PATH'daki tüm cem binary konumlarını bul (birden fazla olabilir)
func findAllBinaries() []string {
	var found []string
	names := []string{"cem", "cemi", "cemir"}
	if runtime.GOOS == "windows" {
		names = []string{"cem.exe", "cemi.exe", "cemir.exe"}
	}

	pathDirs := filepath.SplitList(os.Getenv("PATH"))
	for _, dir := range pathDirs {
		for _, name := range names {
			full := filepath.Join(dir, name)
			if _, err := os.Stat(full); err == nil {
				found = append(found, full)
			}
		}
	}

	// Tekrarları temizle
	seen := map[string]bool{}
	unique := found[:0]
	for _, f := range found {
		key := strings.ToLower(f)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, f)
		}
	}
	return unique
}
