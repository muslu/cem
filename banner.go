package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Banner türleri
type BannerKind int

const (
	BannerCem BannerKind = iota
	BannerCemi
	BannerCemir
)

var (
	colorAccent  = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	colorMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	colorURL     = lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Bold(true)
	colorTagline = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	colorGreen   = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	colorYellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	colorRed     = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
)

const asciiCem = `
   ██████╗███████╗███╗   ███╗
  ██╔════╝██╔════╝████╗ ████║
  ██║     █████╗  ██╔████╔██║
  ██║     ██╔══╝  ██║╚██╔╝██║
  ╚██████╗███████╗██║  ╚═╝ ██║
   ╚═════╝╚══════╝╚═╝      ╚═╝`

func PrintBanner(kind BannerKind) {
	logo := colorAccent.Render(asciiCem)

	var subtitle, badge, tip string

	switch kind {
	case BannerCem:
		badge = colorAccent.Render("  ⚡ Compose · Execute · Multiplex")
		subtitle = colorTagline.Render("  One command, many AIs.")
		tip = colorMuted.Render("  cem -p \"task\"  →  pair mode")

	case BannerCemi:
		badge = colorGreen.Render("  📦 AI Tool Installer")
		subtitle = colorTagline.Render("  Install and update AI CLI tools")
		tip = colorMuted.Render("  cemi all -y  →  install everything (no prompts)")

	case BannerCemir:
		badge = colorRed.Render("  🗑  AI Tool Remover")
		subtitle = colorTagline.Render("  Uninstall AI CLI tools")
		tip = colorMuted.Render("  cemir all -y  →  remove everything (no prompts)")
	}

	url := colorURL.Render("  cem.pw")
	sep := colorMuted.Render("  " + repeat("─", 38))

	fmt.Println(logo)
	fmt.Println()
	fmt.Println(badge + colorMuted.Render("  ·  ") + url)
	fmt.Println(subtitle)
	fmt.Println(sep)
	fmt.Println(tip)
	fmt.Println()
}

// OpenSourceNotice — tüm cem/cemi/cemir başlangıçlarında basılır; kullanıcı
// kaynağın açık olduğunu ve hangi sürümü kullandığını görür.
func OpenSourceNotice() {
	fmt.Println(colorMuted.Render(
		"  ⓘ Open source · https://github.com/muslu/cem  ·  " + version))
}

// ShowConfigSource — hangi config kullanıldığını göster
func ShowConfigSource(rc *ResolvedConfig) {
	if rc.HasProjectConfig() {
		fmt.Println(colorYellow.Render("  📁 Project config: ") +
			colorTagline.Render(".cem.yaml") +
			colorMuted.Render("  (overrides global)"))
		fmt.Println()
	}
	// Global config normal durum — ayrıca bildirmeye gerek yok; sadece proje
	// config'i devredeyken uyarmak bilgiyi anlamlı kılıyor.
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
