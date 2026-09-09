package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleBold    = lipgloss.NewStyle().Bold(true)
	styleTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	styleWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleDim     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	// Rol renkleri: düşünen mavi, yazan yeşil. Aynı ekranda iki AI'ın
	// çıktısı peş peşe akıyor; renk, kimin konuştuğunu okumadan ayırt ettirir.
	styleThinker = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	styleWriter  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("114"))
	styleBox     = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("212")).
			Padding(0, 2)
)

// RunSetupWizard — ilk kurulum sihirbazı
func RunSetupWizard(cfg *GlobalConfig) error {
	toolOrder := orderedToolKeys

	// Dil ilk soru: bundan sonraki tüm sihirbaz metinleri seçilen dilde.
	askLanguage(cfg)

	fmt.Println(styleBox.Render(
		styleTitle.Render(L("Hangi AI düşünür, hangi AI yazar?",
			"Which AI thinks, which AI writes?")) + "\n" +
			styleDim.Render(L("Bir kez seç, istediğin zaman değiştir: cem roles",
				"Choose once, change anytime: cem roles"))))
	fmt.Println()

	// Bulunulan dizinde .cem.yaml varsa, kullanıcıya wizard'ın global'i
	// değiştirdiğini ama proje config'in onu override edeceğini hatırlat.
	if _, err := os.Stat(".cem.yaml"); err == nil {
		fmt.Println(styleWarn.Render(
			L("  ⚠ Bu dizinde .cem.yaml var — global ayarlar burada override edilir.",
				"  ⚠ A .cem.yaml exists here — it overrides the global settings.")))
		fmt.Println(styleDim.Render(
			L("    Wizard bitince proje config'i de güncellemek isteyip istemediğini soracağım.",
				"    After the wizard I will ask whether to update the project config too.")))
		fmt.Println()
	}

	// Araç listesi
	for i, key := range toolOrder {
		meta := KnownTools[key]
		installed := ""
		if _, ok := cfg.Tools[key]; ok {
			installed = styleSuccess.Render(" ✓")
		}
		fmt.Printf("  %s  %-12s %s%s\n",
			colorMuted.Render(fmt.Sprintf("[%d]", i+1)),
			styleBold.Render(meta.Name),
			colorTagline.Render(meta.Description),
			installed,
		)
		if meta.Deprecated != "" {
			fmt.Printf("       %s %s\n",
				styleWarn.Render("⚠"),
				styleDim.Render(meta.Deprecated))
		}
	}
	fmt.Println()

	// Thinker seç
	thinker := pickTool(L("  🧠 Düşünen AI", "  🧠 Thinking AI"), toolOrder, cfg)
	if thinker == "" {
		return fmt.Errorf("iptal edildi")
	}

	// Writer seç
	fmt.Println()
	writer := pickTool("  ✍️  Yazan AI  ", toolOrder, cfg)
	if writer == "" {
		return fmt.Errorf("iptal edildi")
	}

	// Kurulu değilse kur
	fmt.Println()
	for _, key := range []string{thinker, writer} {
		if _, ok := cfg.Tools[key]; !ok {
			meta := KnownTools[key]
			if askYN(fmt.Sprintf("  %s kurulsun mu?", styleBold.Render(meta.Name))) {
				if err := InstallTool(key, cfg); err != nil {
					fmt.Println(styleWarn.Render(L("  ⚠ Kurulum başarısız: ", "  ⚠ Installation failed: ") + err.Error()))
					fmt.Println(styleDim.Render("    Manuel kurabilir, devam edebilirsiniz."))
				}
			}
		}
	}

	// Model seçimi — her rol için ayrı, varsayılan: CLI default'u (boş kalır)
	if !autoYes {
		fmt.Println()
		askModel(thinker, "🧠 thinker", cfg)
		askEffort(thinker, "🧠 thinker", cfg)
		if writer != thinker {
			askModel(writer, "✍️  writer", cfg)
			askEffort(writer, "✍️  writer", cfg)
		}
	}

	cfg.Roles = Roles{Thinker: thinker, Writer: writer}
	cfg.Setup = true

	if err := saveGlobalConfig(cfg); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(styleSuccess.Render(L("  ✓ Hazır!", "  ✓ Ready!")))
	// Proje config'i global'i override ediyor — kullanıcıya bu dizindeki
	// .cem.yaml'i yeni rollere göre güncellemek isteyip istemediğini sor.
	if _, err := os.Stat(".cem.yaml"); err == nil {
		fmt.Println(styleWarn.Render(
			L("  ⚠ Bu dizinde .cem.yaml var — burada çalışırken global override edilir.",
				"  ⚠ A .cem.yaml exists here — it overrides the global config in this directory.")))
		if askYN(L("  Proje config'ini de yeni rollerle güncelleyeyim mi?",
			"  Update the project config with the new roles as well?")) {
			pc := &ProjectConfig{Roles: &Roles{Thinker: thinker, Writer: writer}}
			// Modelleri de aktar — global'de set edilmişse proje override eklenir
			pc.Models = map[string]string{}
			if t, ok := cfg.Tools[thinker]; ok && t.Model != "" {
				pc.Models[thinker] = t.Model
			}
			if writer != thinker {
				if w, ok := cfg.Tools[writer]; ok && w.Model != "" {
					pc.Models[writer] = w.Model
				}
			}
			if len(pc.Models) == 0 {
				pc.Models = nil
			}
			if err := SaveProjectConfig(pc); err != nil {
				fmt.Println(styleError.Render(L("  ✗ .cem.yaml yazılamadı: ", "  ✗ Cannot write .cem.yaml: ") + err.Error()))
			} else {
				fmt.Println(styleSuccess.Render(L("  ✓ .cem.yaml güncellendi", "  ✓ .cem.yaml updated")))
			}
		} else {
			fmt.Println(styleDim.Render("    Sonra: cem init"))
		}
	}
	// Kullanıcı seviye sorusunu boş geçtiyse rol bazlı varsayılanı yaz —
	// deneyimsiz kullanıcı da israfsız bir kurulumla çıksın.
	applyRoleDefaults(cfg, thinker, writer)

	fmt.Println()
	printRolesTable(thinker, writer, &ResolvedConfig{Global: cfg})
	fmt.Println()

	return nil
}

// printRolesTable — rol özetini iki kutu halinde basar:
//
//	┌── Aktif Roller ──────────────────────────┐
//	│ 🧠 thinker  claude       cem "soru"      │
//	│ ✍️  writer   agy          cem -w "görev"  │
//	│ 🤝 pair     claude → agy cem -p "görev"  │
//	└──────────────────────────────────────────┘
//	┌── Değiştir ──────────────────────────────┐
//	│ cem roles claude agy   global'i değiştir │
//	│ cem init               proje config'i    │
//	└──────────────────────────────────────────┘
func printRolesTable(thinker, writer string, rc *ResolvedConfig) {
	// İlk kolon (ikon + rol adı): max genişlik = "✍️  writer "
	// İkinci kolon (model adı kombosu): thinker / writer / pair
	// Üçüncü kolon: örnek komut
	ask := L(`cem "soru"`, `cem "question"`)
	task := L(`cem -w "görev"`, `cem -w "task"`)
	pairTask := L(`cem -p "görev"`, `cem -p "task"`)
	// Rolün yanında model + düşünme seviyesi: pair'de amaç genelde "pahalı
	// model düşünsün, ucuz model yazsın" olduğu için kurulumun doğru olup
	// olmadığı tek bakışta görünmeli.
	tDesc, wDesc := thinker, writer
	if rc != nil {
		if d := describeToolRun(thinker, rc); d != "default" {
			tDesc = thinker + " · " + d
		}
		if d := describeToolRun(writer, rc); d != "default" {
			wDesc = writer + " · " + d
		}
	}
	rows := [][3]string{
		{"🧠 thinker", tDesc, ask},
		{"✍️  writer ", wDesc, task},
		{"🤝 pair    ", thinker + " → " + writer, pairTask},
	}
	helpRows := [][2]string{
		{"cem roles claude agy", L("global'i değiştir", "change globally")},
		{"cem roles --here X Y", L("sadece bu proje", "this project only")},
		{"cem init", L("proje wizard", "project wizard")},
	}

	// Genişlik hesapla
	w1, w2 := 11, 10 // ikon kolonu sabit, isim kolonu min
	for _, r := range rows {
		if l := utf8RuneLen(r[1]); l > w2 {
			w2 = l
		}
	}
	w3 := 0
	for _, r := range rows {
		if l := utf8RuneLen(r[2]); l > w3 {
			w3 = l
		}
	}

	// Aktif Roller kutusu
	innerWidth := w1 + 2 + w2 + 2 + w3
	printBoxTitle(L("Aktif Roller", "Active Roles"), innerWidth)
	for _, r := range rows {
		fmt.Printf("  │ %s  %s  %s │\n",
			padRight(r[0], w1),
			styleBold.Render(padRight(r[1], w2)),
			colorMuted.Render(padRight(r[2], w3)))
	}
	printBoxBottom(innerWidth)

	// Değiştir kutusu
	w4 := 0
	for _, h := range helpRows {
		if l := utf8RuneLen(h[0]); l > w4 {
			w4 = l
		}
	}
	w5 := 0
	for _, h := range helpRows {
		if l := utf8RuneLen(h[1]); l > w5 {
			w5 = l
		}
	}
	hInner := w4 + 2 + w5
	if hInner < innerWidth {
		hInner = innerWidth
	}
	fmt.Println()
	printBoxTitle(L("Değiştir", "Change"), hInner)
	for _, h := range helpRows {
		fmt.Printf("  │ %s  %s │\n",
			styleBold.Render(padRight(h[0], w4)),
			styleDim.Render(padRight(h[1], hInner-w4-2)))
	}
	printBoxBottom(hInner)
}

// utf8RuneLen — emoji ve Türkçe karakter sayan görsel genişlik (yaklaşık).
// Emoji'ler 2 hücre kabul edilir; ANSI escape'ler atılır.
func utf8RuneLen(s string) int {
	// Çok kaba: rune sayısı (emoji'ler 1 sayılır ama lipgloss style olmadığı için
	// pratikte yeterli; ihtiyaç olursa runewidth eklenir).
	n := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == 0x1b {
			inEsc = true
			continue
		}
		n++
	}
	return n
}

func padRight(s string, width int) string {
	gap := width - utf8RuneLen(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

// Kutu üst/alt: │ ile │ arasındaki görsel genişlik = inner + 2 (1 boşluk her yandan).
// Yani ┌─...─┐ arasındaki çizgi sayısı = inner + 2.
func printBoxTitle(title string, inner int) {
	tl := utf8RuneLen(title) + 2 // " title "
	totalDashes := inner + 2     // toplam çizgi inside-width'e eşit
	leadDashes := 2              // ┌── title ...
	tailDashes := totalDashes - leadDashes - tl
	if tailDashes < 1 {
		tailDashes = 1
	}
	fmt.Printf("  ┌%s %s %s┐\n",
		strings.Repeat("─", leadDashes), title, strings.Repeat("─", tailDashes))
}

func printBoxBottom(inner int) {
	fmt.Printf("  └%s┘\n", strings.Repeat("─", inner+2))
}

// printTail — metnin son n satırını dim renkle yazdırır (hata bağlamı için).
func printTail(s string, n int) {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
		fmt.Println(styleDim.Render(L("  ... (çıktı kısaltıldı)", "  ... (output truncated)")))
	}
	for _, l := range lines {
		fmt.Println(styleDim.Render("  " + l))
	}
}

// askModel — kullanıcıya bir tool için model seçtirir; seçimi cfg.Tools[key].Model'a
// kaydeder. Models listesi boşsa sessizce döner. ModelFlag boşsa (örn. agy)
// seçim alınır ama runtime'da kullanılmaz — kullanıcıya uyarı basılır.
func askModel(toolKey, label string, cfg *GlobalConfig) {
	meta, ok := KnownTools[toolKey]
	if !ok || len(meta.Models) == 0 {
		return
	}
	if meta.ModelFlag == "" {
		fmt.Println(styleDim.Render(fmt.Sprintf(
			L("  ⓘ %s CLI'sı henüz --model flag'ini desteklemiyor — seçim kayda alınır, runtime'a etki etmez.",
				"  ⓘ %s CLI has no --model flag yet — the choice is stored but has no runtime effect."),
			meta.Name)))
	}
	fmt.Printf(L("  %s · %s için model:\n", "  %s · model for %s:\n"), styleBold.Render(label), styleBold.Render(meta.Name))
	for i, m := range meta.Models {
		marker := " "
		if t, ok := cfg.Tools[toolKey]; ok && t.Model == m {
			marker = styleSuccess.Render("✓")
		}
		fmt.Printf("    %s [%d] %s\n", marker, i+1, m)
	}
	fmt.Printf(L("      [%d] custom (kendi adını gir)\n", "      [%d] custom (type your own)\n"), len(meta.Models)+1)
	fmt.Print(L("      [0] default (CLI kendi seçer)\n", "      [0] default (the CLI decides)\n"))
	fmt.Print(L("  Seçim: ", "  Choice: "))
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(resp)

	t := cfg.Tools[toolKey] // zero-value ok
	switch resp {
	case "":
		// Enter = mevcut seçimi koru (yukarıda ✓ ile gösterilen)
		// Hiç yoksa CLI default kullanılır (t.Model zaten "" kalır)
	case "0":
		t.Model = "" // explicit default
	default:
		idx, err := strconv.Atoi(resp)
		if err != nil || idx < 1 || idx > len(meta.Models)+1 {
			fmt.Println(styleDim.Render(L("  geçersiz, mevcut/default korundu", "  invalid, kept current/default")))
		} else if idx == len(meta.Models)+1 {
			fmt.Print(L("  Model adı: ", "  Model name: "))
			line, _ := reader.ReadString('\n')
			t.Model = strings.TrimSpace(line)
		} else {
			t.Model = meta.Models[idx-1]
		}
	}
	if cfg.Tools == nil {
		cfg.Tools = map[string]InstalledTool{}
	}
	cfg.Tools[toolKey] = t
	if t.Model != "" {
		fmt.Printf(L("  %s model: %s\n", "  %s model: %s\n"), styleSuccess.Render("✓"), styleBold.Render(t.Model))
	}
}

// askLanguage — sihirbazın İLK sorusu. Seçim hemen uygulanır ki geri kalan
// sihirbaz doğru dilde aksın. Enter = mevcut/otomatik algılanan dil.
func askLanguage(cfg *GlobalConfig) {
	current := Lang()
	fmt.Println()
	fmt.Println(styleBold.Render("  Dil / Language"))
	for i, opt := range []struct{ code, name string }{
		{LangTR, "Türkçe"},
		{LangEN, "English"},
	} {
		marker := " "
		if opt.code == current {
			marker = styleSuccess.Render("✓")
		}
		fmt.Printf("    %s [%d] %s\n", marker, i+1, opt.name)
	}
	fmt.Print("  Seçim / Choice: ")
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')

	switch strings.TrimSpace(resp) {
	case "1":
		cfg.Lang = LangTR
	case "2":
		cfg.Lang = LangEN
	default:
		// Enter veya geçersiz: mevcut dil kalıcı hâle gelsin ki bir dahaki
		// çalıştırmada ortam değişkeni değişse bile arayüz aynı kalsın.
		cfg.Lang = current
	}
	// applyLang() burada ÇAĞRILMAZ: komut açıklamalarına dokunmak
	// rootCmd ↔ wizard arasında initialization cycle yaratıyor. Sihirbazın
	// kendi metinleri L() ile dinamik; help metinleri zaten bir sonraki
	// çalıştırmada main() içindeki applyLang() ile doğru dile geçer.
	setLang(cfg.Lang)
}

// askEffort — kullanıcıya bir tool için düşünme (reasoning) seviyesi seçtirir;
// seçim cfg.Tools[key].Effort'a yazılır. Araç seviye seçimini desteklemiyorsa
// (agy, cursor) sessizce döner.
func askEffort(toolKey, label string, cfg *GlobalConfig) {
	meta, ok := KnownTools[toolKey]
	if !ok || len(meta.Efforts) == 0 || len(meta.EffortArgs) == 0 {
		return
	}
	fmt.Printf(L("  %s · %s için düşünme seviyesi:\n", "  %s · reasoning effort for %s:\n"),
		styleBold.Render(label), styleBold.Render(meta.Name))
	for i, e := range meta.Efforts {
		marker := " "
		if t, ok := cfg.Tools[toolKey]; ok && t.Effort == e {
			marker = styleSuccess.Render("✓")
		}
		fmt.Printf("    %s [%d] %s%s\n", marker, i+1, e, styleDim.Render(effortHint(e)))
	}
	fmt.Print(L("      [0] default (CLI kendi seçer)\n", "      [0] default (the CLI decides)\n"))
	fmt.Print(L("  Seçim: ", "  Choice: "))
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	resp = strings.TrimSpace(resp)

	t := cfg.Tools[toolKey]
	switch resp {
	case "":
		// Enter = mevcut seçimi koru
	case "0":
		t.Effort = ""
	default:
		idx, err := strconv.Atoi(resp)
		if err != nil || idx < 1 || idx > len(meta.Efforts) {
			fmt.Println(styleDim.Render(L("  geçersiz, mevcut/default korundu", "  invalid, kept current/default")))
		} else {
			t.Effort = meta.Efforts[idx-1]
		}
	}
	if cfg.Tools == nil {
		cfg.Tools = map[string]InstalledTool{}
	}
	cfg.Tools[toolKey] = t
	if t.Effort != "" {
		fmt.Printf(L("  %s düşünme: %s\n", "  %s effort: %s\n"), styleSuccess.Render("✓"), styleBold.Render(t.Effort))
	}
}

// effortHint — seviyelerin maliyet/derinlik dengesini tek kelimeyle anlatır.
func effortHint(e string) string {
	switch e {
	case "minimal", "low":
		return L("  (hızlı, ucuz)", "  (fast, cheap)")
	case "medium":
		return L("  (denge)", "  (balanced)")
	case "high":
		return L("  (derin)", "  (deep)")
	case "xhigh", "max":
		return L("  (en derin, yavaş + pahalı)", "  (deepest, slow + expensive)")
	}
	return ""
}

// pickInstallShell — platforma uyan shell-install komutunu döndürür; yoksa "".
func pickInstallShell(meta ToolMeta) string {
	if runtime.GOOS == "windows" {
		return meta.InstallShellWin
	}
	return meta.InstallShellUnix
}

// ensureDep — bir bin'i PATH'da arar; yoksa veya çok eskise kullanıcıya öneri
// sunar ve (kullanıcı kabul ederse) OS paket yöneticisiyle kurmayı dener.
// true → bin artık PATH'da ve yeterince güncel.
func ensureDep(bin string) bool {
	if _, err := exec.LookPath(bin); err == nil {
		if depVersionOK(bin) {
			return true
		}
		fmt.Println(styleDim.Render(fmt.Sprintf(L("  ⚠ %s sürümü çok eski — modern sürüm kuruluyor", "  ⚠ %s is too old — installing a modern version"), bin)))
	}
	install, label := depInstallCommand(bin)
	if install == nil {
		fmt.Println(styleError.Render(fmt.Sprintf(L("  ✗ %s PATH'de yok ve otomatik kurulum tanımlanmamış", "  ✗ %s is not in PATH and has no automatic installer"), bin)))
		return false
	}
	fmt.Println(styleDim.Render(fmt.Sprintf(L("  ⚠ %s eksik — kurmak için: %s", "  ⚠ %s is missing — install it with: %s"), bin, label)))
	if !askYN(L("  Şimdi kurulsun mu?", "  Install it now?")) {
		return false
	}
	// Output'u yakala, başarısız olunca son satırları göster.
	var depBuf strings.Builder
	install.Stdout = &depBuf
	install.Stderr = &depBuf
	sp := StartSpinner(fmt.Sprintf(L("⏳ %s kuruluyor (önkoşul)", "⏳ installing %s (prerequisite)"), bin))
	err := install.Run()
	sp.Stop()
	if err != nil {
		printTail(depBuf.String(), 12)
		fmt.Println(styleError.Render(L("  ✗ kurulum başarısız: ", "  ✗ installation failed: ") + err.Error()))
		return false
	}
	// Linux'ta nvm Node'u ~/.nvm/versions/node/<v>/bin/'e koyar; çalışan
	// cem süreci için PATH'i o dizinle güncelle ki LookPath bulsun.
	if runtime.GOOS == "linux" && (bin == "npm" || bin == "node") {
		home, _ := os.UserHomeDir()
		matches, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin"))
		if len(matches) > 0 {
			// En son sürümü al (lexicographic son)
			nvmBin := matches[len(matches)-1]
			os.Setenv("PATH", nvmBin+string(os.PathListSeparator)+os.Getenv("PATH"))
		}
	}
	_, lookErr := exec.LookPath(bin)
	return lookErr == nil
}

// depVersionOK — bin'in çıktısından major sürümü çekip minimum eşikle karşılaştırır.
// Claude Code/OpenAI Codex CLI gibi paketler Node 18+ ister; npm 3.x (Ubuntu 18.04)
// gibi antika sürümlerde install argv parse hatası verir.
func depVersionOK(bin string) bool {
	minMajor := map[string]int{
		"npm":     9, // Node 18 LTS ile gelir
		"node":    18,
		"python":  3,
		"python3": 3,
	}
	threshold, has := minMajor[bin]
	if !has {
		return true // bilinmeyen → kontrol etme
	}
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return false
	}
	s := strings.TrimSpace(string(out))
	s = strings.TrimPrefix(s, "v")
	major := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			break
		}
		major = major*10 + int(s[i]-'0')
	}
	return major >= threshold
}

// depInstallCommand — bin için OS-spesifik kurulum komutunu döndürür.
func depInstallCommand(bin string) (*exec.Cmd, string) {
	switch bin {
	case "npm", "node":
		switch runtime.GOOS {
		case "windows":
			return exec.Command("winget", "install", "-e", "--id", "OpenJS.NodeJS.LTS", "--accept-package-agreements", "--accept-source-agreements"),
				"winget install OpenJS.NodeJS.LTS"
		case "darwin":
			return exec.Command("brew", "install", "node"), "brew install node"
		case "linux":
			// nvm — prebuilt Node binary. NodeSource artık glibc 2.28+ istiyor;
			// nvm Ubuntu 18.04 (libc 2.27) dahil her distro'da çalışıyor.
			return exec.Command("sh", "-c", `
set -e
if [ ! -s "$HOME/.nvm/nvm.sh" ]; then
  curl -fsSL https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
fi
export NVM_DIR="$HOME/.nvm"
. "$NVM_DIR/nvm.sh"
nvm install --lts
nvm use --lts
`), "nvm + Node LTS (prebuilt, glibc bağımsız)"
		}
	case "python", "python3":
		switch runtime.GOOS {
		case "windows":
			return exec.Command("winget", "install", "-e", "--id", "Python.Python.3.12", "--accept-package-agreements", "--accept-source-agreements"),
				"winget install Python.Python.3.12"
		case "darwin":
			return exec.Command("brew", "install", "python"), "brew install python"
		case "linux":
			return exec.Command("sudo", "apt-get", "install", "-y", "python3", "python3-pip"),
				"sudo apt-get install -y python3 python3-pip"
		}
	}
	return nil, ""
}

// InstallTool — bir AI CLI aracını kur
func InstallTool(toolKey string, cfg *GlobalConfig) error {
	meta, ok := KnownTools[toolKey]
	if !ok {
		return fmt.Errorf("bilinmeyen araç: %s", toolKey)
	}

	shellCmd := pickInstallShell(meta)
	if shellCmd == "" && meta.InstallCmd == nil {
		fmt.Printf("  %s manuel kurulum gerekiyor\n", meta.Name)
		if meta.Description != "" {
			fmt.Printf("  → %s\n", meta.Description)
		}
		return nil
	}

	// Önkoşul: doğrudan komutsa ilk binary (npm, pip, python), shell-install ise sh/cmd zaten var.
	if shellCmd == "" && len(meta.InstallCmd) > 0 {
		if !ensureDep(meta.InstallCmd[0]) {
			return fmt.Errorf("%s eksik — %s kurulamadı", meta.InstallCmd[0], meta.Name)
		}
	}

	var cmd *exec.Cmd
	if shellCmd != "" {
		if runtime.GOOS == "windows" {
			// Doğrudan PowerShell'e ver. cmd /c üzerinden geçince çift-quote
			// nesting bazen child output'unu yutuyor.
			cmd = exec.Command("powershell", "-NoProfile",
				"-ExecutionPolicy", "Bypass", "-Command", shellCmd)
		} else {
			cmd = exec.Command("sh", "-c", shellCmd)
		}
	} else {
		cmd = exec.Command(meta.InstallCmd[0], meta.InstallCmd[1:]...)
	}
	// Sessiz kurulum: çıktıyı buffer'a, hata durumunda son satırlar gösterilir.
	// Kullanıcı süreç donmuş sanmasın diye spinner göster.
	var instBuf strings.Builder
	cmd.Stdout = &instBuf
	cmd.Stderr = &instBuf
	sp := StartSpinner(fmt.Sprintf("⏳ %s kuruluyor", meta.Name))
	runErr := cmd.Run()
	sp.Stop()
	if runErr != nil {
		printTail(instBuf.String(), 15)
		return runErr
	}

	// ToolMeta.Binary set ise PATH'da o adla aranır (örn. cursor → cursor-agent).
	binName := toolKey
	if meta.Binary != "" {
		binName = meta.Binary
	}

	version := ""
	if meta.VersionFlag != "" {
		out, _ := exec.Command(binName, meta.VersionFlag).Output()
		version = strings.TrimSpace(string(out))
	}

	// Command'i her zaman mutlak yol olarak sakla — cemir gibi kaldırma
	// işlemleri PATH lookup'a güvenmeden direkt dosyaya ulaşsın.
	command := binName
	if abs, err := exec.LookPath(binName); err == nil {
		command = abs
	} else if fallback := fallbackInstallPath(toolKey); fallback != "" {
		command = fallback
		fmt.Println(styleDim.Render("    bulundu: " + fallback))
	} else {
		// Henüz PATH'da yok ve fallback yolu da bulunamadı — küçük ihtimal.
		// toolKey'i sakla, cemir manuel sil ipucu verir.
		existing := cfg.Tools[toolKey]
		existing.Version = version
		if existing.Command == "" {
			existing.Command = toolKey
		}
		cfg.Tools[toolKey] = existing
		fmt.Printf(L("  %s %s kuruldu ama %s henüz PATH'de değil\n", "  %s %s installed but %s is not in PATH yet\n"),
			styleWarn.Render("⚠"), meta.Name, toolKey)
		fmt.Println(styleDim.Render(L("    Yeni terminal aç (PATH bu oturumda yenilenmez)", "    Open a new terminal (PATH is not refreshed in this session)")))
		return nil
	}
	// Model alanını koru — wizard daha önce set etmiş olabilir
	existing := cfg.Tools[toolKey]
	existing.Command = command
	existing.Version = version
	cfg.Tools[toolKey] = existing
	fmt.Printf("  %s %s kuruldu\n", styleSuccess.Render("✓"), meta.Name)

	// Post-install auth setup: provider varsa kullanıcıya API key girme şansı ver,
	// yoksa CLI'ın kendi login akışını işaret et.
	postInstallAuthSetup(toolKey, meta, command, cfg)

	// Her kurulumdan sonra .gitignore güvenliği — proje config'i (.cem.yaml)
	// yanlışlıkla repo'ya gitmesin.
	ensureGitignoreSafe()

	return nil
}

// postInstallAuthSetup — provider tanımlı ise kullanıcıyı API key / login
// arasında seçim yapmaya yönlendirir. autoYes modunda atlanır (kullanıcı
// non-interactive istemiş, sessiz bırak).
func postInstallAuthSetup(toolKey string, meta ToolMeta, binPath string, cfg *GlobalConfig) {
	if meta.Provider == "" || meta.APIKeyEnv == "" {
		// OAuth-only araç (agy, cursor) — CLI kendi login'ini yönetir
		return
	}
	if autoYes {
		fmt.Println(styleDim.Render(fmt.Sprintf(
			L("  ⓘ Auth: 'cem keys add %s' ile key gir veya '%s' çalıştırıp login ol",
				"  ⓘ Auth: add a key with 'cem keys add %s' or run '%s' and log in"),
			meta.Provider, filepath.Base(binPath))))
		return
	}
	fmt.Println()
	fmt.Printf(L("  %s için auth:\n", "  auth for %s:\n"), styleBold.Render(meta.Name))
	fmt.Println(styleDim.Render(L("    [1] API key kaydet (çoklu key + rate-limit rotasyonu)", "    [1] Store an API key (multi-key + rate-limit rotation)")))
	fmt.Println(styleDim.Render(L("    [2] Subscription / OAuth login (sonra: '", "    [2] Subscription / OAuth login (then run: '") + filepath.Base(binPath) + L("' çalıştır)", "')")))
	fmt.Println(styleDim.Render(L("    [3] Şimdilik atla", "    [3] Skip for now")))
	fmt.Print(L("  Seçim [1-3]: ", "  Choice [1-3]: "))
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	switch choice {
	case "1":
		for {
			fmt.Print("  API key: ")
			val, _ := reader.ReadString('\n')
			val = strings.TrimSpace(val)
			if val == "" {
				fmt.Println(styleDim.Render("  iptal"))
				return
			}
			fmt.Print(L("  Etiket (opsiyonel, örn. 'personal'): ", "  Label (optional, e.g. 'personal'): "))
			label, _ := reader.ReadString('\n')
			label = strings.TrimSpace(label)
			if cfg.APIKeys == nil {
				cfg.APIKeys = map[string][]APIKey{}
			}
			cfg.APIKeys[meta.Provider] = append(cfg.APIKeys[meta.Provider],
				APIKey{Value: val, Label: label})
			fmt.Printf("  %s %s key #%d kaydedildi\n",
				styleSuccess.Render("✓"), meta.Provider, len(cfg.APIKeys[meta.Provider]))
			if !askYN(L("  Başka key eklemek ister misin?", "  Add another key?")) {
				return
			}
		}
	case "2":
		fmt.Println(styleDim.Render("  → " + filepath.Base(binPath) + L("  (interaktif login akışı açılır)", "  (opens the interactive login flow)")))
	default:
		fmt.Println(styleDim.Render(L("  Atlandı. Sonra: cem keys add ", "  Skipped. Later: cem keys add ") + meta.Provider))
	}
}

// ensureGitignoreSafe — CWD bir git repo ise .gitignore'da .cem.yaml var mı
// kontrol eder; yoksa ekler ve kullanıcıya bildirir. Repo değilse sessizce
// döner. cem'in proje config'i yanlışlıkla repo'ya commit edilmesin diye.
func ensureGitignoreSafe() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	if _, err := os.Stat(filepath.Join(wd, ".git")); err != nil {
		return // git repo değil
	}
	gi := filepath.Join(wd, ".gitignore")
	entry := ".cem.yaml"
	data, err := os.ReadFile(gi)
	if err != nil {
		// .gitignore yok — uyar
		fmt.Println(styleWarn.Render(
			"  ⚠ Git repo'da .gitignore yok — .cem.yaml repo'ya gidebilir"))
		return
	}
	if strings.Contains("\n"+string(data)+"\n", "\n"+entry+"\n") {
		return // zaten var
	}
	f, err := os.OpenFile(gi, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	suffix := ""
	if len(data) > 0 && data[len(data)-1] != '\n' {
		suffix = "\n"
	}
	if _, err := f.WriteString(suffix + "\n# cem proje config\n" + entry + "\n"); err != nil {
		return
	}
	fmt.Println(styleSuccess.Render("  ✓ .gitignore'a '" + entry + "' eklendi"))
}

// RemoveTool — bir AI CLI aracını kaldır
func RemoveTool(toolKey string, cfg *GlobalConfig) error {
	meta, ok := KnownTools[toolKey]
	if !ok {
		return fmt.Errorf("bilinmeyen araç: %s", toolKey)
	}
	if _, installed := cfg.Tools[toolKey]; !installed {
		return fmt.Errorf(L("%s zaten kurulu değil", "%s is not installed"), toolKey)
	}
	if !askYN(fmt.Sprintf(L("  %s kaldırılsın mı?", "  Remove %s?"), styleBold.Render(meta.Name))) {
		fmt.Println(L("  İptal.", "  Cancelled."))
		return nil
	}

	fmt.Printf(L("  ⏳ %s kaldırılıyor...\n", "  ⏳ removing %s...\n"), meta.Name)

	ic := meta.InstallCmd
	var unCmd *exec.Cmd
	if len(ic) >= 2 && ic[0] == "npm" {
		unCmd = exec.Command("npm", "uninstall", "-g", ic[len(ic)-1])
	} else if len(ic) >= 2 && ic[0] == "pip" {
		unCmd = exec.Command("pip", "uninstall", "-y", ic[len(ic)-1])
	} else if pickInstallShell(meta) != "" {
		// Shell-installed tool: silinecek binary'yi bulma sırası:
		// 1) config'deki mutlak yol
		// 2) config'deki düz isim (eski install) → PATH'da resolve
		// 3) meta.Binary'yi PATH'da ara
		// 4) toolKey'i PATH'da ara
		// 5) fallbackInstallPath
		path := ""
		if t, ok := cfg.Tools[toolKey]; ok && t.Command != "" {
			if filepath.IsAbs(t.Command) {
				path = t.Command
			} else if abs, err := exec.LookPath(t.Command); err == nil {
				path = abs
			}
		}
		if path == "" {
			lookupName := toolKey
			if meta.Binary != "" {
				lookupName = meta.Binary
			}
			if abs, err := exec.LookPath(lookupName); err == nil {
				path = abs
			}
		}
		if path == "" {
			path = fallbackInstallPath(toolKey)
		}
		if path == "" {
			return fmt.Errorf("%s", L("binary konumu bulunamadı — manuel sil", "binary location not found — remove it manually"))
		}
		// Tool kendi alt-dizinine kuruluyorsa (örn. \cursor-agent\, \agy\)
		// versions/ ve config dosyalarıyla birlikte komple ağacı sil. Aksi
		// halde sadece tek dosyayı temizle ve boş kalan dizinleri kaldır.
		removed := false
		dir := filepath.Dir(path)
		dirName := filepath.Base(dir)
		grand := filepath.Dir(dir)
		grandName := filepath.Base(grand)
		matchesTool := func(n string) bool {
			return n == toolKey || (meta.Binary != "" && n == meta.Binary)
		}
		switch {
		case matchesTool(dirName):
			// Ör: %LOCALAPPDATA%\cursor-agent\cursor-agent.cmd → wipe cursor-agent\
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("%s silinemedi: %w", dir, err)
			}
			fmt.Println(styleDim.Render(L("    silindi (komple ağaç): ", "    removed (whole tree): ") + dir))
			removed = true
		case matchesTool(grandName):
			// Ör: %LOCALAPPDATA%\agy\bin\agy.exe → wipe agy\
			if err := os.RemoveAll(grand); err != nil {
				return fmt.Errorf("%s silinemedi: %w", grand, err)
			}
			fmt.Println(styleDim.Render(L("    silindi (komple ağaç): ", "    removed (whole tree): ") + grand))
			removed = true
		default:
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("%s silinemedi: %w", path, err)
			}
			fmt.Println(styleDim.Render("    silindi: " + path))
			// Boş kalan dizinleri temizle (best-effort)
			_ = os.Remove(dir)
			_ = os.Remove(grand)
			removed = true
		}
		_ = removed
		delete(cfg.Tools, toolKey)
		fmt.Printf(L("  %s %s kaldırıldı\n", "  %s %s removed\n"), styleSuccess.Render("✓"), meta.Name)
		return saveGlobalConfig(cfg)
	} else {
		return fmt.Errorf("%s", L("otomatik kaldırma desteklenmiyor", "automatic removal is not supported"))
	}

	unCmd.Stdout = os.Stdout
	unCmd.Stderr = os.Stderr
	if err := unCmd.Run(); err != nil {
		return err
	}

	delete(cfg.Tools, toolKey)
	if cfg.Roles.Thinker == toolKey {
		cfg.Roles.Thinker = ""
	}
	if cfg.Roles.Writer == toolKey {
		cfg.Roles.Writer = ""
	}

	fmt.Printf(L("  %s %s kaldırıldı\n", "  %s %s removed\n"), styleSuccess.Render("✓"), meta.Name)
	return saveGlobalConfig(cfg)
}

// ShowRoles — aktif rolleri güzel şekilde göster
func ShowRoles(rc *ResolvedConfig) {
	roles := rc.ActiveRoles()

	src := colorMuted.Render("  kaynak: ~/.cem/config.yaml  ") +
		styleDim.Render("(global)")
	if rc.HasProjectConfig() {
		src = colorYellow.Render("  kaynak: .cem.yaml  ") +
			styleDim.Render("(proje — global override)")
	}

	printRolesTable(roles.Thinker, roles.Writer, rc)
	fmt.Println(src)
	fmt.Println()

	// Kurulu araçlar — tablo formatı
	if len(rc.Global.Tools) > 0 {
		printToolsTable(rc.Global.Tools)
	}
}

// printToolsTable — kurulu araçları sürüm + model ile birlikte tablo halinde basar.
func printToolsTable(tools map[string]InstalledTool) {
	// Genişlikleri hesapla
	wKey, wVer, wModel := 4, 7, 5
	for key, t := range tools {
		if l := utf8RuneLen(key); l > wKey {
			wKey = l
		}
		v := t.Version
		if v == "" {
			v = "—"
		}
		if l := utf8RuneLen(v); l > wVer {
			wVer = l
		}
		m := t.Model
		if m == "" {
			m = "default"
		}
		if l := utf8RuneLen(m); l > wModel {
			wModel = l
		}
	}
	inner := wKey + 2 + wVer + 2 + wModel
	printBoxTitle(L("Kurulu Araçlar", "Installed Tools"), inner)
	for key, t := range tools {
		v := t.Version
		if v == "" {
			v = "—"
		}
		m := t.Model
		if m == "" {
			m = "default"
		}
		fmt.Printf("  │ %s  %s  %s │\n",
			styleBold.Render(padRight(key, wKey)),
			styleDim.Render(padRight(v, wVer)),
			colorMuted.Render(padRight(m, wModel)))
	}
	printBoxBottom(inner)
}

// pickTool — wizard için araç seçtir
func pickTool(prompt string, toolOrder []string, cfg *GlobalConfig) string {
	fmt.Printf("%s [1-%d]: ", styleBold.Render(prompt), len(toolOrder))
	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "q" || input == "" {
			return ""
		}
		var idx int
		if _, err := fmt.Sscanf(input, "%d", &idx); err == nil {
			if idx >= 1 && idx <= len(toolOrder) {
				key := toolOrder[idx-1]
				meta := KnownTools[key]
				suffix := ""
				if _, ok := cfg.Tools[key]; ok {
					suffix = styleSuccess.Render(" (kurulu)")
				}
				fmt.Printf("  → %s%s\n", styleBold.Render(meta.Name), suffix)
				return key
			}
		}
		fmt.Printf("  %s [1-%d]: ", styleWarn.Render("Geçersiz, tekrar gir"), len(toolOrder))
	}
}

// autoYes — cemi -y veya benzeri etkileşimsiz mod aktifse tüm askYN
// çağrıları "y" döner. Komut başlangıcında set edilir, çıkışta sıfırlanmaz
// (kısa ömürlü süreç).
var autoYes bool

func askYN(prompt string) bool {
	if autoYes {
		fmt.Printf("%s (y/N): y  (auto)\n", prompt)
		return true
	}
	fmt.Printf("%s (y/N): ", prompt)
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	resp = strings.ToLower(strings.TrimSpace(resp))
	return resp == "y" || resp == "yes" || resp == "e" || resp == "evet"
}
