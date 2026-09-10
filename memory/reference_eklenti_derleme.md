---
name: reference-eklenti-derleme
description: IntelliJ eklentisini yerelde derlemenin iki tuzağı — JDK 21 yok, tr_TR locale derlemeyi kırıyor
metadata:
  type: reference
---

> Not: Proje yapısı, dosya listesi ve kod kuralları CLAUDE.md'de tutuluyor — burada tekrar yok.

`plugin/intellij` derlemesi (`gradle compileKotlin` / `buildPlugin`) bu makinede
iki nedenle düz çalışmıyor; ikisi de ölçüldü (2026-09-09):

1. **JDK 21 gerekiyor** (`/usr/lib/jvm` yalnız 8/11). Gradle'ın kendisi JVM 17+
   istiyor (JAVA_HOME=temurin-17), toolchain ise 21. 2026-09-10'da Temurin 21
   kalıcı olarak `~/.jdks/jdk-21.0.12.1+1` altına açıldı; derlerken
   `-Porg.gradle.java.installations.paths=$HOME/.jdks/jdk-21.0.12.1+1` ver.
2. **Türkçe locale derlemeyi kırıyor.** tr_TR'de büyük harfe çevirme
   `APPLICATION` yerine `APPLİCATİON` üretiyor ve plugin-structure IDE
   descriptor'ını okuyamıyor:
   `No enum constant ...ServiceType.APPLİCATİON`. Çözüm: gradle'ı
   `LC_ALL=en_US.UTF-8` + `-Dorg.gradle.jvmargs=-Duser.language=en -Duser.country=US`
   ile çalıştır.

Gradle wrapper binary'si depoda yok; `~/.gradle/wrapper/dists/gradle-9.3.1-bin/*/gradle-9.3.1/bin/gradle`
kullanılıyor. `buildPlugin` **`--offline` ile çalışmıyor**:
`java-compiler-ant-tasks` (instrumentCode) cache'te yok, ağ gerekiyor —
`compileKotlin` offline yeterli. Ölçmeden göndermeme kuralı için [[feedback-olcerek-gonder]].

**Yerelde kurma + "hâlâ eski davranıyor" tuzağı.** Eklenti hatası bildirildiğinde
önce KURULU jar'ın sürümünü doğrula, kaynağı değil:
`ls ~/.local/share/JetBrains/<IDE>/cem-intellij/lib`. 2026-09-10'da modal dialog
"hâlâ açılıyor" denen durumun sebebi buydu — kurulu `20260910.01` tag'i, inline-kutu
commit'inden 12 dakika önce atılmıştı. Sembol kontrolü kesin sonuç verir:
`unzip -p <jar> 'dev/cempw/intellij/CemTab$Companion.class' | strings | grep askInInput`.
Kurulum: `gradle buildPlugin -PpluginVersion=<tag>` → eski dizini yedeğe taşı →
zip'i `~/.local/share/JetBrains/<IDE>/` altına aç → IDE yeniden başlat.
