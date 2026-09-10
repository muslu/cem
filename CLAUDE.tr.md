# CLAUDE.md — cem (Türkçe)

> Bu dosya bu projeye özgüdür ve **global `~/.claude/CLAUDE.md`'yi ezer**.
> Çelişen her madde için bu dosya geçerlidir.
> İngilizce sürüm: [CLAUDE.md](CLAUDE.md).

---

## Proje Özeti

**CEM — Compose · Execute · Multiplex.** Birden fazla AI CLI'yi (Claude,
Antigravity, Aider, Gemini, Codex, Goose, Cody, Continue, OpenHands,
Cursor) tek komutla yöneten Go orchestrator'ı. "Thinker" düşünür, "writer"
yazar; pair modunda thinker çıktısı writer'a beslenir.

- **Domain:** `cem.pw`
- **Git:** `https://github.com/muslu/cem.git` (canonical)
- **Dil:** Go 1.25 (cobra + lipgloss + yaml.v3)
- **Binary:** 3 ad — `cem`, `cemi` (installer), `cemir` (remover).
  Hepsi aynı kaynaktan derlenir, `main.go` binary adına göre dispatch eder.

---

Oturum hafızası [`memory/`](memory/MEMORY.md) dizininde — depoyla birlikte
taşınsın diye version-controlled. Kodda görünmeyeni tutar: kullanıcının hedefi,
verdiği geri bildirimler ve ortamına dair ölçülmüş gerçekler. Buraya bilinçli
olarak @import EDİLMEZ; gerektiğinde oku, içeriğini burada tekrarlama.


## Global Kurallarla Farklılaşan Noktalar

Global `~/.claude/CLAUDE.md` Python + PostgreSQL + FastAPI + nginx
merkezli. Bu proje **Go CLI**'dır → aşağıdaki global maddeler
**uygulanmaz**:

| Global madde | Sebep |
|---|---|
| Pydantic / `.env` / Fernet | CLI; secret yok, config = YAML |
| FastAPI middleware / Server-Timing | API değil |
| PostgreSQL / MongoDB / Valkey | Veri katmanı yok |
| Cache dekoratörü / rate limit | İlgisiz |
| Swagger UI / `src/swagger.py` | API değil |

**Uygulanır:** Docker yasağı (bare-metal binary), `nala` zorunluluğu,
nginx production standartları (HTTP/3, Server header, fail2ban) — `cem.pw`
sunucusu için.

---

## Dosya Yapısı

İngilizce sürümdeki "File Layout" bölümüyle aynıdır; bkz.
[CLAUDE.md](CLAUDE.md#file-layout).

---

## Kod Kuralları

1. **`package main`** — tek paket. Alt paket açma (gerekmedikçe).
2. **Logger yok** — kullanıcıya `fmt.Println` + lipgloss style
   (`styleSuccess`, `styleError`, `styleDim`, `styleBold` — `wizard.go`'da
   tanımlı).
3. **Hata mesajları Türkçe** — `styleError.Render("✗ ...")`. Stack trace
   gösterme. `fmt.Errorf` mesajları İngilizce olabilir (debug için).
4. **`exec.Command` ile araç çalıştır** — stdin: input,
   stdout/stderr: passthrough. Pair modunda `cmd.Output()` ile çıktıyı
   yakala.
5. **Config IO:** `~/.cem/config.yaml` permission `0600`, dir `0755`.
6. **YAML marshal:** `gopkg.in/yaml.v3`. Tag'ler küçük harf snake_case.
7. **Yeni AI aracı eklemek:** `KnownTools` map'ine `ToolMeta` ekle ve
   `orderedToolKeys`'e key ekle. Wizard/installer/remover map'i otomatik
   okur.
8. **Cobra subcommand eklemek:** yeni dosya `cmd_<name>.go`, kendi
   `init()` bloğunda `rootCmd.AddCommand(...)`.
9. **Banner sadece help / setup / status / cemi-no-args / cemir-no-args'da
   görünür** — normal komut çıktısında değil.
10. **Build:** her zaman 3 binary üret. Geliştirmede `make build` yeter;
    sistem-wide kurulum için `make install` (sudo).

---

## Yasaklar (bu proje)

- **Docker / docker-compose ekleme** — bare-metal binary.
- **Python köprüsü yazma** — saf Go. AI CLI'ları subprocess olarak
  çağrılır.
- **Versiyon dosyası ayrı tutma** — sürüm
  `LDFLAGS -X main.version=$(git describe)` ile inject edilir.
- **`go run main.go`** — main.go diğer dosyalara bağımlı. `go run .`
  kullan.
- **Binary adı değiştirme** — `cem` / `cemi` / `cemir` sabit. `main.go`
  dispatch ve install scripts buna bağlı.
- **Kullanıcıya gösterilen metinler Türkçe; kod içi mesajlar İngilizce
  olabilir.** Etrafındaki bağlamla uyumlu olsun.

---

## Git & Release

**Kullanıcı "yayınla" dediğinde `./surum-yayinla.sh` çalıştır** — yayının
tamamı bu. Kirli çalışma dizinini reddeder, `go test -race ./...` ve eklenti
derlemesini koşar, sıradaki `YYYYMMDD.NN` tag'ini seçer, dalı ve tag'i push
eder (CI 7 platform × 3 binary, eklenti zip'i, `updatePlugins.xml` ve GitHub
Release'i kendisi yayınlar), sonra eklentiyi imzalayıp JetBrains Marketplace'e
gönderir. `--kuru` ne yapacağını gösterir, `--sadece-github` Marketplace'i
atlar, `--surum X` sürümü elle verir. Tag/push adımlarını elle tekrarlama —
gereken düzeltmeyi script'e yap.

Marketplace adımı iki dosya istiyor, ikisi de `chmod 600` ve repo dışında:
`~/.cem-signing/parola` (imza anahtarı parolası) ve `~/.cem-signing/token`
(plugins.jetbrains.com token'ı). Biri yoksa script GitHub'da kalır ve hangisini
istediğini söyler — etrafından dolaşılacak bir hata değil.

- **JetBrains Marketplace:** plugin id `dev.cempw.cem`, sayfa 34196
  (`https://plugins.jetbrains.com/plugin/34196`). ID `intellij` kelimesini
  içeremiyor — ilk yükleme bu yüzden reddedildi — ve eklenti yayınlandığı için
  artık değiştirilemez.

## Doğrulama Akışı

```sh
make clean && make build
./build/cem --help      # banner + komut listesi
./build/cem doctor      # tanı raporu
./build/cemi            # araç listesi (banner ile)
./build/cemir           # kurulu araçlar (banner ile)
./build/cem roles       # aktif roller + config kaynak
go test -race ./...     # 28 test, hepsi geçmeli
```

İlk çalıştırmada wizard açılır; `~/.cem/config.yaml` oluşur. Test
dizininde `.cem.yaml` ile proje override edilir.

---

## Bilinen Eksikler

`todo.md` güncel listeyi tutar. Açık başlıklar:
- `.claude/agents/` ve `.claude/skills/` `autoinstalltrixie` projesinin
  kalıntısı — silme/değiştirme kararı bekliyor. `.claude/` gitignore'lı.
- Daha kapsamlı testler: `history_test.go` yok (`executor_test.go` ve
  `spinner_test.go` eklendi).
- macOS/Linux/Windows entegrasyon testleri yok.

---

*Bu CLAUDE.md değişirse `todo.md`'ye "doc:CLAUDE update" satırı eklenir.*
