package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Mode int

const (
	ModeThink Mode = iota
	ModeWrite
	ModePair
)

// ReadStdin — pipe ile gelen veriyi okur (interaktif tty ise boş döner)
func ReadStdin() string {
	info, err := os.Stdin.Stat()
	if err != nil {
		return ""
	}
	if (info.Mode() & os.ModeCharDevice) != 0 {
		return ""
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(data), "\n")
}

// Run — seçili moda göre AI(ları) çalıştırır
func Run(input string, mode Mode, rc *ResolvedConfig) error {
	roles := rc.ActiveRoles()

	switch mode {
	case ModeThink:
		if roles.Thinker == "" {
			return errMissingRole("thinker")
		}
		printAIHeader("thinker", roles.Thinker, rc)
		start := time.Now()
		err := runTool(roles.Thinker, rc, input, "🧠")
		if err == nil {
			printElapsed(start, L("düşünme", "thinking"))
		}
		return err

	case ModeWrite:
		if roles.Writer == "" {
			return errMissingRole("writer")
		}
		printAIHeader("writer", roles.Writer, rc)
		start := time.Now()
		err := runTool(roles.Writer, rc, input, "✍️")
		if err == nil {
			printElapsed(start, L("yazma", "writing"))
		}
		return err

	case ModePair:
		if roles.Thinker == "" {
			return errMissingRole("thinker")
		}
		if roles.Writer == "" {
			return errMissingRole("writer")
		}

		thinkerLabel := roles.Thinker + " (" + describeToolRun(roles.Thinker, rc) + ")"
		// Header'ı ÖNCE bas — çıktı streaming geldiği için kullanıcı kimin
		// konuştuğunu hemen bilsin.
		// Plan talimatı SADECE kod görevlerinde uygulanır. Aksi halde writer'ı
		// atlama kararı bozulurdu: thinker plan yazdığı için çıktıda kod bloğu
		// olmaz, istek de kod isteğine benzemiyorsa (aşağıdaki kontrol) ne plan
		// ne kod kalırdı. Kod isteği değilse thinker serbest çalışır ve
		// hasCodeBlock kontrolü eskisi gibi anlamlı olur.
		codeTask := looksLikeCodeRequest(input)
		thinkerInput := input
		if codeTask {
			thinkerInput = buildThinkerPrompt(input)
		}
		printAIHeader("thinker", roles.Thinker, rc)
		pairStart := time.Now()
		sp := StartSpinner("🧠 " + thinkerLabel + L(" düşünüyor...", " is thinking..."))
		thought, err := captureToolWithSpinner(roles.Thinker, rc, thinkerInput, sp)
		sp.Stop() // stopWriter da durdurmuş olabilir; Stop() idempotent (sync.Once)
		if err != nil {
			return err
		}
		thinkDur := time.Since(pairStart)
		printElapsed(pairStart, L("düşünme", "thinking"))
		// thought zaten captureTool tarafından stream edildi; tekrar basmıyoruz.

		// Writer kararı:
		//   - Aynı AI ise (thinker == writer) tekrar çağırmak çıktıyı duplike eder.
		//   - Soru kod istemiyorsa ve thinker zaten kod üretmediyse writer atlanır.
		if roles.Thinker == roles.Writer {
			fmt.Println(styleDim.Render(L("\n  (writer = thinker, ikinci çağrı atlandı)",
				"\n  (writer = thinker, second call skipped)")))
			return nil
		}
		if !codeTask && !hasCodeBlock(thought) {
			fmt.Println(styleDim.Render(L("\n  (yazılacak kod yok, writer atlandı)",
				"\n  (nothing to write, writer skipped)")))
			return nil
		}

		// Düşünme bitti, yazma başlıyor: iki AI'ın çıktısı arka arkaya aktığı
		// için görsel ayraç olmadan nerede bittiği anlaşılmıyor.
		printSeparator()
		printAIHeader("writer", roles.Writer, rc)
		// Writer'a giden metin: önce banner/log gürültüsü, sonra aracın
		// tekrarladığı final mesaj atılır — ikisi de boşuna token.
		writerInput := buildWriterPrompt(input,
			dedupeTrailingEcho(filterText(roles.Thinker, thought)))
		writeStart := time.Now()
		if err := runTool(roles.Writer, rc, writerInput, "✍️"); err != nil {
			return err
		}
		printElapsed(writeStart, L("yazma", "writing"))
		fmt.Println(styleDim.Render(fmt.Sprintf(
			L("  ⏱ toplam %s   (düşünme %s + yazma %s)",
				"  ⏱ total %s   (thinking %s + writing %s)"),
			formatDuration(time.Since(pairStart)),
			formatDuration(thinkDur),
			formatDuration(time.Since(writeStart)))))
		return nil
	}
	return fmt.Errorf("bilinmeyen mod")
}

// buildThinkerPrompt — pair modunda thinker'a giden metin.
//
// Talimatsız bırakıldığında thinker görevi baştan sona ÇÖZÜYOR: kodu tam
// olarak yazıyor, sonra writer aynı kodu bir kez daha yazıyor (ölçüldü:
// "pi'nin ilk 100 basamağını yazan script" isteğinde her ikisi de eksiksiz
// script üretti). Aynı iş iki kez faturalanıyor — üstelik thinker rolüne
// bilerek daha pahalı/derin model konuyor.
//
// Bu yüzden thinker'dan KOD değil PLAN isteniyor: pahalı model kararları
// verir, ucuz model yazar. ModeThink'te (tek başına 'cem "soru"') bu talimat
// UYGULANMAZ — orada kullanıcı doğrudan cevabı ister.
func buildThinkerPrompt(task string) string {
	if Lang() == LangEN {
		return strings.Join([]string{
			"Another AI will write the code for this task. Do NOT write the",
			"implementation yourself. Produce a short, concrete plan instead:",
			"which file(s), which functions and signatures, which algorithm or",
			"approach, and the edge cases that matter. Bullet points, no essay,",
			"no trade-off discussion. Short signatures or a few key lines are",
			"fine; a complete implementation is not.",
			"",
			"=== TASK ===",
			task,
		}, "\n")
	}
	return strings.Join([]string{
		"Bu görevin kodunu BAŞKA bir AI yazacak. Sen implementasyonu yazma.",
		"Onun yerine kısa ve somut bir plan çıkar: hangi dosya(lar), hangi",
		"fonksiyonlar ve imzalar, hangi algoritma/yaklaşım, hangi kenar durumlar",
		"önemli. Madde madde, uzun anlatım yok, trade-off tartışması yok. Kısa",
		"imza ya da birkaç kritik satır olabilir; eksiksiz implementasyon olmaz.",
		"",
		"=== GÖREV ===",
		task,
	}, "\n")
}

// dedupeTrailingEcho — bazı CLI'lar final mesajı stream'in sonunda BİR KEZ
// DAHA basıyor (codex exec: mesaj → "tokens used N" → aynı mesaj). Ekrandaki
// tekrar aracın kendi çıktısı, ona karışmıyoruz; ama writer'a iki kopya
// göndermek prompt'u boşuna iki katına çıkarır.
//
// Yöntem: metnin sonunda yer alan ve DAHA ÖNCE birebir geçen en uzun bloğu
// bulup atar. minEchoLen eşiği, meşru kısa tekrarların (kısa kod parçası,
// tekrarlanan uyarı satırı) yanlışlıkla kırpılmasını önler.
const minEchoLen = 200

func dedupeTrailingEcho(s string) string {
	t := strings.TrimRight(s, "\n")
	n := len(t)
	if n < 2*minEchoLen {
		return s
	}
	best := 0
	lo, hi := minEchoLen, n/2
	for lo <= hi {
		mid := (lo + hi) / 2
		if strings.Contains(t[:n-mid], t[n-mid:]) {
			best = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	if best == 0 {
		return s
	}
	return strings.TrimRight(t[:n-best], "\n")
}

// buildWriterPrompt — writer'a düşünenin çıktısını + net "tekrar analiz
// yapma" talimatı ile besler. Amaç: writer prompt'u yorumlamak yerine
// doğrudan implementasyona geçsin, thinker'ın işini tekrarlamasın.
func buildWriterPrompt(originalTask, thinkerOutput string) string {
	if Lang() == LangEN {
		return strings.Join([]string{
			"Another AI has already analysed/planned this task. Your job: produce the CODE",
			"that implements the analysis below. Do not redo the analysis, do not explain the",
			"plan, do not discuss trade-offs. Write working, complete code only (short inline",
			"comments if needed). If there are multiple files, say so explicitly.",
			"",
			"=== ORIGINAL TASK ===",
			originalTask,
			"",
			"=== THINKER ANALYSIS (FOLLOW IT) ===",
			thinkerOutput,
			"",
			"=== NOW WRITE THE CODE ===",
		}, "\n")
	}
	return strings.Join([]string{
		"Görev için bir başka AI tarafından analiz/plan yapıldı. Senin işin: aşağıdaki",
		"analizi uygulayan KODU üretmek. Tekrar analiz yapma, plan açıklaması ekleme,",
		"trade-off tartışması yapma. Sadece çalışan, eksiksiz kodu yaz (gerekirse kısa",
		"inline yorum). Birden fazla dosya varsa açıkça belirt.",
		"",
		"=== ORİJİNAL GÖREV ===",
		originalTask,
		"",
		"=== THINKER ANALİZİ (TAKİP ET) ===",
		thinkerOutput,
		"",
		"=== ŞİMDİ KODU YAZ ===",
	}, "\n")
}

// buildArgs — bir AI CLI çağrısının komut argümanlarını oluşturur. Sıra:
//
//	ModelBeforeRun=true  → [--model X, RunFlags..., input?]
//	ModelBeforeRun=false → [RunFlags..., --model X, input?]
//
// agy/cursor -p "PROMPT" alır; --model -p ile prompt arasına girerse -p'nin
// değeri "--model" oluyor. Bunu önlemek için ModelBeforeRun=true.
func buildArgs(meta ToolMeta, toolKey string, rc *ResolvedConfig, input string) []string {
	args := buildArgsNoPrompt(meta, toolKey, rc)
	if meta.PromptAsArg {
		args = append(args, input)
	}
	return args
}

// buildArgsNoPrompt — prompt HARİÇ argümanlar. runQuiet, araya kendi
// --output-last-message <dosya> çiftini eklemek için buna ihtiyaç duyuyor:
// prompt her zaman en sonda kalmalı.
func buildArgsNoPrompt(meta ToolMeta, toolKey string, rc *ResolvedConfig) []string {
	model := resolveModel(toolKey, rc)
	includeModel := model != "" && meta.ModelFlag != ""
	effortArgs := buildEffortArgs(meta, toolKey, rc)
	args := []string{}
	if includeModel && meta.ModelBeforeRun {
		args = append(args, meta.ModelFlag, model)
	}
	// Effort de model ile aynı kuralı izler: prompt-flag'i argüman yutan
	// araçlarda (claude -p, cursor -p) RunFlags'ten ÖNCE gelmeli.
	if meta.ModelBeforeRun {
		args = append(args, effortArgs...)
	}
	args = append(args, meta.RunFlags...)
	if includeModel && !meta.ModelBeforeRun {
		args = append(args, meta.ModelFlag, model)
	}
	if !meta.ModelBeforeRun {
		args = append(args, effortArgs...)
	}
	return args
}

// buildEffortArgs — seçili düşünme seviyesini aracın beklediği argüman
// biçimine çevirir. Seviye seçilmemişse veya araç desteklemiyorsa boş döner.
func buildEffortArgs(meta ToolMeta, toolKey string, rc *ResolvedConfig) []string {
	if len(meta.EffortArgs) == 0 {
		return nil
	}
	effort := resolveEffort(toolKey, rc)
	if effort == "" {
		return nil
	}
	out := make([]string, 0, len(meta.EffortArgs))
	for _, tpl := range meta.EffortArgs {
		if strings.Contains(tpl, "%s") {
			out = append(out, fmt.Sprintf(tpl, effort))
		} else {
			out = append(out, tpl)
		}
	}
	return out
}

// resolveEffort — toolKey için kullanılacak düşünme seviyesi. Öncelik
// resolveModel ile aynı: proje .cem.yaml → global config → CLI default.
// Araç seviye seçimini desteklemiyorsa her zaman "" döner.
func resolveEffort(toolKey string, rc *ResolvedConfig) string {
	meta, ok := KnownTools[toolKey]
	if !ok || len(meta.EffortArgs) == 0 {
		return ""
	}
	if rc.Project != nil && rc.Project.Efforts != nil {
		if e, ok := rc.Project.Efforts[toolKey]; ok && e != "" {
			return e
		}
	}
	if t, ok := rc.Global.Tools[toolKey]; ok && t.Effort != "" {
		return t.Effort
	}
	return ""
}

// resolveModel — toolKey için kullanılacak modeli döndürür. Sıra:
// 1) Proje config'i (.cem.yaml > models > <key>)
// 2) Global config (~/.cem/config.yaml > tools > <key> > model)
// 3) Boş → CLI default kullanılır.
//
// Tool'un ModelFlag'i yoksa (örn. agy) hangi config'de ne yazarsa yazsın
// "" döner — CLI default kullanılır, header de doğru "(default)" gösterir.
func resolveModel(toolKey string, rc *ResolvedConfig) string {
	if meta, ok := KnownTools[toolKey]; ok && meta.ModelFlag == "" {
		return "" // tool model seçimini desteklemiyor
	}
	if rc.Project != nil && rc.Project.Models != nil {
		if m, ok := rc.Project.Models[toolKey]; ok && m != "" {
			return m
		}
	}
	if t, ok := rc.Global.Tools[toolKey]; ok && t.Model != "" {
		return t.Model
	}
	return ""
}

// printAIHeader — her AI çıktısının üstüne kim olduğunu + hangi modeli
// kullandığını gösteren başlık. Örnek:
//
//	─── 🧠 thinker · claude (opus) ───
//	─── ✍️  writer · agy (gemini-3-flash) ───
//	─── 🧠 thinker · claude (default) ───   // model seçilmemiş, CLI default
func printAIHeader(kind, toolKey string, rc *ResolvedConfig) {
	icon, label, style := "🧠", L("DÜŞÜNEN", "THINKING"), styleThinker
	if kind == "writer" {
		icon, label, style = "✍️ ", L("YAZAN", "WRITING"), styleWriter
	}
	fmt.Println()
	fmt.Println(style.Render(fmt.Sprintf("  %s %s · %s", icon, label, toolKey)) +
		styleDim.Render("  "+describeToolRun(toolKey, rc)))
}

// formatDuration — kısa, okunur süre: "8.3s", "1m 04s".
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d / time.Minute)
	sec := int((d % time.Minute) / time.Second)
	return fmt.Sprintf("%dm %02ds", m, sec)
}

// printElapsed — bir rolün ne kadar sürdüğünü çıktının altına yazar.
func printElapsed(start time.Time, label string) {
	fmt.Println(styleDim.Render(fmt.Sprintf("  ⏱ %s %s", label, formatDuration(time.Since(start)))))
}

// printSeparator — düşünen ile yazan arasındaki görsel sınır.
func printSeparator() {
	fmt.Println()
	fmt.Println(styleDim.Render("  " + strings.Repeat("─", 52)))
}

// describeToolRun — header/spinner etiketi: "sonnet · high", "gpt-5.6-terra",
// seçim yoksa "default".
func describeToolRun(toolKey string, rc *ResolvedConfig) string {
	model := resolveModel(toolKey, rc)
	effort := resolveEffort(toolKey, rc)
	switch {
	case model == "" && effort == "":
		return "default"
	case model == "":
		return "default · " + effort
	case effort == "":
		return model
	default:
		return model + " · " + effort
	}
}

func errMissingRole(name string) error {
	msg := fmt.Sprintf(L("%s rolü atanmamış — cem roles ile ayarla",
		"%s role is not assigned — set it with: cem roles"), name)
	fmt.Println(styleError.Render("✗ " + msg))
	return fmt.Errorf("%s", msg)
}

// resolveCommand — config'de saklanan command tercih edilir, yoksa tool key
func resolveCommand(toolKey string, rc *ResolvedConfig) string {
	binName := toolKey
	if meta, ok := KnownTools[toolKey]; ok && meta.Binary != "" {
		binName = meta.Binary
	}
	if t, ok := rc.Global.Tools[toolKey]; ok && t.Command != "" {
		// Config'deki yol hâlâ geçerli mi?
		if _, err := exec.LookPath(t.Command); err == nil {
			return t.Command
		}
	}
	// PATH'da düz isimle var mı?
	if _, err := exec.LookPath(binName); err == nil {
		return binName
	}
	// Bilinen kurulum konumlarını dene (bazı installer'lar PATH'i güncellemiyor).
	if p := fallbackInstallPath(toolKey); p != "" {
		return p
	}
	return binName
}

// rateLimitRe — stderr'de rate-limit / quota imzaları (provider'lar arası).
var rateLimitRe = regexp.MustCompile(`(?i)(rate.?limit|quota|429|too many requests|usage limit|overloaded)`)

// authFailRe — stderr'de yetkilendirme hatası imzaları (401, eksik token,
// interaktif OAuth prompt'ları). Rate-limit'ten farklıdır: rotasyonla
// çözülmez, kullanıcı login/key müdahalesi gerekir.
var authFailRe = regexp.MustCompile(`(?i)(401|unauthorized|missing bearer|invalid api key|not.?logged.?in|please run /login|please log in|authentication failed|authentication required|please visit the url|paste the authorization code|authentication interrupted|waiting for authentication)`)

// errRateLimit — withKeyRotation iç sinyali. Dışarı sızmaz; tüm key'ler bittiğinde
// gerçek alt-process hatasına dönüşür.
var errRateLimit = errors.New("rate limit / quota")

// modelUnsupportedRe — seçili modelin hesap/plan tarafından reddedildiği
// durumlar. Auth hatasından ÖNCE denenir: mesajda ikisi birden geçebiliyor
// ve kullanıcıyı boş yere login akışına yollamak istemiyoruz.
var modelUnsupportedRe = regexp.MustCompile(`(?i)` +
	`model[^\n]{0,60}?(is )?not supported` + // codex: "The 'x' model is not supported when using..."
	`|unsupported model|unknown model|model not found|invalid model` +
	`|model ` + "`" + `[^` + "`" + `]+` + "`" + ` does not exist` + // codex 404: model `gpt-5.5` does not exist
	`|does not exist or you do not have access` +
	`|no access to (the )?model|you do not have access to (this |that )?model`)

// effortUnsupportedRe — seçilen düşünme seviyesi araç/model tarafından
// reddedildi. modelUnsupportedRe'den ÖNCE denenmeli: bu mesajlarda "model"
// kelimesi de geçiyor ("'minimal' is not supported with the 'x' model"),
// yoksa kullanıcı modeli değiştirmeye yönlendirilir — oysa sorun seviyede.
var effortUnsupportedRe = regexp.MustCompile(`(?i)(reasoning[._ ]effort|model_reasoning_effort)`)

// noiseLineMax — alt araçların stderr'e döktüğü ham HTTP gövdeleri tek
// satırda 100 KB'ı bulabiliyor. İçlerinde "unauthorized", "429", "quota",
// "authentication failed" gibi ifadeler VERİ olarak geçer; imza taramasına
// sokulursa yanlış teşhis üretir. Ölçüldü (2026-09-09): codex'in
// "failed to refresh available models" logu 105 KB'lık model JSON'u basıyor,
// guardian prompt metninde "authentication failed" geçiyordu → cem gerçek
// hatayı ("model not supported") gizleyip "auth eksik" diyordu.
const noiseLineMax = 400

// bodyDumpRe — "body: {...}" / "body: [...]" sonrası ham gövde.
var bodyDumpRe = regexp.MustCompile(`(?i)\bbody:\s*[\[{].*$`)

// sanitizeStderr — imza taraması öncesi çıktıyı temizler: ham gövde
// dökümlerini kırpar, kalan aşırı uzun satırları tamamen atar.
func sanitizeStderr(s string) string {
	lines := strings.Split(s, "\n")
	kept := make([]string, 0, len(lines))
	for _, ln := range lines {
		ln = bodyDumpRe.ReplaceAllString(ln, "body: <kırpıldı>")
		if len(ln) > noiseLineMax {
			continue
		}
		kept = append(kept, ln)
	}
	return strings.Join(kept, "\n")
}

func looksLikeRateLimit(stderr string) bool {
	return rateLimitRe.MatchString(sanitizeStderr(stderr))
}

func looksLikeAuthFailure(stderr string) bool {
	return authFailRe.MatchString(sanitizeStderr(stderr))
}

func looksLikeModelUnsupported(out string) bool {
	return modelUnsupportedRe.MatchString(sanitizeStderr(out))
}

func looksLikeEffortUnsupported(out string) bool {
	s := sanitizeStderr(out)
	return effortUnsupportedRe.MatchString(s) &&
		(modelUnsupportedRe.MatchString(s) || strings.Contains(strings.ToLower(s), "unsupported"))
}

// tailWriter — sadece son n byte'ı tutar. runTool'un stdout'unu hata imzası
// için taramak gerekiyor ama writer çıktısı MB'larca kod olabilir; tamamını
// buffer'lamak gereksiz.
type tailWriter struct {
	buf []byte
	n   int
}

func (w *tailWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	if len(w.buf) > w.n {
		w.buf = w.buf[len(w.buf)-w.n:]
	}
	return len(p), nil
}

func (w *tailWriter) String() string { return string(w.buf) }

// hintAuth — auth hatası tespit edildiğinde kullanıcıya net düzeltme yolu sun.
// toolKey paramı ile 'cem auth <toolKey>' önerebiliyoruz.
func hintAuth(bin, toolKey string, meta ToolMeta, cfg *GlobalConfig) {
	fmt.Println()
	fmt.Println(styleWarn.Render("  ⚠ " + bin + L(" yetkilendirilmemiş — auth eksik veya interaktif login akışı kesildi",
		" is not authorized — missing auth, or the interactive login flow was interrupted")))
	fmt.Println(styleDim.Render(fmt.Sprintf(L("    Önerilen: cem auth %s         (pano-yapıştır yardımcısı dahil)",
		"    Suggested: cem auth %s        (includes the clipboard-paste helper)"), toolKey)))
	if meta.Provider != "" {
		if len(cfg.APIKeys[meta.Provider]) > 0 {
			fmt.Println(styleDim.Render(fmt.Sprintf(
				L("    Veya kayıtlı %d %s key var ama biri/hepsi geçersiz olabilir:",
					"    Or: %d stored %s key(s) exist but some/all may be invalid:"),
				len(cfg.APIKeys[meta.Provider]), meta.Provider)))
			fmt.Println(styleDim.Render("      cem keys list"))
			fmt.Println(styleDim.Render(fmt.Sprintf("      cem keys remove %s <index>", meta.Provider)))
		} else {
			fmt.Println(styleDim.Render(fmt.Sprintf(L("    Veya yeni API key: cem keys add %s",
				"    Or add a new API key: cem keys add %s"), meta.Provider)))
		}
	}
}

// hintEffort — düşünme seviyesi reddedildiğinde. Aracın kendi hata mesajı
// geçerli listeyi zaten yazıyor; biz nereden değiştirileceğini söylüyoruz.
func hintEffort(bin, toolKey string, meta ToolMeta, rc *ResolvedConfig) {
	level := resolveEffort(toolKey, rc)
	fmt.Println()
	fmt.Println(styleWarn.Render(fmt.Sprintf(
		L("  ⚠ %s: '%s' düşünme seviyesi bu model tarafından kabul edilmedi",
			"  ⚠ %s: reasoning effort '%s' was rejected by this model"), bin, level)))
	if len(meta.Efforts) > 0 {
		fmt.Println(styleDim.Render(L("    Bilinen seviyeler: ", "    Known levels: ") +
			strings.Join(meta.Efforts, ", ")))
	}
	fmt.Println(styleDim.Render(fmt.Sprintf("    cem effort %s high", toolKey)))
	fmt.Println(styleDim.Render(fmt.Sprintf(
		L("    kaldırmak için: cem effort %s default", "    to clear it: cem effort %s default"), toolKey)))
}

// hintModel — seçili model hesap/plan tarafından reddedildiğinde nereden
// değiştirileceğini gösterir. Auth önerisi vermek burada yanlış olur.
func hintModel(bin, toolKey string, meta ToolMeta, rc *ResolvedConfig) {
	model := resolveModel(toolKey, rc)
	if model == "" {
		model = "(CLI default)"
	}
	fmt.Println()
	fmt.Println(styleWarn.Render(fmt.Sprintf(
		L("  ⚠ %s: '%s' modeli bu hesap/plan ile kullanılamıyor",
			"  ⚠ %s: model '%s' is not available on this account/plan"), bin, model)))
	if len(meta.Models) > 0 {
		fmt.Println(styleDim.Render(L("    Bilinen modeller: ", "    Known models: ") +
			strings.Join(meta.Models, ", ")))
	}
	fmt.Println(styleDim.Render(L("    Değiştir: cem setup", "    Change it: cem setup")))
	fmt.Println(styleDim.Render(fmt.Sprintf(
		"      global   ~/.cem/config.yaml → tools.%s.model", toolKey)))
	fmt.Println(styleDim.Render(fmt.Sprintf(
		L("      proje    .cem.yaml → models.%s", "      project  .cem.yaml → models.%s"), toolKey)))
}

// stopWriter — ilk yazımda spinner'ı durdurur, sonrasında verileri inner'a iletir.
// Interactive prompt'ların (OAuth URL'leri, kod yapıştırma çağrıları) spinner
// tarafından üzerine yazılmasını engeller.
type stopWriter struct {
	sp      *Spinner
	inner   io.Writer
	stopped bool
}

func (w *stopWriter) Write(p []byte) (int, error) {
	if !w.stopped && w.sp != nil {
		w.sp.Stop()
		w.stopped = true
	}
	return w.inner.Write(p)
}

// withKeyRotation — meta.Provider varsa cfg.APIKeys[provider] içinden sırayla
// her key'i env değişkeni olarak set edip fn'i çağırır. fn errRateLimit
// dönerse sonraki key denenir. Provider tanımlı değilse fn bir kez OS env'iyle
// çalıştırılır (CLI'ın kendi auth'u devrede).
func withKeyRotation(meta ToolMeta, cfg *GlobalConfig, fn func(env []string) error) error {
	baseEnv := os.Environ()
	if meta.Provider == "" || meta.APIKeyEnv == "" {
		return fn(baseEnv)
	}
	keys := cfg.APIKeys[meta.Provider]
	if len(keys) == 0 {
		// Key tanımlanmamış → CLI'ın mevcut auth'unu kullan
		return fn(baseEnv)
	}
	var lastErr error
	for i, k := range keys {
		env := append(append([]string{}, baseEnv...), meta.APIKeyEnv+"="+k.Value)
		// Aynı env değişkeni baseEnv'de varsa Go'nun exec son tanımı kullanır,
		// yani append yeterli.
		err := fn(env)
		if err == nil {
			return nil
		}
		if !errors.Is(err, errRateLimit) {
			return err // başka bir hata: tekrar denemenin anlamı yok
		}
		lastErr = err
		label := k.Label
		if label == "" {
			label = fmt.Sprintf("#%d", i+1)
		}
		if i+1 < len(keys) {
			fmt.Println(styleWarn.Render(fmt.Sprintf(L("  ⚠ %s rate limit — sonraki key'e geçiliyor",
				"  ⚠ %s rate limited — switching to the next key"), label)))
		} else {
			fmt.Println(styleError.Render(fmt.Sprintf(L("  ✗ tüm %s key'leri rate limit",
				"  ✗ all %s keys are rate limited"), meta.Provider)))
		}
	}
	return lastErr
}

// codeRequestRe — input'ta kod yazma niyetini gösteren kelimeler (TR + EN).
// codeRequestKeywords — pair modunda "bu bir kod işi mi" kararının sözlüğü.
// Karar iki yeri etkiliyor: thinker'a plan talimatı verilip verilmeyeceği ve
// writer'ın atlanıp atlanmayacağı.
const codeRequestKeywords = `yaz|kod|kodla|script|fonksiyon|class|method|implement|` +
	`oluştur|üret|döndür|export|function|code|write|build|generate|refactor|debug|fix|` +
	// Kod işi olduğu hâlde "yaz" geçmeyen istekler.
	`optimize|optimizasyon|düzelt|ekle|sil|kaldır|taşı|dönüştür|çevir|test|port|` +
	`add|remove|rename|update|migrate|patch|extend|convert|wrap|hook`

// codeRequestRe — kelime sınırı için \b KULLANILMAZ: Go'nun \b'si ASCII
// tabanlı, "çevir"/"üret" gibi Türkçe harfle başlayan kelimeler bir boşluktan
// sonra gelse bile hiç eşleşmiyordu (sessizce: writer atlanıyordu).
var codeRequestRe = regexp.MustCompile(`(?i)(^|[^\p{L}])(` + codeRequestKeywords + `)([^\p{L}]|$)`)

// hasCodeBlock — metin markdown kod bloğu içeriyor mu (``` veya satır başı 4-boşluk değil).
func hasCodeBlock(s string) bool {
	return strings.Contains(s, "```")
}

// looksLikeCodeRequest — input metni kod yazılması/üretilmesi gerektiğini ima ediyor mu.
func looksLikeCodeRequest(s string) bool {
	return codeRequestRe.MatchString(s)
}

// fallbackInstallPath — araç PATH'da yoksa standart konumlarda arar.
func fallbackInstallPath(toolKey string) string {
	home, _ := os.UserHomeDir()
	candidates := []string{}
	switch toolKey {
	case "agy":
		if runtime.GOOS == "windows" {
			if lad := os.Getenv("LOCALAPPDATA"); lad != "" {
				candidates = append(candidates, filepath.Join(lad, "agy", "bin", "agy.exe"))
			}
		} else {
			candidates = append(candidates, filepath.Join(home, ".local", "bin", "agy"))
		}
	case "claude":
		// Native installer (claude.ai/install.sh) önce ~/.claude/local/bin/'e koyup
		// shell rc'lerine PATH ekler. Mevcut süreçte hâlâ yoksa direkt yolu deneyelim.
		if runtime.GOOS == "windows" {
			if lap := os.Getenv("LOCALAPPDATA"); lap != "" {
				candidates = append(candidates, filepath.Join(lap, "Claude", "claude.exe"))
			}
			candidates = append(candidates, filepath.Join(home, ".claude", "local", "claude.exe"))
		} else {
			candidates = append(candidates,
				filepath.Join(home, ".claude", "local", "claude"),
				filepath.Join(home, ".local", "bin", "claude"),
			)
		}
	case "cursor":
		if runtime.GOOS == "windows" {
			lad := os.Getenv("LOCALAPPDATA")
			appd := os.Getenv("APPDATA")
			// Cursor native installer Windows'ta .cmd + .ps1 launcher koyar
			// (.exe değil — JS-tabanlı agent), root: %LOCALAPPDATA%\cursor-agent\
			if lad != "" {
				for _, base := range []string{
					filepath.Join(lad, "cursor-agent"),
					filepath.Join(lad, "Programs", "cursor-agent"),
					filepath.Join(lad, "Programs", "cursor"),
				} {
					for _, name := range []string{
						"cursor-agent.cmd", "cursor-agent.exe", "cursor-agent.ps1",
						"agent.cmd", "agent.exe",
					} {
						candidates = append(candidates, filepath.Join(base, name))
					}
				}
			}
			if appd != "" {
				// Legacy npm global bin (eski cemi npm install ile gelmişse)
				candidates = append(candidates,
					filepath.Join(appd, "npm", "cursor-agent.cmd"),
					filepath.Join(appd, "npm", "cursor-agent.ps1"),
					filepath.Join(appd, "npm", "cursor-agent"))
			}
			candidates = append(candidates,
				filepath.Join(home, ".local", "bin", "cursor-agent.exe"),
				filepath.Join(home, ".local", "bin", "cursor-agent.cmd"))
		} else {
			candidates = append(candidates,
				filepath.Join(home, ".local", "bin", "cursor-agent"),
				filepath.Join(home, ".local", "bin", "agent"))
		}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// runQuiet — LastMessageFlag'i olan araçlar için: ham akış ekrana basılmaz,
// çalışırken spinner döner, sonunda araç kendi yazdığı final mesaj dosyadan
// okunup tek parça basılır. Dönen string writer'a beslenecek metindir.
//
// Neden: codex exec, cevabı verirken çalıştırdığı her komutu, ürettiği her
// patch'i ve aynı diff'i defalarca stdout'a döküyor (ölçüldü: tek bir "pi
// script'i yaz" isteğinde aynı 28 satırlık diff 4 kez basıldı). Kullanıcı
// asıl cevabı bulamıyor.
func runQuiet(toolKey string, rc *ResolvedConfig, input string, meta ToolMeta,
	bin string, label string, sp *Spinner) (string, error) {

	tmp, err := os.CreateTemp("", "cem-last-*.txt")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	args := append(buildArgsNoPrompt(meta, toolKey, rc), meta.LastMessageFlag, tmpPath)
	if meta.PromptAsArg {
		args = append(args, input)
	}

	var out string
	var stderrText string
	if sp == nil {
		sp = StartSpinner(label)
	}
	runErr := withKeyRotation(meta, rc.Global, func(env []string) error {
		cmd := exec.Command(bin, args...)
		if !meta.PromptAsArg {
			cmd.Stdin = strings.NewReader(input)
		}
		// Ham akış ekrana gitmez ama hata imzası taraması için tutulur.
		outTail := &tailWriter{n: 8 << 10}
		cmd.Stdout = outTail
		var errBuf bytes.Buffer
		// Quiet modda stderr EKRANA HİÇ BASILMAZ: codex hem banner'ını hem de
		// cevabın kendisini stderr'e yazıyor; ekrana verirsek dosyadan
		// bastığımız final mesajla birlikte cevap İKİ KEZ görünür.
		//
		// Burada bir ara çözüm denendi ve geri alındı: "içinde bağlantı geçen
		// satırları geçir" (interaktif login URL'i kaçmasın diye). Cevabın
		// kendisi de link içerebiliyor ve tek uzun satır olduğunda çıktının
		// tamamı sızıyor — sahada tam olarak bu oldu. Login/hata bilgisi
		// zaten aşağıda gösteriliyor: hata varsa ya da araç final mesaj
		// üretemediyse stderr'in son satırları basılır.
		cmd.Stderr = &errBuf
		cmd.Env = env
		err := cmd.Run()
		stderrText = errBuf.String()
		if err == nil {
			return nil
		}
		combined := stderrText + "\n" + outTail.String()
		if looksLikeRateLimit(combined) {
			return errRateLimit
		}
		switch {
		case looksLikeEffortUnsupported(combined):
			hintEffort(bin, toolKey, meta, rc)
		case looksLikeModelUnsupported(combined):
			hintModel(bin, toolKey, meta, rc)
		case looksLikeAuthFailure(combined):
			hintAuth(bin, toolKey, meta, rc.Global)
		default:
			fmt.Println(styleError.Render("✗ " + bin + L(" hata: ", " error: ") + err.Error()))
			// Quiet modda ham akış gizli; hata varsa son satırları göster.
			printTail(filterText(toolKey, errBuf.String()), 8)
		}
		return err
	})
	sp.Stop()
	if runErr != nil {
		return "", runErr
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil || len(bytes.TrimSpace(data)) == 0 {
		// Araç final mesaj üretmediyse sessiz kalmayalım: çoğu zaman sebebi
		// stderr'de duruyor (login isteği, kota uyarısı).
		fmt.Println(styleDim.Render(L("  (araç final mesaj üretmedi — ham çıktı için: --raw)",
			"  (the tool produced no final message — use --raw for the raw output)")))
		printTail(filterText(toolKey, stderrText), 10)
		return "", nil
	}
	out = strings.TrimRight(string(data), "\n")
	fmt.Println(out)
	return out, nil
}

// usesQuietRun — bu araç için sessiz mod geçerli mi.
func usesQuietRun(meta ToolMeta) bool {
	return meta.LastMessageFlag != "" && !rawOutput
}

// runTool — stdin'i pipe edip stdout/stderr'i kullanıcıya gösterir
func runTool(toolKey string, rc *ResolvedConfig, input, icon string) error {
	bin := resolveCommand(toolKey, rc)
	if _, err := exec.LookPath(bin); err != nil {
		fmt.Println(styleError.Render(
			fmt.Sprintf(L("✗ %s bulunamadı — kurmak için: cemi %s",
				"✗ %s not found — install it with: cemi %s"), bin, toolKey)))
		return err
	}

	meta := KnownTools[toolKey]
	if usesQuietRun(meta) {
		verb := L(" düşünüyor...", " is thinking...")
		if icon == "✍️" {
			verb = L(" yazıyor...", " is writing...")
		}
		_, err := runQuiet(toolKey, rc, input, meta, bin, icon+" "+toolKey+verb, nil)
		return err
	}
	args := buildArgs(meta, toolKey, rc, input)

	return withKeyRotation(meta, rc.Global, func(env []string) error {
		cmd := exec.Command(bin, args...)
		if !meta.PromptAsArg {
			cmd.Stdin = strings.NewReader(input)
		}
		// Bazı CLI'lar (codex) API hatasını stdout'a yazıyor; imza taraması
		// stdout+stderr birleşimi üzerinde yapılmalı.
		outTail := &tailWriter{n: 8 << 10}
		// Ekrana giden kopya filtrelenir (banner/log gürültüsü), hata imzası
		// taraması için tutulan kopya HAM kalır.
		nf := newNoiseFilter(toolKey, os.Stdout, rawOutput)
		cmd.Stdout = io.MultiWriter(nf, outTail)
		// stderr'i hem konsola yansıt hem buffer'a yaz (rate-limit / auth imzasını yakalamak için).
		// runTool zaten spinner çalıştırmıyor, stopWriter pass-through olur.
		var errBuf bytes.Buffer
		enf := newNoiseFilter(toolKey, os.Stderr, rawOutput)
		cmd.Stderr = io.MultiWriter(enf, &errBuf)
		cmd.Env = env
		err := cmd.Run()
		nf.Close()
		enf.Close()
		if err == nil {
			return nil
		}
		combined := errBuf.String() + "\n" + outTail.String()
		if looksLikeRateLimit(combined) {
			return errRateLimit
		}
		switch {
		case looksLikeEffortUnsupported(combined):
			hintEffort(bin, toolKey, meta, rc)
		case looksLikeModelUnsupported(combined):
			hintModel(bin, toolKey, meta, rc)
		case looksLikeAuthFailure(combined):
			hintAuth(bin, toolKey, meta, rc.Global)
		default:
			fmt.Println(styleError.Render("✗ " + bin + L(" hata: ", " error: ") + err.Error()))
		}
		return err
	})
}

// captureTool — pair modu için: çıktıyı yakalar
func captureTool(toolKey string, rc *ResolvedConfig, input string) (string, error) {
	bin := resolveCommand(toolKey, rc)
	if _, err := exec.LookPath(bin); err != nil {
		fmt.Println(styleError.Render(
			fmt.Sprintf(L("✗ %s bulunamadı — kurmak için: cemi %s",
				"✗ %s not found — install it with: cemi %s"), bin, toolKey)))
		return "", err
	}

	return captureToolWithSpinner(toolKey, rc, input, nil)
}

// captureToolWithSpinner — captureTool'un spinner-aware versiyonu. Subprocess
// ilk byte'ı stderr'e yazdığında verilen spinner durur (OAuth prompt'ları görünsün).
// sp nil ise düz capture.
func captureToolWithSpinner(toolKey string, rc *ResolvedConfig, input string, sp *Spinner) (string, error) {
	bin := resolveCommand(toolKey, rc)
	if _, err := exec.LookPath(bin); err != nil {
		fmt.Println(styleError.Render(
			fmt.Sprintf(L("✗ %s bulunamadı — kurmak için: cemi %s",
				"✗ %s not found — install it with: cemi %s"), bin, toolKey)))
		return "", err
	}
	meta := KnownTools[toolKey]
	if usesQuietRun(meta) {
		// Çağıranın spinner'ı varsa onu sürdür: iki ayrı spinner mesajı
		// arka arkaya yanıp sönmesin.
		return runQuiet(toolKey, rc, input, meta, bin,
			"🧠 "+toolKey+L(" düşünüyor...", " is thinking..."), sp)
	}
	args := buildArgs(meta, toolKey, rc, input)
	var captured bytes.Buffer
	err := withKeyRotation(meta, rc.Global, func(env []string) error {
		cmd := exec.Command(bin, args...)
		if !meta.PromptAsArg {
			cmd.Stdin = strings.NewReader(input)
		}
		captured.Reset()
		// Stream + capture: thinker çıktısı plugin/terminal'e ANINDA akar
		// ve buffer'a kopyalanır (writer fazı için).
		nf := newNoiseFilter(toolKey, os.Stdout, rawOutput)
		cmd.Stdout = io.MultiWriter(nf, &captured)
		var errBuf bytes.Buffer
		sw := &stopWriter{sp: sp, inner: os.Stderr}
		enf := newNoiseFilter(toolKey, sw, rawOutput)
		cmd.Stderr = io.MultiWriter(enf, &errBuf)
		cmd.Env = env
		runErr := cmd.Run()
		nf.Close()
		enf.Close()
		if runErr == nil {
			return nil
		}
		combined := errBuf.String() + "\n" + captured.String()
		if looksLikeRateLimit(combined) {
			return errRateLimit
		}
		switch {
		case looksLikeEffortUnsupported(combined):
			hintEffort(bin, toolKey, meta, rc)
		case looksLikeModelUnsupported(combined):
			hintModel(bin, toolKey, meta, rc)
		case looksLikeAuthFailure(combined):
			hintAuth(bin, toolKey, meta, rc.Global)
		default:
			// Sessiz çıkmayalım: pair modunda thinker patlarsa kullanıcı
			// tek satır cem mesajı görmeden exit 1 alıyordu.
			fmt.Println(styleError.Render("✗ " + bin + L(" hata: ", " error: ") + runErr.Error()))
		}
		return runErr
	})
	if err != nil {
		return "", err
	}
	return strings.TrimRight(captured.String(), "\n"), nil
}
