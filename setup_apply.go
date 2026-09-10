package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Etkileşimsiz setup ve makine-okur durum çıktısı.
//
// Neden gerekli: setup zorunlu tutuluyor (loadAndCheckSetup) ama sihirbaz TTY
// istiyor. IDE eklentisi, kurulum betikleri ve CI'da TTY yok — o ortamlarda
// kurulum yapılamadığı için cem hiç çalışmıyordu. Buradaki iki yüzey (bayraklı
// `cem setup` ve `cem status --json`) GUI'nin ve betiklerin aynı yapılandırma
// mantığını kullanmasını sağlıyor: config'i yazan tek yer hâlâ cem.

// machineReadableOutput — çağıran çıktıyı ayrıştıracak mı (--json).
//
// Bayrağı cobra ayrıştırmadan önce bilmemiz gerekiyor: PersistentPreRun
// komuttan ÖNCE çalışıyor ve orada basılan tek satır JSON'u bozuyor.
func machineReadableOutput() bool {
	for _, a := range os.Args[1:] {
		if a == "--json" {
			return true
		}
	}
	return false
}

// SetupOptions — bayraklarla verilen kurulum seçimleri.
type SetupOptions struct {
	Thinker         string
	Writer          string
	ModelThinker    string
	ModelWriter     string
	EffortThinker   string
	EffortWriter    string
	EndpointThinker string
	EndpointWriter  string
	Lang            string
}

// Given — kullanıcı en az bir seçim verdi mi (verdiyse sihirbaz açılmaz).
func (o SetupOptions) Given() bool {
	return o.Thinker != "" || o.Writer != "" || o.Lang != ""
}

// ApplySetup — seçimleri doğrular ve config'e yazar.
//
// Doğrulama sırası bilinçli: önce araç adı, sonra HTTP/CLI'ya göre gereklilik.
// Yanlış yapılandırmayı kaydetmek, sonraki çalıştırmada anlaşılmaz bir hataya
// dönüşürdü; hata mesajı burada, seçimin yanında verilir.
func ApplySetup(cfg *GlobalConfig, o SetupOptions) error {
	if o.Lang != "" {
		if o.Lang != "tr" && o.Lang != "en" {
			return fmt.Errorf(L("dil 'tr' veya 'en' olmalı: %s", "language must be 'tr' or 'en': %s"), o.Lang)
		}
		cfg.Lang = o.Lang
		setLang(cfg.Lang)
	}

	// Rol verilmediyse mevcut değeri koru: `cem setup --lang en` sadece dili
	// değiştirmeli.
	thinker := firstNonEmpty(o.Thinker, cfg.Roles.Thinker)
	writer := firstNonEmpty(o.Writer, cfg.Roles.Writer)
	if thinker == "" || writer == "" {
		return fmt.Errorf("%s", L("düşünen ve yazan rolü gerekli (--thinker, --writer)",
			"thinker and writer roles are required (--thinker, --writer)"))
	}

	for rol, key := range map[string]string{"thinker": thinker, "writer": writer} {
		if _, ok := KnownTools[key]; !ok {
			öneri := suggestTool(key)
			if öneri != "" {
				return fmt.Errorf(L("bilinmeyen araç (%s): %s — %s mi?", "unknown tool (%s): %s — did you mean %s?"), rol, key, öneri)
			}
			return fmt.Errorf(L("bilinmeyen araç (%s): %s", "unknown tool (%s): %s"), rol, key)
		}
	}

	if cfg.Tools == nil {
		cfg.Tools = map[string]InstalledTool{}
	}

	roller := []struct {
		key      string
		model    string
		effort   string
		endpoint string
	}{
		{thinker, o.ModelThinker, o.EffortThinker, o.EndpointThinker},
		{writer, o.ModelWriter, o.EffortWriter, o.EndpointWriter},
	}

	for _, r := range roller {
		if isHTTPTool(r.key) {
			if err := applyEndpointSetup(cfg, r.key, r.endpoint, r.model); err != nil {
				return err
			}
			continue
		}
		if err := applyCLIToolSetup(cfg, r.key, r.model, r.effort); err != nil {
			return err
		}
	}

	cfg.Roles = Roles{Thinker: thinker, Writer: writer}
	cfg.Setup = true
	return saveGlobalConfig(cfg)
}

// applyEndpointSetup — HTTP aracın adresi ve modeli. Model adı tahmin
// edilmiyor: yanlış ad, yerel sunucuda istenmeyen modeli çalıştırmak demek.
func applyEndpointSetup(cfg *GlobalConfig, key, endpoint, model string) error {
	if cfg.Endpoints == nil {
		cfg.Endpoints = map[string]Endpoint{}
	}
	ep := cfg.Endpoints[key]
	if endpoint != "" {
		ep.BaseURL = normalizeBaseURL(endpoint)
	}
	if ep.BaseURL == "" {
		ep.BaseURL = KnownTools[key].DefaultBaseURL
	}
	if model != "" {
		ep.Model = model
	}
	if ep.Model == "" {
		return fmt.Errorf(L("%s için model adı gerekli (--model-...) — sunucudaki modeller: cem endpoint %s --modeller",
			"%s needs a model name (--model-...) — models on the server: cem endpoint %s --modeller"), key, key)
	}
	cfg.Endpoints[key] = ep
	return nil
}

// applyCLIToolSetup — binary tabanlı araç: PATH'te varsa config'e kaydedilir.
// Yoksa hata DEĞİL uyarı: kullanıcı aracı sonra kurabilir, ama seçim
// kaydedilmezse cem yine "kurulum yapılmadı" der ve döngüye girerdi.
func applyCLIToolSetup(cfg *GlobalConfig, key, model, effort string) error {
	if effort != "" {
		if !slicesContains(KnownTools[key].Efforts, effort) && len(KnownTools[key].Efforts) > 0 {
			return fmt.Errorf(L("%s için geçersiz effort: %s (geçerli: %s)", "invalid effort for %s: %s (valid: %s)"),
				key, effort, strings.Join(KnownTools[key].Efforts, ", "))
		}
	}
	t := cfg.Tools[key]
	bin := KnownTools[key].Binary
	if bin == "" {
		bin = key
	}
	if p, err := exec.LookPath(bin); err == nil {
		t.Command = p
	} else {
		fmt.Println(styleWarn.Render(fmt.Sprintf(
			L("  ⚠ %s PATH'te yok — kurmak için: cemi %s", "  ⚠ %s is not in PATH — install it with: cemi %s"), bin, key)))
	}
	if model != "" {
		t.Model = model
	}
	if effort != "" {
		t.Effort = effort
	}
	cfg.Tools[key] = t
	return nil
}

// statusJSON — GUI ve betikler için makine-okur durum.
//
// Eklenti bununla iki şeyi öğreniyor: kurulum yapılmış mı (yapılmadıysa
// kullanıcıya form gösterilir) ve mevcut seçimler ne (form önceden dolar).
// Ekrana basılan tablo insan içindir; onu ayrıştırmak kırılgan olurdu.
type statusTool struct {
	Key       string   `json:"key"`
	Name      string   `json:"name"`
	HTTP      bool     `json:"http"`
	Installed bool     `json:"installed"`
	Path      string   `json:"path,omitempty"`
	Model     string   `json:"model,omitempty"`
	Effort    string   `json:"effort,omitempty"`
	Endpoint  string   `json:"endpoint,omitempty"`
	Models    []string `json:"models,omitempty"`
	Efforts   []string `json:"efforts,omitempty"`
}

type statusOut struct {
	Version     string       `json:"version"`
	Setup       bool         `json:"setup"`
	Lang        string       `json:"lang"`
	Thinker     string       `json:"thinker"`
	Writer      string       `json:"writer"`
	ConfigPath  string       `json:"config_path"`
	ProjectFile bool         `json:"project_config"`
	Tools       []statusTool `json:"tools"`
}

func printStatusJSON(rc *ResolvedConfig) error {
	roles := rc.ActiveRoles()
	out := statusOut{
		Version:     version,
		Setup:       rc.Global.Setup,
		Lang:        rc.Global.Lang,
		Thinker:     roles.Thinker,
		Writer:      roles.Writer,
		ConfigPath:  globalConfigPathOrEmpty(),
		ProjectFile: rc.HasProjectConfig(),
	}
	for _, key := range orderedToolKeys {
		meta := KnownTools[key]
		st := statusTool{
			Key:     key,
			Name:    meta.Name,
			HTTP:    isHTTPTool(key),
			Models:  meta.Models,
			Efforts: meta.Efforts,
		}
		if st.HTTP {
			ep := resolveEndpoint(key, rc)
			st.Endpoint = ep.BaseURL
			if m, err := endpointModel(key, rc, ep); err == nil {
				st.Model = m
			}
			// HTTP araçta "kurulu" = adres ayarlanmış.
			_, st.Installed = rc.Global.Endpoints[key]
		} else {
			bin := resolveCommand(key, rc)
			if p, err := exec.LookPath(bin); err == nil {
				st.Installed = true
				st.Path = p
			}
			st.Model = resolveModel(key, rc)
			st.Effort = resolveEffort(key, rc)
		}
		out.Tools = append(out.Tools, st)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// globalConfigPathOrEmpty — JSON çıktısında yol bilgisi; hata durumunda boş
// bırakılır (durum raporu bu yüzden hiç dönmemesin).
func globalConfigPathOrEmpty() string {
	p, err := globalConfigPath()
	if err != nil {
		return ""
	}
	return p
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func slicesContains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
