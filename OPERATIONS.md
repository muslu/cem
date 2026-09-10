##############################################################################
# /etc/nginx/  →  cem.pw Operations Runbook
##############################################################################
# Turkish version: OPERATIONS.tr.md

## TEST & RELOAD
nginx -t                                    # Syntax check
nginx -T                                    # Full config dump
systemctl reload nginx                      # Reload config (zero downtime)
systemctl restart nginx                     # Full restart

## LOG TAILING
tail -f /var/log/nginx/cem.pw.access.log    # Live traffic
tail -f /var/log/nginx/cem.pw.error.log     # Errors
tail -f /var/log/nginx/cem.pw.downloads.log # Install-script downloads

# Top 20 client IPs over the recent log window:
awk '{print $1}' /var/log/nginx/cem.pw.access.log | sort | uniq -c | sort -rn | head -20

# IPs hitting 429 (rate-limited):
grep ' 429 ' /var/log/nginx/cem.pw.access.log | awk '{print $1}' | sort | uniq -c | sort -rn

# Count of 444 (bad-bot drop) events:
grep '" 444' /var/log/nginx/cem.pw.access.log | wc -l

## FAIL2BAN
fail2ban-client status                              # All jails
fail2ban-client status nginx-limit-req              # Specific jail
fail2ban-client set nginx-limit-req unbanip 1.2.3.4 # Unban an IP
fail2ban-client banned                              # All currently banned IPs
iptables -L -n | grep DROP                          # Kernel-level drops

## TLS / CERTIFICATE
certbot certificates                        # List certs and their expiry
certbot renew --dry-run                     # Test renewal flow
openssl s_client -connect cem.pw:443 -tls1_3 </dev/null \
  | openssl x509 -noout -dates              # Show valid_from / valid_to

## SECURITY HEADERS CHECK
curl -sI https://cem.pw/health | grep -E '^(Strict-Transport-Security|Content-Security-Policy|X-Frame-Options|Permissions-Policy|Server):'

## RATE-LIMIT TEST
# Trigger the install proxy 10 times in a row; the burst should kick in at >5.
for i in $(seq 1 10); do
  curl -s -o /dev/null -w '%{http_code} ' https://cem.pw/r/cem-linux-amd64
done; echo

## RELEASE FLOW (canonical: GitHub)
# 1. Tag locally
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
# 2. GitHub Actions builds 7 platforms × 3 binaries + SHA256SUMS
# 3. nginx /r/* now serves the latest assets via GitHub Releases proxy
# 4. install.sh / install.ps1 download directly from
#    github.com/muslu/cem/releases/latest/download/

## ROLLBACK
# If a release is bad, delete the GitHub Release (web UI or:)
gh release delete v1.0.0 --yes
# Re-point /r/* to a prior tag temporarily by editing
# nginx/sites-available/cem.pw.conf → proxy_pass with explicit tag in URL.

## SMOKE TEST AFTER DEPLOY
./build/cem doctor                          # local sanity
./build/cem --version                       # confirms LDFLAGS injection
curl -fsSL https://cem.pw/r/cem-linux-amd64 -o /tmp/cem-smoke && \
  file /tmp/cem-smoke && /tmp/cem-smoke --version

## MARKETPLACE WIDGET (cem.pw — NOT GitHub)
# GitHub sanitises <iframe> and <script> out of README markdown, so the
# JetBrains embeddable widgets only work on the cem.pw pages. The README uses
# shields.io badges instead (plugin id 34196).
#
# Plugin card (384×319) and one-click install button (245×48):
#   <iframe width="384px" height="319px"
#           src="https://plugins.jetbrains.com/embeddable/card/34196"></iframe>
#   <iframe width="245px" height="48px"
#           src="https://plugins.jetbrains.com/embeddable/install/34196"></iframe>
#
# Script variant — needs a target element on the page:
#   <div id="cem-install"></div>
#   <script src="https://plugins.jetbrains.com/assets/scripts/mp-widget.js"></script>
#   <script>MarketplaceWidget.setupMarketplaceWidget('install', 34196, '#cem-install');</script>
#
# CSP: both variants load from plugins.jetbrains.com, so that host must be
# allowed in frame-src (iframe) or script-src (script variant). Adding the
# script variant without touching CSP fails silently — the button never renders.
