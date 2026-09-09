# JetBrains IDE'lerde cem

**PyCharm, IntelliJ IDEA, GoLand, WebStorm, RubyMine, PhpStorm, Rider, DataGrip, CLion, AppCode, RustRover, AquaIDE** — IntelliJ Platform 2023.3+ (build 233+) üzerine kurulu her şeyde çalışır.

English: [INTELLIJ.md](INTELLIJ.md)

---

## Kurulum

1. `cem` PATH'de olmalı (`curl -fsSL cem.pw/install | sh` veya PowerShell karşılığı — [README](../README.tr.md)).
2. Son sürümün plugin zip'ini indir:
   ```
   https://github.com/muslu/cem/releases/latest/download/cem-intellij-<sürüm>.zip
   ```
3. IDE → **Settings → Plugins → ⚙ → Install Plugin from Disk** → zip'i seç.
4. IDE'yi yeniden başlat.

Artık **Tools → cem** menüsünde 3 aksiyon ve editör sağ-tık menüsünde `cem` alt menüsü var.

## Otomatik güncelleme

Eklenti JetBrains Marketplace'te değil; diskten kurulan eklentiler **hiç
güncellenmez**. Güncellemeleri IDE'nin kendisinin bulması için tek seferlik
şunu ekle:

**Settings → Plugins → ⚙ → Manage Plugin Repositories → `+`**

```
https://github.com/muslu/cem/releases/latest/download/updatePlugins.xml
```

Bundan sonra IDE bu adresi periyodik yoklar; yeni sürüm çıktığında **Plugins →
Updates** altında görünür ve otomatik güncelleme açıksa kendisi kurar. Adres
sabittir, her cem sürümünde içeriği yenilenir.

## Kullanım

| Aksiyon | Kısayol | Etkisi |
|---|---|---|
| `cem: think on selection` | `Ctrl+Alt+I` (`⌥⌘I` macOS) | `cem "<seçim>"` — thinker |
| `cem: write on selection` | `Ctrl+Alt+W` (`⌥⌘W`) | `cem -w "<seçim>"` — writer |
| `cem: pair on selection`  | `Ctrl+Alt+P` (`⌥⌘P`) | `cem -p "<seçim>"` — thinker → writer |

Seçim yoksa **tüm aktif dosya** gönderilir. Çıktı IDE altındaki **cem** tool window'una stream olur.

Working directory = proje kökü, yani `.cem.yaml` (proje config'i) ve `~/.cem/config.yaml`'daki API key'ler beklenen şekilde çalışır.

## Ayarlar

**Settings → Tools → cem**:

| Alan | Varsayılan | Notlar |
|---|---|---|
| `cem binary path` | `cem` | IDE PATH'inde değilse mutlak yol gerekli (macOS/Linux'ta IDE non-shell ortamdan başlatıldıysa olabilir). |

## Kısayolları değiştirme

**Settings → Keymap → Plug-ins → cem** → aksiyona sağ tık → **Add Keyboard Shortcut**. Varsayılanlar JetBrains standart keymap'iyle çakışmıyor (2026.1 itibariyle).

## Sorun giderme

### `0x80004002` "Interface not supported" (Windows)

Plugin IDE'nin desteklediğinden daha yeni bir IntelliJ Platform için derlenmiş. IDE'yi güncelle (Help → Check for Updates) veya yapamıyorsan build numaranı (Help → About → Build #) ile issue aç.

### Tool window boş

1. `cem` gerçekten PATH'te mi? IDE terminalinde `where cem` (Windows) / `which cem` (Unix).
2. `cem` olağandışı yerdeyse (örn. `C:\Users\...\cem\bin\cem.exe`), **Settings → Tools → cem**'de mutlak yolu yaz.
3. **cem** tool window'undaki stderr satırlarına bak.

### Aksiyon greyed out

Editör (kod alanı) focus'ta olmalı. Aksiyonlar `editorTextFocus`'a bağlı.

### `idea.log`

**Help → Show Log in Explorer / Finder** → `dev.cempw.intellij` ara. Stack trace ile GitHub issue aç.

## Kaynaktan derleme

```sh
cd plugin/intellij
./gradlew buildPlugin
```
JDK 21 gerekli. Mimari + yol haritası: [docs/PYCHARM-PLUGIN.md](PYCHARM-PLUGIN.md).
