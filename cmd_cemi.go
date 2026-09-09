package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// cemi [araç]    → direkt kur
// cemi all       → hepsini kur (onay sorarak)
// cemi           → listele + bilgi
// cemi update    → güncelle
// cemi update agy → sadece agy güncelle

var cemiRootCmd = &cobra.Command{
	Use:     "cemi [tool]",
	Short:   "Install AI CLI tools",
	Version: version,
	Args:    cobra.ArbitraryArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		OpenSourceNotice()
		checkUpdateNotice()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if yes, _ := cmd.Flags().GetBool("yes"); yes {
			autoYes = true
		}
		PrintBanner(BannerCemi)

		cfg, err := loadGlobalConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		// Argüman yoksa listele
		if len(args) == 0 {
			printToolList(cfg)
			return
		}

		target := strings.ToLower(args[0])

		if target == "all" {
			installAll(cfg)
			return
		}

		if target == "update" {
			// cemi update [araç]
			if len(args) >= 2 {
				updateTool(args[1], cfg)
			} else {
				updateAll(cfg)
			}
			return
		}

		// cemi claude / cemi agy / cemi gpt / cemi cursor + typo toleransı
		if _, ok := KnownTools[target]; !ok {
			suggestion := suggestTool(target)
			if suggestion != "" && askYN(fmt.Sprintf("  '%s' bilinmiyor — '%s' demek istedin mi?",
				target, styleBold.Render(suggestion))) {
				target = suggestion
			} else {
				fmt.Println(styleError.Render("✗ Unknown tool: " + target))
				fmt.Println()
				printToolList(cfg)
				os.Exit(1)
			}
		}

		if err := InstallTool(target, cfg); err != nil {
			fmt.Println(styleError.Render(L("✗ Kurulum başarısız: ", "✗ Installation failed: ") + err.Error()))
			os.Exit(1)
		}

		if err := saveGlobalConfig(cfg); err != nil {
			fmt.Println(styleError.Render("✗ Config kaydedilemedi: " + err.Error()))
			os.Exit(1)
		}
	},
}

func initCemiCmd() {
	cemiRootCmd.Flags().BoolP("yes", "y", false, "auto-accept all prompts")
}

// ─── Yardımcı fonksiyonlar ────────────────────────────────────────────────────

func installAll(cfg *GlobalConfig) {
	// Sıralı liste (map rastgele sıralı)
	order := orderedToolKeys

	for _, key := range order {
		meta, ok := KnownTools[key]
		if !ok {
			continue
		}
		// Hem direkt komut hem shell-install yoksa kurulamaz, atla.
		if meta.InstallCmd == nil && pickInstallShell(meta) == "" {
			continue
		}
		if _, installed := cfg.Tools[key]; installed {
			fmt.Printf(L("  %s %-10s zaten kurulu, atlandı\n", "  %s %-10s already installed, skipped\n"),
				styleSuccess.Render("✓"), styleBold.Render(key))
			continue
		}
		if !askYN(fmt.Sprintf("\n  %s kurulsun mu?", styleBold.Render(meta.Name))) {
			fmt.Println(styleDim.Render(L("  atlandı", "  skipped")))
			continue
		}
		if err := InstallTool(key, cfg); err != nil {
			fmt.Println(styleWarn.Render("  ⚠ " + err.Error()))
		}
	}
	saveGlobalConfig(cfg)
	fmt.Println()
	fmt.Println(styleSuccess.Render(L("✓ Tamamlandı", "✓ Done")))
	fmt.Println(styleDim.Render(L("  cem roles  →  kimlerin aktif olduğunu gör", "  cem roles  →  see which tools are active")))
}

func updateTool(name string, cfg *GlobalConfig) {
	if _, ok := cfg.Tools[name]; !ok {
		fmt.Printf(L("  %s kurulu değil — önce: cemi %s\n", "  %s is not installed — first: cemi %s\n"), name, name)
		return
	}
	before := toolVersion(name, cfg)
	fmt.Printf(L("  🔄 %s güncelleniyor...\n", "  🔄 updating %s...\n"), styleBold.Render(name))

	// Önce aracın KENDİ update komutu (claude/codex/agy/cursor-agent update):
	// kurulum betiğini yeniden indirmekten hızlı ve araç kendi güncel mi
	// olduğunu zaten biliyor. Yoksa kurulum komutuna düşülür.
	var out strings.Builder
	sp := StartSpinner(fmt.Sprintf(L("⏳ %s güncelleniyor", "⏳ updating %s"), name))
	native, err := updateToolNative(name, cfg, &out)
	sp.Stop()

	if !native {
		if err := InstallTool(name, cfg); err != nil {
			fmt.Println(styleWarn.Render("  ⚠ " + err.Error()))
			return
		}
	} else if err != nil {
		printTail(out.String(), 10)
		fmt.Println(styleWarn.Render("  ⚠ " + name + L(" güncellenemedi: ", " could not be updated: ") + err.Error()))
		return
	}

	after := toolVersion(name, cfg)
	if after != "" {
		t := cfg.Tools[name]
		t.Version = after
		cfg.Tools[name] = t
	}
	switch {
	case before != "" && after != "" && before != after:
		fmt.Printf("  %s %s → %s\n", styleSuccess.Render("✓"),
			styleDim.Render(before), styleBold.Render(after))
	case after != "":
		fmt.Printf(L("  %s %s zaten güncel (%s)\n", "  %s %s already up to date (%s)\n"), styleSuccess.Render("✓"),
			styleBold.Render(name), styleDim.Render(after))
	default:
		fmt.Printf(L("  %s %s güncellendi\n", "  %s %s updated\n"), styleSuccess.Render("✓"), styleBold.Render(name))
	}
	saveGlobalConfig(cfg)
}

func updateAll(cfg *GlobalConfig) {
	if len(cfg.Tools) == 0 {
		fmt.Println(styleDim.Render(L("  Kurulu araç yok.", "  No tools installed.")))
		return
	}
	for key := range cfg.Tools {
		updateTool(key, cfg)
	}
}

func printToolList(cfg *GlobalConfig) {
	installed := len(cfg.Tools)
	available := len(KnownTools)

	fmt.Printf(L("  Kurulu: %s / %d araç\n\n", "  Installed: %s / %d tools\n\n"),
		styleBold.Render(fmt.Sprintf("%d", installed)), available)

	order := orderedToolKeys
	for _, key := range order {
		meta := KnownTools[key]
		if t, ok := cfg.Tools[key]; ok {
			v := t.Version
			if v == "" {
				v = "kurulu"
			}
			fmt.Printf("  %s %-10s %s\n",
				styleSuccess.Render("✓"),
				styleBold.Render(key),
				styleDim.Render(v))
		} else {
			fmt.Printf("  %s %-10s %s\n",
				styleDim.Render("○"),
				key,
				styleDim.Render(meta.Description))
		}
		if meta.Deprecated != "" {
			fmt.Printf("    %s %s\n",
				styleWarn.Render("⚠ deprecated:"),
				styleDim.Render(meta.Deprecated))
		}
	}

	fmt.Println()
	fmt.Println(styleDim.Render("  cemi claude     → Claude kur"))
	fmt.Println(styleDim.Render("  cemi agy        → Agy kur"))
	fmt.Println(styleDim.Render("  cemi all        → hepsini kur"))
	fmt.Println(styleDim.Render(L("  cemi update     → hepsini güncelle", "  cemi update     → update everything")))
	fmt.Println(styleDim.Render(L("  cemi update agy → sadece agy güncelle", "  cemi update agy → update agy only")))
	fmt.Println()
}
