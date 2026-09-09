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
)

var rootCmd = &cobra.Command{
	Use:     "cem [input]",
	Short:   "⚡ Compose · Execute · Multiplex — one command, many AIs",
	Version: version,
	Args:    cobra.ArbitraryArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
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

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Re-run the setup wizard",
	Run: func(cmd *cobra.Command, args []string) {
		PrintBanner(BannerCem)
		cfg, err := loadGlobalConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}
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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installation status",
	Run: func(cmd *cobra.Command, args []string) {
		PrintBanner(BannerCem)
		rc, err := LoadConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}
		ShowRoles(rc)
	},
}

// ─── init & execute ──────────────────────────────────────────────────────────

func init() {
	rootCmd.Flags().BoolVarP(&flagWrite, "write", "w", false, "use the writer AI")
	rootCmd.Flags().BoolVarP(&flagPair, "pair", "p", false, "pair: thinker → writer")
	rootCmd.Flags().StringVarP(&flagFile, "file", "f", "", "send the contents of a file")
	rootCmd.PersistentFlags().BoolVar(&rawOutput, "raw", false,
		"show the AI CLI output unfiltered (banners, tool steps, diffs)")

	rolesCmd.Flags().BoolVar(&rolesCmdHere, "here", false, "this project only (.cem.yaml)")

	rootCmd.AddCommand(rolesCmd)
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
		PrintBanner(BannerCem)
		fmt.Println(styleWarn.Render("  ⚡ İlk çalıştırma — sihirbaz başlatılıyor...\n"))
		if err := RunSetupWizard(rc.Global); err != nil {
			return nil, err
		}
		rc, err = LoadConfig()
		if err != nil {
			return nil, err
		}
	}
	return rc, nil
}

func readLine() string {
	var s string
	fmt.Scanln(&s)
	return strings.TrimSpace(s)
}
