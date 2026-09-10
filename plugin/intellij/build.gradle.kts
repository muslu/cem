// IntelliJ Platform plugin for cem — runs `cem`/`cemi`/`cemir` on editor selection.
//
// Build:    ./gradlew buildPlugin
// Run IDE:  ./gradlew runIde
// Verify:   ./gradlew verifyPlugin
// Sign:     ./gradlew signPlugin    -Pcem.signDir=$HOME/.cem-signing
// Publish:  ./gradlew publishPlugin (PUBLISH_TOKEN + imza anahtarları gerekir;
//           ilk yayın Marketplace'e ELLE yüklenmek zorunda)
//
// Output:   build/distributions/cem-intellij-*.zip — installable via
//           PyCharm → Settings → Plugins → ⚙ → Install Plugin from Disk.

import org.jetbrains.intellij.platform.gradle.TestFrameworkType
import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import org.jetbrains.kotlin.gradle.tasks.KotlinCompile

plugins {
    id("java")
    id("org.jetbrains.kotlin.jvm") version "2.0.21"
    id("org.jetbrains.intellij.platform") version "2.16.0"
}

group = "dev.cempw"
version = providers.gradleProperty("pluginVersion").get()

// Derleme JDK'si 21, ÜRETİLEN bytecode 17.
// sinceBuild=233 (2023.3) IDE'leri JBR 17 ile çalışıyor; Java 21 bytecode o
// sürümlerde UnsupportedClassVersionError ile hiç YÜKLENMEZ (yalnızca 2024.2+
// JBR 21'e geçti). verifyPluginProjectConfiguration bunu 4 ayrı uyarı olarak
// bildiriyordu. Toolchain'i 17'ye çekmek yerine hedefi düşürüyoruz: makinede
// kurulu tek JDK 21 ve offline build 17 toolchain'i indiremez.
kotlin {
    jvmToolchain(21)
}

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}

repositories {
    mavenCentral()
    intellijPlatform { defaultRepositories() }
}

dependencies {
    intellijPlatform {
        // 2023.3 = IntelliJ Platform 233.x — sinceBuild ile aynı baseline.
        intellijIdeaCommunity("2023.3")
        bundledPlugin("com.intellij.platform.images")
        testFramework(TestFrameworkType.Platform)
    }
    // ~/.cem/config.yaml okuma/yazma için
    implementation("org.snakeyaml:snakeyaml-engine:2.7")
    testImplementation("junit:junit:4.13.2")
}

intellijPlatform {
    pluginConfiguration {
        ideaVersion {
            // PyCharm/IDEA/GoLand 2023.3+ (Kasım 2023). IntelliJ Platform Gradle
            // Plugin 2.16.0 minimum 233 destekliyor — 232 'too low' diye reddediyor.
            sinceBuild = "233"
            untilBuild = provider { null }
        }
        changeNotes = providers.gradleProperty("pluginVersion").map { v ->
            "Plugin version $v. See <a href=\"https://github.com/muslu/cem/blob/main/CHANGELOG.md\">CHANGELOG</a>."
        }
    }
    // Marketplace imzası ZORUNLU: "Before publishing a plugin, make sure it is
    // signed". Anahtarlar repoda tutulmaz.
    //   CI    → CERTIFICATE_CHAIN / PRIVATE_KEY / PRIVATE_KEY_PASSWORD (secret)
    //   Yerel → ./gradlew signPlugin -Pcem.signDir=$HOME/.cem-signing
    //           (dizinde chain.crt + private.pem bulunur)
    // String property'ler file olanlara göre önceliklidir; env yoksa dosya devreye
    // girer, ikisi de yoksa yalnızca signPlugin/publishPlugin hata verir —
    // buildPlugin imzasız da çalışmaya devam eder.
    signing {
        certificateChain = providers.environmentVariable("CERTIFICATE_CHAIN")
        privateKey       = providers.environmentVariable("PRIVATE_KEY")
        password         = providers.environmentVariable("PRIVATE_KEY_PASSWORD")

        providers.gradleProperty("cem.signDir").orNull?.let { dir ->
            certificateChainFile = file("$dir/chain.crt")
            privateKeyFile       = file("$dir/private.pem")
        }
    }

    publishing {
        token = providers.environmentVariable("PUBLISH_TOKEN")
    }

    // IntelliJ Plugin Verifier — API uyumsuzluklarını Marketplace moderasyonundan
    // ÖNCE yakalar. `./gradlew verifyPlugin`
    pluginVerification {
        ides { recommended() }
    }
}

tasks {
    withType<JavaCompile>().configureEach {
        options.release = 17
    }
    // jvmToolchain(21) jvmTarget'ı kendisi 21'e set ediyor ve extension
    // seviyesindeki ayarı eziyor — task üzerinde set etmek gerekiyor.
    withType<KotlinCompile>().configureEach {
        compilerOptions.jvmTarget = JvmTarget.JVM_17
    }

    // buildSearchableOptions kapalıyken prepareJarSearchableOptions var olmayan
    // build/tmp/buildSearchableOptions dizinini girdi bekliyor ve TEMİZ bir
    // checkout'ta derleme patlıyor ("An input file was expected to be present").
    // Yerelde eski build çıktısı dizini hayatta tuttuğu için gizli kalmıştı.
    // Zinciri baştan sona kapat.
    prepareJarSearchableOptions {
        enabled = false
    }
    jarSearchableOptions {
        enabled = false
    }

    test {
        useJUnit()
    }
    // CI'da headless IDE bazen exit 255 atıyor; arama indeksini pre-build etmek
    // opsiyonel (Search Everywhere'de plugin settings'i bulmaya yarar). Bizim
    // settings tek bir 'cem.path' field'ı, indeksleme yok = pratik sorun yok.
    buildSearchableOptions {
        enabled = false
    }
}
