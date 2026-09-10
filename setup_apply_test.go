package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ApplySetup config'i ~/.cem altına yazıyor: testler kendi HOME'unda çalışır.
func izoleHome(t *testing.T) string {
	t.Helper()
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h) // Windows
	return h
}

func TestApplySetupRolleriKaydeder(t *testing.T) {
	h := izoleHome(t)
	cfg := &GlobalConfig{}
	if err := ApplySetup(cfg, SetupOptions{Thinker: "gpt", Writer: "claude"}); err != nil {
		t.Fatal(err)
	}
	if !cfg.Setup {
		t.Error("setup_done true olmalı — yoksa cem yine kurulum ister")
	}
	if cfg.Roles.Thinker != "gpt" || cfg.Roles.Writer != "claude" {
		t.Errorf("roller: %+v", cfg.Roles)
	}
	if _, err := os.Stat(filepath.Join(h, ".cem", "config.yaml")); err != nil {
		t.Errorf("config dosyası yazılmadı: %v", err)
	}
}

// Bilinmeyen araç kaydedilmemeli; yakın bir ad varsa önerilmeli.
func TestApplySetupBilinmeyenAraciReddeder(t *testing.T) {
	izoleHome(t)
	cfg := &GlobalConfig{}
	err := ApplySetup(cfg, SetupOptions{Thinker: "claudee", Writer: "gpt"})
	if err == nil {
		t.Fatal("bilinmeyen araç kabul edildi")
	}
	if cfg.Setup {
		t.Error("hata durumunda setup_done yazılmamalı")
	}
}

// HTTP araçta model adı tahmin edilmez: verilmezse hata.
func TestApplySetupHTTPAracModelIster(t *testing.T) {
	izoleHome(t)
	cfg := &GlobalConfig{}
	if err := ApplySetup(cfg, SetupOptions{Thinker: "ollama", Writer: "claude"}); err == nil {
		t.Fatal("model verilmeden kabul edildi")
	}

	cfg = &GlobalConfig{}
	err := ApplySetup(cfg, SetupOptions{
		Thinker: "ollama", ModelThinker: "qwen3-coder",
		EndpointThinker: "192.168.1.10:11434", Writer: "claude",
	})
	if err != nil {
		t.Fatal(err)
	}
	ep := cfg.Endpoints["ollama"]
	if ep.BaseURL != "http://192.168.1.10:11434" || ep.Model != "qwen3-coder" {
		t.Errorf("endpoint kaydı: %+v", ep)
	}
}

// Adres verilmezse ToolMeta varsayılanı kullanılır (yerel sunucu).
func TestApplySetupHTTPAracVarsayilanAdres(t *testing.T) {
	izoleHome(t)
	cfg := &GlobalConfig{}
	if err := ApplySetup(cfg, SetupOptions{
		Thinker: "claude", Writer: "lmstudio", ModelWriter: "mistral-7b",
	}); err != nil {
		t.Fatal(err)
	}
	if got := cfg.Endpoints["lmstudio"].BaseURL; got != "http://127.0.0.1:1234" {
		t.Errorf("varsayılan adres: %q", got)
	}
}

// Sadece dil değiştiren çağrı mevcut rolleri korumalı.
func TestApplySetupDilRolleriBozmaz(t *testing.T) {
	izoleHome(t)
	cfg := &GlobalConfig{Roles: Roles{Thinker: "gpt", Writer: "claude"}, Setup: true}
	if err := ApplySetup(cfg, SetupOptions{Lang: "en"}); err != nil {
		t.Fatal(err)
	}
	if cfg.Lang != "en" || cfg.Roles.Thinker != "gpt" || cfg.Roles.Writer != "claude" {
		t.Errorf("dil değişimi rolleri bozdu: %+v", cfg)
	}
	setLang("tr") // testler arası sızmasın

	if err := ApplySetup(cfg, SetupOptions{Lang: "de"}); err == nil {
		t.Error("desteklenmeyen dil kabul edildi")
	}
}

// Rol hiç verilmemiş ve config boşsa hata: sessizce yarım kurulum yazmak,
// sonraki çalıştırmada anlaşılmaz hataya dönüşüyordu.
func TestApplySetupRolsuzHata(t *testing.T) {
	izoleHome(t)
	cfg := &GlobalConfig{}
	if err := ApplySetup(cfg, SetupOptions{Lang: "tr"}); err == nil {
		t.Fatal("rolsüz kurulum kabul edildi")
	}
}

// Geçersiz effort seçimi kaydedilmemeli.
func TestApplySetupGecersizEffort(t *testing.T) {
	izoleHome(t)
	cfg := &GlobalConfig{}
	err := ApplySetup(cfg, SetupOptions{
		Thinker: "gpt", Writer: "claude", EffortThinker: "ultra",
	})
	if err == nil {
		t.Fatal("geçersiz effort kabul edildi")
	}
}
