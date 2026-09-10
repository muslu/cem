# JetBrains IDE'lerde cem

**PyCharm, IntelliJ IDEA, GoLand, WebStorm, RubyMine, PhpStorm, Rider, DataGrip, CLion, AppCode, RustRover, AquaIDE** — IntelliJ Platform 2023.3+ (build 233+) üzerine kurulu her şeyde çalışır.

English: [INTELLIJ.md](INTELLIJ.md)

---

## Kurulum

1. `cem` PATH'de olmalı (`curl -fsSL cem.pw/install | sh` veya PowerShell karşılığı — [README](../README.tr.md)).
2. IDE → **Settings → Plugins → Marketplace** → **cem** ara → *Install*
   ([sayfa](https://plugins.jetbrains.com/plugin/34196)).
3. IDE'yi yeniden başlat.

**Diskten kurmak istersen** (Marketplace'e henüz çıkmamış bir derleme veya
internetsiz makine):

```
https://github.com/muslu/cem/releases/latest/download/cem-intellij-<sürüm>.zip
```

IDE → **Settings → Plugins → ⚙ → Install Plugin from Disk** → zip'i seç.
Diskten kurulan eklenti kendiliğinden güncellenmez.

Artık **Tools → cem** menüsünde 3 aksiyon ve editör sağ-tık menüsünde `cem` alt menüsü var.

## IDE içinden kurulum

**Settings → Tools → cem** kurulumun tamamı: düşünen ve yazan rolü, modelleri,
düşünme seviyesi ve model sunucusu (ollama / LM Studio / unsloth) kullanıyorsan
adresi — `192.168.1.10:11434`. Apply kaydediyor.

Panel `cem status --json` okuyor, Apply `cem setup` çalıştırıyor; yani seçimleri
cem doğruluyor ve `~/.cem/config.yaml`'i yazan tek yer cem kalıyor. En üstteki
satır kurulumun tamam olup olmadığını söylüyor; tamamlanmadan cem çalışmayı
reddediyor ve bir bildirim bu sayfayı açmayı öneriyor.

## Otomatik güncelleme

Marketplace'ten kurulduysa IDE eklentiyi kendisi güncelliyor — ayar gerekmez.

**Diskten** kurulduysa güncelleme hiç gelmez. Ya Marketplace'ten yeniden kur,
ya da IDE'yi tek seferlik sürüm akışına yönlendir:

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

### Çalıştırma sekmesinde sohbete devam

Her çalıştırma kendi sekmesini açıyor ve her sekmenin altında bir giriş kutusu
var. Oraya yazınca aynı sekmede yeni bir tur başlıyor ve önceki turlar bağlam
olarak taşınıyor — aracın sorduğu soruya ("tümünü geri almak mı istiyorsunuz?")
böyle cevap veriyorsun, ya da yaptığı işe ekleme yapıyorsun.

Bağlam son birkaç bin karaktere kırpılıyor. cem'in oturumu yok: her çağrı yeni
bir süreç, yani devam etmek önceki turları yeniden göndermek demek ve her tur
yeniden faturalanıyor. Kırpma bunun sessizce büyümesini engelliyor.

### Terminal sekmesi

`Terminal` sekmesi komutları proje kökünde çalıştırıyor — `go test ./...`,
`git diff`, `npm run build` — böylece cevap ve komut çıktısı aynı pencerede
kalıyor. Kabuk üzerinden geçtiği için pipe, yönlendirme ve `&&` çalışıyor. Tam
bir pty değil: `vim`, `top` gibi gerçek terminal bekleyen programlar için
IDE'nin kendi Terminal'ini kullan. `⏹` çalışan komutu durduruyor, `＋` yeni bir
terminal sekmesi açıyor — :8080'de servis eden bir `go run` aksi halde her şeyi
kilitliyordu. İlk sekme kapanmıyor; sonrakiler kapanabiliyor ve kapatılan sekme
içindeki komutu da öldürüyor.

### Seçim ve kısayollar

Seçim yoksa imleç, **cem** tool window'unun altındaki giriş kutusuna gelir ve
mod önceden seçilidir — modal pencere yok, açık dosya yanlışlıkla gönderilmez.
Çıktı aynı tool window'a akar, her çalıştırma kendi tab'ında.

Working directory = proje kökü, yani `.cem.yaml` (proje config'i) ve `~/.cem/config.yaml`'daki API key'ler beklenen şekilde çalışır.

## Ayarlar

**Settings → Tools → cem** artık şunları da yönetiyor (hepsi
`~/.cem/config.yaml`'a yazılır, terminaldeki `cem` ile aynı dosya):

| Alan | Ne işe yarar |
|---|---|
| Thinker / Writer | hangi AI düşünür, hangisi yazar |
| Model | rol başına model (boş = CLI'ın kendi varsayılanı) |
| **Düşünme seviyesi** | öneri: thinker `high`, writer `low` |
| **Hızlı mod** | aracın hook/izin kurallarını yüklemeden çalıştırır — ölçüldü: 124s yerine 8s. Kapatırsan kendi hook'ların çalışır. |


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

**Help → Show Log in Explorer / Finder** → `dev.cempw.cem` ara. Stack trace ile GitHub issue aç.

## Kaynaktan derleme

```sh
cd plugin/intellij
./gradlew buildPlugin
```
JDK 21 gerekli. Mimari + yol haritası: [docs/PYCHARM-PLUGIN.md](PYCHARM-PLUGIN.md).
