package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// toolUpdateInterval — otomatik araç güncellemesi arası minimum süre.
const toolUpdateInterval = 24 * time.Hour

// autoUpdateLogName — arka plan güncellemesinin çıktısı buraya yazılır.
const autoUpdateLogName = "auto-update.log"

// toolVersion — aracın kurulu sürümünü çalıştırarak okur. Okunamazsa "".
func toolVersion(toolKey string, cfg *GlobalConfig) string {
	meta, ok := KnownTools[toolKey]
	if !ok || meta.VersionFlag == "" {
		return ""
	}
	bin := toolBinary(toolKey, cfg)
	if bin == "" {
		return ""
	}
	out, err := exec.Command(bin, meta.VersionFlag).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// toolBinary — config'deki mutlak yolu tercih eder, yoksa PATH'da arar.
func toolBinary(toolKey string, cfg *GlobalConfig) string {
	meta := KnownTools[toolKey]
	if t, ok := cfg.Tools[toolKey]; ok && t.Command != "" {
		if _, err := exec.LookPath(t.Command); err == nil {
			return t.Command
		}
	}
	name := toolKey
	if meta.Binary != "" {
		name = meta.Binary
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return fallbackInstallPath(toolKey)
}

// updateToolNative — aracın KENDİ update subcommand'ını çalıştırır
// (claude update / codex update / agy update / cursor-agent update).
// UpdateCmd tanımlı değilse false döner; çağıran kurulum komutuna düşer.
func updateToolNative(toolKey string, cfg *GlobalConfig, out *strings.Builder) (bool, error) {
	meta, ok := KnownTools[toolKey]
	if !ok || len(meta.UpdateCmd) == 0 {
		return false, nil
	}
	bin := toolBinary(toolKey, cfg)
	if bin == "" {
		return false, fmt.Errorf("%s bulunamadı", toolKey)
	}
	cmd := exec.Command(bin, meta.UpdateCmd...)
	cmd.Stdout = out
	cmd.Stderr = out
	return true, cmd.Run()
}

// autoUpdateLogPath — ~/.cem/auto-update.log
func autoUpdateLogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cem", autoUpdateLogName), nil
}

// autoUpdateEnabled — config'de kapatılmadıysa açık (varsayılan: açık).
func autoUpdateEnabled(cfg *GlobalConfig) bool {
	return cfg.AutoUpdateTools == nil || *cfg.AutoUpdateTools
}

// maybeAutoUpdateTools — kurulu AI CLI'larını günde bir kez arka planda
// günceller. Araçların kendi 'update' komutu zaten "güncel" durumunu kendisi
// tespit ettiği için ayrıca sürüm sorgusu yapmıyoruz.
//
// Güncelleme cem'in kendi süreciyle bağlı DEĞİL: cem çıktıktan sonra da
// devam eder, çıktısı ~/.cem/auto-update.log'a yazılır. Böylece kullanıcının
// asıl komutu (AI çağrısı) hiç beklemez.
func maybeAutoUpdateTools() {
	cfg, err := loadGlobalConfig()
	if err != nil || cfg == nil || !cfg.Setup {
		return
	}
	if !autoUpdateEnabled(cfg) || len(cfg.Tools) == 0 {
		return
	}
	if time.Since(cfg.ToolsLastUpdate) < toolUpdateInterval {
		return
	}

	targets := make([]string, 0, len(cfg.Tools))
	for _, key := range orderedToolKeys {
		if _, ok := cfg.Tools[key]; !ok {
			continue
		}
		if len(KnownTools[key].UpdateCmd) == 0 {
			continue
		}
		if toolBinary(key, cfg) == "" {
			continue
		}
		targets = append(targets, key)
	}
	if len(targets) == 0 {
		return
	}

	// Bir ÖNCEKİ turun arka plan güncellemesi config'e yazamamıştı (detached
	// süreç cem'den bağımsız bitiyor). Sürümleri burada tazeliyoruz ki
	// 'cemi' listesi ve 'cem doctor' eskimiş numara göstermesin.
	for _, key := range targets {
		v := toolVersion(key, cfg)
		if v == "" {
			continue
		}
		if t := cfg.Tools[key]; t.Version != v {
			t.Version = v
			cfg.Tools[key] = t
		}
	}

	// Zaman damgasını ÖNCE yaz: güncelleme başarısız olsa bile her çalıştırmada
	// yeniden denenip kullanıcıyı yavaşlatmasın.
	cfg.ToolsLastUpdate = time.Now()
	if err := saveGlobalConfig(cfg); err != nil {
		return
	}

	logPath, err := autoUpdateLogPath()
	if err != nil {
		return
	}
	started := []string{}
	for _, key := range targets {
		if spawnDetachedUpdate(key, cfg, logPath) {
			started = append(started, key)
		}
	}
	if len(started) == 0 {
		return
	}
	fmt.Printf("  %s %s · log: %s\n",
		styleDim.Render("🔄 araç güncellemesi arka planda:"),
		styleDim.Render(strings.Join(started, ", ")),
		styleDim.Render("~/.cem/"+autoUpdateLogName))
}

// spawnDetachedUpdate — tek bir aracın update komutunu cem'den bağımsız
// çalışacak şekilde başlatır. true = başlatıldı.
func spawnDetachedUpdate(toolKey string, cfg *GlobalConfig, logPath string) bool {
	meta := KnownTools[toolKey]
	bin := toolBinary(toolKey, cfg)
	if bin == "" || len(meta.UpdateCmd) == 0 {
		return false
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return false
	}
	defer f.Close()
	fmt.Fprintf(f, "\n=== %s · %s %s · %s\n",
		time.Now().Format(time.RFC3339), bin,
		strings.Join(meta.UpdateCmd, " "), toolVersion(toolKey, cfg))

	cmd := exec.Command(bin, meta.UpdateCmd...)
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.Stdin = nil
	detachProcess(cmd)
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(f, "başlatılamadı: %v\n", err)
		return false
	}
	// Süreci sahiplenme: cem çıkarken beklemesin.
	_ = cmd.Process.Release()
	return true
}
