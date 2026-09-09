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
	// AutoUpdateTools — kurulu AI CLI'larını günde bir kez arka planda
	// güncelle. nil = açık (varsayılan). Kapatmak: auto_update_tools: false
	AutoUpdateTools *bool `yaml:"auto_update_tools,omitempty"`
	// ToolsLastUpdate — son otomatik güncelleme denemesi (başarı/başarısızlık
	// fark etmez; sık tekrar denemeyi engeller).
	ToolsLastUpdate time.Time `yaml:"tools_last_update,omitempty"`
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
		RunFlags:         []string{"-p"}, // print mode (non-interactive)
		// claude -p PROMPT pozisyonel argüman alır. Stdin'le bırakırsak stdout
		// TTY olduğunda (ModeThink/Write) REPL'e geçip kilitleniyor.
		PromptAsArg:    true,
		ModelFlag:      "--model",
		ModelBeforeRun: true, // -p PROMPT'un arasına --model girmesin
		Models:         []string{"opus", "sonnet", "haiku"},
		// claude --help: --effort <level> (low, medium, high, xhigh, max)
		EffortArgs: []string{"--effort", "%s"},
		Efforts:    []string{"low", "medium", "high", "xhigh", "max"},
		UpdateCmd:  []string{"update"},
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
		Models:      []string{"gpt-5.5", "gpt-5-mini", "gpt-5"},
		// codex effort'u flag değil config anahtarı: -c model_reasoning_effort=X.
		// Geçerli değerler codex'in kendi hata mesajından: none, minimal, low,
		// medium, high, xhigh ("none" listelenmiyor — düşünmeyi kapatmak için
		// model seçimi daha doğru).
		EffortArgs: []string{"-c", "model_reasoning_effort=%s"},
		Efforts:    []string{"minimal", "low", "medium", "high", "xhigh"},
		UpdateCmd:  []string{"update"},
		AuthCmd:    []string{"login"},
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
		return nil, fmt.Errorf("global config parse: %w", err)
	}
	if cfg.Tools == nil {
		cfg.Tools = map[string]InstalledTool{}
	}
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
