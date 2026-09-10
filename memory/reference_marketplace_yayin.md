---
name: reference-marketplace-yayin
description: cem IntelliJ eklentisinin JetBrains Marketplace yayın durumu — repo tarafı hazır, elle yapılacaklar bekliyor
metadata:
  type: reference
---

> Not: Eklenti derleme tuzakları (JDK 21, tr_TR locale) [[reference-eklenti-derleme]]'de;
> imza/jvmTarget gerekçeleri `plugin/intellij/build.gradle.kts` yorumlarında tutuluyor — burada tekrar yok.

**Durum (2026-09-10):** Eklenti henüz Marketplace'te DEĞİL; dağıtım
`updatePlugins.xml` custom repository ile GitHub releases üzerinden yapılıyor.
Repo tarafı yayına hazır (imza yapılandırması, `verifyPlugin`, `CHANGELOG.md`,
ad `cem`, CalVer sürüm, uyuyan `publish-intellij-plugin` CI job'ı).

**Yayın kimliği:** plugin id `dev.cempw.intellij` (DEĞİŞMEZ), vendor maili
`musluyuksektepe@gmail.com` (kullanıcının seçimi — `hi@cem.pw` yerine, çünkü
JetBrains moderasyon yazışması oraya gidiyor ve çalışan bir kutu olmalı).

**Elle yapılacaklar** `todo.tr.md` → "JetBrains Marketplace (açık)" başlığında.
Kritik iki nokta: ilk yayın **elle** yüklenmek zorunda ve **ZIP** yüklenir
(JAR'da `snakeyaml-engine` kaybolur). CI job'ı yalnızca repo değişkeni
`PUBLISH_MARKETPLACE=true` olduğunda çalışır — moderasyon onayı gelmeden açma.
