##############################################################################
# /etc/nginx/  →  cem.pw Operasyon Rehberi (Türkçe)
##############################################################################
# English version: OPERATIONS.md
# Canonical repo: https://github.com/muslu/cem.git

## TEST & RELOAD
nginx -t                                    # Syntax test
nginx -T                                    # Tam config dump
systemctl reload nginx                      # Config reload (downtime yok)
systemctl restart nginx                     # Tam restart

## LOG TAKİBİ
tail -f /var/log/nginx/cem.pw.access.log   # Canlı trafik
tail -f /var/log/nginx/cem.pw.error.log    # Hatalar
tail -f /var/log/nginx/cem.pw.downloads.log  # Install script indirmeleri

# Son 1 saatte en çok istek atan IP'ler:
awk '{print $1}' /var/log/nginx/cem.pw.access.log | sort | uniq -c | sort -rn | head -20

# 429 (rate limit) alan IP'ler:
grep ' 429 ' /var/log/nginx/cem.pw.access.log | awk '{print $1}' | sort | uniq -c | sort -rn

# 444 (bad bot) olayları:
grep '" 444' /var/log/nginx/cem.pw.access.log | wc -l

## FAIL2BAN
fail2ban-client status                      # Tüm jail durumu
fail2ban-client status nginx-limit-req      # Belirli jail
fail2ban-client set nginx-limit-req unbanip 1.2.3.4   # IP ban kaldır
fail2ban-client banned                      # Tüm banlı IP'ler
iptables -L -n | grep DROP                  # Kernel seviyesi bloklar

## SSL
certbot renew --dry-run                     # Yenileme testi
certbot certificates                        # Sertifika durumu
openssl s_client -connect cem.pw:443 -tls1_3 </dev/null 2>&1 | grep "Protocol"
# SSLLabs tam test: https://www.ssllabs.com/ssltest/analyze.html?d=cem.pw

## INSTALL SCRIPT GÜNCELLEME
# 1. cem projesinde install.sh / install.ps1 güncelle
# 2. Sunucuya kopyala:
scp cem/install.sh   user@cem.pw:/var/www/cem.pw/scripts/
scp cem/install.ps1  user@cem.pw:/var/www/cem.pw/scripts/
# Cache sıfırla gerekmez (Cache-Control: no-cache, max-age=300)

## MANUEL IP BAN
iptables -A INPUT -s 1.2.3.4 -j DROP
# Kalıcı için:
echo "1.2.3.4" >> /etc/nginx/blocked-ips.conf
# nginx.conf'a ekle: deny 1.2.3.4;

## HIZLI GÜVENLİK TARAMASI
# Açık portları kontrol et:
ss -tlnp
# Nginx process:
ps aux | grep nginx
# Bağlantı sayısı:
ss -s

## RATE LIMIT AYARLAMA
# nginx.conf'daki zone'ları düzenle:
# limit_req_zone $binary_remote_addr zone=download:20m rate=5r/m;
# Ardından: nginx -t && systemctl reload nginx

## PERFORMANS
# Bağlantı istatistikleri:
curl -s https://cem.pw/health
# Nginx durum (stub_status modülü gerekir):
# curl http://localhost/nginx_status

## MARKETPLACE WIDGET (cem.pw — GitHub'da ÇALIŞMAZ)
# GitHub, README markdown'ından <iframe> ve <script> etiketlerini temizliyor;
# JetBrains gömülebilir widget'ları yalnız cem.pw sayfalarında çalışır. README
# bunun yerine shields.io rozetleri kullanıyor (plugin id 34196).
#
# Eklenti kartı (384×319) ve tek tık kurulum butonu (245×48):
#   <iframe width="384px" height="319px"
#           src="https://plugins.jetbrains.com/embeddable/card/34196"></iframe>
#   <iframe width="245px" height="48px"
#           src="https://plugins.jetbrains.com/embeddable/install/34196"></iframe>
#
# Script sürümü — sayfada hedef bir element gerekiyor:
#   <div id="cem-install"></div>
#   <script src="https://plugins.jetbrains.com/assets/scripts/mp-widget.js"></script>
#   <script>MarketplaceWidget.setupMarketplaceWidget('install', 34196, '#cem-install');</script>
#
# CSP: iki sürüm de plugins.jetbrains.com'dan yükleniyor; bu host frame-src
# (iframe) veya script-src (script sürümü) içinde izinli olmalı. CSP'ye
# dokunmadan script sürümünü eklersen sessizce çalışmaz — buton hiç görünmez.
