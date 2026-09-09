package main

import "testing"

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
	rc := rcWith(map[string]InstalledTool{
		"claude": {Model: "sonnet", Effort: "xhigh"},
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
	if got[len(got)-1] != "merhaba" {
		t.Error("prompt son argüman değil")
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
