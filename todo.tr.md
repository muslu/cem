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
