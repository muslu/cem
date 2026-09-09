package main

import (
	"bytes"
	"io"
	"regexp"
	"strings"
)

// AI CLI'ları asıl cevaptan önce kendi banner'larını, oturum kimliklerini ve
// iç loglarını basıyor. İlk kez cem kullanan biri ekranı doldurmuş bu
// satırlarla asıl cevabı ayırt edemiyor (ölçüldü: codex'in banner'ı 13 satır,
// cevap 1 satır). noiseFilter bunları eler; ham çıktı gerekiyorsa `--raw`.

// commonNoise — araçtan bağımsız gürültü: zaman damgalı iç loglar ve
// Claude Code'un izin-kuralı uyarısı.
var commonNoise = []*regexp.Regexp{
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*\s(ERROR|WARN|INFO|DEBUG)\s`),
	regexp.MustCompile(`^Permission allow rule \(`),
	regexp.MustCompile(`^\s*$`), // arka arkaya boş satırlar aşağıda tekilleştirilir
}

// toolNoise — araca özel banner satırları.
var toolNoise = map[string][]*regexp.Regexp{
	"gpt": {
		regexp.MustCompile(`^Reading additional input from stdin`),
		regexp.MustCompile(`^OpenAI Codex v`),
		regexp.MustCompile(`^-{4,}$`),
		regexp.MustCompile(`^(workdir|model|provider|approval|sandbox|reasoning effort|reasoning summaries|session id):`),
		regexp.MustCompile(`^(user|codex)$`),
		regexp.MustCompile(`^ERROR: Reconnecting\.\.\. \d+/\d+$`),
	},
	"agy": {
		regexp.MustCompile(`^Loaded cached credentials`),
	},
	"cursor": {
		regexp.MustCompile(`^Using model:`),
	},
	"claude": {
		// Kullanıcının hook'ları her çağrıda çalışıyor ve başarısız olduklarında
		// komutun tamamını (yüzlerce karakter) ekrana döküyorlar. cem'in
		// cevabıyla ilgisi yok; ham çıktı gerekiyorsa --raw.
		regexp.MustCompile(`^\w+ hook \[`),
		regexp.MustCompile(`hook .*\] failed:`),
		regexp.MustCompile(`^Warning: no stdin data received`),
	},
}

// stopAfter — bu satır görüldükten SONRAKİ her şey atılır. codex, cevabı
// yazdıktan sonra "tokens used" özetini ve ardından cevabın TAMAMINI bir kez
// daha basıyor; kullanıcı aynı metni iki kez okuyor.
var stopAfter = map[string]*regexp.Regexp{
	"gpt": regexp.MustCompile(`^tokens used$`),
}

// noiseFilter — satır tamponlu yazıcı. Write çağrıları satır sınırında
// bölünmediği için yarım satırlar tamponda bekletilir; Close kalanı basar.
type noiseFilter struct {
	out      io.Writer
	drop     []*regexp.Regexp
	stop     *regexp.Regexp
	buf      bytes.Buffer
	stopped  bool
	lastLine string
	blankRun int // arka arkaya kaç boş satır basıldı
}

// newNoiseFilter — toolKey'e uygun filtreyi kurar. raw=true ise çıktı hiç
// dokunulmadan geçer (--raw).
func newNoiseFilter(toolKey string, out io.Writer, raw bool) io.WriteCloser {
	if raw {
		return nopCloser{out}
	}
	rules := append([]*regexp.Regexp{}, commonNoise...)
	rules = append(rules, toolNoise[toolKey]...)
	return &noiseFilter{out: out, drop: rules, stop: stopAfter[toolKey]}
}

func (f *noiseFilter) Write(p []byte) (int, error) {
	n := len(p)
	f.buf.Write(p)
	for {
		line, err := f.buf.ReadString('\n')
		if err != nil {
			// Satır sonu gelmemiş: kalanı tampona geri koy.
			f.buf.Reset()
			f.buf.WriteString(line)
			return n, nil
		}
		if werr := f.emit(strings.TrimRight(line, "\r\n")); werr != nil {
			return n, werr
		}
	}
}

// emit — tek satırı kurallardan geçirip yazar.
func (f *noiseFilter) emit(line string) error {
	if f.stopped {
		return nil
	}
	if f.stop != nil && f.stop.MatchString(strings.TrimSpace(line)) {
		f.stopped = true
		return nil
	}
	trimmed := strings.TrimSpace(line)

	// Boş satır: baştakiler atılır, arka arkaya en fazla İKİ tanesi geçer.
	// Teke indirmek cazip ama kod bozuyor: Python'da üst düzey tanımlar
	// arasında iki boş satır PEP8 gereği, Markdown'da da anlamlı olabiliyor.
	if trimmed == "" {
		if f.lastLine == "" || f.blankRun >= 2 {
			return nil
		}
		f.blankRun++
		_, err := io.WriteString(f.out, "\n")
		return err
	}
	for _, re := range f.drop {
		if re.MatchString(line) {
			return nil
		}
	}
	// BURADA "aynı satır peş peşe geldiyse ikincisini at" kuralı VARDI ve
	// KALDIRILDI: kod çıktısını bozuyordu. Girintili bir blok kapanırken
	// arka arkaya iki "}" satırı gelir; ikincisi atılınca kullanıcıya
	// DERLENMEYEN kod gidiyordu (sahada görüldü: quicksort'ta for döngüsünün
	// kapanışı yutuldu, gofmt "expected '(', found main" verdi).
	//
	// Bu kural aracın kendi cevabını iki kez basmasına karşı eklenmişti; o
	// sorun artık kaynağında çözülü (codex --output-last-message ile sessiz
	// çalışıyor, dedupeTrailingEcho writer prompt'unu temizliyor). Çıktının
	// bütünlüğü, kozmetik tekrarları ayıklamaktan önce gelir.
	f.lastLine = trimmed
	f.blankRun = 0
	_, err := io.WriteString(f.out, line+"\n")
	return err
}

// Close — tamponda kalan yarım satırı basar.
func (f *noiseFilter) Close() error {
	if rest := f.buf.String(); rest != "" && !f.stopped {
		f.buf.Reset()
		return f.emit(strings.TrimRight(rest, "\r\n"))
	}
	return nil
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

// filterText — bir metni ekrandakiyle aynı gürültü filtresinden geçirir.
// Writer'a gönderilecek thinker çıktısı için kullanılır.
func filterText(toolKey, s string) string {
	var buf bytes.Buffer
	nf := newNoiseFilter(toolKey, &buf, false)
	_, _ = io.WriteString(nf, s)
	_ = nf.Close()
	return strings.TrimRight(buf.String(), "\n")
}
