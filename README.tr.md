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

![cem pair modu: düşünen planlar, yazan kodlar](docs/img/cem-pair.tr.svg)

---

## Kur

```sh
curl -fsSL cem.pw/install | sh        # macOS / Linux / WSL
```
```powershell
irm cem.pw/install | iex              # Windows (PowerShell)
```

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
