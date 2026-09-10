#!/usr/bin/env bash
# surum-yayinla.sh — tek komutla sürüm yayınlar: GitHub + JetBrains Marketplace.
#
#   ./surum-yayinla.sh                 sıradaki takvim sürümünü yayınla
#   ./surum-yayinla.sh --surum X       sürümü elle ver (YYYYMMDD.NN)
#   ./surum-yayinla.sh --kuru          hiçbir şey yapmadan ne olacağını göster
#   ./surum-yayinla.sh --sadece-github Marketplace'i atla
#   ./surum-yayinla.sh --yardim
#
# Akış: testler → sıradaki tag → git tag + push (CI 7 platform binary + eklenti
# zip'i + release'i kendisi yayınlar) → eklentiyi imzala ve Marketplace'e gönder.
#
# Marketplace adımı için iki dosya gerekiyor (yoksa GitHub'da kalır, uyarır):
#   ~/.cem-signing/parola   imza anahtarının parolası   (chmod 600)
#   ~/.cem-signing/token    plugins.jetbrains.com token (chmod 600)
set -euo pipefail

readonly KOK="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly IMZA_DIZIN="${CEM_SIGN_DIR:-$HOME/.cem-signing}"
readonly DAL=main

KURU=0
SADECE_GITHUB=0
SURUM=""

bilgi() { printf '\033[36m→\033[0m %s\n' "$*" >&2; }
tamam() { printf '\033[32m✓\033[0m %s\n' "$*" >&2; }
uyari() { printf '\033[33m!\033[0m %s\n' "$*" >&2; }
hata()  { printf '\033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
    case "$1" in
        --surum)         SURUM="${2:-}"; [[ -n "$SURUM" ]] || hata "--surum bir değer bekliyor"; shift 2 ;;
        --kuru)          KURU=1; shift ;;
        --sadece-github) SADECE_GITHUB=1; shift ;;
        --yardim|-h)     awk 'NR>1 && /^#/ { sub(/^# ?/, ""); print; next } NR>1 { exit }' "${BASH_SOURCE[0]}"; exit 0 ;;
        *)               hata "bilinmeyen argüman: $1 (--yardim)" ;;
    esac
done

cd "$KOK"

# ── sıradaki takvim sürümü ───────────────────────────────────────────────────
# Şema YYYYMMDD.NN — aynı gün ikinci yayın .02 olur.
sonraki_surum() {
    local gun; gun="$(date +%Y%m%d)"
    local en_buyuk=0 t n
    for t in $(git tag -l "${gun}.*"); do
        n="${t#*.}"
        [[ "$n" =~ ^[0-9]+$ ]] && (( 10#$n > en_buyuk )) && en_buyuk=$((10#$n))
    done
    printf '%s.%02d' "$gun" $((en_buyuk + 1))
}

# ── ön kontroller ────────────────────────────────────────────────────────────
[[ "$(git rev-parse --abbrev-ref HEAD)" == "$DAL" ]] || hata "dal $DAL değil: $(git rev-parse --abbrev-ref HEAD)"
[[ -z "$(git status --porcelain)" ]] || {
    git status --short >&2
    hata "çalışma dizini temiz değil — önce commit et"
}

bilgi "origin ile senkron kontrolü"
git fetch --quiet origin "$DAL"
[[ "$(git rev-parse HEAD)" == "$(git rev-parse "origin/$DAL")" ]] || \
    uyari "HEAD ile origin/$DAL farklı — push edilecek commit'ler var"

[[ -n "$SURUM" ]] || SURUM="$(sonraki_surum)"
git rev-parse -q --verify "refs/tags/$SURUM" >/dev/null && hata "tag zaten var: $SURUM"

# Marketplace için gereken sırlar; eksikse GitHub'da kalınır.
MARKET=0
if [[ $SADECE_GITHUB -eq 0 ]]; then
    if [[ -f "$IMZA_DIZIN/parola" && -f "$IMZA_DIZIN/token" ]]; then
        MARKET=1
    else
        uyari "Marketplace atlanıyor — eksik: $(
            [[ -f "$IMZA_DIZIN/parola" ]] || printf '%s ' "$IMZA_DIZIN/parola"
            [[ -f "$IMZA_DIZIN/token" ]]  || printf '%s ' "$IMZA_DIZIN/token")"
    fi
fi

echo
bilgi "sürüm       : $SURUM"
bilgi "GitHub      : tag + push → CI release"
bilgi "Marketplace : $([[ $MARKET -eq 1 ]] && echo "evet (publishPlugin)" || echo "hayır")"
echo

if [[ $KURU -eq 1 ]]; then
    tamam "kuru çalışma — hiçbir şey yapılmadı"
    exit 0
fi

# ── testler ──────────────────────────────────────────────────────────────────
bilgi "go test -race ./..."
go test -race ./... >/dev/null
tamam "Go testleri geçti"

bilgi "eklenti derlemesi (compileKotlin)"
( cd plugin/intellij && LC_ALL=en_US.UTF-8 ./gradlew --quiet compileKotlin )
tamam "eklenti derlendi"

# ── tag + push ───────────────────────────────────────────────────────────────
onceki="$(git describe --tags --abbrev=0 2>/dev/null || true)"
notlar="$(git log --format='- %s' ${onceki:+$onceki..}HEAD | head -20)"

bilgi "tag atılıyor: $SURUM"
git tag -a "$SURUM" -m "$SURUM

$notlar"
git push --quiet origin "$DAL"
git push --quiet origin "$SURUM"
tamam "GitHub: $SURUM push edildi — CI release'i yayınlıyor"

if command -v gh >/dev/null 2>&1; then
    printf '   %s\n' "https://github.com/muslu/cem/actions"
fi

# ── Marketplace ──────────────────────────────────────────────────────────────
if [[ $MARKET -eq 1 ]]; then
    bilgi "eklenti imzalanıp Marketplace'e gönderiliyor"
    ./plugin/intellij/yayinla.sh --surum "$SURUM" --yayinla
    tamam "Marketplace: $SURUM gönderildi — https://plugins.jetbrains.com/plugin/34196"
else
    bilgi "eklenti yalnızca imzalanıyor (Marketplace'e gönderilmiyor)"
    ./plugin/intellij/yayinla.sh --surum "$SURUM" >/dev/null || \
        uyari "imzalama atlandı (parola dosyası yok)"
fi

echo
tamam "$SURUM yayınlandı"
