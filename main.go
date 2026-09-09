package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LDFLAGS ile enjekte edilir: -X main.version=$(git describe --tags --always)
var version = "dev"

func main() {
	// Dil, cobra Execute'dan ÖNCE belirlenmeli: help/menü metinleri
	// paket-init'te sabitlendiği için applyLang() onları yeniden yazar.
	preloadLang()
	applyLang()

	bin := filepath.Base(os.Args[0])
	// Windows uzantısını at: cem.exe → cem
	bin = strings.TrimSuffix(bin, ".exe")
	// Geliştirme: `go run .` çıktısında "cem" / "main" olabilir
	bin = strings.ToLower(bin)

	switch bin {
	case "cemi":
		initCemiCmd()
		if err := cemiRootCmd.Execute(); err != nil {
			os.Exit(1)
		}
	case "cemir":
		initCemirCmd()
		if err := cemirRootCmd.Execute(); err != nil {
			os.Exit(1)
		}
	default:
		// cem ve diğer adlar
		init_uninstall()
		if err := rootCmd.Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
