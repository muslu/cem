package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	endpointHere   bool
	endpointKey    string
	endpointModelF string
	endpointTest   bool
	endpointList   bool
	endpointClear  bool
)

// endpointCmd — HTTP konuşan araçların (ollama, lmstudio, unsloth) adresi,
// anahtarı ve modeli. Bu araçlar kurulmaz: bir sunucu adresi verilir.
var endpointCmd = &cobra.Command{
	Use:   "endpoint [tool] [ip:port]",
	Short: "Show or change the address/key/model of HTTP model servers",
	Long: `  cem endpoint                            → show configured servers
  cem endpoint ollama 192.168.1.10:11434  → set the address
  cem endpoint ollama --model qwen3-coder → set the model
  cem endpoint unsloth --key sk-...       → set the API key (Bearer)
  cem endpoint ollama --test              → is the server up?
  cem endpoint ollama --modeller          → models the server offers
  cem endpoint --here lmstudio 10.0.0.5:1234  → this project only (.cem.yaml)
  cem endpoint ollama --sil                → drop the setting`,
	Args: cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		rc, err := LoadConfig()
		if err != nil {
			fmt.Println(styleError.Render("✗ " + err.Error()))
			os.Exit(1)
		}

		if len(args) == 0 {
			showEndpoints(rc)
			return
		}

		toolKey := strings.ToLower(args[0])
		if !isHTTPTool(toolKey) {
			fmt.Println(styleError.Render(fmt.Sprintf(
				L("✗ %s bir HTTP sunucusu değil", "✗ %s is not an HTTP server tool"), toolKey)))
			fmt.Println(styleDim.Render(L("  HTTP araçları: ", "  HTTP tools: ") + strings.Join(httpToolKeys(), ", ")))
			fmt.Println(styleDim.Render(L("  CLI araçlarının modeli: cem model <araç> <ad>",
				"  For CLI tools use: cem model <tool> <name>")))
			os.Exit(1)
		}

		switch {
		case endpointTest:
			testEndpoint(toolKey, rc)
			return
		case endpointList:
			listEndpointModels(toolKey, rc)
			return
		}

		ep := Endpoint{}
		if len(args) == 2 {
			ep.BaseURL = normalizeBaseURL(args[1])
		}
		ep.APIKey = endpointKey
		ep.Model = strings.TrimSpace(endpointModelF)

		if !endpointClear && ep.BaseURL == "" && ep.APIKey == "" && ep.Model == "" {
			showEndpoint(toolKey, rc)
			return
		}

		if endpointHere {
			setProjectEndpoint(rc, toolKey, ep, endpointClear)
			return
		}
		setGlobalEndpoint(rc, toolKey, ep, endpointClear)
	},
}

func init() {
	endpointCmd.Flags().BoolVar(&endpointHere, "here", false, "only this project (.cem.yaml)")
	endpointCmd.Flags().StringVar(&endpointKey, "key", "", "API key sent as a Bearer token")
	endpointCmd.Flags().StringVar(&endpointModelF, "model", "", "model name to run on that server")
	endpointCmd.Flags().BoolVar(&endpointTest, "test", false, "check whether the server answers")
	endpointCmd.Flags().BoolVar(&endpointList, "modeller", false, "list the models the server offers")
	endpointCmd.Flags().BoolVar(&endpointClear, "sil", false, "drop the stored setting")
	rootCmd.AddCommand(endpointCmd)
}

// httpToolKeys — sabit sırada HTTP araçları.
func httpToolKeys() []string {
	var out []string
	for _, k := range orderedToolKeys {
		if isHTTPTool(k) {
			out = append(out, k)
		}
	}
	return out
}

// Anahtar EKRANA BASILMAZ: sadece var/yok gösterilir. Config dosyası 0600
// olsa da terminal kaydı, ekran paylaşımı ve tmux geçmişi anahtarı sızdırır.
func keyState(ep Endpoint) string {
	if ep.APIKey == "" {
		return styleDim.Render(L("anahtar yok", "no key"))
	}
	return styleSuccess.Render(L("anahtar var", "key set"))
}

func showEndpoints(rc *ResolvedConfig) {
	fmt.Println()
	fmt.Println(styleBold.Render(L("  Model sunucuları", "  Model servers")))
	for _, k := range httpToolKeys() {
		ep := resolveEndpoint(k, rc)
		model, err := endpointModel(k, rc, ep)
		if err != nil {
			model = styleWarn.Render(L("model seçilmedi", "no model set"))
		}
		fmt.Printf("  %-10s %-26s %-22s %s\n", styleBold.Render(k),
			ep.BaseURL, model, keyState(ep))
	}
	fmt.Println()
	fmt.Println(styleDim.Render(L("  cem endpoint ollama 192.168.1.10:11434   adres ver",
		"  cem endpoint ollama 192.168.1.10:11434   set the address")))
	fmt.Println(styleDim.Render(L("  cem endpoint ollama --test               sunucuyu dene",
		"  cem endpoint ollama --test               probe the server")))
	fmt.Println()
}

func showEndpoint(toolKey string, rc *ResolvedConfig) {
	ep := resolveEndpoint(toolKey, rc)
	model, err := endpointModel(toolKey, rc, ep)
	if err != nil {
		model = L("(seçilmedi)", "(not set)")
	}
	fmt.Printf(L("  %s  adres: %s   model: %s   %s\n", "  %s  address: %s   model: %s   %s\n"),
		styleBold.Render(KnownTools[toolKey].Name), ep.BaseURL, model, keyState(ep))
}

func setGlobalEndpoint(rc *ResolvedConfig, toolKey string, yeni Endpoint, sil bool) {
	if rc.Global.Endpoints == nil {
		rc.Global.Endpoints = map[string]Endpoint{}
	}
	if sil {
		delete(rc.Global.Endpoints, toolKey)
	} else {
		// Verilmeyen alan silinmez: `--model X` yazan kişi adresini
		// kaybetmemeli.
		cur := rc.Global.Endpoints[toolKey]
		if yeni.BaseURL != "" {
			cur.BaseURL = yeni.BaseURL
		}
		if yeni.APIKey != "" {
			cur.APIKey = yeni.APIKey
		}
		if yeni.Model != "" {
			cur.Model = yeni.Model
		}
		rc.Global.Endpoints[toolKey] = cur
	}
	if err := saveGlobalConfig(rc.Global); err != nil {
		fmt.Println(styleError.Render("✗ " + err.Error()))
		os.Exit(1)
	}
	fmt.Printf("  %s %s\n", styleSuccess.Render("✓"), L("kaydedildi", "saved"))
	showEndpoint(toolKey, rc)
}

func setProjectEndpoint(rc *ResolvedConfig, toolKey string, yeni Endpoint, sil bool) {
	pc := rc.Project
	if pc == nil {
		pc = &ProjectConfig{}
	}
	if sil {
		delete(pc.Endpoints, toolKey)
	} else {
		if pc.Endpoints == nil {
			pc.Endpoints = map[string]Endpoint{}
		}
		cur := pc.Endpoints[toolKey]
		if yeni.BaseURL != "" {
			cur.BaseURL = yeni.BaseURL
		}
		if yeni.APIKey != "" {
			cur.APIKey = yeni.APIKey
		}
		if yeni.Model != "" {
			cur.Model = yeni.Model
		}
		pc.Endpoints[toolKey] = cur
	}
	// .cem.yaml tek başına anlamlı olsun diye roller de taşınır (cem model
	// ile aynı davranış).
	if pc.Roles == nil {
		r := rc.Global.Roles
		pc.Roles = &r
	}
	if err := SaveProjectConfig(pc); err != nil {
		fmt.Println(styleError.Render("✗ " + err.Error()))
		os.Exit(1)
	}
	fmt.Printf("  %s .cem.yaml → endpoints.%s\n", styleSuccess.Render("✓"), toolKey)
	showEndpoint(toolKey, rc)
}

func testEndpoint(toolKey string, rc *ResolvedConfig) {
	ep := resolveEndpoint(toolKey, rc)
	fmt.Printf(L("  %s deneniyor: %s\n", "  probing %s: %s\n"), styleBold.Render(toolKey), ep.BaseURL)
	modeller, err := probeEndpoint(toolKey, rc)
	if err != nil {
		fmt.Println(styleError.Render("  ✗ " + err.Error()))
		fmt.Println(styleDim.Render(L("    Sunucu ayakta mı, port doğru mu, güvenlik duvarı kapatıyor mu?",
			"    Is the server running, is the port right, is a firewall blocking it?")))
		os.Exit(1)
	}
	fmt.Println(styleSuccess.Render(fmt.Sprintf(
		L("  ✓ cevap verdi — %d model sunuyor", "  ✓ answered — offering %d models"), len(modeller))))
	if len(modeller) > 0 {
		fmt.Println(styleDim.Render("    " + strings.Join(modeller, ", ")))
	}
}

func listEndpointModels(toolKey string, rc *ResolvedConfig) {
	modeller, err := probeEndpoint(toolKey, rc)
	if err != nil {
		fmt.Println(styleError.Render("  ✗ " + err.Error()))
		os.Exit(1)
	}
	if len(modeller) == 0 {
		fmt.Println(styleWarn.Render(L("  sunucu hiç model bildirmedi — önce model indir/yükle",
			"  the server reported no models — pull/load one first")))
		return
	}
	for _, m := range modeller {
		fmt.Println("  " + m)
	}
	fmt.Println(styleDim.Render(fmt.Sprintf("  cem endpoint %s --model %s", toolKey, modeller[0])))
}
