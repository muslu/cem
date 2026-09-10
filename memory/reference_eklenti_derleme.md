---
name: reference-eklenti-derleme
description: IntelliJ eklentisini yerelde derleme — wrapper + JDK 21 ayarı yapıldı, tr_TR locale tuzağı sürüyor
metadata:
  type: reference
---

> Not: Proje yapısı, dosya listesi ve kod kuralları CLAUDE.md'de tutuluyor — burada tekrar yok.

`plugin/intellij` derlemesi (`gradle compileKotlin` / `buildPlugin`) bu makinede
iki nedenle düz çalışmıyor; ikisi de ölçüldü (2026-09-09):

1. **JDK 21 gerekiyor** (`/usr/lib/jvm` yalnız 8/11, PATH'teki java 11 — Gradle 9
   en az 17 istiyor). Temurin 21 `~/.jdks/jdk-21.0.12.1+1` altında.
   **2026-09-10'da çözüldü:** `~/.gradle/gradle.properties` içine
   `org.gradle.java.home=/home/muslu/.jdks/jdk-21.0.12.1+1` yazıldı (makineye
   özgü olduğu için repoya değil, kullanıcı düzeyine). Artık komut başına
   JAVA_HOME/`-P` vermek gerekmiyor; başka makinede aynı satırı eklemek gerekir.
2. **Türkçe locale derlemeyi kırıyor.** tr_TR'de büyük harfe çevirme
   `APPLICATION` yerine `APPLİCATİON` üretiyor ve plugin-structure IDE
   descriptor'ını okuyamıyor:
   `No enum constant ...ServiceType.APPLİCATİON`. Çözüm: gradle'ı
   `LC_ALL=en_US.UTF-8` + `-Dorg.gradle.jvmargs=-Duser.language=en -Duser.country=US`
   ile çalıştır.

**2026-09-10:** Gradle wrapper depoya eklendi (`plugin/intellij/gradlew`,
Gradle 9.3.1) — `gradle` PATH'te yok, artık `./gradlew` kullan; CI de wrapper'a
çevrildi. `buildPlugin --offline` artık ÇALIŞIYOR (`java-compiler-ant-tasks`
cache'e girdi); ilk kez indirmek gerekirse ağ şart. Ölçmeden göndermeme kuralı için [[feedback-olcerek-gonder]].

**Yerelde kurma + "hâlâ eski davranıyor" tuzağı.** Eklenti hatası bildirildiğinde
önce KURULU jar'ın sürümünü doğrula, kaynağı değil:
`ls ~/.local/share/JetBrains/<IDE>/cem-intellij/lib`. 2026-09-10'da modal dialog
"hâlâ açılıyor" denen durumun sebebi buydu — kurulu `20260910.01` tag'i, inline-kutu
commit'inden 12 dakika önce atılmıştı. Sembol kontrolü kesin sonuç verir:
`unzip -p <jar> 'dev/cempw/intellij/CemTab$Companion.class' | strings | grep askInInput`.
Kurulum: `gradle buildPlugin -PpluginVersion=<tag>` → eski dizini yedeğe taşı →
zip'i `~/.local/share/JetBrains/<IDE>/` altına aç → IDE yeniden başlat.
