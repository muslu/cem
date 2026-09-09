package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDosyaGeriYazma — yazan rolün önbelleği ancak ürettiği dosyaları da geri
// getirirse anlamlı: aksi hâlde "dosya oluşturuldu" cevabı basılır ama ortada
// dosya olmaz.
func TestDosyaGeriYazma(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"onbellek.py":   "class TTLCache:\n    pass\n",
		"alt/yardim.py": "def yardim():\n    return 1\n",
	}

	written, same, conflict := restoreFiles(dir, files)
	if written != 2 || same != 0 || conflict != 0 {
		t.Fatalf("ilk geri yazma: yazılan=%d aynı=%d çakışan=%d", written, same, conflict)
	}
	for rel, want := range files {
		got, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil || string(got) != want {
			t.Errorf("%s geri yazılmadı: %v", rel, err)
		}
	}

	// İkinci kez: dosyalar zaten aynı, dokunulmamalı.
	written, same, _ = restoreFiles(dir, files)
	if written != 0 || same != 2 {
		t.Errorf("aynı içerik tekrar yazıldı: yazılan=%d aynı=%d", written, same)
	}
}

// TestElleDegisenDosyaKorunur — kullanıcı dosyayı düzenlediyse önbellek onun
// işini silmemeli.
func TestElleDegisenDosyaKorunur(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kod.py")
	elle := "# kullanıcının kendi düzenlemesi\n"
	if err := os.WriteFile(path, []byte(elle), 0o644); err != nil {
		t.Fatal(err)
	}

	written, same, conflict := restoreFiles(dir, map[string]string{"kod.py": "# önbellekten\n"})
	if conflict != 1 || written != 0 || same != 0 {
		t.Fatalf("çakışma bildirilmedi: yazılan=%d aynı=%d çakışan=%d", written, same, conflict)
	}
	got, _ := os.ReadFile(path)
	if string(got) != elle {
		t.Error("kullanıcının düzenlemesi önbellekle EZİLDİ")
	}
}

// TestAnlikGoruntuDegisenleriYakalar — koşu sonrası yalnızca yeni/değişen
// dosyalar saklanmalı.
func TestAnlikGoruntuDegisenleriYakalar(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "eski.py"), []byte("x=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := fileSnapshot(dir)

	if err := os.WriteFile(filepath.Join(dir, "yeni.py"), []byte("y=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := capturedFiles(dir, before)
	if len(got) != 1 || got["yeni.py"] != "y=2\n" {
		t.Errorf("yalnızca yeni dosya beklenirken: %v", got)
	}

	// .git gibi dizinler hiç gezilmemeli.
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if snap := fileSnapshot(dir); len(snap) != 2 {
		t.Errorf(".git gezildi: %v", snap)
	}
}

// TestIkiliDosyaOnbellegeAlinmaz — ikili içerik JSON'a gömülmemeli; böyle bir
// koşu eksik saklanmaktansa hiç saklanmamalı.
func TestIkiliDosyaOnbellegeAlinmaz(t *testing.T) {
	dir := t.TempDir()
	before := fileSnapshot(dir)
	if err := os.WriteFile(filepath.Join(dir, "resim.bin"), []byte{0xff, 0xfe, 0x00, 0x01}, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := capturedFiles(dir, before); got != nil {
		t.Errorf("ikili dosya saklandı: %v", got)
	}
}
