# CEM — Yapılacaklar

> English version: [todo.md](todo.md). Bu dosya geçmişin Türkçe arşividir;
> canonical (güncel) liste İngilizce `todo.md` dosyasıdır.

## 1. Klasör & Dosya Düzeni
- [x] `uninstall/cmd_uninstall.go` → root'a taşı (`cmd_uninstall.go`)
- [x] `uninstall/uninstall.sh` → root'a taşı
- [x] `uninstall/uninstall.ps1` → root'a taşı
- [x] `uninstall/` klasörünü sil
- [x] `nginx/` altını düzenle (snippets/, sites-available/, fail2ban/ alt klasörleri)

## 2. Eksik Çekirdek Go Dosyaları
- [x] `main.go` → binary adına göre dispatch (cem / cemi / cemir)
- [x] `config.go` → GlobalConfig + ProjectConfig + ResolvedConfig + Roles + InstalledTool + KnownTools + LoadConfig / saveGlobalConfig / loadGlobalConfig / SaveProjectConfig
- [x] `executor.go` → ModeThink/ModeWrite/ModePair + Run + ReadStdin

## 3. Build & Bağımlılıklar
- [x] `go.mod` doldur (cobra, lipgloss, yaml.v3)
- [x] `go mod tidy`
- [x] `Makefile` (build / dev / install / clean)
- [x] `go build` testi — 3 binary üretildi (5.6 MB → 3.9 MB ldflags ile)

## 4. CI/CD
- [x] `.github/workflows/release.yml` (7 platform binary + SHA256SUMS)

## 5. Doğrulama
- [x] `./build/cem --help` çalışıyor
- [x] `./build/cemi --version` → cemi version 1.0.0
- [x] `./build/cemir --version` → cemir version 1.0.0

## 6. Ek Görevler
- [x] `.claude/` agents / skill / hooks kontrolü
- [x] `CLAUDE.md` güncelle (proje-spesifik kılavuz)
- [x] `install.sh` + `install.ps1` → git URL `gitlab.makdos.biz/makdos/cem`,
      binary download `cem.pw/r`

## 7. Yeni Tamamlananlar (2026-05-25 oturumu)
- [x] `cem doctor` komutu (sistem + roller + araçlar PATH + binary'ler)
- [x] `cemir all` toplu kaldırma (onaylı + hatalı özet)
- [x] LDFLAGS versiyon enjeksiyonu (`-X main.version=$(git describe)`)
- [x] `config_test.go` — 5 test (ActiveRoles override + KnownTools sanity)
- [x] `.gitlab-ci.yml` — 7 platform binary + Release tag
- [x] `.gitignore` CEM'e özel yeniden yazıldı
- [x] Git init + gitlab.makdos.biz/makdos/cem origin
- [x] doc:CLAUDE update — proje-spesifik kılavuz yazıldı (2026-05-25)

## 8. Tamamlanan (devam oturumu)
- [x] `~/.cem/history.log` (komut geçmişi) + `cem history` (-n / --clear)
- [x] `cem -p` spinner — TTY-aware, bubbletea olmadan, sessiz pipe modu
- [x] nginx `/r/` proxy → GitLab Releases permalink (önceki: GitHub placeholder)
- [x] `.gitlab-ci.yml` release stage: 21 binary asset link + SHA256SUMS

## 9. Açık (kullanıcı kararı bekliyor)
- [ ] `.claude/agents/` ve `.claude/skills/` — `autoinstalltrixie` kalıntısı.
      Sil/değiştir kararı bekleniyor. `.claude/` artık gitignore'lı.

## 12. Kararlılık + effort/otomatik güncelleme (2026-09-09)
- [x] `Spinner.Stop()` panic: `stopWriter.Write` ve `Run`/`ModePair` ikisi
      birden durdurunca `close of closed channel` — artık `sync.Once`,
      nil-güvenli, race testli.
- [x] Hata teşhisi artık dökülen HTTP gövdelerinin içindeki kelimelere
      takılmıyor (`sanitizeStderr`); codex'in 105 KB'lık model JSON'u auth
      hatası sanılıyordu, gerçek hata "model desteklenmiyor" idi.
- [x] Model hataları kendi ipucunu alıyor (`hintModel`) — kullanıcı boşuna
      login akışına yollanmıyor.
- [x] `captureToolWithSpinner` sessiz çıkmıyor — pair modunda hiç mesaj
      basmadan exit 1 dönüyordu.
- [x] Writer prompt'u thinker'ın tekrarlanan son mesajını kırpıyor
      (`dedupeTrailingEcho`) — codex exec iki kez basıyor, token iki katına
      çıkıyordu.
- [x] Düşünme seviyesi seçilebilir: `cem effort`, wizard adımı, `cem init`
      adımı, `.cem.yaml > efforts`, çalıştırma başlığında görünür.
- [x] `cemi update` araçların kendi `update` komutunu kullanıyor; kurulu
      CLI'lar için günlük arka plan güncellemesi (`auto_update_tools: false`
      ile kapatılır).
- [x] `executor_test.go` + `spinner_test.go` eklendi (28 test, `-race` temiz).
- [ ] doc:CLAUDE update — File Layout, Runtime Gotchas, kural #7 ve
      doğrulama akışı yukarıdakine göre güncellendi.

## 13. Dil + çıktı netliği (2026-09-09)
- [x] İki dilli arayüz (tr/en): `L(tr, en)` yardımcısı, `cem lang`, sihirbaz
      ilk soruda dili soruyor, `CEM_LANG`/`LANG` algılama, cobra Short/Long
      için `applyLang()` (aksi halde paket-init döngüsü).
- [x] Çıktı gürültü filtresi (`noise.go`) + `--raw` kaçış kapısı.
- [x] codex artık `--output-last-message` ile çalışıyor: exec/apply-patch/diff
      yığını ve tekrarlanan final cevap yok.
- [x] Net başlıklar: `🧠 DÜŞÜNEN · gpt` (mavi) / `✍️ YAZAN · claude` (yeşil);
      "Open source" satırı yalnız banner ekranında, global config satırı
      sessiz (proje config'i hâlâ duyuruluyor).
- [ ] README / README.tr: `cem lang`, `cem effort`, `--raw` belgelenecek.
- [x] Quiet modda çift basım: URL geçiş kuralı cevabın tamamını sızdırıyordu, kaldırıldı.
- [x] `cem model` komutu (`cem effort` ile simetrik); plugin artık çıktıdaki
      her URL için auth balonu göstermiyor.
- [x] Pair modu: thinker plan çıkarır, writer kodu yazar (önceden ikisi de tam
      kodu yazıyordu). Roller tablosu artık rol başına model · seviye gösteriyor.
- [x] Rol başına süre (`⏱ düşünme 18.4s` / `⏱ yazma 9.2s` / toplam) ve düşünen
      ile yazan çıktısı arasında ayraç.
- [x] IntelliJ eklentisi custom plugin repository ile otomatik güncelleniyor
      (her release'de `updatePlugins.xml`).
- [x] codex'te `minimal` seviyesi artık geçersiz; seviye reddi model hatası
      olarak değil kendi ipucuyla raporlanıyor.
- [x] Cevap önbelleği (`cem cache`, `--no-cache`), spinner'da canlı süre,
      dizin bazlı güven onayı, sadeleşen README + README_DETAILS, SVG
      diyagramlar, zenginleşen eklenti Overview'ı.
- [x] Hızlı mod (`cem fast`): aracın kullanıcı ayarlarını atlar — aynı görevde
      124s → 8s ölçüldü. Varsayılan kapalı.
- [x] Yazan rolde canlı süreli spinner; hook gürültüsü filtrelendi.
- [x] Dile göre diyagramlar (`*.svg` İngilizce, `*.tr.svg` Türkçe).
- [x] Hızlı mod artık varsayılan; rol bazlı seviye varsayılanları
      (`applyRoleDefaults`); `cem doctor` maliyet kurulumunu denetliyor;
      düşünen plan yerine bilgi istediyse yazan atlanıyor.
- [x] `--no-cache` artık saklanan cevabı tazeliyor; eskisini yerinde
      bırakmıyor (okumayı atlar, yazmayı değil).
- [x] Önbellek doğruluğu: anahtar çalışma dizinini içeriyor; bilgi talebi
      cevapları ('dosya bulunamadı, paylaşın') hiç saklanmıyor.
- [x] Çıktı filtresi kodu bozuyordu: "tekrar eden satırı at" kuralı iç içe
      blokların ikinci `}` satırını yiyordu. Kaldırıldı, testi eklendi. Boş
      satır tekilleştirmesi ikiye gevşetildi (Python aralıkları korunsun).
- [x] Config'teki bozuk zaman damgası artık tüm komutları kilitlemiyor.
- [x] Yazan artık test çalıştırıp onay istemiyor; plan 15 satırla sınırlı.
      Aynı görev: 1m 38s → 1m 06s.
- [x] Çalıştırma başlıklarında saat var; pair toplamında tam tarih.
- [x] IntelliJ ayarlarına düşünme seviyesi + hızlı mod eklendi, model listesi
      tazelendi (gpt-5.6-terra).
- [x] Kod-isteği sözlüğü Türkçe karaktersiz yazımı da tanıyor (olustur,
      duzelt, cevir): eksik varyantlar 49 saniyelik boş bir koşu üretiyordu.
- [x] Düşünen artık görev istemediyse test planlamıyor, yazana da "en yalın
      kodu yaz" deniyor: aynı sınıf görevde 2 dosyada 8.2 KB yerine tek
      dosyada 1.9 KB.
- [x] `memory/` depoya taşındı (harness yolu artık symlink) ve projenin hedefi,
      ölçmeden gönderme geri bildirimi ve kullanıcı ortamı kısıtlarıyla
      tazelendi.
- [x] Yazan önbelleği: kayıt ürettiği dosyaları taşıyor ve isabet ettiğinde
      geri yazıyor; elle değişmiş dosyanın üzerine yazılmıyor.
- [x] IntelliJ eklentisi: seçim yokken açık dosyanın tamamı prompt olarak
      gönderilmiyor. README.md açıkken Ctrl+Alt+P, 112 saniye harcayıp
      "yazılacak yeni kod yok" cevabı döndürüyordu; artık `plugin.xml`'in
      zaten söylediği gibi prompt kutusu açılıyor. Sekme başlıklarından
      markdown/emoji ayıklanıyor, çıktı paneli satırı kesmek yerine sarıyor.
- [x] Soru artık dosya üretmiyor: "lua da helloworld nasıl yazılır?" isteğinde
      düşünenin cevabında kod bloğu olduğu için yazan çalışıyor ve çalışma
      dizinine istenmeyen `hello.lua` bırakıyordu. Sorularda yazan atlanıyor;
      rica kipi ("siler misin?") iş sayılmaya devam ediyor.
- [x] IntelliJ eklentisi: prompt artık modal dialog'da istenmiyor. Seçim
      yokken kısayol, araç penceresinin altındaki çok satırlı kutuya odaklanıp
      modu seçiyor (solda `pair`/`think`/`write` seçici); Enter gönderiyor,
      Shift+Enter satır atlıyor, ↑/↓ tarihçede geziyor. "dosya hakkında sor…"
      dosyayı bağlam olarak iliştirip talimatı aynı kutuda bekliyor.

## JetBrains Marketplace (açık)

Repo tarafı tamam: imza + `verifyPlugin` `build.gradle.kts`'e bağlandı,
`CHANGELOG.md` eklendi (`changeNotes` var olmayan dosyaya link veriyordu),
eklenti adı `cem`'e kısaltıldı (Marketplace ayırıcı noktalamayı reddediyor,
≤20 karakter istiyor), `pluginVersion` release tag'leriyle aynı CalVer'e çekildi
ve repo değişkeni `PUBLISH_MARKETPLACE=true` yapılınca çalışan uyuyan bir
`publish-intellij-plugin` CI job'ı eklendi.

Doğrulama sırasında iki gizli hata çıktı:
- Bytecode hedefi Java 21'di, ama `sinceBuild=233` IDE'leri JBR 17 ile
  çalışıyor — eklenti 2023.3–2024.1'de hiç yüklenmiyordu
  (`UnsupportedClassVersionError`). Artık JDK 21 toolchain'i üzerinde
  `--release 17` / `jvmTarget = 17` ile derleniyor.
- `buildSearchableOptions { enabled = false }`, `prepareJarSearchableOptions`
  görevini temiz bir checkout'ta hiç oluşmayan bir dizini girdi beklemeye
  bırakıyordu; `clean buildPlugin` her zaman patlıyordu. Yerelde eski build
  çıktısı dizini hayatta tuttuğu için görünmüyordu. Zincir tamamen kapatıldı.

Kullanıcıya kalanlar (otomatikleştirilemez):
- [ ] JetBrains hesabı + Marketplace satıcı profili.
- [ ] İmza anahtarını üret (`openssl genpkey` → `private.pem`, `chain.crt`),
      repoya sokma (`.gitignore` `*.pem` / `*.crt`'yi kapsıyor), sonra
      `./gradlew signPlugin -Pcem.signDir=$HOME/.cem-signing`.
- [ ] **ZIP**'i elle yükle (JAR değil — `snakeyaml-engine` zip'in içinde);
      plugins.jetbrains.com/plugin/add. İlk yayın elle yapılmak zorunda ve yeni
      eklentiler moderasyondan geçiyor.
- [ ] ≥1200×760 gerçek IDE ekran görüntüleri — `docs/img/cem-intellij.png`
      900×410 ve ekran görüntüsü değil, çizim.
- [ ] Etiket/kategori seç, listeleme formunda MIT lisansını onayla.
- [ ] Onay sonrası: `PUBLISH_TOKEN`, `CERTIFICATE_CHAIN`, `PRIVATE_KEY`,
      `PRIVATE_KEY_PASSWORD` secret'larını oluştur ve
      `PUBLISH_MARKETPLACE=true` yap.
