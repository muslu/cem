package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var fastHere bool

// fastCmd — hızlı modu görüntüle / değiştir.
//
// Hızlı mod, aracın kullanıcı ayarlarını (hook'lar, izin kuralları, MCP
// sunucuları) yüklemesini atlar. Ölçülen fark büyük — aynı görev için 124s
// yerine 8s — ama bedeli var, o yüzden varsayılan kapalı ve komut ne
// atlandığını açıkça yazıyor.
var fastCmd = &cobra.Command{
	Use:   "fast [tool] [on|off]",
	Short: "Skip the tool's user settings for speed",
	Long: `  cem fast                → which tools run in fast mode
  cem fast claude on      → enable globally
  cem fast --here claude on → this project only (.cem.yaml)
  cem fast claude off     → disable

  Fast mode skips the tool's user settings: hooks, permission rules and MCP
  servers are not loaded, and file edits are auto-approved. Measured on
  claude 2.1.266 with the same task: 124s normally, 8s in fast mode — the
  difference is the hook/plugin setup starting up on every single call.

  Turn it on when the tool is only writing code for you. Leave it off if you
  rely on your own hooks or deny-rules while it works.`,
	Args: cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		rc, err := LoadConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		if len(args) == 0 {
			showFast(rc)
			return
		}

		toolKey := args[0]
		meta, ok := KnownTools[toolKey]
		if !ok {
			fmt.Println(styleError.Render(L("✗ bilinmeyen araç: ", "✗ unknown tool: ") + toolKey))
			fmt.Println(styleDim.Render(L("  geçerli: ", "  valid: ") + strings.Join(orderedToolKeys, ", ")))
			os.Exit(1)
		}
		if len(meta.FastArgs) == 0 {
			fmt.Println(styleWarn.Render(fmt.Sprintf(
				L("  ⚠ %s için hızlı mod tanımlı değil", "  ⚠ fast mode is not defined for %s"),
				meta.Name)))
			os.Exit(1)
		}

		if len(args) == 1 {
			fmt.Printf(L("  %s hızlı mod: %s\n", "  %s fast mode: %s\n"),
				styleBold.Render(meta.Name), onOff(resolveFast(toolKey, rc)))
			return
		}

		var on bool
		switch strings.ToLower(args[1]) {
		case "on", "true", "1", "aç", "ac", "evet":
			on = true
		case "off", "false", "0", "kapat", "hayır", "hayir":
			on = false
		default:
			fmt.Println(styleError.Render(L("✗ geçersiz değer — on veya off",
				"✗ invalid value — use on or off")))
			os.Exit(1)
		}

		if fastHere {
			setProjectFast(rc, toolKey, on)
		} else {
			setGlobalFast(rc, toolKey, on, meta)
		}
		if on {
			fmt.Println(styleDim.Render(L(
				"    hook'lar, izin kuralları ve MCP sunucuları artık yüklenmiyor;",
				"    hooks, permission rules and MCP servers are no longer loaded;")))
			fmt.Println(styleDim.Render(L(
				"    dosya düzenlemeleri otomatik onaylanıyor.",
				"    file edits are auto-approved.")))
		}
	},
}

func onOff(v bool) string {
	if v {
		return styleSuccess.Render("on")
	}
	return styleDim.Render("off")
}

func setGlobalFast(rc *ResolvedConfig, toolKey string, on bool, meta ToolMeta) {
	if rc.Global.Tools == nil {
		rc.Global.Tools = map[string]InstalledTool{}
	}
	t := rc.Global.Tools[toolKey]
	t.Fast = &on
	rc.Global.Tools[toolKey] = t
	if err := saveGlobalConfig(rc.Global); err != nil {
		fmt.Println(styleError.Render("✗ " + err.Error()))
		os.Exit(1)
	}
	fmt.Printf(L("  %s %s hızlı mod: %s\n", "  %s %s fast mode: %s\n"),
		styleSuccess.Render("✓"), styleBold.Render(meta.Name), onOff(on))
}

func setProjectFast(rc *ResolvedConfig, toolKey string, on bool) {
	pc := rc.Project
	if pc == nil {
		pc = &ProjectConfig{}
	}
	if pc.Fast == nil {
		pc.Fast = map[string]bool{}
	}
	pc.Fast[toolKey] = on
	if pc.Roles == nil {
		r := rc.Global.Roles
		pc.Roles = &r
	}
	if err := SaveProjectConfig(pc); err != nil {
		fmt.Println(styleError.Render("✗ " + err.Error()))
		os.Exit(1)
	}
	fmt.Printf("  %s .cem.yaml → fast.%s = %v\n", styleSuccess.Render("✓"), toolKey, on)
}

func showFast(rc *ResolvedConfig) {
	fmt.Println()
	fmt.Println(styleBold.Render(L("  Hızlı mod", "  Fast mode")))
	fmt.Println()
	for _, key := range orderedToolKeys {
		meta := KnownTools[key]
		if len(meta.FastArgs) == 0 {
			fmt.Printf("  %s %-8s %s\n", styleDim.Render("○"), key,
				styleDim.Render(L("tanımlı değil", "not defined")))
			continue
		}
		fmt.Printf("  %s %-8s %s\n", styleSuccess.Render("✓"), key, onOff(resolveFast(key, rc)))
	}
	fmt.Println()
	fmt.Println(styleDim.Render(L(
		"  Açıkken: hook / izin kuralı / MCP yüklenmez, dosya düzenlemeleri otomatik onaylanır.",
		"  When on: hooks / permission rules / MCP are not loaded, file edits are auto-approved.")))
	fmt.Println(styleDim.Render(L("  cem fast claude on", "  cem fast claude on")))
	fmt.Println()
}

func init() {
	fastCmd.Flags().BoolVar(&fastHere, "here", false, "this project only (.cem.yaml)")
	rootCmd.AddCommand(fastCmd)
}
