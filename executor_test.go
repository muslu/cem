package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// codexNoiseLine — gerçek olay (2026-09-09): codex "failed to refresh available
// models" logunu 100 KB'lık model JSON'uyla stderr'e basıyor. Gövdenin içindeki
// guardian prompt metninde "authentication failed" geçiyor. Kırpma olmadan cem
// bunu auth hatası sanıp gerçek hatayı ("model not supported") gizliyordu.
const codexNoiseLine = `2026-09-09T09:36:38.520101Z ERROR codex_models_manager::manager: ` +
	`failed to refresh available models: stream disconnected before completion; ` +
	`body: {"models":[{"slug":"gpt-5.5","guardian":"...perform an action after ` +
	`normal authentication failed as ` + "`high`" + ` risk. unauthorized 429 quota..."}]}`

func TestSanitizeStderrGovdeyiKirpar(t *testing.T) {
	got := sanitizeStderr(codexNoiseLine)
	if len(got) > noiseLineMax {
		t.Fatalf("kırpılmış satır hâlâ uzun: %d bayt", len(got))
	}
	if want := "body: <kırpıldı>"; !contains(got, want) {
		t.Errorf("gövde kırpma işareti yok:\n%s", got)
	}
}

func TestGovdeIcindekiKelimelerImzaSaymaz(t *testing.T) {
	if looksLikeAuthFailure(codexNoiseLine) {
		t.Error("JSON gövdesindeki 'authentication failed' auth hatası sayıldı")
	}
	if looksLikeRateLimit(codexNoiseLine) {
		t.Error("JSON gövdesindeki '429/quota' rate limit sayıldı — boşuna key rotasyonu")
	}
}

func TestGercekImzalarHalaYakalanir(t *testing.T) {
	cases := []struct {
		name string
		in   string
		fn   func(string) bool
	}{
		{"auth-401", "ERROR: 401 Unauthorized", looksLikeAuthFailure},
		{"auth-login", "Please run /login to continue", looksLikeAuthFailure},
		{"rate-429", "ERROR: 429 too many requests", looksLikeRateLimit},
		{"rate-quota", "You exceeded your current quota", looksLikeRateLimit},
	}
	for _, c := range cases {
		if !c.fn(c.in) {
			t.Errorf("%s: imza yakalanmadı: %q", c.name, c.in)
		}
	}
}

// TestModelDesteklenmiyorAuthdanAyrilir — asıl regresyon: bu mesaj auth
// hatası DEĞİL, model hatası. Kullanıcıyı login akışına yollamamalı.
func TestModelDesteklenmiyorAuthdanAyrilir(t *testing.T) {
	msg := `ERROR: {"type":"error","status":400,"error":{"type":"invalid_request_error",` +
		`"message":"The 'gpt-5-mini' model is not supported when using Codex with a ChatGPT account."}}`
	if !looksLikeModelUnsupported(msg) {
		t.Error("model desteklenmiyor hatası tanınmadı")
	}
	if looksLikeAuthFailure(msg) {
		t.Error("model hatası auth hatası olarak da eşleşiyor — hintAuth yanlış tetiklenir")
	}
}

func TestTailWriterSonByteLariTutar(t *testing.T) {
	w := &tailWriter{n: 8}
	for _, chunk := range []string{"aaaaaa", "bbbbbb", "cccc"} {
		if _, err := w.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := w.String(), "bbcccc"; len(got) > 8 || got[len(got)-4:] != "cccc" {
		t.Errorf("tail yanlış: %q (beklenen son parça %q)", got, want)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestModelHataVaryantlari — sahadan toplanan gerçek mesajlar (codex 0.133 ve
// 0.153 farklı metin üretiyor; ikisi de model hatası, auth hatası değil).
func TestModelHataVaryantlari(t *testing.T) {
	msgs := []string{
		"ERROR: unexpected status 404 Not Found: The model `gpt-5.5` does not exist or you do not have access to it., url: https://chatgpt.com/backend-api/codex/responses",
		`{"message":"The 'gpt-5-mini' model is not supported when using Codex with a ChatGPT account."}`,
		"error: unknown model: sonnet-9",
	}
	for _, m := range msgs {
		if !looksLikeModelUnsupported(m) {
			t.Errorf("model hatası tanınmadı: %.80s", m)
		}
		if looksLikeAuthFailure(m) {
			t.Errorf("model hatası auth sanıldı: %.80s", m)
		}
	}
}

// TestDedupeTrailingEcho — codex exec final mesajı iki kez basıyor; writer'a
// iki kopya gitmemeli (ölçüldü 2026-09-09: pair çıktısında blok aynen 2 kez).
func TestDedupeTrailingEcho(t *testing.T) {
	body := "```go\nfunc UniqueStrings(items []string) []string {\n\tseen := make(map[string]struct{}, len(items))\n\tresult := make([]string, 0, len(items))\n\tfor _, item := range items {\n\t\tif _, ok := seen[item]; ok {\n\t\t\tcontinue\n\t\t}\n\t\tseen[item] = struct{}{}\n\t\tresult = append(result, item)\n\t}\n\treturn result\n}\n```\nİlk görülen elemanı korur; sıralamayı değiştirmez."
	echoed := body + "\ntokens used\n6.092\n" + body

	got := dedupeTrailingEcho(echoed)
	if len(got) >= len(echoed) {
		t.Fatalf("kırpılmadı: %d → %d bayt", len(echoed), len(got))
	}
	if contains(got, "tokens used") == false {
		t.Log("not: 'tokens used' ayracı da kırpıldı, sorun değil")
	}
	// Gövde bir kez kalmalı, iki kez değil.
	if occurrences(got, "func UniqueStrings") != 1 {
		t.Errorf("gövde %d kez kaldı, 1 bekleniyordu", occurrences(got, "func UniqueStrings"))
	}
}

// TestDedupeMesruTekrariKorur — eşiğin altındaki tekrarlar kırpılmamalı.
func TestDedupeMesruTekrariKorur(t *testing.T) {
	s := "kısa uyarı satırı\nbaşka içerik\nkısa uyarı satırı"
	if got := dedupeTrailingEcho(s); got != s {
		t.Errorf("kısa tekrar yanlışlıkla kırpıldı:\n%q", got)
	}
}

func occurrences(h, n string) int {
	c, i := 0, 0
	for i+len(n) <= len(h) {
		if h[i:i+len(n)] == n {
			c++
			i += len(n)
		} else {
			i++
		}
	}
	return c
}

// ─── Düşünme seviyesi (effort) ───────────────────────────────────────────

func rcWith(tools map[string]InstalledTool, proj *ProjectConfig) *ResolvedConfig {
	return &ResolvedConfig{Global: &GlobalConfig{Tools: tools}, Project: proj}
}

// TestBuildArgsClaudeEffortPromptuYutmaz — claude'da -p PROMPT pozisyonel;
// --effort ile --model, -p'den ÖNCE gelmeli yoksa prompt kaybolur.
func TestBuildArgsClaudeEffortPromptuYutmaz(t *testing.T) {
	// Hızlı mod kapalı: sadece model + effort + -p + prompt.
	off := false
	rc := rcWith(map[string]InstalledTool{
		"claude": {Model: "sonnet", Effort: "xhigh", Fast: &off},
	}, nil)
	got := buildArgs(KnownTools["claude"], "claude", rc, "merhaba")
	want := []string{"--model", "sonnet", "--effort", "xhigh", "-p", "merhaba"}
	if len(got) != len(want) {
		t.Fatalf("args = %q\nwant %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args = %q\nwant %q", got, want)
		}
	}

	// Hızlı mod açıkken de prompt EN SONDA kalmalı: -p argümanını yutarsa
	// istek yolda kaybolur.
	rcFast := rcWith(map[string]InstalledTool{
		"claude": {Model: "sonnet", Effort: "xhigh"},
	}, nil)
	gotFast := buildArgs(KnownTools["claude"], "claude", rcFast, "merhaba")
	if gotFast[len(gotFast)-1] != "merhaba" {
		t.Errorf("prompt son argüman değil: %q", gotFast)
	}
	if i := indexOf(gotFast, "-p"); i != len(gotFast)-2 {
		t.Errorf("-p prompt'tan hemen önce değil: %q", gotFast)
	}
}

// TestBuildArgsCodexEffortConfigAnahtari — codex'te effort flag değil,
// -c model_reasoning_effort=X biçiminde config override.
func TestBuildArgsCodexEffortConfigAnahtari(t *testing.T) {
	rc := rcWith(map[string]InstalledTool{
		"gpt": {Model: "gpt-5.5", Effort: "high"},
	}, nil)
	got := buildArgs(KnownTools["gpt"], "gpt", rc, "görev")
	joined := ""
	for _, a := range got {
		joined += a + " "
	}
	for _, want := range []string{"exec", "--model gpt-5.5", "-c model_reasoning_effort=high", "görev"} {
		if !contains(joined, want) {
			t.Errorf("args %q içinde %q yok", joined, want)
		}
	}
	if got[len(got)-1] != "görev" {
		t.Errorf("prompt son argüman değil: %q", got)
	}
}

// TestEffortDesteklemeyenAracaEklenmez — agy/cursor'da EffortArgs boş.
func TestEffortDesteklemeyenAracaEklenmez(t *testing.T) {
	rc := rcWith(map[string]InstalledTool{"agy": {Effort: "high"}}, nil)
	if e := resolveEffort("agy", rc); e != "" {
		t.Errorf("agy effort döndürdü: %q", e)
	}
	for _, a := range buildArgs(KnownTools["agy"], "agy", rc, "x") {
		if a == "high" || contains(a, "effort") {
			t.Errorf("agy args'ına effort sızdı: %q", a)
		}
	}
}

// TestProjeEffortGlobaliEzer — .cem.yaml > global config.
func TestProjeEffortGlobaliEzer(t *testing.T) {
	rc := rcWith(
		map[string]InstalledTool{"gpt": {Effort: "low"}},
		&ProjectConfig{Efforts: map[string]string{"gpt": "xhigh"}},
	)
	if got := resolveEffort("gpt", rc); got != "xhigh" {
		t.Errorf("proje override çalışmadı: %q", got)
	}
}

func TestDescribeToolRun(t *testing.T) {
	// Hızlı mod varsayılan: başlıkta belirtilmez.
	cases := []struct{ model, effort, want string }{
		{"sonnet", "high", "sonnet · high"},
		{"sonnet", "", "sonnet"},
		{"", "high", "default · high"},
		{"", "", "default"},
	}
	for _, c := range cases {
		rc := rcWith(map[string]InstalledTool{"claude": {Model: c.model, Effort: c.effort}}, nil)
		if got := describeToolRun("claude", rc); got != c.want {
			t.Errorf("model=%q effort=%q → %q, beklenen %q", c.model, c.effort, got, c.want)
		}
	}

	// Kapatıldığında görünür olmalı: kullanıcı yavaşlığın sebebini bilsin.
	off := false
	rc := rcWith(map[string]InstalledTool{"claude": {Model: "sonnet", Fast: &off}}, nil)
	if got := describeToolRun("claude", rc); !contains(got, "settings") && !contains(got, "ayarlar") {
		t.Errorf("hızlı mod kapalıyken başlıkta belirtilmedi: %q", got)
	}
}

// TestTumAraclarinUpdateKomutuVar — otomatik güncelleme bunlara dayanıyor.
func TestTumAraclarinUpdateKomutuVar(t *testing.T) {
	for _, key := range orderedToolKeys {
		if len(KnownTools[key].UpdateCmd) == 0 {
			t.Errorf("%s için UpdateCmd tanımsız — otomatik güncelleme atlanır", key)
		}
	}
}

// TestEffortSeviyeleriArtanSirada — wizard listesi ve effortHint bu sıraya güvenir.
func TestEffortSeviyeleriArtanSirada(t *testing.T) {
	rank := map[string]int{"none": 0, "minimal": 1, "low": 2, "medium": 3, "high": 4, "xhigh": 5, "max": 6}
	for key, meta := range KnownTools {
		prev := -1
		for _, e := range meta.Efforts {
			r, ok := rank[e]
			if !ok {
				t.Errorf("%s: bilinmeyen seviye %q", key, e)
				continue
			}
			if r <= prev {
				t.Errorf("%s: seviyeler artan sırada değil (%q)", key, e)
			}
			prev = r
		}
	}
}

// TestKodIstegiSozlugu — pair modunda writer'ın atlanıp atlanmayacağına bu
// karar veriyor; "yaz" geçmeyen kod istekleri de yakalanmalı.
func TestKodIstegiSozlugu(t *testing.T) {
	kodİstekleri := []string{
		"bu dosyayı optimize et",
		"şu fonksiyonu refactor et",
		"testleri ekle",
		"kullanılmayan importları sil",
		"bu tipi Go'ya çevir",
		"add a retry to the client",
		"rename the handler",
	}
	for _, s := range kodİstekleri {
		if !looksLikeCodeRequest(s) {
			t.Errorf("kod isteği sayılmadı, writer atlanır: %q", s)
		}
	}
	kodOlmayan := []string{
		"muslu yüksektepe kimdir",
		"bu mimarinin artıları neler",
	}
	for _, s := range kodOlmayan {
		if looksLikeCodeRequest(s) {
			t.Errorf("kod isteği sanıldı: %q", s)
		}
	}
}

// ─── Önbellek ────────────────────────────────────────────────────────────

// TestCacheKeyKurulumaDuyarli — model/effort değişince cevap da değişir;
// anahtar aynı kalırsa eski cevap yeni kurulumda geri gelir.
func TestCacheKeyKurulumaDuyarli(t *testing.T) {
	base := rcWith(map[string]InstalledTool{"gpt": {Model: "gpt-5.5", Effort: "high"}}, nil)
	k1 := cacheKey("thinker", "gpt", "aynı soru", base)

	cases := map[string]*ResolvedConfig{
		"model değişti":  rcWith(map[string]InstalledTool{"gpt": {Model: "gpt-5", Effort: "high"}}, nil),
		"effort değişti": rcWith(map[string]InstalledTool{"gpt": {Model: "gpt-5.5", Effort: "low"}}, nil),
	}
	for name, rc := range cases {
		if cacheKey("thinker", "gpt", "aynı soru", rc) == k1 {
			t.Errorf("%s ama anahtar aynı kaldı", name)
		}
	}
	if cacheKey("thinker", "gpt", "başka soru", base) == k1 {
		t.Error("farklı girdi aynı anahtarı üretti")
	}
	if cacheKey("writer", "gpt", "aynı soru", base) == k1 {
		t.Error("farklı rol aynı anahtarı üretti")
	}
	if cacheKey("thinker", "gpt", "aynı soru", base) != k1 {
		t.Error("aynı girdi + aynı kurulum farklı anahtar üretti")
	}
}

// TestCacheWriterVarsayilanKapali — writer dosya oluşturuyor; önbellekten
// dönen bir cevap ortada dosya bırakmaz.
func TestCacheWriterVarsayilanKapali(t *testing.T) {
	noCache = false
	defer func() { noCache = false }()

	cfg := &GlobalConfig{}
	if cacheEnabled("writer", cfg) {
		t.Error("writer önbelleği varsayılan olarak açık — dosyalar yazılmadan 'yazıldı' denir")
	}
	if !cacheEnabled("thinker", cfg) {
		t.Error("thinker önbelleği varsayılan olarak kapalı")
	}

	cfg.CacheWriter = true
	if !cacheEnabled("writer", cfg) {
		t.Error("cache_writer: true dikkate alınmadı")
	}

	cfg.CacheDisabled = true
	if cacheEnabled("thinker", cfg) || cacheEnabled("writer", cfg) {
		t.Error("cache_disabled: true dikkate alınmadı")
	}

	noCache = true
	if cacheEnabled("thinker", &GlobalConfig{}) {
		t.Error("--no-cache bayrağı her şeyi ezmeli")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{8900 * time.Millisecond, "8.9s"},
		{99 * time.Second, "1m 39s"},
		{time.Hour + 4*time.Second, "60m 04s"},
	}
	for _, c := range cases {
		if got := formatDuration(c.d); got != c.want {
			t.Errorf("formatDuration(%v) = %q, beklenen %q", c.d, got, c.want)
		}
	}
}

// TestFastArgsSirasi — hızlı mod argümanları da prompt'tan ÖNCE gelmeli;
// claude'da -p prompt'u yutuyor.
func TestFastArgsSirasi(t *testing.T) {
	rc := rcWith(map[string]InstalledTool{
		"claude": {Model: "sonnet", Effort: "low"},
	}, nil)
	got := buildArgs(KnownTools["claude"], "claude", rc, "görev")
	joined := strings.Join(got, " ")
	for _, want := range []string{"--model sonnet", "--effort low", "--setting-sources", "--permission-mode acceptEdits"} {
		if !contains(joined, want) {
			t.Errorf("args %q içinde %q yok", joined, want)
		}
	}
	if got[len(got)-1] != "görev" {
		t.Errorf("prompt son argüman değil: %q", got)
	}
	if i := indexOf(got, "-p"); i < 0 || i != len(got)-2 {
		t.Errorf("-p prompt'tan hemen önce değil: %q", got)
	}
}

// TestFastVarsayilanAcik — ayarlanmamışsa hızlı mod açık olmalı (ölçülen
// 124s → 8s), açıkça kapatıldığında argümanlar hiç eklenmemeli.
func TestFastVarsayilanAcik(t *testing.T) {
	rc := rcWith(map[string]InstalledTool{"claude": {Model: "sonnet"}}, nil)
	if !resolveFast("claude", rc) {
		t.Error("hızlı mod varsayılan olarak kapalı")
	}

	off := false
	rcOff := rcWith(map[string]InstalledTool{"claude": {Model: "sonnet", Fast: &off}}, nil)
	if resolveFast("claude", rcOff) {
		t.Error("cem fast claude off dikkate alınmadı")
	}
	for _, a := range buildArgs(KnownTools["claude"], "claude", rcOff, "x") {
		if a == "--setting-sources" || a == "--permission-mode" {
			t.Errorf("kapalıyken hızlı mod argümanı sızdı: %q", a)
		}
	}

	// Araç desteklemiyorsa (FastArgs boş) hiçbir zaman açılmaz.
	on := true
	rcGpt := rcWith(map[string]InstalledTool{"gpt": {Fast: &on}}, nil)
	if resolveFast("gpt", rcGpt) {
		t.Error("FastArgs tanımsız araçta hızlı mod açıldı")
	}
}

// TestRolVarsayilanlari — deneyimsiz kullanıcı hiçbir şey ayarlamasa bile
// thinker derin, writer ucuz çalışmalı.
func TestRolVarsayilanlari(t *testing.T) {
	if got := defaultEffortForRole("claude", "thinker"); got != "high" {
		t.Errorf("thinker varsayılanı %q, beklenen high", got)
	}
	if got := defaultEffortForRole("claude", "writer"); got != "low" {
		t.Errorf("writer varsayılanı %q, beklenen low", got)
	}
	if got := defaultEffortForRole("agy", "thinker"); got != "" {
		t.Errorf("seviye desteklemeyen araç için %q döndü", got)
	}

	cfg := &GlobalConfig{Tools: map[string]InstalledTool{
		"gpt":    {},
		"claude": {Effort: "xhigh"}, // kullanıcının açık seçimi
	}}
	applyRoleDefaults(cfg, "gpt", "claude")
	if cfg.Tools["gpt"].Effort != "high" {
		t.Errorf("thinker'a varsayılan yazılmadı: %q", cfg.Tools["gpt"].Effort)
	}
	if cfg.Tools["claude"].Effort != "xhigh" {
		t.Error("kullanıcının açık seçimi ezildi")
	}
}

// TestProjeFastGlobaliEzer — .cem.yaml global'i ezmeli (false dahil).
func TestProjeFastGlobaliEzer(t *testing.T) {
	on := true
	rc := rcWith(
		map[string]InstalledTool{"claude": {Fast: &on}},
		&ProjectConfig{Fast: map[string]bool{"claude": false}},
	)
	if resolveFast("claude", rc) {
		t.Error("proje 'false' override'ı global 'true' tarafından eziliyor")
	}
}

func indexOf(xs []string, want string) int {
	for i, x := range xs {
		if x == want {
			return i
		}
	}
	return -1
}

// TestNoCacheYazmayiKapatmaz — --no-cache "eski cevabı kullanma, tazesini al
// ve KAYDET" demek. Yazmayı da kapatınca kullanıcı taze cevabı görüyor ama
// bir sonraki çağrıda eski kayıt geri geliyordu (sahada görüldü).
func TestNoCacheYazmayiKapatmaz(t *testing.T) {
	defer func() { noCache = false }()
	cfg := &GlobalConfig{}

	noCache = true
	if cacheEnabled("thinker", cfg) {
		t.Error("--no-cache okumayı kapatmadı")
	}
	if !cacheWriteEnabled("thinker", cfg) {
		t.Error("--no-cache yazmayı da kapattı — taze cevap saklanmıyor")
	}

	noCache = false
	cfg.CacheDisabled = true
	if cacheEnabled("thinker", cfg) || cacheWriteEnabled("thinker", cfg) {
		t.Error("cache_disabled: true hem okumayı hem yazmayı kapatmalı")
	}
}

// TestBilgiTalebiOnbellegeYazilmaz — "dosya bulunamadı, şunu paylaşın"
// cevabı saklanırsa, kullanıcı dosyayı ekledikten sonra bile günlerce aynı
// cevabı alır. Sahada görüldü: "add retries to client.go" cevabı önbellekten
// 11m 48s sonra tekrar geldi.
func TestBilgiTalebiTespiti(t *testing.T) {
	talepler := []string{
		"Depoda `client.go` bulunmuyor; dosyayı paylaşın.",
		"`client.go` dosyasını veya ilgili depo yolunu paylaşın",
		"I could not find client.go in the repo — could you share it?",
		"Which file did you mean?",
	}
	for _, s := range talepler {
		if !looksLikeClarification(s) {
			t.Errorf("bilgi talebi tanınmadı: %.60s", s)
		}
	}

	planlar := []string{
		"- Dosya: client.go\n- Fonksiyon: func withRetry() error\n- Kenar durumlar: iptal",
		"- File: retry.go\n- Approach: exponential backoff",
	}
	for _, s := range planlar {
		if looksLikeClarification(s) {
			t.Errorf("plan bilgi talebi sanıldı: %.60s", s)
		}
	}
}

// TestCacheKeyDizineDuyarli — aynı soru farklı depoda farklı cevap gerektirir.
func TestCacheKeyDizineDuyarli(t *testing.T) {
	rc := rcWith(map[string]InstalledTool{"gpt": {Model: "gpt-5.5"}}, nil)
	dir1, err := os.Getwd()
	if err != nil {
		t.Skip("çalışma dizini alınamadı")
	}
	k1 := cacheKey("thinker", "gpt", "add retries to client.go", rc)

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Skip("dizin değiştirilemedi")
	}
	defer os.Chdir(dir1)

	if cacheKey("thinker", "gpt", "add retries to client.go", rc) == k1 {
		t.Error("farklı dizinde aynı anahtar üretildi — bir projenin cevabı diğerine sızar")
	}
}

// TestAsciiTurkceKodIstegi — terminalde Türkçe karakter kullanmamak yaygın.
// "olustur" listede yokken zincirin tamamı sessizce kırılıyordu: istek kod
// işi sayılmıyor, thinker'a plan talimatı gitmiyor, araç emri kendisi
// uygulamaya kalkıp sandbox'a takılıyor, writer atlanıyor.
func TestAsciiTurkceKodIstegi(t *testing.T) {
	istekler := []string{
		"retry_dekorator.py olustur: ustel backoff ile yeniden deneyen dekorator",
		"bu fonksiyonu duzelt",
		"su tipi Go'ya cevir",
		"kullanilmayan importlari kaldir",
		"dosyalari yeni dizine tasi",
		"json'a donustur",
		"ilk n asal sayiyi uret",
	}
	for _, s := range istekler {
		if !looksLikeCodeRequest(s) {
			t.Errorf("ASCII Türkçe kod isteği tanınmadı: %q", s)
		}
	}
}
