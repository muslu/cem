package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagWrite bool
	flagPair  bool
	flagFile  string
	// rawOutput — AI CLI çıktısını filtresiz göster (banner, iç loglar dahil).
	rawOutput bool
	// noCache — bu çalıştırmada önbelleği hem okuma hem yazma dışı bırak.
	noCache bool
)

var rootCmd = &cobra.Command{
	Use:     "cem [input]",
	Short:   "⚡ Compose · Execute · Multiplex — one command, many AIs",
	Version: version,
	Args:    cobra.ArbitraryArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Makine-okur çıktı isteniyorsa HİÇBİR ŞEY basılmaz: "yeni sürüm var"
		// bildirimi ya da "araç güncellemesi arka planda" satırı JSON'un
		// başına eklendiğinde çağıran taraf (eklenti GUI'si, betik) çıktıyı
		// ayrıştıramıyor.
		if machineReadableOutput() {
			return
		}
		// OpenSourceNotice sadece banner ekranında (aşağıda) — her komutun
		// başına basılınca asıl çıktıyı bastırıyordu.
		checkUpdateNotice()
		maybeAutoUpdateTools()
	},
	// Banner her çalıştırmada değil sadece help'te görünsün
	// Kullanım sırasında kısa prefix yeterli
	Run: func(cmd *cobra.Command, args []string) {
		rc, err := loadAndCheckSetup()
		if err != nil {
			os.Exit(1)
		}

		// Config kaynağını göster
		ShowConfigSource(rc)

		// Input: pipe > -f > args
		input := ReadStdin()

		if flagFile != "" {
			data, err := os.ReadFile(flagFile)
			if err != nil {
				fmt.Println(styleError.Render(L("✗ Dosya okunamadı: ", "✗ Cannot read file: ") + err.Error()))
				os.Exit(1)
			}
			if len(args) > 0 {
				input = strings.Join(args, " ") + "\n\n" + string(data)
			} else {
				input = string(data)
			}
		} else if len(args) > 0 {
			if input != "" {
				input = strings.Join(args, " ") + "\n\n" + input
			} else {
				input = strings.Join(args, " ")
			}
		}

		if input == "" {
			OpenSourceNotice()
			PrintBanner(BannerCem)
			cmd.Help()
			return
		}

		// Araçlar bu dizinde dosya oluşturacak — yanlış dizinde olduğunu
		// çıktıyı gördükten sonra fark etmek yerine önce sor.
		if !ensureWorkdirTrusted(rc.Global) {
			os.Exit(1)
		}

		mode := ModeThink
		if flagPair {
			mode = ModePair
		} else if flagWrite {
			mode = ModeWrite
		}

		runErr := Run(input, mode, rc)

		// History (rol: pair → thinker+writer, write → writer, think → thinker)
		roles := rc.ActiveRoles()
		var role string
		switch mode {
		case ModeWrite:
			role = roles.Writer
		case ModePair:
			role = roles.Thinker + "+" + roles.Writer
		default:
			role = roles.Thinker
		}
		exit := 0
		if runErr != nil {
			exit = 1
		}
		AppendHistory(mode, role, input, exit)

		if runErr != nil {
			os.Exit(1)
		}
	},
}

// ─── cem roles ───────────────────────────────────────────────────────────────

var rolesCmdHere bool

var rolesCmd = &cobra.Command{
	Use:   "roles [thinker] [writer]",
	Short: "Show or change roles (thinker / writer)",
	Long: `  cem roles                    → show current roles
  cem roles claude agy         → set global
  cem roles claude             → only thinker
  cem roles --here claude agy  → project-only (.cem.yaml)`,
	Args: cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		rc, err := LoadConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		if len(args) == 0 {
			PrintBanner(BannerCem)
			ShowRoles(rc)
			return
		}

		current := rc.ActiveRoles()
		if len(args) >= 1 {
			current.Thinker = args[0]
		}
		if len(args) >= 2 {
			current.Writer = args[1]
		}

		if rolesCmdHere {
			pc := &ProjectConfig{Roles: &current}
			if err := SaveProjectConfig(pc); err != nil {
				fmt.Println(styleError.Render("✗ " + err.Error()))
				os.Exit(1)
			}
			fmt.Println(styleSuccess.Render("✓ Proje rolleri güncellendi → .cem.yaml"))
		} else {
			rc.Global.Roles = current
			applyRoleDefaults(rc.Global, rc.Global.Roles.Thinker, rc.Global.Roles.Writer)
			if err := saveGlobalConfig(rc.Global); err != nil {
				fmt.Println(styleError.Render("✗ " + err.Error()))
				os.Exit(1)
			}
			fmt.Println(styleSuccess.Render("✓ Global roller güncellendi → ~/.cem/config.yaml"))
		}

		fmt.Printf("  🧠 Thinker → %s\n", styleBold.Render(current.Thinker))
		fmt.Printf("  ✍️  Writer  → %s\n", styleBold.Render(current.Writer))
		fmt.Println()
		fmt.Println(styleDim.Render("  cem roles  →  kontrol et"))
	},
}

// ─── cem setup ───────────────────────────────────────────────────────────────

// setupOpts — bayraklı (etkileşimsiz) kurulum. Sihirbaz TTY istiyor; eklenti,
// kurulum betiği ve CI'da TTY yok. Bayraklar verildiğinde soru sorulmaz.
var setupOpts SetupOptions

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Re-run the setup wizard (or configure it with flags)",
	Long: `  cem setup                                     → interactive wizard
  cem setup --thinker gpt --writer claude       → no questions asked
  cem setup --thinker ollama --endpoint-thinker 192.168.1.10:11434             --model-thinker qwen3-coder --writer claude
  cem setup --lang en                           → only change the language

Flags let a GUI or a script configure cem where no terminal is available
(the IDE plugin, an installer, CI). cem stays the only writer of the config.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadGlobalConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		if setupOpts.Given() {
			if err := ApplySetup(cfg, setupOpts); err != nil {
				fmt.Println(styleError.Render("✗ " + err.Error()))
				os.Exit(1)
			}
			fmt.Println(styleSuccess.Render(fmt.Sprintf(
				L("  ✓ kurulum kaydedildi: düşünen %s · yazan %s",
					"  ✓ setup saved: thinker %s · writer %s"),
				cfg.Roles.Thinker, cfg.Roles.Writer)))
			return
		}

		PrintBanner(BannerCem)
		if err := RunSetupWizard(cfg); err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}
	},
}

// ─── cem init ────────────────────────────────────────────────────────────────

var initCmd = &cobra.Command{
	Use:   "init [thinker] [writer]",
	Short: "Create .cem.yaml for this project",
	Long: `  cem init                 → interactive wizard
  cem init claude agy      → direkt oluştur`,
	Args: cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		rc, err := LoadConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		global := rc.Global.Roles
		var t, w string

		if len(args) >= 2 {
			t, w = args[0], args[1]
		} else {
			fmt.Println()
			fmt.Println(styleBold.Render("  Proje rolleri") +
				styleDim.Render("  (boş bırak → global değeri kullanılır)"))
			fmt.Println()

			toolOrder := orderedToolKeys
			for i, key := range toolOrder {
				meta := KnownTools[key]
				installed := ""
				if _, ok := rc.Global.Tools[key]; ok {
					installed = styleSuccess.Render(" ✓")
				}
				fmt.Printf("  [%d]  %-12s %s%s\n",
					i+1, styleBold.Render(meta.Name),
					styleDim.Render(meta.Description), installed)
			}
			fmt.Println()

			t = pickToolWithDefault("  🧠 Thinker", toolOrder, rc.Global, global.Thinker)
			w = pickToolWithDefault("  ✍️  Writer ", toolOrder, rc.Global, global.Writer)
		}

		pc := &ProjectConfig{Roles: &Roles{Thinker: t, Writer: w}}

		// Model seçimi — sadece interaktif modda (len(args) < 2) ve autoYes'siz
		if len(args) < 2 && !autoYes {
			pc.Models = map[string]string{}
			fmt.Println()
			if m := pickProjectModel(t, "🧠 thinker", rc.Global); m != "" {
				pc.Models[t] = m
			}
			if e := pickProjectEffort(t, "🧠 thinker", rc.Global); e != "" {
				if pc.Efforts == nil {
					pc.Efforts = map[string]string{}
				}
				pc.Efforts[t] = e
			}
			if w != t {
				if m := pickProjectModel(w, "✍️  writer", rc.Global); m != "" {
					pc.Models[w] = m
				}
				if e := pickProjectEffort(w, "✍️  writer", rc.Global); e != "" {
					if pc.Efforts == nil {
						pc.Efforts = map[string]string{}
					}
					pc.Efforts[w] = e
				}
			}
			if len(pc.Models) == 0 {
				pc.Models = nil // YAML'da boş key görünmesin
			}
		}

		if err := SaveProjectConfig(pc); err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(styleSuccess.Render("✓ .cem.yaml oluşturuldu"))
		fmt.Printf("  🧠 Thinker → %s", styleBold.Render(t))
		if m, ok := pc.Models[t]; ok && m != "" {
			fmt.Printf("  (model: %s)", styleBold.Render(m))
		}
		fmt.Println()
		fmt.Printf("  ✍️  Writer  → %s", styleBold.Render(w))
		if m, ok := pc.Models[w]; ok && m != "" {
			fmt.Printf("  (model: %s)", styleBold.Render(m))
		}
		fmt.Println()
		fmt.Println()
		fmt.Println(styleDim.Render("  Bu dizinden çalışırken proje config geçerli olur."))
		fmt.Println(styleDim.Render("  cem roles  →  aktif rolleri gör"))
	},
}

// pickToolWithDefault — pickTool gibi numarayla seçtirir ama boş Enter'da
// 'fallback' anahtarını döndürür. cem init için: kullanıcı bir sayı yazmadan
// Enter'a basarsa global rolü kullansın.
func pickToolWithDefault(label string, toolOrder []string, cfg *GlobalConfig, fallback string) string {
	fmt.Printf("%s [1-%d, Enter=%s]: ",
		styleBold.Render(label), len(toolOrder), styleBold.Render(fallback))
	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			return fallback
		}
		idx, err := strconv.Atoi(input)
		if err == nil && idx >= 1 && idx <= len(toolOrder) {
			key := toolOrder[idx-1]
			meta := KnownTools[key]
			suffix := ""
			if _, ok := cfg.Tools[key]; ok {
				suffix = styleSuccess.Render(" (kurulu)")
			}
			fmt.Printf("  → %s%s\n", styleBold.Render(meta.Name), suffix)
			return key
		}
		fmt.Printf("  %s [1-%d, Enter=%s]: ",
			styleWarn.Render("Geçersiz, tekrar gir"),
			len(toolOrder), styleBold.Render(fallback))
	}
}

// pickProjectModel — cem init için: kullanıcıya proje-spesifik model seçtirir.
// Mevcut global model (varsa) varsayılan olarak sunulur. Boş seçim = global'i
// devral (proje override yok).
// pickProjectEffort — 'cem init' sırasında proje-bazlı düşünme seviyesi.
// Boş dönerse .cem.yaml'a yazılmaz, global geçerli kalır.
func pickProjectEffort(toolKey, label string, global *GlobalConfig) string {
	meta, ok := KnownTools[toolKey]
	if !ok || len(meta.Efforts) == 0 || len(meta.EffortArgs) == 0 {
		return ""
	}
	current := ""
	if t, ok := global.Tools[toolKey]; ok {
		current = t.Effort
	}
	if current == "" {
		current = "CLI default"
	}
	fmt.Printf("  %s · %s için proje düşünme seviyesi (global: %s):\n",
		styleBold.Render(label), styleBold.Render(meta.Name), styleDim.Render(current))
	for i, e := range meta.Efforts {
		fmt.Printf("      [%d] %s%s\n", i+1, e, styleDim.Render(effortHint(e)))
	}
	fmt.Printf("      [0] global (no override)\n")
	fmt.Print(L("  Seçim: ", "  Choice: "))
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(resp)
	if resp == "" || resp == "0" {
		return ""
	}
	idx, err := strconv.Atoi(resp)
	if err != nil || idx < 1 || idx > len(meta.Efforts) {
		fmt.Println(styleDim.Render(L("  geçersiz, global kullanılacak", "  invalid, the global value will be used")))
		return ""
	}
	return meta.Efforts[idx-1]
}

func pickProjectModel(toolKey, label string, global *GlobalConfig) string {
	meta, ok := KnownTools[toolKey]
	if !ok || meta.ModelFlag == "" || len(meta.Models) == 0 {
		return ""
	}
	current := ""
	if t, ok := global.Tools[toolKey]; ok {
		current = t.Model
	}
	if current == "" {
		current = "CLI default"
	}
	fmt.Printf("  %s · %s için proje modeli (global: %s):\n",
		styleBold.Render(label), styleBold.Render(meta.Name),
		styleDim.Render(current))
	for i, m := range meta.Models {
		fmt.Printf("      [%d] %s\n", i+1, m)
	}
	fmt.Printf("      [%d] custom\n", len(meta.Models)+1)
	fmt.Printf("      [0] global (no override)\n")
	fmt.Print(L("  Seçim: ", "  Choice: "))
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(resp)
	switch resp {
	case "", "0":
		return ""
	}
	idx, err := strconv.Atoi(resp)
	if err != nil || idx < 1 || idx > len(meta.Models)+1 {
		fmt.Println(styleDim.Render(L("  geçersiz, global kullanılacak", "  invalid, the global value will be used")))
		return ""
	}
	if idx == len(meta.Models)+1 {
		fmt.Print(L("  Model adı: ", "  Model name: "))
		line, _ := reader.ReadString('\n')
		return strings.TrimSpace(line)
	}
	return meta.Models[idx-1]
}

// ─── cem status ──────────────────────────────────────────────────────────────

var statusJSONFlag bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installation status (--json for machines)",
	Run: func(cmd *cobra.Command, args []string) {
		rc, err := LoadConfig()
		if err != nil {
			// JSON isteyen çağıran (GUI) için hata da JSON olmalı; yoksa
			// ayrıştırma tarafında "beklenmeyen çıktı" olarak görünür.
			if statusJSONFlag {
				fmt.Printf("{\"error\":%q}\n", err.Error())
				os.Exit(1)
			}
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}
		if statusJSONFlag {
			// Banner/ANSI YOK: çıktı ayrıştırılacak.
			if err := printStatusJSON(rc); err != nil {
				os.Exit(1)
			}
			return
		}
		PrintBanner(BannerCem)
		ShowRoles(rc)
	},
}

// ─── init & execute ──────────────────────────────────────────────────────────

func init() {
	rootCmd.Flags().BoolVarP(&flagWrite, "write", "w", false, "use the writer AI")
	rootCmd.Flags().BoolVarP(&flagPair, "pair", "p", false, "pair: thinker → writer")
	rootCmd.Flags().StringVarP(&flagFile, "file", "f", "", "send the contents of a file")
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false,
		"ignore the cache: ask the AI again and overwrite the stored answer")
	rootCmd.PersistentFlags().BoolVar(&rawOutput, "raw", false,
		"show the AI CLI output unfiltered (banners, tool steps, diffs)")

	rolesCmd.Flags().BoolVar(&rolesCmdHere, "here", false, "this project only (.cem.yaml)")

	rootCmd.AddCommand(rolesCmd)
	setupCmd.Flags().StringVar(&setupOpts.Thinker, "thinker", "", "tool that thinks")
	setupCmd.Flags().StringVar(&setupOpts.Writer, "writer", "", "tool that writes the code")
	setupCmd.Flags().StringVar(&setupOpts.ModelThinker, "model-thinker", "", "model for the thinker")
	setupCmd.Flags().StringVar(&setupOpts.ModelWriter, "model-writer", "", "model for the writer")
	setupCmd.Flags().StringVar(&setupOpts.EffortThinker, "effort-thinker", "", "reasoning effort for the thinker")
	setupCmd.Flags().StringVar(&setupOpts.EffortWriter, "effort-writer", "", "reasoning effort for the writer")
	setupCmd.Flags().StringVar(&setupOpts.EndpointThinker, "endpoint-thinker", "", "server address if the thinker is an HTTP tool")
	setupCmd.Flags().StringVar(&setupOpts.EndpointWriter, "endpoint-writer", "", "server address if the writer is an HTTP tool")
	setupCmd.Flags().StringVar(&setupOpts.Lang, "lang", "", "interface language: tr | en")
	statusCmd.Flags().BoolVar(&statusJSONFlag, "json", false, "machine-readable output (no banner, no colours)")
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
}

func loadAndCheckSetup() (*ResolvedConfig, error) {
	rc, err := LoadConfig()
	if err != nil {
		fmt.Println(styleError.Render("✗ Config yüklenemedi: " + err.Error()))
		return nil, err
	}
	if !rc.Global.Setup || rc.Global.Roles.Thinker == "" {
		// Setup ZORUNLU. Yarım yapılandırmayla çalıştırmak, kullanıcının
		// seçmediği bir araca istek göndermek (ve faturasını ödemek) demek.
		//
		// Terminal yoksa sihirbaz da çalıştırılamaz: eklentiden, pipe'tan ya
		// da CI'dan gelen çağrıda sorular ekrana basılır, cevap gelmez ve
		// akış körlemesine ilerler. O durumda hata verip ne yapılacağını
		// söylemek tek doğru davranış.
		PrintBanner(BannerCem)
		fmt.Println(styleWarn.Render(L("  ⚡ cem henüz yapılandırılmadı — kurulum yapılmadan çalışmaz.",
			"  ⚡ cem is not configured yet — it does not run before setup.")))
		fmt.Println()

		if !isInteractiveStdin() {
			fmt.Println(styleError.Render(L("✗ Terminal yok, sihirbaz çalıştırılamaz.",
				"✗ No terminal available, the wizard cannot run.")))
			fmt.Println(styleDim.Render(L("    Bir terminalde bir kez çalıştır:  cem setup",
				"    Run this once in a terminal:  cem setup")))
			return nil, fmt.Errorf("%s", L("kurulum yapılmadı", "setup not completed"))
		}

		if !askYN(L("  Sihirbazı şimdi çalıştıralım mı?", "  Run the setup wizard now?")) {
			fmt.Println(styleDim.Render(L("    Hazır olduğunda:  cem setup", "    When you are ready:  cem setup")))
			return nil, fmt.Errorf("%s", L("kurulum yapılmadı", "setup not completed"))
		}

		if err := RunSetupWizard(rc.Global); err != nil {
			return nil, err
		}
		rc, err = LoadConfig()
		if err != nil {
			return nil, err
		}
		// Sihirbaz yarıda kesilmiş olabilir (Ctrl+C, boş seçim): kaydı
		// doğrulamadan devam etmek aynı yarım-yapılandırma sorununa döner.
		if !rc.Global.Setup || rc.Global.Roles.Thinker == "" {
			fmt.Println(styleError.Render(L("✗ Kurulum tamamlanmadı.", "✗ Setup was not completed.")))
			return nil, fmt.Errorf("%s", L("kurulum yapılmadı", "setup not completed"))
		}
	}
	return rc, nil
}

func readLine() string {
	var s string
	fmt.Scanln(&s)
	return strings.TrimSpace(s)
}
