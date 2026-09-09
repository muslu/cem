package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ─── Tipler ──────────────────────────────────────────────────────────────────

type Roles struct {
	Thinker string `yaml:"thinker"`
	Writer  string `yaml:"writer"`
}

type InstalledTool struct {
	Command string `yaml:"command"`
	Version string `yaml:"version,omitempty"`
	// Model — CLI'a aktif çağrıda --model olarak verilecek model adı.
	// Boş ise CLI kendi default'unu kullanır.
	Model string `yaml:"model,omitempty"`
	// Effort — düşünme/akıl yürütme seviyesi (claude: low..max,
	// codex: minimal..xhigh). Boş ise CLI kendi default'unu kullanır.
	Effort string `yaml:"effort,omitempty"`
	// Fast — hızlı mod: aracın kullanıcı ayarlarını (hook, izin kuralı, MCP)
	// yüklemesini atla. nil = ayarlanmamış → VARSAYILAN AÇIK (ölçülen fark
	// 124s → 8s). Kapatmak için: cem fast <araç> off. Bkz. ToolMeta.FastArgs.
	Fast *bool `yaml:"fast,omitempty"`
}

// APIKey — bir provider için saklanan tek bir API key. Label opsiyonel (insan
// okunabilir etiket: "personal", "company", "free-tier").
type APIKey struct {
	Value string `yaml:"value"`
	Label string `yaml:"label,omitempty"`
}

type GlobalConfig struct {
	Version string                   `yaml:"version"`
	Tools   map[string]InstalledTool `yaml:"tools"`
	Roles   Roles                    `yaml:"roles"`
	Setup   bool                     `yaml:"setup_done"`
	// APIKeys — provider → key listesi. cem aracı çağırırken sırayla denenir;
	// rate-limit hatasında bir sonrakine geçer. Provider adları:
	// "anthropic" (Claude), "openai" (Codex). agy/cursor OAuth ile çalışır.
	APIKeys map[string][]APIKey `yaml:"api_keys,omitempty"`
	// Lang — arayüz dili: "tr" | "en". Boş ise ortamdan tahmin edilir
	// (CEM_LANG > LANG); setup sihirbazı ilk açılışta kullanıcıya sorar.
	Lang string `yaml:"lang,omitempty"`
	// TrustedDirs — kullanıcının "burada çalışabilirsin" dediği dizinler.
	// Alt dizinleri de kapsar.
	TrustedDirs []string `yaml:"trusted_dirs,omitempty"`
	// CacheDisabled — true ise önbellek tamamen kapalı.
	CacheDisabled bool `yaml:"cache_disabled,omitempty"`
	// CacheWriter — yazan rolün önbelleği. nil = AÇIK (varsayılan). Kayıt
	// üretilen dosyaları da taşıdığı için önbellekten dönmek dosyasız
	// bırakmıyor; yine de kapatmak isteyen cache_writer: false yazar.
	CacheWriter *bool `yaml:"cache_writer,omitempty"`
	// CacheTTLHours — bundan eski kayıtlar yok sayılır (varsayılan 168 = 7 gün).
	CacheTTLHours int `yaml:"cache_ttl_hours,omitempty"`
	// AutoUpdateTools — kurulu AI CLI'larını günde bir kez arka planda
	// güncelle. nil = açık (varsayılan). Kapatmak: auto_update_tools: false
	AutoUpdateTools *bool `yaml:"auto_update_tools,omitempty"`
	// ToolsLastUpdate — son otomatik güncelleme denemesi (başarı/başarısızlık
	// fark etmez; sık tekrar denemeyi engeller).
	//
	// time.Time DEĞİL, string: yaml.v3 zaman alanını çözemediğinde TÜM
	// config'i reddediyor ve cem tek bir bozuk satır yüzünden hiç açılmıyordu
	// — 'cem doctor' dahil. Elle düzenlenmiş ya da başka bir araçla yeniden
	// yazılmış bir config bu hatayı kolayca üretiyor. Artık çözülemeyen
	// damga yok sayılıyor: en fazla bir kez fazladan güncelleme kontrolü olur.
	ToolsLastUpdate string `yaml:"tools_last_update,omitempty"`
}

type ProjectConfig struct {
	Roles *Roles `yaml:"roles,omitempty"`
	// Models — proje-spesifik model override'ları. Anahtar = toolKey (claude, agy, ...);
	// değer = CLI'a verilecek model adı. Boş veya tanımsızsa global config'deki
	// tools.<key>.model kullanılır; o da yoksa CLI default'u.
	Models map[string]string `yaml:"models,omitempty"`
	// Efforts — proje-spesifik düşünme seviyesi override'ları.
	// Anahtar = toolKey; değer = effort seviyesi.
	Efforts map[string]string `yaml:"efforts,omitempty"`
	// Fast — proje-spesifik hızlı mod override'ı (araç → açık/kapalı).
	Fast map[string]bool `yaml:"fast,omitempty"`
}

type ResolvedConfig struct {
	Global  *GlobalConfig
	Project *ProjectConfig
}

func (rc *ResolvedConfig) HasProjectConfig() bool {
	return rc.Project != nil && rc.Project.Roles != nil
}

func (rc *ResolvedConfig) ActiveRoles() Roles {
	if rc.HasProjectConfig() {
		r := *rc.Project.Roles
		if r.Thinker == "" {
			r.Thinker = rc.Global.Roles.Thinker
		}
		if r.Writer == "" {
			r.Writer = rc.Global.Roles.Writer
		}
		return r
	}
	return rc.Global.Roles
}

// ─── KnownTools ──────────────────────────────────────────────────────────────

type ToolMeta struct {
	Name string
	// Binary — PATH'da aranan asıl binary adı. Boş ise map'teki anahtar
	// (toolKey) kullanılır. Cursor için anahtar "cursor", binary "cursor-agent".
	Binary      string
	Description string
	// Deprecated boş değilse setup/cemi listelerinde uyarı satırı olarak basılır.
	Deprecated string
	// InstallCmd — doğrudan exec (kabuk yok). nil → manuel kurulum.
	InstallCmd []string
	// InstallShellUnix — sh -c ile çalıştırılır (curl|bash gibi pipe'lar için).
	// Linux + macOS'ta InstallCmd'den önce gelir.
	InstallShellUnix string
	// InstallShellWin — cmd /c ile çalıştırılır. Windows'ta InstallCmd'den önce gelir.
	InstallShellWin string
	// VersionFlag — cemi listesinde sürümü göstermek için (örn. "--version").
	VersionFlag string
	// RunFlags — cem aracı çalıştırırken eklenen flag'ler. Boşsa stdin'le çağrılır;
	// "-p" gibi tek-atış flag'i, AI CLI'larının interaktif REPL'e geçmesini önler.
	RunFlags []string
	// PromptAsArg true ise input, RunFlags'ten sonra son pozisyonel arg olarak verilir;
	// false ise (varsayılan) stdin üzerinden pipe edilir. Codex 'exec "prompt"'
	// gibi pozisyonel pattern bekleyen araçlar için gerekli.
	PromptAsArg bool
	// Provider — bu tool'un kullandığı API provider'ı ("anthropic", "openai").
	// Boş ise API key rotasyonu devre dışı (OAuth-only tool: agy, cursor).
	Provider string
	// APIKeyEnv — provider'ın aktif key'i hangi env değişkeniyle alacağı.
	// Boş ise key inject edilmez (CLI kendi auth'unu kullanır).
	APIKeyEnv string
	// ModelFlag — model seçimi için CLI bayrağı (örn. "--model").
	// Boş ise model seçimi devre dışı.
	ModelFlag string
	// ModelBeforeRun true ise --model X, RunFlags'ten ÖNCE yerleştirilir.
	// Gerekli olduğu durum: tool'un prompt-flag'i (örn. agy -p, cursor -p)
	// argüman alır; --model -p ile prompt arasına girerse -p'nin değeri
	// "--model" olur ve prompt yolda kaybolur. Codex 'exec' subcommand'ı
	// olduğu için RunFlags sonuna konur (default false).
	ModelBeforeRun bool
	// Models — wizard model seçicide gösterilecek öneriler. Kullanıcı bunlardan
	// birini seçebilir veya "custom" ile manuel string girebilir.
	Models []string
	// EffortArgs — düşünme seviyesi argüman şablonu; her eleman fmt.Sprintf
	// ile seviyeye göre doldurulur. Örnekler:
	//   claude: {"--effort", "%s"}              → --effort high
	//   codex : {"-c", "model_reasoning_effort=%s"}
	// Boş ise araç seviye seçimini desteklemiyor demektir.
	EffortArgs []string
	// Efforts — geçerli seviyeler (wizard listesi + doğrulama). İlk eleman
	// listenin en ucuzu olacak şekilde artan sırada tutulur.
	Efforts []string
	// LastMessageFlag — aracın SADECE final cevabını bir dosyaya yazdıran
	// bayrağı (codex: -o/--output-last-message). Doluysa cem ham akışı ekrana
	// basmaz: araç kendi tool-use adımlarını (exec, apply patch, tam diff'ler)
	// stdout'a döküyor ve aynı diff'i defalarca tekrarlıyor. Bunun yerine
	// spinner gösterilip sonunda tek, temiz cevap basılır. --raw ile devre dışı.
	LastMessageFlag string
	// FastArgs — "hızlı mod" argümanları. Aracın kullanıcı ayarlarını
	// (hook'lar, izin kuralları, MCP) yüklemesini atlatır. Ölçüldü
	// (2026-09-09, claude 2.1.266, aynı görev "not.txt'ye merhaba yaz",
	// dosya her iki durumda da oluştu):
	//     normal ............ 124s
	//     hızlı mod .......... 8s
	// Farkın tamamı kullanıcının hook/plugin kurulumunun her çağrıda
	// yeniden ayağa kalkmasından geliyor. VARSAYILAN KAPALI: kullanıcının
	// izin kurallarını sessizce atlamak doğru olmaz.
	FastArgs []string
	// UpdateCmd — aracın kendi güncelleme subcommand'ı (örn. {"update"}).
	// Boş ise güncelleme, kurulum komutunun yeniden çalıştırılmasına düşer.
	UpdateCmd []string
	// AuthCmd — 'cem auth <tool>' tarafından çağrılacak login subcommand.
	// Boş ise sadece binary çalıştırılır (CLI ilk açılışta kendi prompt'unu açar,
	// Claude Code böyle çalışır).
	AuthCmd []string
}

// KnownTools — desteklenen AI CLI araçları. Description kullanıcıya gösterilir.
// Deprecated alanı doluysa setup/cemi listelerinde uyarı görüntülenir.
var KnownTools = map[string]ToolMeta{
	"claude": {
		Name:             "Claude",
		Description:      "Anthropic Claude Code (code.claude.com) — native installer, auto-update",
		InstallShellUnix: "curl -fsSL https://claude.ai/install.sh | bash",
		InstallShellWin:  "irm https://claude.ai/install.ps1 | iex",
		VersionFlag:      "--version",
		// NOT: --bare (hook/LSP/plugin yüklemesini atlar) hız için denendi ve
		// GERİ ALINDI: oturum bilgisini de atlıyor, her çağrı
		// "Not logged in · Please run /login" ile düşüyor (ölçüldü 2026-09-09,
		// claude 2.1.266). Tekrar eklemeden önce `claude --bare -p "x"`
		// çalıştırıp gerçekten cevap döndüğünü gör.
		RunFlags: []string{"-p"}, // print mode (non-interactive)
		// claude -p PROMPT pozisyonel argüman alır. Stdin'le bırakırsak stdout
		// TTY olduğunda (ModeThink/Write) REPL'e geçip kilitleniyor.
		PromptAsArg:    true,
		ModelFlag:      "--model",
		ModelBeforeRun: true, // -p PROMPT'un arasına --model girmesin
		Models:         []string{"opus", "sonnet", "haiku"},
		// claude --help: --effort <level> (low, medium, high, xhigh, max)
		EffortArgs: []string{"--effort", "%s"},
		Efforts:    []string{"low", "medium", "high", "xhigh", "max"},
		// --setting-sources "" tek başına yetmiyor: izin kuralları da o
		// dosyalardan geldiği için claude "yazma izni verilmedi" deyip dosya
		// oluşturmuyordu. acceptEdits dosya düzenlemelerini onaylıyor; Bash
		// gibi komut çalıştıran araçlar hâlâ onay istiyor.
		FastArgs:  []string{"--setting-sources", "", "--permission-mode", "acceptEdits"},
		UpdateCmd: []string{"update"},
	},
	"agy": {
		Name:             "Antigravity",
		Description:      "Google Antigravity CLI — Gemini CLI'ın halefi (antigravity.google)",
		InstallShellUnix: "curl -fsSL https://antigravity.google/cli/install.sh | bash",
		InstallShellWin:  "irm https://antigravity.google/cli/install.ps1 | iex",
		VersionFlag:      "--version",
		RunFlags:         []string{"-p"},
		PromptAsArg:      true, // agy -p "prompt" (— -p bir argüman bekliyor)
		// NOT: Antigravity CLI'nın --model flag'i henüz yok (agy --help: -p, -c,
		// --sandbox, --print-timeout). Model Google tarafında seçiliyor. Yine de
		// Models listesi gösterilir ki wizard'da tercih kaydedilebilsin —
		// CLI ileride --model eklerse ModelFlag'i set etmek yeterli olacak.
		Models:    []string{"gemini-3-pro", "gemini-3-flash"},
		UpdateCmd: []string{"update"},
		AuthCmd:   []string{"login"},
	},
	"gpt": {
		Name:        "Codex",
		Binary:      "codex", // npm @openai/codex 'codex' adıyla PATH'e koyar
		Description: "OpenAI Codex CLI (developers.openai.com/codex)",
		InstallCmd:  []string{"npm", "install", "-g", "@openai/codex"},
		VersionFlag: "--version",
		RunFlags:    []string{"exec", "--skip-git-repo-check"}, // non-interactive, herhangi bir dizinden
		PromptAsArg: true,                                      // codex exec "prompt"
		Provider:    "openai",
		APIKeyEnv:   "OPENAI_API_KEY",
		ModelFlag:   "--model",
		Models:      []string{"gpt-5.6-terra", "gpt-5.5", "gpt-5-mini", "gpt-5"},
		// codex effort'u flag değil config anahtarı: -c model_reasoning_effort=X.
		// Geçerli değerler codex'in kendi hata mesajından: none, minimal, low,
		// medium, high, xhigh ("none" listelenmiyor — düşünmeyi kapatmak için
		// model seçimi daha doğru).
		EffortArgs: []string{"-c", "model_reasoning_effort=%s"},
		// Geçerli seviyeler MODELE göre değişiyor: gpt-5.6-terra 'minimal'i
		// reddediyor ("supported values: none, low, medium, high, xhigh, max"),
		// eski modeller kabul ediyordu. Bu liste yalnızca öneri; araç reddederse
		// hintEffort kullanıcıya kendi listesini gösterir. "none" bilerek yok:
		// düşünmeyi tamamen kapatmak isteyen modeli değiştirsin.
		Efforts:         []string{"low", "medium", "high", "xhigh", "max"},
		LastMessageFlag: "--output-last-message",
		UpdateCmd:       []string{"update"},
		AuthCmd:         []string{"login"},
	},
	"cursor": {
		Name:             "Cursor",
		Binary:           "cursor-agent", // install legacy symlink + 'agent'
		Description:      "Cursor terminal agent (cursor.com/cli)",
		InstallShellUnix: "curl -fsS https://cursor.com/install | bash",
		InstallShellWin:  "irm 'https://cursor.com/install?win32=true' | iex",
		VersionFlag:      "--version",
		RunFlags:         []string{"-p"},
		PromptAsArg:      true,
		ModelFlag:        "--model",
		ModelBeforeRun:   true, // cursor-agent -p arg yutmasın diye
		Models:           []string{"claude-4.6", "gpt-5.2", "gemini-3-pro"},
		UpdateCmd:        []string{"update"},
		AuthCmd:          []string{"login"},
	},
}

// orderedToolKeys — wizard/installer listelerinin sabit sırası.
// KnownTools map iterasyonu rastgele; UI tutarlılığı için bu liste kullanılır.
var orderedToolKeys = []string{
	"claude", "agy", "gpt", "cursor",
}

// lastToolUpdate — damgayı zamana çevirir; çözülemezse sıfır değer.
func (g *GlobalConfig) lastToolUpdate() time.Time {
	if g == nil || g.ToolsLastUpdate == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, g.ToolsLastUpdate); err == nil {
			return t
		}
	}
	return time.Time{}
}

// setLastToolUpdate — her zaman RFC3339 yazar.
func (g *GlobalConfig) setLastToolUpdate(t time.Time) {
	g.ToolsLastUpdate = t.Format(time.RFC3339)
}

// ─── Rol bazlı varsayılanlar ─────────────────────────────────────────────

// Kullanıcı hiçbir şey ayarlamasa bile kurulum israfsız olmalı. Rol seçildiği
// anda maliyet-optimal düşünme seviyesi yazılır:
//
//	thinker → planı O çıkarıyor; derin düşünsün, çıktısı zaten kısa
//	writer  → planı uyguluyor; derin düşünmesine gerek yok, çıktısı uzun
//
// Model seçimi bilinçli olarak CLI'ın kendi varsayılanına bırakılıyor:
// sağlayıcılar model adlarını sık değiştiriyor, sabit bir ad yazmak
// kullanıcıyı erişemediği bir modele kilitleyebilir (sahada yaşandı:
// gpt-5-mini ChatGPT hesabıyla, gpt-5.5 ücretsiz planla çalışmıyor).

// defaultEffortForRole — rol için önerilen seviye; araç desteklemiyorsa "".
func defaultEffortForRole(toolKey, role string) string {
	meta, ok := KnownTools[toolKey]
	if !ok || len(meta.Efforts) == 0 || len(meta.EffortArgs) == 0 {
		return ""
	}
	want := "high"
	if role == "writer" {
		want = "low"
	}
	for _, e := range meta.Efforts {
		if e == want {
			return e
		}
	}
	return ""
}

// applyRoleDefaults — rol atandığında (setup / cem roles) boş bırakılmış
// seviyeleri doldurur. Kullanıcının açık seçimini ASLA ezmez.
func applyRoleDefaults(cfg *GlobalConfig, thinker, writer string) {
	if cfg.Tools == nil {
		return
	}
	for _, pair := range []struct{ key, role string }{
		{thinker, "thinker"},
		{writer, "writer"},
	} {
		if pair.key == "" {
			continue
		}
		t, ok := cfg.Tools[pair.key]
		if !ok || t.Effort != "" {
			continue
		}
		if e := defaultEffortForRole(pair.key, pair.role); e != "" {
			t.Effort = e
			cfg.Tools[pair.key] = t
		}
	}
}

// ─── Yollar ──────────────────────────────────────────────────────────────────

func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cem", "config.yaml"), nil
}

const projectConfigName = ".cem.yaml"

// ─── Global config ───────────────────────────────────────────────────────────

func loadGlobalConfig() (*GlobalConfig, error) {
	path, err := globalConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &GlobalConfig{
		Version: "1.0",
		Tools:   map[string]InstalledTool{},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		// Tek bozuk satır cem'i tamamen kullanılamaz hâle getiriyordu; en
		// azından nerede olduğunu ve nasıl kurtarılacağını söyleyelim.
		return nil, fmt.Errorf("%s\n  %s\n  %s: %w",
			L("global config okunamadı", "cannot read the global config"),
			path,
			L("düzelt ya da sil (cem setup yeniden oluşturur)",
				"fix it or delete it (cem setup recreates it)"),
			err)
	}
	if cfg.Tools == nil {
		cfg.Tools = map[string]InstalledTool{}
	}
	// Dil, config okunur okunmaz devreye girsin: bundan sonraki her L()
	// çağrısı doğru dili görür.
	setLang(cfg.Lang)
	return cfg, nil
}

func saveGlobalConfig(cfg *GlobalConfig) error {
	path, err := globalConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// ─── Project config ──────────────────────────────────────────────────────────

func loadProjectConfig() (*ProjectConfig, error) {
	data, err := os.ReadFile(projectConfigName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	pc := &ProjectConfig{}
	if err := yaml.Unmarshal(data, pc); err != nil {
		return nil, fmt.Errorf("project config parse: %w", err)
	}
	if pc.Roles == nil {
		return nil, nil
	}
	return pc, nil
}

func SaveProjectConfig(pc *ProjectConfig) error {
	data, err := yaml.Marshal(pc)
	if err != nil {
		return err
	}
	return os.WriteFile(projectConfigName, data, 0o644)
}

// ─── Resolved ────────────────────────────────────────────────────────────────

func LoadConfig() (*ResolvedConfig, error) {
	g, err := loadGlobalConfig()
	if err != nil {
		return nil, err
	}
	p, err := loadProjectConfig()
	if err != nil {
		return nil, err
	}
	return &ResolvedConfig{Global: g, Project: p}, nil
}
