package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Aynı soruyu ikinci kez sormak, aynı modele aynı token'ı ikinci kez ödemek
// demek. cem düşünme/analiz çıktısını diske yazıp aynı girdi + aynı kurulum
// tekrar geldiğinde oradan veriyor.
//
// YAZAN rol varsayılan olarak önbelleklenmez ve bu bilinçli: writer dosya
// oluşturuyor, komut çalıştırıyor. Cevabı önbellekten basmak "dosya
// oluşturuldu" yazıp ortada dosya bırakmamak olurdu. Açmak isteyen
// ~/.cem/config.yaml içine cache_writer: true yazar.

const cacheDirName = "cache"

// defaultCacheTTLHours — bundan eski kayıtlar yok sayılır. Kod ve API'ler
// değişiyor; süresiz önbellek eskimiş cevabı taze gibi gösterir.
const defaultCacheTTLHours = 168 // 7 gün

// cacheEntry — diskteki kayıt.
type cacheEntry struct {
	Created time.Time `json:"created"`
	Role    string    `json:"role"`  // thinker | writer
	Tool    string    `json:"tool"`  // claude, gpt, ...
	Model   string    `json:"model"` // boşsa CLI default
	Effort  string    `json:"effort,omitempty"`
	Input   string    `json:"input"`  // ilk 200 karakter — 'cem cache list' için
	Output  string    `json:"output"` // asıl cevap
}

func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cem", cacheDirName), nil
}

// cacheKey — girdi + kurulumun tamamı. Model veya düşünme seviyesi
// değiştiğinde cevap da değişeceği için anahtar da değişmeli.
func cacheKey(role, toolKey, input string, rc *ResolvedConfig) string {
	h := sha256.New()
	parts := []string{
		"v1", role, toolKey,
		resolveModel(toolKey, rc),
		resolveEffort(toolKey, rc),
		Lang(),
		input,
	}
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// cacheTTL — config'den, yoksa varsayılan.
func cacheTTL(cfg *GlobalConfig) time.Duration {
	hours := defaultCacheTTLHours
	if cfg != nil && cfg.CacheTTLHours > 0 {
		hours = cfg.CacheTTLHours
	}
	return time.Duration(hours) * time.Hour
}

// cacheEnabled — --no-cache bayrağı her şeyi ezer; writer ayrıca opt-in.
func cacheEnabled(role string, cfg *GlobalConfig) bool {
	if noCache {
		return false
	}
	if cfg != nil && cfg.CacheDisabled {
		return false
	}
	if role == "writer" {
		return cfg != nil && cfg.CacheWriter
	}
	return true
}

// cacheGet — geçerli (süresi dolmamış) kayıt varsa çıktıyı ve yaşını döndürür.
func cacheGet(key string, cfg *GlobalConfig) (string, time.Duration, bool) {
	dir, err := cacheDir()
	if err != nil {
		return "", 0, false
	}
	data, err := os.ReadFile(filepath.Join(dir, key+".json"))
	if err != nil {
		return "", 0, false
	}
	var e cacheEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return "", 0, false
	}
	age := time.Since(e.Created)
	if age > cacheTTL(cfg) {
		return "", 0, false
	}
	if strings.TrimSpace(e.Output) == "" {
		return "", 0, false
	}
	return e.Output, age, true
}

// cachePut — sessizce başarısız olur: önbellek yazılamıyorsa iş durmamalı.
func cachePut(key, role, toolKey, input, output string, rc *ResolvedConfig) {
	if strings.TrimSpace(output) == "" {
		return
	}
	dir, err := cacheDir()
	if err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	short := input
	if len(short) > 200 {
		short = short[:200]
	}
	e := cacheEntry{
		Created: time.Now(),
		Role:    role,
		Tool:    toolKey,
		Model:   resolveModel(toolKey, rc),
		Effort:  resolveEffort(toolKey, rc),
		Input:   short,
		Output:  output,
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, key+".json"), data, 0o600)
}

// printCacheHit — önbellekten geldiğini gizlemeyelim: kullanıcı cevabın taze
// olmadığını bilmeli ve nasıl atlayacağını görmeli.
func printCacheHit(age time.Duration) {
	fmt.Println(styleDim.Render(fmt.Sprintf(
		L("  ♻ önbellekten (%s önce) · yeniden çalıştırmak için: --no-cache",
			"  ♻ from cache (%s ago) · re-run it with: --no-cache"),
		formatDuration(age))))
}

// nowSince — cmd_cache.go'nun time'ı ayrıca import etmesine gerek kalmasın.
func nowSince(t time.Time) time.Duration { return time.Since(t) }

// loadCacheEntries — 'cem cache list/clear' için.
func loadCacheEntries() ([]cacheEntry, []string, error) {
	dir, err := cacheDir()
	if err != nil {
		return nil, nil, err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var entries []cacheEntry
	var paths []string
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		p := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var e cacheEntry
		if err := json.Unmarshal(data, &e); err != nil {
			continue
		}
		entries = append(entries, e)
		paths = append(paths, p)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Created.After(entries[j].Created)
	})
	return entries, paths, nil
}
