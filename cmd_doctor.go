package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose installation health",
	Long: `  cem doctor   → installed tools, PATH, config, binary locations
                  ve roller tutarlılığı için tanı raporu.`,
	Run: func(cmd *cobra.Command, args []string) {
		PrintBanner(BannerCem)
		runDoctor()
	},
}

func runDoctor() {
	rc, err := LoadConfig()
	if err != nil {
		fmt.Println(styleError.Render(L("✗ config yüklenemedi: ", "✗ cannot load config: ") + err.Error()))
		os.Exit(1)
	}

	ok, warn, fail := 0, 0, 0
	tick := func(state string, line string) {
		switch state {
		case "ok":
			fmt.Println("  " + styleSuccess.Render("✓") + " " + line)
			ok++
		case "warn":
			fmt.Println("  " + styleWarn.Render("⚠") + " " + line)
			warn++
		case "fail":
			fmt.Println("  " + styleError.Render("✗") + " " + line)
			fail++
		}
	}

	fmt.Println(styleBold.Render("  Sistem"))
	tick("ok", fmt.Sprintf("%s/%s · Go runtime: %s",
		runtime.GOOS, runtime.GOARCH, runtime.Version()))

	home, _ := os.UserHomeDir()
	cemDir := filepath.Join(home, ".cem")
	if _, err := os.Stat(cemDir); err == nil {
		tick("ok", "~/.cem dizini → "+styleDim.Render(cemDir))
	} else {
		tick("warn", L("~/.cem dizini yok (ilk çalıştırmada oluşur)", "~/.cem directory missing (created on first run)"))
	}

	gp, _ := globalConfigPath()
	if _, err := os.Stat(gp); err == nil {
		tick("ok", "global config → "+styleDim.Render(gp))
	} else {
		tick("warn", L("global config yok → ", "no global config → ")+styleDim.Render(L("cem setup ile oluştur", "create it with: cem setup")))
	}

	if rc.HasProjectConfig() {
		tick("ok", "proje config → "+styleDim.Render(".cem.yaml (global override aktif)"))
	} else {
		tick("ok", L("proje config yok (global geçerli)", "no project config (global applies)"))
	}

	fmt.Println()
	fmt.Println(styleBold.Render("  Roller"))
	roles := rc.ActiveRoles()
	if roles.Thinker == "" {
		tick("fail", L("thinker atanmamış → ", "thinker not assigned → ")+styleDim.Render("cem roles claude"))
	} else if _, found := rc.Global.Tools[roles.Thinker]; !found {
		tick("warn", "thinker '"+roles.Thinker+L("' config'de kayıtlı değil → ", "' is not registered in the config → ")+
			styleDim.Render("cemi "+roles.Thinker))
	} else {
		tick("ok", "thinker → "+styleBold.Render(roles.Thinker))
	}
	if roles.Writer == "" {
		tick("fail", L("writer atanmamış → ", "writer not assigned → ")+styleDim.Render("cem roles - agy"))
	} else if _, found := rc.Global.Tools[roles.Writer]; !found {
		tick("warn", "writer '"+roles.Writer+L("' config'de kayıtlı değil → ", "' is not registered in the config → ")+
			styleDim.Render("cemi "+roles.Writer))
	} else {
		tick("ok", "writer → "+styleBold.Render(roles.Writer))
	}

	fmt.Println()
	fmt.Println(styleBold.Render(L("  Araçlar (PATH kontrolü)", "  Tools (PATH check)")))
	order := orderedToolKeys
	for _, key := range order {
		meta := KnownTools[key]
		cmd := resolveCommand(key, rc)
		path, lerr := exec.LookPath(cmd)
		_, registered := rc.Global.Tools[key]

		switch {
		case lerr == nil && registered:
			tick("ok", fmt.Sprintf("%-8s %s", styleBold.Render(meta.Name), styleDim.Render(path)))
		case lerr == nil && !registered:
			tick("warn", fmt.Sprintf(L("%-8s PATH'da var ama config'e kayıtlı değil → cemi %s", "%-8s found in PATH but not registered in config → cemi %s"),
				styleBold.Render(meta.Name), key))
		case lerr != nil && registered:
			tick("fail", fmt.Sprintf(L("%-8s config'de kayıtlı ama PATH'da yok", "%-8s registered in config but missing from PATH"),
				styleBold.Render(meta.Name)))
		default:
			tick("ok", fmt.Sprintf("%-8s %s", meta.Name, styleDim.Render(L("kurulu değil", "not installed"))))
		}
	}

	fmt.Println()
	fmt.Println(styleBold.Render("  Binary'ler"))
	for _, name := range []string{"cem", "cemi", "cemir"} {
		path, err := exec.LookPath(name)
		if err != nil {
			tick("warn", name+" → "+styleDim.Render(L("PATH'da bulunamadı", "not found in PATH")))
			continue
		}
		tick("ok", fmt.Sprintf("%-6s %s", styleBold.Render(name), styleDim.Render(path)))
	}

	fmt.Println()
	pathParts := filepath.SplitList(os.Getenv("PATH"))
	tick("ok", fmt.Sprintf(L("PATH girişi: %d dizin", "PATH entries: %d directories"), len(pathParts)))
	hasLocal := false
	for _, p := range pathParts {
		if strings.Contains(p, ".local/bin") || strings.Contains(p, "/usr/local/bin") {
			hasLocal = true
			break
		}
	}
	if !hasLocal {
		tick("warn", L("~/.local/bin veya /usr/local/bin PATH'da değil", "neither ~/.local/bin nor /usr/local/bin is in PATH"))
	}

	fmt.Println()
	summary := fmt.Sprintf(L("ok:%d  uyarı:%d  hata:%d", "ok:%d  warn:%d  fail:%d"), ok, warn, fail)
	switch {
	case fail > 0:
		fmt.Println("  " + styleError.Render("● Sistem sorunlu — ") + summary)
	case warn > 0:
		fmt.Println("  " + styleWarn.Render(L("● Sistem çalışıyor, eksikler var — ", "● Working, with gaps — ")) + summary)
	default:
		fmt.Println("  " + styleSuccess.Render(L("● Sistem sağlıklı — ", "● Healthy — ")) + summary)
	}
	fmt.Println()
}
