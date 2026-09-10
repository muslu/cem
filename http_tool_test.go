package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeBaseURL(t *testing.T) {
	durumlar := map[string]string{
		"192.168.1.10:11434":     "http://192.168.1.10:11434",
		"localhost:1234":         "http://localhost:1234",
		"http://127.0.0.1:8000/": "http://127.0.0.1:8000",
		"https://gpu.lan/v1":     "https://gpu.lan/v1",
		"  10.0.0.5:1234  ":      "http://10.0.0.5:1234",
		"":                       "",
	}
	for girdi, beklenen := range durumlar {
		if got := normalizeBaseURL(girdi); got != beklenen {
			t.Errorf("normalizeBaseURL(%q) = %q, beklenen %q", girdi, got, beklenen)
		}
	}
}

// Kullanıcı adresi ".../v1" olarak verdiyse yol iki kez eklenmemeli.
func TestChatURLV1YiTekrarlamaz(t *testing.T) {
	openai := KnownTools["lmstudio"]
	if got := chatURL(openai, "http://h:1234"); got != "http://h:1234/v1/chat/completions" {
		t.Errorf("openai: %q", got)
	}
	if got := chatURL(openai, "https://gpu.lan/v1"); got != "https://gpu.lan/v1/chat/completions" {
		t.Errorf("openai /v1: %q", got)
	}
	if got := chatURL(KnownTools["ollama"], "http://h:11434"); got != "http://h:11434/api/chat" {
		t.Errorf("ollama: %q", got)
	}
}

func TestChatChunkIkiBicimiCozer(t *testing.T) {
	if metin, _ := chatChunk("openai", `{"choices":[{"delta":{"content":"mer"}}]}`); metin != "mer" {
		t.Errorf("openai delta: %q", metin)
	}
	if _, bitti := chatChunk("openai", `{"choices":[{"delta":{},"finish_reason":"stop"}]}`); !bitti {
		t.Error("finish_reason=stop bitiş sayılmalı")
	}
	if metin, bitti := chatChunk("ollama", `{"message":{"content":"haba"},"done":false}`); metin != "haba" || bitti {
		t.Errorf("ollama: %q %v", metin, bitti)
	}
	// Sunucular arada istatistik/ısıtma nesnesi yollayabiliyor; bunlar hata
	// değil, atlanacak satır.
	if metin, bitti := chatChunk("openai", `{"beklenmeyen":1}`); metin != "" || bitti {
		t.Error("tanınmayan satır sessizce atlanmalı")
	}
	if metin, _ := chatChunk("ollama", "bozuk json"); metin != "" {
		t.Error("bozuk satır hata vermeden atlanmalı")
	}
}

func TestParseModelList(t *testing.T) {
	ollama := `{"models":[{"name":"qwen3-coder:30b"},{"name":"llama3.3"}]}`
	adlar, err := parseModelList("ollama", strings.NewReader(ollama))
	if err != nil || len(adlar) != 2 || adlar[0] != "qwen3-coder:30b" {
		t.Fatalf("ollama listesi: %v %v", adlar, err)
	}
	openai := `{"data":[{"id":"mistral-7b"},{"id":"gemma-3"}]}`
	adlar, err = parseModelList("openai", strings.NewReader(openai))
	if err != nil || len(adlar) != 2 || adlar[1] != "gemma-3" {
		t.Fatalf("openai listesi: %v %v", adlar, err)
	}
}

// Proje config'i global'i, global de ToolMeta varsayılanını ezer.
func TestResolveEndpointOncelikSirasi(t *testing.T) {
	rc := &ResolvedConfig{Global: &GlobalConfig{}}
	if got := resolveEndpoint("ollama", rc).BaseURL; got != "http://127.0.0.1:11434" {
		t.Errorf("varsayılan: %q", got)
	}

	rc.Global.Endpoints = map[string]Endpoint{"ollama": {BaseURL: "10.0.0.9:11434", Model: "global-model"}}
	ep := resolveEndpoint("ollama", rc)
	if ep.BaseURL != "http://10.0.0.9:11434" || ep.Model != "global-model" {
		t.Errorf("global: %+v", ep)
	}

	rc.Project = &ProjectConfig{Endpoints: map[string]Endpoint{"ollama": {BaseURL: "gpu.lan:11434"}}}
	ep = resolveEndpoint("ollama", rc)
	if ep.BaseURL != "http://gpu.lan:11434" {
		t.Errorf("proje adresi global'i ezmeli: %+v", ep)
	}
	// Proje yalnız adresi verdiyse global'in modeli korunur.
	if ep.Model != "global-model" {
		t.Errorf("proje override'ı modeli silmemeli: %+v", ep)
	}
}

// Model adı tahmin edilmez: hiçbir yerde tanımlı değilse hata döner.
func TestEndpointModelTahminEtmez(t *testing.T) {
	rc := &ResolvedConfig{Global: &GlobalConfig{}}
	if _, err := endpointModel("ollama", rc, resolveEndpoint("ollama", rc)); err == nil {
		t.Fatal("model tanımsızken hata beklenir")
	}
}

// Model sunucuları kurulmaz: cemi/cemir/auto-update listesine girmemeli.
func TestInstallableToolKeysSunucularıDisar(t *testing.T) {
	for _, k := range installableToolKeys() {
		if isHTTPTool(k) {
			t.Fatalf("%s kurulabilir listede olmamalı", k)
		}
	}
	if len(installableToolKeys()) == len(orderedToolKeys) {
		t.Fatal("HTTP araçları hiç ayrılmamış")
	}
}

// Uçtan uca: OpenAI uyumlu SSE akışı parça parça basılır ve birleşir.
func TestRunHTTPToolOpenAIAkisi(t *testing.T) {
	var alinanYol, yetki string
	var istek map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		alinanYol = r.URL.Path
		yetki = r.Header.Get("Authorization")
		gövde, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(gövde, &istek)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"iki \"}}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"parça\"}}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	rc := &ResolvedConfig{Global: &GlobalConfig{Endpoints: map[string]Endpoint{
		"lmstudio": {BaseURL: srv.URL, APIKey: "sk-test", Model: "mistral-7b"},
	}}}
	out, err := runHTTPTool("lmstudio", rc, "soru", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if out != "iki parça" {
		t.Errorf("akış birleşmedi: %q", out)
	}
	if alinanYol != "/v1/chat/completions" {
		t.Errorf("yol: %q", alinanYol)
	}
	if yetki != "Bearer sk-test" {
		t.Errorf("anahtar Bearer olarak gitmeli: %q", yetki)
	}
	if istek["model"] != "mistral-7b" || istek["stream"] != true {
		t.Errorf("istek gövdesi: %+v", istek)
	}
}

// Ollama'nın satır-başına-JSON akışı.
func TestRunHTTPToolOllamaAkisi(t *testing.T) {
	var yol string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		yol = r.URL.Path
		io.WriteString(w, "{\"message\":{\"content\":\"yerel \"},\"done\":false}\n")
		io.WriteString(w, "{\"message\":{\"content\":\"model\"},\"done\":true}\n")
	}))
	defer srv.Close()

	rc := &ResolvedConfig{Global: &GlobalConfig{Endpoints: map[string]Endpoint{
		"ollama": {BaseURL: srv.URL, Model: "qwen3-coder"},
	}}}
	out, err := runHTTPTool("ollama", rc, "soru", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if out != "yerel model" {
		t.Errorf("akış: %q", out)
	}
	if yol != "/api/chat" {
		t.Errorf("yol: %q", yol)
	}
}

// Anahtarsız yerel sunucuda Authorization başlığı HİÇ gönderilmemeli.
func TestRunHTTPToolAnahtarsizBaslikGondermez(t *testing.T) {
	var vardi bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, vardi = r.Header["Authorization"]
		io.WriteString(w, "{\"message\":{\"content\":\"ok\"},\"done\":true}\n")
	}))
	defer srv.Close()
	rc := &ResolvedConfig{Global: &GlobalConfig{Endpoints: map[string]Endpoint{
		"ollama": {BaseURL: srv.URL, Model: "m"},
	}}}
	if _, err := runHTTPTool("ollama", rc, "soru", nil, false); err != nil {
		t.Fatal(err)
	}
	if vardi {
		t.Error("anahtar yokken Authorization başlığı gönderilmemeli")
	}
}

// 2xx dışı yanıt okunur hataya çevrilir (gövde de kırpılarak gösterilir).
func TestRunHTTPToolHTTPHatasi(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":"invalid api key"}`)
	}))
	defer srv.Close()
	rc := &ResolvedConfig{Global: &GlobalConfig{Endpoints: map[string]Endpoint{
		"unsloth": {BaseURL: srv.URL, Model: "m"},
	}}}
	_, err := runHTTPTool("unsloth", rc, "soru", nil, false)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("401 hatası beklenir, gelen: %v", err)
	}
}

// Boş cevap sessizce "başarılı" sayılmamalı: model yüklenmemiş sunucular
// içeriksiz akış döndürüyor.
func TestRunHTTPToolBosCevapHatasi(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "{\"message\":{\"content\":\"\"},\"done\":true}\n")
	}))
	defer srv.Close()
	rc := &ResolvedConfig{Global: &GlobalConfig{Endpoints: map[string]Endpoint{
		"ollama": {BaseURL: srv.URL, Model: "m"},
	}}}
	if _, err := runHTTPTool("ollama", rc, "soru", nil, false); err == nil {
		t.Fatal("boş cevapta hata beklenir")
	}
}
