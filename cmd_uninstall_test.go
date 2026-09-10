package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// IDE eklenti dizinleri ürün+sürüm başına ayrı (GoLand2026.2, PyCharm2026.1);
// kullanıcının elle bulması zor olduğu için uninstall bunları listeliyor.
// Regresyon riski: yol şeması yanlışsa liste sessizce boş döner ve eklenti
// diskte kalıp her IDE açılışında "cem bulunamadı" verir.
func TestIdeEklentiDizinleriBulunur(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("yol şeması platforma özgü; test Linux yolunu kuruyor")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	jb := filepath.Join(home, ".local", "share", "JetBrains")
	istenen := []string{
		filepath.Join(jb, "GoLand2026.2", "cem-intellij"),
		filepath.Join(jb, "PyCharm2026.1", "cem-intellij"),
	}
	for _, d := range istenen {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Alakasız eklenti listeye girmemeli.
	if err := os.MkdirAll(filepath.Join(jb, "GoLand2026.2", "some-other-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	// VS Code eklentisi de bulunmalı.
	vsc := filepath.Join(home, ".vscode", "extensions", "cempw.cem-0.1.0")
	if err := os.MkdirAll(vsc, 0o755); err != nil {
		t.Fatal(err)
	}

	bulunan := ideEklentiDizinleri()
	if len(bulunan) != 3 {
		t.Fatalf("3 dizin beklenir, gelen %d: %v", len(bulunan), bulunan)
	}
	for _, d := range append(istenen, vsc) {
		var var_ bool
		for _, b := range bulunan {
			if b == d {
				var_ = true
			}
		}
		if !var_ {
			t.Errorf("bulunamadı: %s", d)
		}
	}
}

// Kurulu eklenti yoksa liste boş dönmeli (uyarı basılmasın).
func TestIdeEklentiDizinleriBosOlabilir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if d := ideEklentiDizinleri(); len(d) != 0 {
		t.Errorf("boş beklenir: %v", d)
	}
}
