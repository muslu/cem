# ⚡ CEM

```
   ██████╗███████╗███╗   ███╗
  ██╔════╝██╔════╝████╗ ████║
  ██║     █████╗  ██╔████╔██║
  ██║     ██╔══╝  ██║╚██╔╝██║
  ╚██████╗███████╗██║  ╚═╝ ██║
   ╚═════╝╚══════╝╚═╝      ╚═╝
```

**Tek komut, çok AI.** Bir AI düşünür, başka bir AI kodu yazar.

Düşünmeyi güçlü modele, yazmayı ucuz modele bırakırsın — kararlar iyileşir,
her satır kod için pahalı model çalıştırmazsın.

[![JetBrains Marketplace](https://img.shields.io/badge/JetBrains%20Marketplace-cem-000?logo=jetbrains)](https://plugins.jetbrains.com/plugin/34196)
[![Marketplace downloads](https://img.shields.io/jetbrains/plugin/d/34196?label=plugin%20downloads)](https://plugins.jetbrains.com/plugin/34196)
[![Release](https://img.shields.io/github/v/release/muslu/cem?sort=date&label=release)](https://github.com/muslu/cem/releases/latest)

![cem pair modu: düşünen planlar, yazan kodlar](docs/img/cem-pair.tr.svg)


---

## Kur

```sh
curl -fsSL cem.pw/install | sh        # macOS / Linux / WSL
```
```powershell
irm cem.pw/install | iex              # Windows (PowerShell)
```

JetBrains IDE eklentisi: **Settings → Plugins → Marketplace → `cem` ara**
([sayfa](https://plugins.jetbrains.com/plugin/34196)). VS Code ve diğer
editörler: [README_DETAILS.tr.md](README_DETAILS.tr.md).

### Lua HTTP Sunucusu (İsteğe Bağlı)

Dahil edilen Lua HTTP sunucusunu (`server.lua`) çalıştırmak istiyorsan:

```sh
# LuaSocket kur
luarocks install luasocket        # veya
apt-get install lua-socket        # Debian/Ubuntu
brew install lua-socket           # macOS
dnf install lua-socket            # Fedora

# Sunucuyu başlat
lua5.4 server.lua
```

**Not:** LuaSocket, çalıştırdığın Lua sürümü için kurulu olmalı.

## Kullan

```sh
cem "B-tree nedir?"                  # düşünene sor
cem -w "Go'da quicksort yaz"         # yazana yaptır
cem -p "client.go'ya retry ekle"     # pair: düşünen planlar, yazan kodlar
```

İlk çalıştırmada kısa bir sihirbaz açılır: dilini seç, hangi AI düşünsün,
hangisi yazsın. Hepsi bu.

```
  🧠 DÜŞÜNEN · gpt  gpt-5.6-terra · xhigh
  - Dosya: retry.go
  - Fonksiyon: func withRetry(fn func() error, n int) error
  - Kenar durumlar: context iptali, backoff üst sınırı
  ⏱ düşünme 8.9s
  ────────────────────────────────────────────────────
  ✍️  YAZAN · claude  sonnet · low
  retry.go oluşturuldu — withRetry n kez, sınırlı backoff ile yeniden dener.
  ⏱ yazma 24.1s
  ⏱ toplam 33.0s   (düşünme 8.9s + yazma 24.1s)
```

---

## Daha fazlası

- **[README_DETAILS.tr.md](README_DETAILS.tr.md)** — tüm komutlar, IDE
  eklentileri, API key'ler, modeller, düşünme seviyesi, sorun giderme
- [ADVANCED.tr.md](ADVANCED.tr.md) — proje config'leri, kaynaktan derleme
- English: [README.md](README.md)
- Site: [cem.pw](https://cem.pw)

MIT — [LICENSE](LICENSE).
