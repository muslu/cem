package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// /cem slash command markdown — `cem install-slash` ile target IDE'ye kopyalanır.
//
//go:embed assets/commands/cem.md
var slashCemMarkdown string

// slashTarget — destekleyen her CLI için hedef commands/ dizini.
type slashTarget struct {
	tool string // toolKey
	dir  func(home string) string
}

var slashTargets = []slashTarget{
	{"claude", func(h string) string { return filepath.Join(h, ".claude", "commands") }},
	{"agy", func(h string) string { return filepath.Join(h, ".antigravity", "commands") }},
}

var slashCmd = &cobra.Command{
	Use:     "install-slash",
	Aliases: []string{"slash"},
	Short:   "Install /cem slash command into supported AI CLIs",
	Long: `  cem install-slash                 → tüm desteklenen CLI'lara kur (claude + agy)
  cem install-slash --for claude    → sadece Claude Code
  cem install-slash --for agy       → sadece Antigravity
  cem install-slash --here          → .claude/commands/cem.md (proje, sadece claude)
  cem install-slash --target X      → özel hedef dizin

Slash command kurulduktan sonra session'da '/cem <görev>' yazarak cem pair
flow'unu (thinker → writer) tetikleyebilirsin. Aktif CLI sadece komutu
çalıştırıp çıktıyı yansıtır — pahalı analiz/yazımı diğer AI'lar yapar.

Antigravity slash command konumu (~/.antigravity/commands/) henüz resmi belge
yayımlanmadan tahmin edilmiş; agy ileriki sürümde başka bir dizin kullanırsa
'cem install-slash --target' ile manuel yol verebilirsin.`,
	Run: func(cmd *cobra.Command, args []string) {
		here, _ := cmd.Flags().GetBool("here")
		custom, _ := cmd.Flags().GetString("target")
		forTool, _ := cmd.Flags().GetString("for")

		// --here ve --target tek bir konuma kurar; --for olmadan tüm CLI'lara.
		switch {
		case custom != "":
			writeSlash(custom)
		case here:
			writeSlash(filepath.Join(".claude", "commands"))
		case forTool != "":
			home, _ := os.UserHomeDir()
			t := findTarget(forTool)
			if t == nil {
				fmt.Println(styleError.Render(L("✗ desteklenen --for değerleri: claude, agy", "✗ supported --for values: claude, agy")))
				os.Exit(1)
			}
			writeSlash(t.dir(home))
		default:
			home, _ := os.UserHomeDir()
			installed := 0
			for _, t := range slashTargets {
				if writeSlash(t.dir(home)) {
					installed++
				}
			}
			if installed == 0 {
				fmt.Println(styleWarn.Render(L("  ⚠ hiçbir hedefe kurulamadı", "  ⚠ could not install to any target")))
				os.Exit(1)
			}
		}
		printSlashUsage()
	},
}

func findTarget(tool string) *slashTarget {
	for i := range slashTargets {
		if slashTargets[i].tool == tool {
			return &slashTargets[i]
		}
	}
	return nil
}

func writeSlash(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Println(styleWarn.Render("  ⚠ " + dir + ": " + err.Error()))
		return false
	}
	target := filepath.Join(dir, "cem.md")
	if err := os.WriteFile(target, []byte(slashCemMarkdown), 0o644); err != nil {
		fmt.Println(styleWarn.Render("  ⚠ " + target + ": " + err.Error()))
		return false
	}
	fmt.Println(styleSuccess.Render("  ✓ kuruldu: " + target))
	return true
}

func printSlashUsage() {
	fmt.Println()
	fmt.Println(styleBold.Render(L("  Kullanım — herhangi bir CLI session'ında:", "  Usage — in any CLI session:")))
	fmt.Println(styleDim.Render(`    /cem fibonacci için python kodu yaz`))
	fmt.Println(styleDim.Render(`    /cem TBMM kaç yılında kuruldu`))
	fmt.Println()
	fmt.Println(styleBold.Render("  VS Code / Cursor:"))
	fmt.Println(styleDim.Render(L("    Native slash command yok — cem-vscode extension kısayolları kullan:", "    No native slash command — use the cem-vscode extension shortcuts:")))
	fmt.Println(styleDim.Render("      Ctrl+Alt+I  cem: think on selection"))
	fmt.Println(styleDim.Render("      Ctrl+Alt+W  cem: write on selection"))
	fmt.Println(styleDim.Render("      Ctrl+Alt+P  cem: pair on selection"))
	fmt.Println(styleDim.Render("    Extension: https://github.com/muslu/cem/releases/latest/download/cem-vscode.vsix"))
	fmt.Println()
	fmt.Println(styleBold.Render("  Continue.dev:"))
	fmt.Println(styleDim.Render(L("    ~/.continue/config.json içine customCommands ekle:", "    Add customCommands to ~/.continue/config.json:")))
	fmt.Println(styleDim.Render(`      { "name": "cem", "description": "cem pair", "prompt": "Run cem -p {{{ input }}} and return output" }`))
	fmt.Println()
	fmt.Println(styleDim.Render("  Roller/modeller: cem setup veya IDE plugin Settings → Tools → cem"))
}

func init() {
	slashCmd.Flags().Bool("here", false, "install into .claude/commands/ in the project root (claude only, this project)")
	slashCmd.Flags().String("target", "", "custom target directory (override)")
	slashCmd.Flags().String("for", "", "sadece bir CLI'a kur: claude veya agy")
	rootCmd.AddCommand(slashCmd)
}
