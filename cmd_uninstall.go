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

var (
	uninstallYes    bool
	uninstallConfig bool
	uninstallPlugin bool
	uninstallAll    bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove CEM from the system",
	Long: `  cem uninstall              → delete cem, cemi, cemir binaries
                               (asks about the config directory)
  cem uninstall --yes        → no questions
  cem uninstall --config     → also delete ~/.cem
  cem uninstall --plugin     → also delete the IDE plugins (JetBrains, VS Code)
  cem uninstall --all        → all of the above, no questions

The AI CLIs themselves (claude, agy, gpt, cursor) are NOT touched: cemir
removes those. Binaries under /usr/local/bin need sudo — cem retries the
delete with sudo and prints the manual command if that fails too.`,
	Run: func(cmd *cobra.Command, args []string) {
		if uninstallAll {
			uninstallYes, uninstallConfig, uninstallPlugin = true, true, true
		}
		// Terminal yoksa soru sorulamaz: onaysız silmek yerine ne yapılacağını
		// söylemek gerekiyor (setup kapısıyla aynı gerekçe).
		if !uninstallYes && !isInteractiveStdin() {
			fmt.Println(styleError.Render(L("✗ Terminal yok — onay sorulamaz.",
				"✗ No terminal — cannot ask for confirmation.")))
			fmt.Println(styleDim.Render("    cem uninstall --yes"))
			os.Exit(1)
		}
		PrintBanner(BannerCem)
		runUninstall()
	},
}

func init_uninstall() {
	uninstallCmd.Flags().BoolVar(&uninstallYes, "yes", false, "do not ask")
	uninstallCmd.Flags().BoolVar(&uninstallConfig, "config", false, "also delete ~/.cem")
	uninstallCmd.Flags().BoolVar(&uninstallPlugin, "plugin", false, "also delete the IDE plugins")
	uninstallCmd.Flags().BoolVar(&uninstallAll, "all", false, "binaries + config + IDE plugins, no questions")
	rootCmd.AddCommand(uninstallCmd)
}

// onay — --yes verilmişse sormaz.
func onay(soru string) bool {
	if uninstallYes {
		return true
	}
	return askYN(soru)
}

func runUninstall() {
	fmt.Println(styleBold.Render(L("  CEM kaldırılacak.", "  CEM will be removed.")))
	fmt.Println(styleDim.Render(L("  Bu işlem cem, cemi ve cemir komutlarını siler.", "  This removes the cem, cemi and cemir commands.")))
	fmt.Println()

	if !onay("  Devam edilsin mi?") {
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

	// ── IDE eklentileri ───────────────────────────────────────────────────────
	// Binary silinince eklenti işlevsiz kalıyor ama diskte duruyor ve IDE
	// açılışta yüklemeye devam ediyor: her çalıştırmada "cem bulunamadı".
	// Kullanıcının bunları elle bulması zor (JetBrains dizinleri ürün+sürüm
	// başına ayrı), o yüzden burada listeleniyor.
	if dizinler := ideEklentiDizinleri(); len(dizinler) > 0 {
		fmt.Println()
		fmt.Println(styleBold.Render(L("  IDE eklentileri:", "  IDE plugins:")))
		for _, d := range dizinler {
			fmt.Println(styleDim.Render("  " + d))
		}
		if uninstallPlugin || (!uninstallYes && askYN(L("  Bunlar da silinsin mi?", "  Delete these too?"))) {
			for _, d := range dizinler {
				if err := os.RemoveAll(d); err != nil {
					fmt.Printf("  %s %s: %v\n", styleError.Render("✗"), d, err)
					continue
				}
				fmt.Printf("  %s %s\n", styleSuccess.Render("✓"), styleDim.Render(d))
			}
			fmt.Println(styleDim.Render(L("    IDE'yi yeniden başlat.", "    Restart the IDE.")))
		}
	}

	// ── Config klasörü ────────────────────────────────────────────────────────
	fmt.Println()
	home, _ := os.UserHomeDir()
	cemDir := filepath.Join(home, ".cem")

	if _, err := os.Stat(cemDir); err == nil {
		fmt.Printf(L("  Config klasörü: %s\n", "  Config directory: %s\n"), styleDim.Render(cemDir))
		if uninstallConfig || (!uninstallYes && askYN("  Config ve ayarlar da silinsin mi?")) {
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
		if uninstallConfig || (!uninstallYes && askYN("  Bu dizindeki .cem.yaml da silinsin mi?")) {
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

// ideEklentiDizinleri — kurulu cem eklentilerinin dizinleri.
//
// JetBrains eklentileri ürün ve sürüm başına ayrı dizinde duruyor
// (GoLand2026.2, PyCharm2026.1 …), bu yüzden tek tek aramak yerine JetBrains
// kökünün altındaki her ürün dizininde 'cem-intellij' aranıyor. VS Code
// eklentisi tek bir extensions dizininde ve sürüm adıyla bitiyor.
func ideEklentiDizinleri() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var kökler []string
	switch runtime.GOOS {
	case "darwin":
		kökler = []string{filepath.Join(home, "Library", "Application Support", "JetBrains")}
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			kökler = []string{filepath.Join(appData, "JetBrains")}
		}
	default:
		kökler = []string{filepath.Join(home, ".local", "share", "JetBrains")}
	}

	var bulunan []string
	for _, kök := range kökler {
		ürünler, err := os.ReadDir(kök)
		if err != nil {
			continue
		}
		for _, ü := range ürünler {
			if !ü.IsDir() {
				continue
			}
			d := filepath.Join(kök, ü.Name(), "cem-intellij")
			if st, err := os.Stat(d); err == nil && st.IsDir() {
				bulunan = append(bulunan, d)
			}
		}
	}

	// VS Code: ~/.vscode/extensions/<yayıncı>.cem-<sürüm>
	vsc := filepath.Join(home, ".vscode", "extensions")
	if girdiler, err := os.ReadDir(vsc); err == nil {
		for _, g := range girdiler {
			if g.IsDir() && strings.Contains(strings.ToLower(g.Name()), "cem-") {
				bulunan = append(bulunan, filepath.Join(vsc, g.Name()))
			}
		}
	}
	return bulunan
}
