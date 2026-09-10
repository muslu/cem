package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var modelHere bool

// modelCmd — bir aracın kullanacağı modeli görüntüle / değiştir.
// 'cem effort' ile simetrik: aynı öncelik sırası (.cem.yaml > global > CLI).
var modelCmd = &cobra.Command{
	Use:   "model [tool] [name]",
	Short: "Show or change the model per tool",
	Long: `  cem model                       → show current models
  cem model gpt                   → list known models for that tool
  cem model gpt gpt-5.6-terra     → set globally
  cem model --here claude sonnet  → this project only (.cem.yaml)
  cem model gpt default           → clear the choice (the CLI decides)`,
	Args: cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		rc, err := LoadConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		if len(args) == 0 {
			showModels(rc)
			return
		}

		toolKey := args[0]
		meta, ok := KnownTools[toolKey]
		if !ok {
			fmt.Println(styleError.Render(L("✗ bilinmeyen araç: ", "✗ unknown tool: ") + toolKey))
			fmt.Println(styleDim.Render(L("  geçerli: ", "  valid: ") + strings.Join(orderedToolKeys, ", ")))
			os.Exit(1)
		}

		if len(args) == 1 {
			fmt.Printf(L("  %s bilinen modeller: %s\n", "  %s known models: %s\n"),
				styleBold.Render(meta.Name), strings.Join(meta.Models, ", "))
			fmt.Println(styleDim.Render(L("  (liste öneri — CLI'nın kabul ettiği başka bir ad da yazabilirsin)",
				"  (suggestions only — any name the CLI accepts also works)")))
			if len(meta.Models) > 0 {
				fmt.Println(styleDim.Render("  cem model " + toolKey + " " + meta.Models[0]))
			}
			return
		}

		name := strings.TrimSpace(args[1])
		if name == "default" {
			name = ""
		}
		// Model adı serbest: sağlayıcılar sık sık yeni ad yayınlıyor, sabit
		// listeye kilitlemek kullanıcıyı config dosyasına gitmeye zorlar.
		if meta.ModelFlag == "" && name != "" {
			fmt.Println(styleWarn.Render(fmt.Sprintf(
				L("  ⚠ %s CLI'sının --model bayrağı yok — seçim kaydedilir ama çalıştırmaya etki etmez",
					"  ⚠ %s CLI has no --model flag — the choice is stored but has no runtime effect"),
				meta.Name)))
		}

		if modelHere {
			setProjectModel(rc, toolKey, name)
			return
		}
		setGlobalModel(rc, toolKey, name, meta)
	},
}

func setGlobalModel(rc *ResolvedConfig, toolKey, name string, meta ToolMeta) {
	if rc.Global.Tools == nil {
		rc.Global.Tools = map[string]InstalledTool{}
	}
	t := rc.Global.Tools[toolKey]
	t.Model = name
	rc.Global.Tools[toolKey] = t
	if err := saveGlobalConfig(rc.Global); err != nil {
		fmt.Println(styleError.Render("✗ " + err.Error()))
		os.Exit(1)
	}
	if name == "" {
		fmt.Printf(L("  %s %s modeli: CLI default\n", "  %s %s model: CLI default\n"),
			styleSuccess.Render("✓"), styleBold.Render(meta.Name))
		return
	}
	fmt.Printf(L("  %s %s modeli: %s\n", "  %s %s model: %s\n"),
		styleSuccess.Render("✓"), styleBold.Render(meta.Name), styleBold.Render(name))
}

func setProjectModel(rc *ResolvedConfig, toolKey, name string) {
	pc := rc.Project
	if pc == nil {
		pc = &ProjectConfig{}
	}
	if name == "" {
		delete(pc.Models, toolKey)
	} else {
		if pc.Models == nil {
			pc.Models = map[string]string{}
		}
		pc.Models[toolKey] = name
	}
	// .cem.yaml tek başına anlamlı olsun diye roller de taşınır.
	if pc.Roles == nil {
		r := rc.Global.Roles
		pc.Roles = &r
	}
	if err := SaveProjectConfig(pc); err != nil {
		fmt.Println(styleError.Render("✗ " + err.Error()))
		os.Exit(1)
	}
	fmt.Printf("  %s .cem.yaml → models.%s = %s\n",
		styleSuccess.Render("✓"), toolKey, styleBold.Render(orDefault(name)))
}

// showModels — kurulu araçların aktif modeli ve ayarın nereden geldiği.
func showModels(rc *ResolvedConfig) {
	fmt.Println()
	fmt.Println(styleBold.Render(L("  Modeller", "  Models")))
	fmt.Println()
	for _, key := range orderedToolKeys {
		// HTTP sunucularının modeli endpoint ayarında duruyor; burada
		// göstermek iki ayrı doğruluk kaynağı gibi görünürdü.
		if isHTTPTool(key) {
			continue
		}
		meta := KnownTools[key]
		active := resolveModel(key, rc)
		src := "global"
		if rc.Project != nil && rc.Project.Models != nil {
			if _, ok := rc.Project.Models[key]; ok {
				src = ".cem.yaml"
			}
		}
		if active == "" {
			active, src = "default", "CLI"
		}
		mark := styleSuccess.Render("✓")
		note := src
		if meta.ModelFlag == "" {
			mark = styleDim.Render("○")
			note = L("--model bayrağı yok", "no --model flag")
		}
		fmt.Printf("  %s %-8s %-16s %s\n", mark, key, styleBold.Render(active), styleDim.Render(note))
	}
	fmt.Println()
	fmt.Println(styleDim.Render("  cem model gpt gpt-5.6-terra    → global"))
	fmt.Println(styleDim.Render(L("  cem model --here claude sonnet → sadece bu proje",
		"  cem model --here claude sonnet → this project only")))
	fmt.Println()
}

func init() {
	modelCmd.Flags().BoolVar(&modelHere, "here", false, "this project only (.cem.yaml)")
	rootCmd.AddCommand(modelCmd)
}
