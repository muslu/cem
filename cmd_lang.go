package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// langCmd — arayüz dilini göster / değiştir.
var langCmd = &cobra.Command{
	Use: "lang [tr|en]",
	// NOT: Short/Long paket-init'te değerlendirilir; config o an henüz
	// okunmamıştır. Türkçe karşılıkları applyLang() yazar (lang_apply.go).
	Short: "Show or change the interface language",
	Long: `  cem lang        → show current language
  cem lang tr     → Turkish
  cem lang en     → English

  Priority: CEM_LANG env var > config > system locale (LANG)`,
	Args: cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		rc, err := LoadConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		if len(args) == 0 {
			showLang(rc.Global)
			return
		}

		code := args[0]
		switch code {
		case LangTR, LangEN:
		default:
			fmt.Println(styleError.Render(L("✗ geçersiz dil: ", "✗ invalid language: ") + code))
			fmt.Println(styleDim.Render(L("  geçerli: tr, en", "  valid: tr, en")))
			os.Exit(1)
		}

		rc.Global.Lang = code
		if err := saveGlobalConfig(rc.Global); err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}
		setLang(code)
		fmt.Printf("  %s %s\n", styleSuccess.Render("✓"),
			L("dil: Türkçe", "language: English"))
	},
}

// showLang — aktif dil + nereden geldiği.
func showLang(cfg *GlobalConfig) {
	src := L("sistem dili (LANG)", "system locale (LANG)")
	if cfg.Lang != "" {
		src = "~/.cem/config.yaml"
	}
	if os.Getenv("CEM_LANG") != "" {
		src = "CEM_LANG"
	}
	name := "Türkçe"
	if Lang() == LangEN {
		name = "English"
	}
	fmt.Printf("\n  %s %s  %s\n\n",
		styleBold.Render(L("Dil:", "Language:")),
		styleBold.Render(name), styleDim.Render("("+src+")"))
	fmt.Println(styleDim.Render("  cem lang tr    → Türkçe"))
	fmt.Println(styleDim.Render("  cem lang en    → English"))
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(langCmd)
}
