# CEM — İleri Kullanıcı Kılavuzu

> Hiç cem kullanmadıysan [README.tr.md](README.tr.md) ile başla. Bu doküman her ayara hâkim olmak isteyenler için.

English: [ADVANCED.md](ADVANCED.md)

---

## 30 saniyede mimari

cem bir **LLM client değildir**; bir süreç orkestratörüdür. Her "thinker" / "writer" çağrısı `exec.Command(<tool>, --flags..., <prompt>)`. cem karar verir:

- Hangi binary çağrılacak (PATH + tool-spesifik fallback listesi)
- Hangi model flag'i eklenecek (her tool'un `ModelFlag`'i)
- API key hangi env değişkenine inject edilecek (her tool'un `Provider` + `APIKeyEnv`'i)
- Rate-limit sinyali görünce sonraki key'le retry yapılıp yapılmayacağı

`exec.Command` + flag sıralamasını anlıyorsan cem'i anlıyorsun.

---

## Tool registry

```go
type ToolMeta struct {
    Name             string   // görünen ad
    Binary           string   // toolKey'den farklıysa PATH adı (cursor → cursor-agent)
    Description      string
    Deprecated       string   // cemi listesinde uyarı satırı basar

    InstallCmd       []string // exec.Command formu, örn. ["npm", "install", "-g", "@openai/codex"]
    InstallShellUnix string   // sh -c için shell komutu (Linux/macOS)
    InstallShellWin  string   // powershell için shell komutu (Windows)

    VersionFlag      string   // örn. "--version"
    RunFlags         []string // her çağrıda eklenir, örn. ["-p"] veya ["exec", "--skip-git-repo-check"]
    PromptAsArg      bool     // true → input son pozisyonel arg; false → stdin'den pipe
    ModelFlag        string   // örn. "--model" — CLI model seçimini desteklemiyorsa boş
    ModelBeforeRun   bool     // true → --model X RunFlags'ten ÖNCE konur (-p'nin arg yutmasını engellemek için)
    Models           []string // wizard model picker önerileri

    Provider         string   // "anthropic", "openai" — boş ise API key rotasyonu kapalı
    APIKeyEnv        string   // "ANTHROPIC_API_KEY", "OPENAI_API_KEY"

    AuthCmd          []string // `cem auth <tool>` için subcommand (örn. ["login"])
}
```

Mevcut 4-tool kadrosu:

| Anahtar | Binary | Kurulum kaynağı | Model flag | API key rotasyonu |
|---|---|---|---|---|
| `claude` | `claude` | [claude.ai/install.sh](https://code.claude.com/docs/en/quickstart) | `--model opus\|sonnet\|haiku` | anthropic (ANTHROPIC_API_KEY) |
| `agy` | `agy` | [antigravity.google/cli/install.sh](https://antigravity.google/docs/cli-getting-started) | — (CLI'da yok) | — (OAuth only) |
| `gpt` | `codex` | `npm i -g @openai/codex` | `--model gpt-5.5\|...` (`exec` sonrası) | openai (OPENAI_API_KEY) |
| `cursor` | `cursor-agent` | [cursor.com/install](https://cursor.com/cli) | `--model claude-4.6\|gpt-5.2\|...` | — (OAuth only) |

Notlar:
- **Antigravity'de `--model` flag'i yok** — `agy --help` çıktısı: `-p`, `-c`, `--sandbox`, `--print-timeout` var ama model seçici yok. cem'in `ModelFlag`'i bilinçli olarak boş.
- **Cursor binary'si `cursor-agent`** (legacy symlink); installer ayrıca `agent.*` alias'ları da bırakıyor.
- **Codex binary'si `codex`** (npm paketi `@openai/codex` ama PATH adı `codex`).

---

## Çözümleme sırası (cem bir tool çağırırken)

```
1. cfg.Tools[toolKey].Command  (install sonrası kaydedilen mutlak yol) — dosya hâlâ varsa kullanılır
2. exec.LookPath(meta.Binary veya toolKey)
3. fallbackInstallPath(toolKey) — bilinen kurulum konumları:
     agy    → %LOCALAPPDATA%\agy\bin\agy.exe         ~/.local/bin/agy
     claude → %LOCALAPPDATA%\Claude\claude.exe       ~/.claude/local/claude  ~/.local/bin/claude
     cursor → %LOCALAPPDATA%\cursor-agent\{cursor-agent,agent}.{cmd,ps1,exe}
              %APPDATA%\npm\cursor-agent.{cmd,ps1}   ~/.local/bin/cursor-agent
4. Pes et → "✗ <tool> bulunamadı"
```

Native installer binary'yi beklenmedik bir yere koyuyorsa `executor.go`'daki `fallbackInstallPath`'a aday ekle. PR açabilirsin.

---

## Komut argüman sıralaması

cem args'ı şöyle kurar:

```
ModelBeforeRun = true   →  [--model X, RunFlags..., promptArg?]
ModelBeforeRun = false  →  [RunFlags..., --model X, promptArg?]
```

Neden önemli: `agy -p` ve `cursor-agent -p` *bir sonraki* arg'ı prompt olarak yer. `--model X` araya girerse tool "--model"u prompt sanır. Run flag'i argüman yutuyorsa `ModelBeforeRun = true` yap.

Codex `exec` subcommand kullandığı için `--model` ondan SONRA gelmeli: `codex exec --model gpt-5.5 --skip-git-repo-check "prompt"`. Bu yüzden `ModelBeforeRun = false` (default) + `RunFlags: ["exec", "--skip-git-repo-check"]`.

---

## Pair modu skip kuralları

`cem -p "..."` thinker → writer akışı kurar. cem boşa giden writer çağrısını atlar:

| Durum | Davranış |
|---|---|
| `thinker == writer` (ikisi de `claude` vs.) | Writer atlanır — aynı model duplikasyon olur |
| Soru kod istemiyor **ve** thinker çıktısında ` ``` ` yok | Writer atlanır — yazılacak şey yok |
| Aksi halde | Writer thinker analizini bağlam alarak çalışır |

Kod-niyet algılaması: `yaz|kod|script|fonksiyon|class|method|implement|kodla|oluştur|üret|döndür|export|function|code|write|build|generate|refactor|debug|fix` (case-insensitive, kelime sınırı).

---

## HTTP model sunucuları (ollama · LM Studio · unsloth)

Kayıttaki üç araç subprocess değil. Karar `ToolMeta.HTTPAPI` ile veriliyor:

| Değer | İstek |
|---|---|
| `"openai"` | `POST {base}/v1/chat/completions` — LM Studio, unsloth/vLLM |
| `"ollama"` | `POST {base}/api/chat` |
| `""` | HTTP aracı değil; cem bir binary çalıştırır |

`isHTTPTool(key)` binary varsayan her yolu kapatıyor:
`installableToolKeys()` bunları `cemi` / `cemir` / auto-update dışında
tutuyor, effort ve hızlı mod atlanıyor, `cem doctor` PATH yerine
erişilebilirliği yokluyor.

Adres çözümü: proje `.cem.yaml` → global `endpoints.<araç>` →
`ToolMeta.DefaultBaseURL`. Şema yoksa `http://` ekleniyor — LAN'daki sunucu
nadiren TLS konuşuyor — ve adres `/v1` ile bitiyorsa yol iki kez eklenmiyor.
Anahtar tanımlıysa `Authorization: Bearer` olarak gidiyor; ollama ve LM Studio
yerelde anahtar istemediği için boşken başlık hiç gönderilmiyor.

Cevap akıtılıyor: OpenAI'nin SSE'si (`data: {...}`, `data: [DONE]`) ve
ollama'nın satır-başına-JSON'u satır tabanlı olduğu için tek tarayıcı yetiyor.
Çözülemeyen satır hata değil, atlanıyor — sunucular arada ısıtma ve istatistik
nesnesi yolluyor. Boş cevap ise hata: modeli yüklenmemiş sunucu içeriksiz akış
döndürüyor ve buna "başarılı" demek kullanıcıyı hem çıktısız hem gerekçesiz
bırakırdı.

Zaman aşımları ayrı: istek için 30 dakika (yerel 70B model uzun cevapta
dakikalar harcayabiliyor), bağlanmak için 5 saniye — kapalı port yarım saat
bekletmesin.

Model adı asla tahmin edilmiyor. `endpointModel` hata dönüp
`cem endpoint <araç> --model`'i gösteriyor: yerel sunucuda yanlış ad ya 404
veriyor ya da sessizce başka bir modeli çalıştırıyor.

---

## Terminalsiz kurulum

`loadAndCheckSetup` yapılandırılmadan çalışmayı reddediyor — yarım
yapılandırma, kullanıcının seçmediği araca istek göndermek demek. TTY varsa
sihirbaz teklif ediliyor ve sonrasında kayıt yeniden doğrulanıyor (Ctrl+C ile
kesilen sihirbaz aynı yarım durumu bırakıyordu). TTY yoksa `cem setup` deyip
çıkıyor, çünkü sorular cevaplayacak kimse olmadan akıp geçerdi.

Geriye GUI'ler ve betikler kalıyor; bayraklar bunun için:

```sh
cem setup --thinker gpt --writer claude
cem setup --thinker ollama --endpoint-thinker 1.2.3.4:11434 \
          --model-thinker qwen3-coder --writer claude
cem status --json
```

`ApplySetup` kaydetmeden doğruluyor: bilinmeyen araç (yakın ad önerisiyle),
model adı olmayan HTTP aracı, aracın desteklemediği effort. Verilmeyen değer
korunuyor; yani `cem setup --lang en` rolleri silmiyor.

`cem status --json` config'i ve araç başına kurulu mu, yolu, modeli, effort'u,
endpoint'i ve bilinen model/effort listelerini basıyor. stdout'a başka hiçbir
şey çıkamaz: `machineReadableOutput()` cobra ayrıştırmadan önce `os.Args`'ta
`--json` arıyor ve güncelleme bildirimi ile auto-update satırını atlıyor —
ikisi de JSON'un başına eklenip çağıranın ayrıştırmasını kırıyordu.

---

## API key rotasyonu

Depolama (`~/.cem/config.yaml`):

```yaml
api_keys:
  anthropic:
    - value: sk-ant-...
      label: personal
    - value: sk-ant-...
      label: company-backup
  openai:
    - value: sk-proj-...
```

Runtime: her çağrıda cem `<provider>.APIKeyEnv = <key>` set eder ve tool'u exec eder. Tool exit-non-zero **ve** stderr şu pattern'lerden birine uyuyorsa:

```
rate.?limit | quota | 429 | too many requests | usage limit | overloaded
```

cem sıradaki key ile retry yapar (sırayı korur, çağrı başına tüm key'ler denenir). Stderr auth-failure pattern'ine uyuyorsa:

```
401 | unauthorized | missing bearer | invalid api key | not.?logged.?in | please run /login | please log in | authentication failed
```

cem rotasyon yapmaz (farklı key bozuk auth'u düzeltmez); `cem keys list/remove/add` ya da tool'un interaktif login'ine yönlendirir.

---

## Interaktif oturum açma ve OAuth kodu

Prompt'u argümanla alan araçlarda (agy, claude, cursor) stdin serbest ve
şimdiye kadar boş bırakılıyordu — yani `/dev/null`. Böyle bir araç OAuth kodu
istediğinde ("Or, paste the authorization code here and press Enter")
okuyacak bir şey bulamıyordu: 60 saniye bekleyip düşüyor, kullanıcının
yapıştırdığı kod ise kabuğa gidiyordu. Windows'ta PowerShell `4/0ATs…`'yi
komut sanıp üç denemede de *"You must provide a value expression following the
'/' operator"* verdi.

`attachStdin`, cem'in kendi stdin'i TTY ise terminali araca devrediyor. Pipe
ise devretmiyor — o veri `ReadStdin` ile okundu, araç EOF beklerken
kilitlenirdi — quiet modda da devretmiyor, çünkü ham akış gizli ve kullanıcı
körlemesine yazardı. İstem akışta görüldüğünde tek satır ipucu basılıyor:
kod kabuğa değil buraya yazılacak.

### PSReadline yapıştırma yardımcısı

PowerShell PSReadline uzun yapıştırmaları sessizce kırpıyor (Google OAuth kodları sıklıkla bozuluyor). Çözüm:

```powershell
cem auth agy --code 4/0AY29...uzun-google-oauth-kodu
```

`--code` kodu sistem panosuna yazar (Windows'ta `Set-Clipboard`, macOS'ta `pbcopy`, Linux'ta `wl-copy`/`xclip`/`xsel`). Sonra cem `<bin> login` çalıştırır. CLI prompt'unda **sağ-tık paste** — PSReadline'ı tamamen bypass eder.

`--code` olmadan komut sadece `<bin> login`'i TTY passthrough ile çalıştırır (pano dokunulmaz).

---

## Proje-bazlı config

`.cem.yaml` proje kökünde global ayarları o ağaç için override eder:

```yaml
roles:
  thinker: claude
  writer:  agy
models:
  claude: opus
  agy: gemini-3-pro    # ignore edilir (agy'nin --model'i yok) ama dokümantasyon için tutulabilir
```

Model çözümleme sırası: proje `models.<key>` → global `tools.<key>.model` → boş (CLI default).

`cem init` bu dosyayı interaktif oluşturur. Her install sonrası cem mevcut git repo'da `.gitignore`'a `.cem.yaml` ekler (yoksa uyarı) — proje-bazlı config repo'ya sızmasın.

`endpoints` de aynı şekilde çalışıyor: aynı projeyi iki makinede açan biri
farklı model sunucularına bakabilir.

---

## Linux Node.js — neden nvm?

NodeSource (resmi Node apt/dnf kaynağı) 2024'te "nodistro" geçişi yaptı: tüm build'ler artık `libc6 >= 2.28` istiyor. Ubuntu 18.04 (bionic) `libc6 2.27`. Sonuç: bionic gibi eski distro'larda `apt install nodejs` NodeSource'tan "unmet dependencies" hatası verir.

cem'in `ensureDep("npm")` fonksiyonu Linux'ta **nvm** kullanır:

```sh
curl -fsSL https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
. ~/.nvm/nvm.sh
nvm install --lts
```

Install sonrası cem `~/.nvm/versions/node/v*/bin`'i kendi PATH'inin başına ekler ki yeni npm aynı oturumda kullanılabilsin.

**Önemli not:** nvm'nin nodejs.org/dist'ten indirdiği prebuilt Node'lar da modern glibc'ye link'li. Ubuntu 18.04'te nvm Node 24 için bile `GLIBC_2.28 not found` hatası gördük. Bionic'te npm-tabanlı tool'ları (Codex) zorunlu kullanmak istiyorsan Node'u kaynaktan derle ya da glibc 2.27'yi destekleyen eski Node sürümüne pin'le. Daha kolay: OS upgrade.

---

## `cem doctor`

Sistem + roller + araçlar + PATH tanı raporu. Bir şey ters görünüyorsa:

```
✓ linux/amd64 · Go runtime: go1.25.10
✓ ~/.cem dizini → /home/muslu/.cem
✓ global config → /home/muslu/.cem/config.yaml
✓ proje config yok (global geçerli)

✓ thinker → claude
✓ writer → agy

✓ Claude   /home/muslu/.local/bin/claude
⚠ Antigravity PATH'da var ama config'e kayıtlı değil → cemi agy
✗ Cursor   config'de kayıtlı ama PATH'da yok
```

---

## Komut geçmişi

`~/.cem/history.log` her cem invokasyonunu TSV olarak kaydeder:

```
2026-05-25T17:30:14Z    think   claude    "bu projeyi anlat"
```

`cem history -n 50` son 50'yi gösterir; `cem history --clear` log'u temizler.

---

## Update kontrolü

cem `api.github.com/repos/muslu/cem/releases/latest`'i saatte en fazla bir kez sorgular ve `~/.cem/update-check.json`'a cache'ler:

```json
{"last_check":"2026-05-25T19:00:00Z","latest_version":"v0.1.33"}
```

`latest_version != version` ise bir sonraki cem/cemi/cemir çağrısı renkli uyarı basar:

```
🔔 yeni sürüm: v0.1.32 → v0.1.33 · run cem update
```

`cem update`'in kendisi API'yi indirmeden ÖNCE sorgular, son tag her zaman önce gösterilir ("ⓘ son sürüm: v0.1.33 (mevcut: v0.1.32)").

---

## Kaynaktan derleme

```sh
git clone https://github.com/muslu/cem.git
cd cem
make build               # ./build/{cem,cemi,cemir} üretir
make install             # sudo cp /usr/local/bin'e
go test ./...
```

Build flag'leri `git describe --tags --always --dirty` çıktısını `main.version`'a `LDFLAGS -X main.version=...` ile inject eder. `dev` build'leri (git tag bağlamı yok) update-check uyarısını bastırır — yerel iterasyonda işine yarar.

---

## Self-update iç işleyişi

`cem update` saf Go (`cmd_update.go`). Adımlar:

1. GitHub API'sinden son tag (sync, ~200ms).
2. `main.version` ile karşılaştır; eşitse "zaten güncel" diyip çık.
3. Install dizinine `canWriteDir` ön-kontrolü. Unix'te yazılamıyorsa → kendini `sudo` ile yeniden başlat. Windows'ta → admin PowerShell hatırlatmaası.
4. `cem.pw/r/<binary>`'yi `os.TempDir()`'a indir.
5. Çalışan binary'yi değiştir:
   - **Unix**: `os.Rename(tmp, dst)` — atomik, kernel çalışan exe'yi değiştirmeye izin verir.
   - **Windows**: önce `dst → dst.old` rename'i (çalışan exe için bile çalışır), sonra `os.Rename(tmp, dst)`. `.old` dosyası kalır — `cem uninstall` temizler.
6. `cemi`, `cemir` için tekrarla.

---

## Self-uninstall iç işleyişi

`cem uninstall`:

- `[cem, cemi, cemir]` üzerinde döner, her birini PATH'tan siler.
- Windows'ta çalışan `cem.exe` kendini silemez; cem detached `cmd /c "ping 127.0.0.1 -n 2 >nul & del /f /q <path>"` planlar — parent çıkışından sonra çalışır.
- İsteğe bağlı: `~/.cem/` (config, history log) siler.
- İsteğe bağlı: projedeki `.cem.yaml`'i siler.

`cemir <tool>` AI tool'un binary'sini siler. Shell-installed (claude, agy, cursor) için parent dizini kontrol edilir: `dirname` toolKey veya binary adıyla eşleşirse tüm ağaç silinir (`%LOCALAPPDATA%\cursor-agent\` dahil `versions/`). Aksi halde sadece tek dosya + boş parent dir cleanup.

`cemir all` ayrıca `cfg.Tools`'taki artık `KnownTools`'da olmayan orphan girdileri siler (v0.1.13 kadro daraltması bazı kullanıcıların config'inde `gemini` bıraktı).

Bayraklar: `--yes` (soru sormaz), `--config` (`~/.cem` ve yereldeki
`.cem.yaml`), `--plugin` (IDE eklentileri), `--all` (hepsi, soru sormadan).
Terminal yoksa onaysız silmek yerine reddediyor.

`ideEklentiDizinleri()` JetBrains config kökünü tarayıp her ürün dizininde
(`GoLand2026.2`, `PyCharm2026.1` …) `cem-intellij` arıyor, ayrıca
`~/.vscode/extensions/*cem-*`. Bunları binary'yle birlikte silmek önemli:
geride kalan eklenti her IDE açılışında yüklenip cem'in bulunmadığını
bildiriyor ve ürün başına ayrı yolları elle bulmak zahmetli. AI CLI'larına
dokunulmuyor — onlar `cemir`'in işi.

---

## Fuzzy komut eşleştirme

```
PS> cemi cluade
  'cluade' bilinmiyor — 'claude' demek istedin mi? (y/N): y
  ⏳ Claude kuruluyor...
```

`suggestTool` `KnownTools` anahtarlarına Levenshtein mesafesi ≤ 2 uygular. Tam eşleşme öncelikli; aksi halde en yakın komşu önerilir. `fuzzy.go`'da implement edildi.

---

## Sunucu tarafı (cem.pw)

Install/update URL'leri cem.pw host'undaki Apache vhost tarafından servisleniyor. Routing:

- `/install` ve `/uninstall` User-Agent algılayıcı: PowerShell veya WindowsPowerShell → `.ps1`; gerisi → `.sh`
- İki script de `Content-Type: text/plain; charset=utf-8` (Türkçe karakter için) + `Cache-Control: no-cache, max-age=300`
- `/r/<asset>` 302 redirect → `github.com/muslu/cem/releases/latest/download/<asset>` — install script'leri GitHub'ı doğrudan bilmek zorunda değil

vhost config + UA kuralları + Apache rewrite kuralları production host'ta (bu repo'da değil); ilgili snippet'lar `OPERATIONS.md`'de.

---

## Bilinen sınırlamalar

- **Ubuntu 18.04 + Codex**: glibc 2.27 < 2.28 → modern Node çalışmaz → npm-installed Codex CLI çuvallar. OS upgrade veya Node'u kaynaktan derlemeden çözüm yok.
- **Antigravity Windows + PSReadline**: uzun OAuth kodları bracketed paste tarafından bozuluyor. `cem auth agy --code <kod>` clipboard helper'ı kullan.
- **Antigravity `--model`**: CLI desteklemiyor. Model seçimi cem dışında.
- **Cursor CLI login**: OAuth el sıkışması için Cursor masaüstü uygulamasının kurulu olması gerekiyor.

---

## Dosya yapısı

```
cem/
├── main.go                — binary-adı dispatch (cem/cemi/cemir tek binary'den)
├── banner.go              — ASCII banner + OpenSourceNotice
├── config.go              — GlobalConfig + ProjectConfig + ToolMeta + KnownTools
├── config_test.go         — KnownTools sanity + Roles çözümleme
├── executor.go            — runTool, captureTool, withKeyRotation, resolveModel, fallbackInstallPath
├── wizard.go              — RunSetupWizard, InstallTool, RemoveTool, ensureDep, askYN, askModel
├── spinner.go             — TTY-aware spinner
├── history.go             — AppendHistory → ~/.cem/history.log
├── fuzzy.go / fuzzy_test  — Levenshtein-tabanlı typo önerici
├── update_notice.go       — Saatlik GitHub Releases poll + cache'li renkli banner
├── cmd_cem.go             — kök cobra: cem [input] + roles + setup + init + status
├── cmd_cemi.go            — cemi: install (-y desteği)
├── cmd_cemir.go           — cemir: uninstall (-y, orphan cleanup)
├── cmd_update.go          — cem update (Unix'te sudo escalation)
├── cmd_uninstall.go       — cem uninstall (Windows self-delete via detached cmd)
├── cmd_doctor.go          — cem doctor
├── cmd_history.go         — cem history
├── cmd_keys.go            — cem keys add/list/remove
├── cmd_auth.go            — cem auth <tool> [--code]
├── cmd_endpoint.go        — cem endpoint (HTTP sunucuların adresi / anahtarı / modeli)
├── http_tool.go           — HTTP model sunucuları: akışlı sohbet, yoklama, model listesi
├── setup_apply.go         — bayraklı kurulum + cem status --json
├── surum-yayinla.sh       — tek komut: testler → tag → push → Marketplace
├── plugin/intellij/       — JetBrains eklentisi (docs/PYCHARM-PLUGIN.md)
└── .github/workflows/     — release.yml (7 platform × 3 binary + SHA256SUMS)
```
