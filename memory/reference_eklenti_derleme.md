---
name: reference-eklenti-derleme
description: IntelliJ eklentisini yerelde derlemenin iki tuzağı — JDK 21 yok, tr_TR locale derlemeyi kırıyor
metadata:
  type: reference
---

> Not: Proje yapısı, dosya listesi ve kod kuralları CLAUDE.md'de tutuluyor — burada tekrar yok.

`plugin/intellij` derlemesi (`gradle compileKotlin` / `buildPlugin`) bu makinede
iki nedenle düz çalışmıyor; ikisi de ölçüldü (2026-09-09):

1. **JDK 21 kurulu değil.** `/usr/lib/jvm` yalnız 8/11, `~/.jdks` yalnız
   temurin-17. Gradle'ın kendisi JVM 17+ istiyor (JAVA_HOME=temurin-17),
   toolchain ise 21 istiyor. Kalıcı çözüm: `sudo nala install openjdk-21-jdk`
   ya da Temurin 21'i `~/.jdks` altına açıp
   `-Porg.gradle.java.installations.paths=<yol>` ile göstermek.
2. **Türkçe locale derlemeyi kırıyor.** tr_TR'de büyük harfe çevirme
   `APPLICATION` yerine `APPLİCATİON` üretiyor ve plugin-structure IDE
   descriptor'ını okuyamıyor:
   `No enum constant ...ServiceType.APPLİCATİON`. Çözüm: gradle'ı
   `LC_ALL=en_US.UTF-8` + `-Dorg.gradle.jvmargs=-Duser.language=en -Duser.country=US`
   ile çalıştır.

Gradle wrapper binary'si depoda yok; `~/.gradle/wrapper/dists/gradle-9.3.1-bin/*/gradle-9.3.1/bin/gradle`
kullanılıyor. Ölçmeden göndermeme kuralı için [[feedback-olcerek-gonder]].
