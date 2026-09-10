#!/usr/bin/env bash
# yayinla.sh — cem IntelliJ eklentisini Ubuntu üzerinde derler, İMZALAR ve
# Marketplace'e yüklemeye hazır zip'i çıkarır.
#
#   ./yayinla.sh                      derle + imzala (varsayılan)
#   ./yayinla.sh --surum 20260911.01  sürümü tag ile ver (gradle.properties'i ezer)
#   ./yayinla.sh --dogrula            ek olarak IntelliJ Plugin Verifier çalıştır (ağ + ~1 GB indirme)
#   ./yayinla.sh --yayinla            imzalı zip'i Marketplace'e YÜKLE (yalnız onaydan sonra)
#   ./yayinla.sh --anahtar-yenile     mevcut imza anahtarını silip yenisini üret
#   ./yayinla.sh --yardim
#
# İLK yayın Marketplace'e ELLE yüklenmek zorunda (JetBrains kuralı):
# plugins.jetbrains.com/plugin/add → ZIP yükle. --yayinla ancak eklenti
# onaylandıktan ve token alındıktan sonra çalışır.
set -euo pipefail

readonly IMZA_DIZIN="${CEM_SIGN_DIR:-$HOME/.cem-signing}"
readonly PROJE_DIZIN="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PAKETLER=(openssl unzip curl)

DOGRULA=0
YAYINLA=0
ANAHTAR_YENILE=0
SURUM=""

# tr_TR locale'de büyük harfe çevirme APPLICATION yerine APPLİCATİON üretiyor ve
# plugin-structure IDE descriptor'ını okuyamıyor (ölçüldü 2026-09-09). Sabit.
export LC_ALL=en_US.UTF-8

# İlerleme mesajları stderr'e: jdk21_kur / paket_yoneticisi gibi fonksiyonların
# stdout'u DÖNÜŞ DEĞERİ olarak $( ) ile okunuyor, mesaj oraya karışırsa
# org.gradle.java.home'a çöp yol gider (yaşandı).
bilgi()  { printf '\033[36m→\033[0m %s\n' "$*" >&2; }
tamam()  { printf '\033[32m✓\033[0m %s\n' "$*" >&2; }
uyari()  { printf '\033[33m!\033[0m %s\n' "$*" >&2; }
hata()   { printf '\033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

yardim() {
    # Başlıktaki yorum bloğunu, ilk kod satırına kadar bas (sabit satır aralığı
    # dosya büyüdükçe koda taşıyordu).
    awk 'NR>1 && /^#/ { sub(/^# ?/, ""); print; next } NR>1 { exit }' "${BASH_SOURCE[0]}"
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --surum)          SURUM="${2:-}"; [[ -n "$SURUM" ]] || hata "--surum bir değer bekliyor"; shift 2 ;;
        --dogrula)        DOGRULA=1; shift ;;
        --yayinla)        YAYINLA=1; shift ;;
        --anahtar-yenile) ANAHTAR_YENILE=1; shift ;;
        --yardim|-h)      yardim ;;
        *)                hata "bilinmeyen argüman: $1 (--yardim)" ;;
    esac
done

# ─────────────────────────────────────────────────────────────────────────────
# 1) Paketler
# ─────────────────────────────────────────────────────────────────────────────
paket_yoneticisi() {
    # Proje kuralı: apt yerine nala. nala yoksa apt ile bir kez kurulur.
    if command -v nala >/dev/null 2>&1; then
        echo "nala"
    else
        uyari "nala yok, apt ile kuruluyor (proje kuralı: apt yerine nala)"
        sudo apt-get update -qq && sudo apt-get install -y nala >/dev/null
        command -v nala >/dev/null 2>&1 && echo "nala" || echo "apt-get"
    fi
}

paketleri_kur() {
    local eksik=()
    for p in "${PAKETLER[@]}"; do
        command -v "$p" >/dev/null 2>&1 || eksik+=("$p")
    done
    if [[ ${#eksik[@]} -gt 0 ]]; then
        bilgi "eksik paketler kuruluyor: ${eksik[*]}"
        local pm; pm="$(paket_yoneticisi)"
        sudo "$pm" install -y "${eksik[@]}"
    fi
    tamam "openssl / unzip / curl hazır"
}

# JDK 21: eklenti JDK 21 toolchain'i ile derleniyor, PATH'teki java 11 olabilir.
jdk21_bul() {
    local aday
    for aday in "$HOME"/.jdks/jdk-21* /usr/lib/jvm/java-21-openjdk-* /usr/lib/jvm/temurin-21-* /usr/lib/jvm/jdk-21*; do
        [[ -x "$aday/bin/javac" ]] && { echo "$aday"; return 0; }
    done
    return 1
}

jdk21_kur() {
    local jdk
    if jdk="$(jdk21_bul)"; then
        tamam "JDK 21 bulundu: $jdk"
        echo "$jdk"; return 0
    fi
    bilgi "JDK 21 yok, openjdk-21-jdk kuruluyor"
    local pm; pm="$(paket_yoneticisi)"
    sudo "$pm" install -y openjdk-21-jdk
    jdk="$(jdk21_bul)" || hata "openjdk-21-jdk kurulduğu hâlde JDK 21 bulunamadı"
    tamam "JDK 21 kuruldu: $jdk"
    echo "$jdk"
}

# ─────────────────────────────────────────────────────────────────────────────
# 2) İmza anahtarı
# ─────────────────────────────────────────────────────────────────────────────
# Marketplace imzayı ZORUNLU tutuyor. Anahtar repoya değil, $IMZA_DIZIN'e yazılır
# (.gitignore *.pem / *.crt / .cem-signing/ kapsıyor).
anahtar_uret() {
    mkdir -p "$IMZA_DIZIN"; chmod 700 "$IMZA_DIZIN"

    local p1 p2
    while :; do
        read -rsp "İmza anahtarı için parola (en az 8 karakter): " p1; echo
        read -rsp "Parolayı yeniden gir: " p2; echo
        [[ "$p1" == "$p2" ]] || { uyari "parolalar aynı değil, tekrar"; continue; }
        # openssl 4 karakterin altını reddediyor; 8 sınırı bilinçli olarak daha yüksek.
        [[ ${#p1} -ge 8 ]] || { uyari "parola çok kısa (${#p1} karakter)"; continue; }
        break
    done

    export CEM_KEYPASS="$p1"
    unset p1 p2

    bilgi "4096 bit RSA anahtarı üretiliyor (şifreli)"
    openssl genpkey -aes-256-cbc -algorithm RSA \
        -out "$IMZA_DIZIN/private.pem" -pkeyopt rsa_keygen_bits:4096 \
        -pass env:CEM_KEYPASS

    bilgi "kendinden imzalı sertifika zinciri üretiliyor (10 yıl)"
    openssl req -new -x509 -days 3650 \
        -key "$IMZA_DIZIN/private.pem" -passin env:CEM_KEYPASS \
        -out "$IMZA_DIZIN/chain.crt" \
        -subj "/CN=cem.pw/O=cem/emailAddress=musluyuksektepe@gmail.com"

    chmod 600 "$IMZA_DIZIN/private.pem" "$IMZA_DIZIN/chain.crt"
    tamam "anahtar üretildi: $IMZA_DIZIN/{private.pem,chain.crt}"
    uyari "BU DOSYALARI YEDEKLE. Kaybedersen aynı anahtarla imzalanmış"
    uyari "güncelleme gönderemezsin; sertifika süresi 2036'ya kadar geçerli."
}

anahtar_hazirla() {
    if [[ $ANAHTAR_YENILE -eq 1 ]]; then
        uyari "mevcut anahtar siliniyor: $IMZA_DIZIN"
        rm -f "$IMZA_DIZIN"/private*.pem "$IMZA_DIZIN"/chain.crt
        anahtar_uret; return
    fi
    if [[ -f "$IMZA_DIZIN/private.pem" && -f "$IMZA_DIZIN/chain.crt" ]]; then
        tamam "imza anahtarı mevcut: $IMZA_DIZIN"
        parola_al
    else
        bilgi "imza anahtarı yok, üretiliyor"
        anahtar_uret
    fi
}

# Parola sırası: env → $IMZA_DIZIN/parola (0600) → sor.
parola_al() {
    if [[ -n "${PRIVATE_KEY_PASSWORD:-}" ]]; then
        export CEM_KEYPASS="$PRIVATE_KEY_PASSWORD"
        tamam "parola PRIVATE_KEY_PASSWORD ortam değişkeninden alındı"
        return
    fi
    local dosya="$IMZA_DIZIN/parola"
    if [[ -f "$dosya" ]]; then
        # Dünyaya açık parola dosyasını kullanmıyoruz — sessizce kabul etmek
        # kullanıcıyı yanlış güvene sokar.
        local mod; mod="$(stat -c '%a' "$dosya")"
        [[ "$mod" == "600" ]] || hata "$dosya izni $mod — 'chmod 600 $dosya' yapıp tekrar dene"
        export CEM_KEYPASS="$(<"$dosya")"
        tamam "parola $dosya dosyasından alındı"
        return
    fi
    read -rsp "İmza anahtarı parolası: " CEM_KEYPASS; echo
    export CEM_KEYPASS
    [[ -n "$CEM_KEYPASS" ]] || hata "parola boş"
}

# Yanlış parola signPlugin'de BouncyCastle yığın izi olarak çıkıyor
# ("pad block corrupted" → 20 satır Java stack trace, sebebi görünmüyor).
# Anahtarı önce openssl ile açmayı dene: hata tek satırda anlaşılsın.
parola_dogrula() {
    openssl pkey -in "$IMZA_DIZIN/private.pem" -passin env:CEM_KEYPASS -noout 2>/dev/null && return
    hata "parola özel anahtarı açmıyor ($IMZA_DIZIN/private.pem).
   Doğru parolayı yaz:  printf '%s' 'PAROLA' > $IMZA_DIZIN/parola && chmod 600 $IMZA_DIZIN/parola
   Parolayı hatırlamıyorsan:  ./yayinla.sh --anahtar-yenile"
}

# ─────────────────────────────────────────────────────────────────────────────
# 3) Derle + imzala
# ─────────────────────────────────────────────────────────────────────────────
main() {
    cd "$PROJE_DIZIN"
    [[ -x ./gradlew ]] || hata "./gradlew bulunamadı — $PROJE_DIZIN doğru dizin mi?"

    paketleri_kur
    local JDK; JDK="$(jdk21_kur)"
    anahtar_hazirla
    parola_dogrula

    local -a gopt=(
        "-Dorg.gradle.java.home=$JDK"   # PATH'teki java 11 olabilir; Gradle 9 en az 17 istiyor
        "-Pcem.signDir=$IMZA_DIZIN"
        --console=plain
    )
    [[ -n "$SURUM" ]] && gopt+=("-PpluginVersion=$SURUM")

    bilgi "derleniyor + imzalanıyor${SURUM:+ (sürüm $SURUM)}"
    PRIVATE_KEY_PASSWORD="$CEM_KEYPASS" ./gradlew "${gopt[@]}" \
        clean verifyPluginProjectConfiguration buildPlugin signPlugin verifyPluginSignature

    if [[ $DOGRULA -eq 1 ]]; then
        bilgi "IntelliJ Plugin Verifier (IDE indirme uzun sürebilir)"
        PRIVATE_KEY_PASSWORD="$CEM_KEYPASS" ./gradlew "${gopt[@]}" verifyPlugin
    fi

    local zip
    zip="$(ls -1t build/distributions/*-signed.zip 2>/dev/null | head -1)" \
        || hata "imzalı zip üretilmedi"
    [[ -n "$zip" ]] || hata "imzalı zip üretilmedi"

    if [[ $YAYINLA -eq 1 ]]; then
        local token="${PUBLISH_TOKEN:-}"
        if [[ -z "$token" && -f "$IMZA_DIZIN/token" ]]; then
            local mod; mod="$(stat -c '%a' "$IMZA_DIZIN/token")"
            [[ "$mod" == "600" ]] || hata "$IMZA_DIZIN/token izni $mod — chmod 600 yap"
            token="$(<"$IMZA_DIZIN/token")"
        fi
        [[ -n "$token" ]] || hata "PUBLISH_TOKEN yok (plugins.jetbrains.com/author/me/tokens → $IMZA_DIZIN/token)"
        bilgi "Marketplace'e yükleniyor"
        PRIVATE_KEY_PASSWORD="$CEM_KEYPASS" PUBLISH_TOKEN="$token" \
            ./gradlew "${gopt[@]}" publishPlugin
        tamam "yayınlandı: $(basename "$zip")"
        return
    fi

    echo
    tamam "imzalı paket hazır:"
    printf '   %s\n' "$PROJE_DIZIN/$zip"
    printf '   %s\n' "$(sha256sum "$zip" | cut -d' ' -f1)"
    echo
    echo "Sıradaki adım:"
    echo "  • İlk yayın (elle, zorunlu): https://plugins.jetbrains.com/plugin/add"
    echo "    → ZIP'i yükle (JAR değil), lisans MIT, en az bir kategori seç."
    echo "  • Onaydan sonra güncellemeler: ./yayinla.sh --surum <tag> --yayinla"
}

main "$@"
