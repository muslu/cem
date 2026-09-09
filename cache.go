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
	"unicode/utf8"
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
	// Files — yazan rolün ürettiği dosyalar (göreli yol → içerik). Cevabı
	// tek başına saklamak yetmiyordu: writer dosya oluşturuyor, önbellekten
	// dönen "dosya oluşturuldu" cümlesi ortada dosya bırakmazdı.
	Files map[string]string `json:"files,omitempty"`
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
//
// Çalışma dizini de anahtara dahil: araçlar bulundukları dizini okuyor ve
// cevap ona göre değişiyor. "add retries to client.go" sorusunun cevabı
// deponun içeriğine bağlı — dizinsiz anahtar, bir projenin cevabını başka
// projede geri verirdi.
func cacheKey(role, toolKey, input string, rc *ResolvedConfig) string {
	wd, _ := os.Getwd()
	h := sha256.New()
	parts := []string{
		"v2", role, toolKey,
		resolveModel(toolKey, rc),
		resolveEffort(toolKey, rc),
		Lang(),
		wd,
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

// cacheAllowed — bu rol için önbellek hiç devrede mi? (okuma/yazma ortak
// koşullar: config'te kapatılmış olabilir, writer opt-in.)
func cacheAllowed(role string, cfg *GlobalConfig) bool {
	if cfg != nil && cfg.CacheDisabled {
		return false
	}
	if role == "writer" {
		// Yazan rol de önbellekli: aynı istek aynı çıktıyı vermeli. Güvenli
		// olmasının sebebi kaydın ÜRETİLEN DOSYALARI da taşıması —
		// önbellekten dönüldüğünde dosyalar geri yazılıyor. Kapatmak için
		// config'e cache_writer: false.
		return cfg == nil || cfg.CacheWriter == nil || *cfg.CacheWriter
	}
	return true
}

// cacheEnabled — önbellekten OKUMA. --no-cache okumayı atlar.
func cacheEnabled(role string, cfg *GlobalConfig) bool {
	if noCache {
		return false
	}
	return cacheAllowed(role, cfg)
}

// cacheWriteEnabled — önbelleğe YAZMA. --no-cache yazmayı KAPATMAZ, tam
// tersine amacı budur: taze cevabı alıp saklananın üzerine yazmak.
//
// Önceden --no-cache yazmayı da kapatıyordu ve sonuç kafa karıştırıcıydı:
// kullanıcı --no-cache ile taze cevabı alıyor, bir sonraki normal çağrıda
// dakikalar önceki eski cevap geri geliyordu (sahada görüldü: --no-cache'ten
// hemen sonra "♻ önbellekten (11m 47s önce)").
func cacheWriteEnabled(role string, cfg *GlobalConfig) bool {
	return cacheAllowed(role, cfg)
}

// Dosya yakalamanın sınırları. Aşan koşu önbelleğe alınmaz: önbellek bir
// yedekleme aracı değil, tekrarlanan aynı isteği ucuzlatma aracı.
const (
	maxCachedFiles     = 40
	maxCachedFileBytes = 256 << 10 // tek dosya
	maxCachedTotal     = 1 << 20   // toplam
)

// skipDirs — anlık görüntüde hiç gezilmeyen dizinler.
var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "__pycache__": true,
	".venv": true, "venv": true, "target": true, "dist": true, "build": true,
	".idea": true, ".mypy_cache": true, ".pytest_cache": true,
}

// fileSnapshot — çalışma dizinindeki dosyaların göreli yol → içerik özeti.
// Hata durumunda kısmi harita döner; anlık görüntü en iyi çaba işidir.
func fileSnapshot(root string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxCachedFileBytes {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		sum := sha256.Sum256(data)
		out[rel] = hex.EncodeToString(sum[:8])
		return nil
	})
	return out
}

// capturedFiles — koşu sonrası yeni veya değişmiş dosyaların İÇERİĞİ.
// Sınırlar aşılırsa nil döner (kayıt dosyasız saklanmaz, bkz. cachePutFiles).
func capturedFiles(root string, before map[string]string) map[string]string {
	after := fileSnapshot(root)
	files := map[string]string{}
	total := 0
	for rel, sum := range after {
		if before[rel] == sum {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil || !utf8.Valid(data) {
			// İkili dosyayı JSON'a gömmek doğru değil; böyle bir koşuyu
			// eksik saklamaktansa hiç saklamamak yeğ.
			return nil
		}
		total += len(data)
		if len(files) >= maxCachedFiles || total > maxCachedTotal {
			return nil
		}
		files[rel] = string(data)
	}
	return files
}

// restoreFiles — önbellekteki dosyaları geri yazar.
//
// Var olan ve İÇERİĞİ FARKLI bir dosyanın üzerine ASLA yazmaz: kullanıcı o
// dosyayı bu arada elle düzenlemiş olabilir ve önbellek onun işini silmemeli.
func restoreFiles(root string, files map[string]string) (written, same, conflict int) {
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if existing, err := os.ReadFile(path); err == nil {
			if string(existing) == content {
				same++
			} else {
				conflict++
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err == nil {
			written++
		}
	}
	return
}

// cacheGetEntry — kaydın tamamı (dosyalar dahil).
func cacheGetEntry(key string, cfg *GlobalConfig) (cacheEntry, time.Duration, bool) {
	dir, err := cacheDir()
	if err != nil {
		return cacheEntry{}, 0, false
	}
	data, err := os.ReadFile(filepath.Join(dir, key+".json"))
	if err != nil {
		return cacheEntry{}, 0, false
	}
	var e cacheEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return cacheEntry{}, 0, false
	}
	age := time.Since(e.Created)
	if age > cacheTTL(cfg) || strings.TrimSpace(e.Output) == "" {
		return cacheEntry{}, 0, false
	}
	return e, age, true
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
	cachePutFiles(key, role, toolKey, input, output, nil, rc)
}

// cachePutFiles — cevabı ve (yazan rolde) üretilen dosyaları saklar.
func cachePutFiles(key, role, toolKey, input, output string, files map[string]string, rc *ResolvedConfig) {
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
		Files:   files,
	}
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, key+".json"), data, 0o600)
}

// cacheDelete — tek bir kaydı siler (artık geçerli olmayan cevaplar için).
func cacheDelete(key string) {
	dir, err := cacheDir()
	if err != nil {
		return
	}
	_ = os.Remove(filepath.Join(dir, key+".json"))
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
