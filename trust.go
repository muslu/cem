package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AI araçları cem'i çalıştırdığın dizinde çalışır: writer dosya oluşturur,
// komut çalıştırır. Bunu yanlış dizinde fark etmek kolay — ev dizininde
// "binary search yaz" demek $HOME altına src/algorithms/ açar.
//
// Bu yüzden her yeni dizinde bir kez onay isteniyor; onaylanan dizinler
// ~/.cem/config.yaml içinde saklanıyor ve bir daha sorulmuyor.

// isTrustedDir — dizin daha önce onaylandı mı? Onaylanmış bir dizinin ALT
// dizinleri de kapsanır: proje kökünü onaylayan kişi her paket için yeniden
// onay vermek zorunda kalmasın.
func isTrustedDir(dir string, cfg *GlobalConfig) bool {
	if cfg == nil {
		return false
	}
	for _, t := range cfg.TrustedDirs {
		if t == "" {
			continue
		}
		if dir == t || strings.HasPrefix(dir, strings.TrimSuffix(t, string(os.PathSeparator))+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// ensureWorkdirTrusted — gerekiyorsa onay ister. false dönerse çalıştırma
// iptal edilmeli.
//
// Etkileşimli olmayan ortamlarda (IDE eklentisi, pipe, CI) soru sorulamaz:
// orada tek satır uyarı basılıp devam edilir — aksi halde eklenti sessizce
// kilitlenirdi.
func ensureWorkdirTrusted(cfg *GlobalConfig) bool {
	dir, err := os.Getwd()
	if err != nil {
		return true
	}
	if isTrustedDir(dir, cfg) {
		return true
	}

	home, _ := os.UserHomeDir()
	warnHome := home != "" && dir == home

	if !isInteractive() {
		fmt.Println(styleWarn.Render(fmt.Sprintf(
			L("  ⚠ çalışma dizini: %s — oluşan dosyalar buraya yazılır",
				"  ⚠ working directory: %s — files are created here"), dir)))
		return true
	}

	fmt.Println()
	fmt.Println(styleWarn.Render(L("  ⚠ Bu dizinde ilk kez çalıştırıyorsun.",
		"  ⚠ First run in this directory.")))
	fmt.Printf("    %s\n", styleBold.Render(dir))
	fmt.Println(styleDim.Render(L(
		"    AI araçları burada çalışacak: oluşacak dosya ve klasörler",
		"    The AI tools will run here: any files and folders they create")))
	fmt.Println(styleDim.Render(L(
		"    doğrudan bu dizinin altına yazılır.",
		"    are written directly under this directory.")))
	if warnHome {
		fmt.Println(styleWarn.Render(L(
			"    Bu senin ev dizinin — proje dizinine geçmek isteyebilirsin.",
			"    This is your home directory — you may want a project directory instead.")))
	}
	fmt.Println()
	fmt.Print(L("  Bu dizine güveniyor musun? [e/H] ", "  Do you trust this directory? [y/N] "))

	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(resp)) {
	case "e", "evet", "y", "yes":
	default:
		fmt.Println(styleDim.Render(L("  İptal. Doğru dizine geçip tekrar çalıştır.",
			"  Cancelled. Change to the right directory and run again.")))
		return false
	}

	cfg.TrustedDirs = append(cfg.TrustedDirs, filepath.Clean(dir))
	if err := saveGlobalConfig(cfg); err != nil {
		fmt.Println(styleDim.Render(L("  (dizin kaydedilemedi, bir dahaki sefere yine sorulur)",
			"  (could not store the directory; you will be asked again)")))
	}
	fmt.Println()
	return true
}

// isInteractive — stdin'den gerçekten cevap alabilir miyiz? Pipe'la gelen
// girdide soru sormak, cevabı prompt'un kendisinden okumak olurdu.
//
// ModeCharDevice tek başına yetmiyor: /dev/null da bir karakter aygıtı ve
// `cem … < /dev/null` (script, CI, IDE) o kontrolü geçip soruyu soruyor,
// ardından boş cevabı "hayır" sayıp çalıştırmayı iptal ediyordu.
func isInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil || (info.Mode()&os.ModeCharDevice) == 0 {
		return false
	}
	if dn, err := os.Stat(os.DevNull); err == nil && os.SameFile(info, dn) {
		return false
	}
	return true
}
