package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var effortHere bool

// effortCmd — düşünme (reasoning) seviyesini görüntüle / değiştir.
// Seviye sadece destekleyen araçlara uygulanır (claude --effort,
// codex -c model_reasoning_effort); agy/cursor için flag yok.
var effortCmd = &cobra.Command{
	Use:   "effort [tool] [level]",
	Short: "Show or change reasoning effort (düşünme seviyesi)",
	Long: `  cem effort                    → mevcut seviyeleri göster
  cem effort gpt high           → global ayarla
  cem effort claude xhigh       → global ayarla
  cem effort --here gpt xhigh   → sadece bu proje (.cem.yaml)
  cem effort gpt default        → seçimi kaldır (CLI kendi seçer)`,
	Args: cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		rc, err := LoadConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		if len(args) == 0 {
			showEfforts(rc)
			return
		}

		toolKey := args[0]
		meta, ok := KnownTools[toolKey]
		if !ok {
			fmt.Println(styleError.Render("✗ bilinmeyen araç: " + toolKey))
			fmt.Println(styleDim.Render("  geçerli: " + strings.Join(orderedToolKeys, ", ")))
			os.Exit(1)
		}
		if len(meta.Efforts) == 0 || len(meta.EffortArgs) == 0 {
			fmt.Println(styleWarn.Render(fmt.Sprintf(
				"  ⚠ %s CLI'sı düşünme seviyesi seçimini desteklemiyor", meta.Name)))
			os.Exit(1)
		}

		if len(args) == 1 {
			fmt.Printf("  %s geçerli seviyeler: %s\n",
				styleBold.Render(meta.Name), strings.Join(meta.Efforts, ", "))
			fmt.Println(styleDim.Render("  ayarlamak için: cem effort " + toolKey + " " + meta.Efforts[len(meta.Efforts)-1]))
			return
		}

		level := strings.ToLower(strings.TrimSpace(args[1]))
		if level != "default" && !validEffort(meta, level) {
			fmt.Println(styleError.Render("✗ geçersiz seviye: " + level))
			fmt.Println(styleDim.Render("  geçerli: " + strings.Join(meta.Efforts, ", ") + ", default"))
			os.Exit(1)
		}
		if level == "default" {
			level = ""
		}

		if effortHere {
			setProjectEffort(rc, toolKey, level)
			return
		}
		setGlobalEffort(rc, toolKey, level, meta)
	},
}

func validEffort(meta ToolMeta, level string) bool {
	for _, e := range meta.Efforts {
		if e == level {
			return true
		}
	}
	return false
}

func setGlobalEffort(rc *ResolvedConfig, toolKey, level string, meta ToolMeta) {
	t := rc.Global.Tools[toolKey]
	t.Effort = level
	if rc.Global.Tools == nil {
		rc.Global.Tools = map[string]InstalledTool{}
	}
	rc.Global.Tools[toolKey] = t
	if err := saveGlobalConfig(rc.Global); err != nil {
		fmt.Println(styleError.Render("✗ " + err.Error()))
		os.Exit(1)
	}
	if level == "" {
		fmt.Printf("  %s %s düşünme seviyesi: CLI default\n",
			styleSuccess.Render("✓"), styleBold.Render(meta.Name))
		return
	}
	fmt.Printf("  %s %s düşünme seviyesi: %s%s\n",
		styleSuccess.Render("✓"), styleBold.Render(meta.Name),
		styleBold.Render(level), styleDim.Render(effortHint(level)))
}

func setProjectEffort(rc *ResolvedConfig, toolKey, level string) {
	pc := rc.Project
	if pc == nil {
		pc = &ProjectConfig{}
	}
	if level == "" {
		delete(pc.Efforts, toolKey)
	} else {
		if pc.Efforts == nil {
			pc.Efforts = map[string]string{}
		}
		pc.Efforts[toolKey] = level
	}
	// .cem.yaml sadece roles yoksa da geçerli olsun diye global rolleri taşı.
	if pc.Roles == nil {
		r := rc.Global.Roles
		pc.Roles = &r
	}
	if err := SaveProjectConfig(pc); err != nil {
		fmt.Println(styleError.Render("✗ " + err.Error()))
		os.Exit(1)
	}
	fmt.Printf("  %s .cem.yaml → efforts.%s = %s\n",
		styleSuccess.Render("✓"), toolKey, styleBold.Render(orDefault(level)))
}

func orDefault(s string) string {
	if s == "" {
		return "(kaldırıldı)"
	}
	return s
}

// showEfforts — kurulu araçların aktif seviyelerini ve kaynağını listeler.
func showEfforts(rc *ResolvedConfig) {
	fmt.Println()
	fmt.Println(styleBold.Render("  Düşünme seviyeleri"))
	fmt.Println()
	for _, key := range orderedToolKeys {
		meta := KnownTools[key]
		if len(meta.Efforts) == 0 || len(meta.EffortArgs) == 0 {
			fmt.Printf("  %s %-8s %s\n", styleDim.Render("○"), key,
				styleDim.Render("seviye seçimi desteklenmiyor"))
			continue
		}
		active := resolveEffort(key, rc)
		src := "global"
		if rc.Project != nil && rc.Project.Efforts != nil {
			if _, ok := rc.Project.Efforts[key]; ok {
				src = ".cem.yaml"
			}
		}
		if active == "" {
			active, src = "default", "CLI"
		}
		fmt.Printf("  %s %-8s %-9s %s\n",
			styleSuccess.Render("✓"), key, styleBold.Render(active),
			styleDim.Render(src+"  ("+strings.Join(meta.Efforts, ", ")+")"))
	}
	fmt.Println()
	fmt.Println(styleDim.Render("  cem effort gpt xhigh        → global"))
	fmt.Println(styleDim.Render("  cem effort --here gpt high  → sadece bu proje"))
	fmt.Println()
}

func init() {
	effortCmd.Flags().BoolVar(&effortHere, "here", false, "sadece bu proje (.cem.yaml)")
	rootCmd.AddCommand(effortCmd)
}
