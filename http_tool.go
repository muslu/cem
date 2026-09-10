package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// HTTP endpoint araçları (ollama, lmstudio, unsloth) subprocess değildir:
// cem model sunucusuyla doğrudan konuşur. İki API konuşulur —
// OpenAI uyumlu /v1/chat/completions ve Ollama'nın kendi /api/chat'i.
//
// Neden akışlı (stream): yerel modeller ilk token'ı saniyeler sonra veriyor ve
// uzun cevap dakikalar sürüyor. Tek parça beklemek kullanıcıya "takıldı" gibi
// görünürdü; abonelikli CLI'larda olduğu gibi cevap aktıkça basılır.
const (
	// httpToolCeiling — bir isteğin üst sınırı. Yerel 70B model uzun cevapta
	// dakikalar harcayabiliyor; sınır tamamen kaldırılırsa asılı kalan bir
	// sunucu cem'i süresiz bekletir.
	httpToolCeiling = 30 * time.Minute
	// httpDialTimeout — BAĞLANMA sınırı ayrı ve kısa: kapalı porta 30 dakika
	// beklemek yerine hemen "sunucu ayakta mı?" diye sormak gerekiyor.
	httpDialTimeout = 5 * time.Second
	// httpProbeTimeout — `--test` ve model listesi için.
	httpProbeTimeout = 8 * time.Second
)

var errNoEndpointModel = errors.New("model adı tanımlı değil")

// httpClient — bağlanma kısa, toplam uzun.
func httpClient(total time.Duration) *http.Client {
	return &http.Client{
		Timeout: total,
		Transport: &http.Transport{
			DialContext:         (&net.Dialer{Timeout: httpDialTimeout}).DialContext,
			TLSHandshakeTimeout: httpDialTimeout,
		},
	}
}

// normalizeBaseURL — kullanıcının yazdığı adresi kullanılabilir URL'e çevirir.
//
// Kabul edilenler: "1.2.3.4:11434", "localhost:1234", "http://host:8000",
// "https://gpu.lan/v1", "host" (portsuz). Şema yoksa http:// eklenir —
// yerel/LAN sunucularında TLS beklemek yanlış varsayım olurdu. Sondaki "/"
// atılır ki yol ekleyince "//" oluşmasın.
func normalizeBaseURL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = "http://" + s
	}
	return strings.TrimRight(s, "/")
}

// resolveEndpoint — proje config'i global'i ezer; hiçbiri yoksa ToolMeta'daki
// varsayılan yerel adres kullanılır.
func resolveEndpoint(toolKey string, rc *ResolvedConfig) Endpoint {
	meta := KnownTools[toolKey]
	ep := Endpoint{BaseURL: meta.DefaultBaseURL}

	if rc != nil && rc.Global != nil {
		if g, ok := rc.Global.Endpoints[toolKey]; ok {
			if g.BaseURL != "" {
				ep.BaseURL = g.BaseURL
			}
			if g.APIKey != "" {
				ep.APIKey = g.APIKey
			}
			if g.Model != "" {
				ep.Model = g.Model
			}
		}
	}
	if rc != nil && rc.Project != nil {
		if p, ok := rc.Project.Endpoints[toolKey]; ok {
			if p.BaseURL != "" {
				ep.BaseURL = p.BaseURL
			}
			if p.APIKey != "" {
				ep.APIKey = p.APIKey
			}
			if p.Model != "" {
				ep.Model = p.Model
			}
		}
	}
	ep.BaseURL = normalizeBaseURL(ep.BaseURL)
	return ep
}

// endpointModel — endpoint.Model > tools.<key>.model. Tahmin YOK: yerel
// sunucuda yanlış model adı sessizce başka bir modeli çalıştırmak demek.
func endpointModel(toolKey string, rc *ResolvedConfig, ep Endpoint) (string, error) {
	if ep.Model != "" {
		return ep.Model, nil
	}
	if m := resolveModel(toolKey, rc); m != "" {
		return m, nil
	}
	return "", errNoEndpointModel
}

// chatURL — API tipine göre istek adresi.
func chatURL(meta ToolMeta, base string) string {
	if meta.HTTPAPI == "ollama" {
		return base + "/api/chat"
	}
	// Kullanıcı adresi ".../v1" olarak verdiyse tekrar eklemiyoruz.
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

// httpChatRequest — iki API'nin gövdesi de aynı alanlardan oluşuyor.
func httpChatRequest(model, input string) map[string]any {
	return map[string]any{
		"model":  model,
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": input},
		},
	}
}

// runHTTPTool — isteği gönderir, cevabı akıtır ve tam metni döner.
// echo=false ise ekrana basılmaz (pair modunda düşünenin çıktısı zaten
// başka yerde gösteriliyor olabilir).
func runHTTPTool(toolKey string, rc *ResolvedConfig, input string, sp *Spinner, echo bool) (string, error) {
	meta := KnownTools[toolKey]
	ep := resolveEndpoint(toolKey, rc)
	if ep.BaseURL == "" {
		if sp != nil {
			sp.Stop()
		}
		fmt.Println(styleError.Render(fmt.Sprintf(
			L("✗ %s için adres tanımlı değil", "✗ no address configured for %s"), toolKey)))
		fmt.Println(styleDim.Render(fmt.Sprintf("    cem endpoint %s <ip:port>", toolKey)))
		return "", fmt.Errorf("%s: endpoint yok", toolKey)
	}

	model, err := endpointModel(toolKey, rc, ep)
	if err != nil {
		if sp != nil {
			sp.Stop()
		}
		fmt.Println(styleError.Render(fmt.Sprintf(
			L("✗ %s için model adı gerekli", "✗ %s needs a model name"), toolKey)))
		fmt.Println(styleDim.Render(fmt.Sprintf(
			L("    cem endpoint %s --model <ad>     (kurulu modeller: cem endpoint %s --modeller)",
				"    cem endpoint %s --model <name>   (available models: cem endpoint %s --modeller)"),
			toolKey, toolKey)))
		return "", err
	}

	body, err := json.Marshal(httpChatRequest(model, input))
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpToolCeiling)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL(meta, ep.BaseURL), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if ep.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+ep.APIKey)
	}

	resp, err := httpClient(httpToolCeiling).Do(req)
	if err != nil {
		if sp != nil {
			sp.Stop()
		}
		hintEndpointUnreachable(toolKey, ep, err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if sp != nil {
			sp.Stop()
		}
		return "", httpToolStatusError(toolKey, ep, resp)
	}

	var dst io.Writer = io.Discard
	if echo {
		dst = os.Stdout
	}
	text, err := streamChat(meta.HTTPAPI, resp.Body, dst, sp)
	if err != nil {
		return text, err
	}
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf(L("%s boş cevap döndü (model: %s)", "%s returned an empty answer (model: %s)"), toolKey, model)
	}
	return text, nil
}

// streamChat — akıştaki parçaları hem ekrana yazar hem biriktirir.
//
// İki biçim var: OpenAI SSE ("data: {...}", sonunda "data: [DONE]") ve
// Ollama'nın satır-başına-JSON'u. İkisi de satır tabanlı olduğu için tek
// tarayıcı yeterli.
func streamChat(api string, r io.Reader, dst io.Writer, sp *Spinner) (string, error) {
	var acc strings.Builder
	sc := bufio.NewScanner(r)
	// Uzun satırlar: tek parçada büyük içerik gelebiliyor.
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if api != "ollama" {
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if line == "[DONE]" {
				break
			}
		}

		parca, bitti := chatChunk(api, line)
		if parca != "" {
			// Spinner'ı ilk içerik gelince durdur: aynı satıra yazıp cevabın
			// üzerine binmesin (aynı hata daha önce pair modunda yaşandı).
			if first {
				if sp != nil {
					sp.Stop()
				}
				first = false
			}
			acc.WriteString(parca)
			fmt.Fprint(dst, parca)
		}
		if bitti {
			break
		}
	}
	if err := sc.Err(); err != nil {
		return acc.String(), err
	}
	if !first {
		fmt.Fprintln(dst)
	}
	return acc.String(), nil
}

// chatChunk — tek satırdan metin parçasını ve "bitti" bilgisini çıkarır.
// Çözülemeyen satır sessizce atlanır: sunucular arada ısıtma/istatistik
// nesneleri yollayabiliyor ve bunlar için hata vermek gereksiz.
func chatChunk(api, line string) (string, bool) {
	if api == "ollama" {
		var o struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Done bool `json:"done"`
		}
		if json.Unmarshal([]byte(line), &o) != nil {
			return "", false
		}
		return o.Message.Content, o.Done
	}
	var o struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
	}
	if json.Unmarshal([]byte(line), &o) != nil || len(o.Choices) == 0 {
		return "", false
	}
	c := o.Choices[0]
	return c.Delta.Content, c.FinishReason != nil && *c.FinishReason != ""
}

// hintEndpointUnreachable — bağlanamama en sık görülen hata; kullanıcıya
// adresi ve düzeltme yolunu göster.
func hintEndpointUnreachable(toolKey string, ep Endpoint, err error) {
	fmt.Println(styleError.Render(fmt.Sprintf(
		L("✗ %s adresine bağlanılamadı: %s", "✗ cannot reach %s at %s"), toolKey, ep.BaseURL)))
	fmt.Println(styleDim.Render("    " + err.Error()))
	fmt.Println(styleDim.Render(fmt.Sprintf(
		L("    Sunucu ayakta mı: cem endpoint %s --test", "    Is the server up: cem endpoint %s --test"), toolKey)))
}

// httpToolStatusError — 2xx dışı yanıtı okunur hataya çevirir.
func httpToolStatusError(toolKey string, ep Endpoint, resp *http.Response) error {
	gövde, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
	özet := strings.TrimSpace(string(gövde))
	if len(özet) > 300 {
		özet = özet[:300] + "…"
	}
	fmt.Println(styleError.Render(fmt.Sprintf("✗ %s HTTP %d — %s", toolKey, resp.StatusCode, ep.BaseURL)))
	if özet != "" {
		fmt.Println(styleDim.Render("    " + özet))
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		fmt.Println(styleDim.Render(fmt.Sprintf(
			L("    Anahtar gerekiyor: cem endpoint %s --key <anahtar>",
				"    A key is required: cem endpoint %s --key <key>"), toolKey)))
	case http.StatusNotFound:
		fmt.Println(styleDim.Render(fmt.Sprintf(
			L("    Adres yolu yanlış olabilir; şu an denenen: %s",
				"    The path may be wrong; currently trying: %s"), chatURL(KnownTools[toolKey], ep.BaseURL))))
	}
	return fmt.Errorf("%s: HTTP %d", toolKey, resp.StatusCode)
}

// probeEndpoint — sunucu ayakta mı ve hangi modelleri sunuyor.
func probeEndpoint(toolKey string, rc *ResolvedConfig) ([]string, error) {
	meta := KnownTools[toolKey]
	ep := resolveEndpoint(toolKey, rc)
	if ep.BaseURL == "" {
		return nil, fmt.Errorf(L("%s için adres tanımlı değil", "no address configured for %s"), toolKey)
	}

	adres := ep.BaseURL + "/v1/models"
	if meta.HTTPAPI == "ollama" {
		adres = ep.BaseURL + "/api/tags"
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpProbeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adres, nil)
	if err != nil {
		return nil, err
	}
	if ep.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+ep.APIKey)
	}
	resp, err := httpClient(httpProbeTimeout).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return parseModelList(meta.HTTPAPI, resp.Body)
}

// parseModelList — /api/tags ve /v1/models cevaplarından model adlarını çıkarır.
func parseModelList(api string, r io.Reader) ([]string, error) {
	data, err := io.ReadAll(io.LimitReader(r, 1<<20))
	if err != nil {
		return nil, err
	}
	var adlar []string
	if api == "ollama" {
		var o struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.Unmarshal(data, &o); err != nil {
			return nil, err
		}
		for _, m := range o.Models {
			adlar = append(adlar, m.Name)
		}
		return adlar, nil
	}
	var o struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	for _, m := range o.Data {
		adlar = append(adlar, m.ID)
	}
	return adlar, nil
}

// endpointHost — ekranda gösterim için şemasız kısa adres.
func endpointHost(base string) string {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return base
	}
	return u.Host
}

// configuredHTTPTools — kullanıcının adres verdiği HTTP araçları (varsayılan
// yerel adresle bırakılmış olanlar dahil değil).
func configuredHTTPTools(rc *ResolvedConfig) []string {
	var out []string
	for _, k := range orderedToolKeys {
		if !isHTTPTool(k) {
			continue
		}
		var ayarli bool
		if rc != nil && rc.Global != nil {
			_, ayarli = rc.Global.Endpoints[k]
		}
		if !ayarli && rc != nil && rc.Project != nil {
			_, ayarli = rc.Project.Endpoints[k]
		}
		if ayarli {
			out = append(out, k)
		}
	}
	return out
}
