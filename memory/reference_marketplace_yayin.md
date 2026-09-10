---
name: reference-marketplace-yayin
description: cem IntelliJ eklentisinin JetBrains Marketplace yayın durumu — repo tarafı hazır, elle yapılacaklar bekliyor
metadata:
  type: reference
---

> Not: Eklenti derleme tuzakları (JDK 21, tr_TR locale) [[reference-eklenti-derleme]]'de;
> imza/jvmTarget gerekçeleri `plugin/intellij/build.gradle.kts` yorumlarında tutuluyor — burada tekrar yok.

**Durum (2026-09-10):** Eklenti Marketplace'te — **sayfa 34196**
(`plugins.jetbrains.com/plugin/34196`), ilk yükleme kullanıcı tarafından elle
yapıldı. `updatePlugins.xml` custom repository akışı yedek olarak duruyor.

**Son yayın:** `20260910.03` (2026-09-10) — GitHub release 32 asset ile çıktı,
Marketplace'e imzalı zip yüklendi. Marketplace API'si eklenti için hâlâ
`approve: false` diyor: yeni eklenti moderasyonda, onay gelene kadar
`/api/plugins/34196/updates` boş liste döner ve sürüm listede görünmez.

**Tek komutla yayın:** `./surum-yayinla.sh` (kullanıcı "yayınla" dediğinde
çalıştırılacak — ayrıntı CLAUDE.tr.md "Git & Release"). Marketplace adımı
`~/.cem-signing/parola` + `~/.cem-signing/token` dosyalarına bakıyor (chmod
600); yoksa GitHub'da kalıyor.

**Uyumluluk raporu (verifier 1.410, 2026-09-10):** iki uyarı geldi ve
düzeltildi — `SimpleListCellRenderer.create(String, Function)` kaldırılmak
üzereydi (3 argümanlı customizer'a geçildi), `doWhenFocusSettlesDown(Runnable)`
deprecated'di (ModalityState'li imza). 2024.3.7.1 zaten Success'ti; uyarılar
2025.1+ sürümlerde çıkıyordu.

**Marketplace görselleri:** `docs/img/market-{pair,input,menu}[.tr].png`,
tam 1200x760 (Marketplace minimumu). `docs/img/market-gorseller.py` ile
yeniden üretiliyor (headless Chrome 2x → convert ile küçültme). Bunlar IDE
mock-up'ı; gerçek ekran görüntüsü değil.

**Yayın kimliği:** plugin id `dev.cempw.cem` (DEĞİŞMEZ; `dev.cempw.intellij` Marketplace tarafından reddedildi — ID 'intellij' içeremiyor), vendor maili
`musluyuksektepe@gmail.com` (kullanıcının seçimi — `hi@cem.pw` yerine, çünkü
JetBrains moderasyon yazışması oraya gidiyor ve çalışan bir kutu olmalı).

**Yerel yayın script'i:** `plugin/intellij/yayinla.sh` — paket kurulumu, JDK 21,
anahtar üretimi, derle+imzala+imza doğrula, `--yayinla` ile publishPlugin.
İmza anahtarı `~/.cem-signing/{private.pem,chain.crt}` (2026-09-10'da üretildi,
sertifika 2036'ya kadar). Anahtar kaybolursa aynı kimlikle güncelleme
gönderilemez — yedeklenmesi gerekiyor.

**Elle yapılacaklar** `todo.tr.md` → "JetBrains Marketplace (açık)" başlığında.
Kritik iki nokta: ilk yayın **elle** yüklenmek zorunda ve **ZIP** yüklenir
(JAR'da `snakeyaml-engine` kaybolur). CI job'ı yalnızca repo değişkeni
`PUBLISH_MARKETPLACE=true` olduğunda çalışır — moderasyon onayı gelmeden açma.
