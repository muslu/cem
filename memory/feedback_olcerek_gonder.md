---
name: feedback-olcerek-gonder
description: Performans için eklenen her bayrak, gönderilmeden önce gerçek çağrıyla doğrulanmalı
metadata:
  type: feedback
---

> Not: `--bare` yasağının teknik gerekçesi CLAUDE.md'de — burada tekrar yok.

Hız kazandırdığını düşündüğün bir bayrağı, gerçek bir çağrıyla çalıştırıp
sonucu görmeden sürüme koyma.

**Why:** 2026-09-09'da `claude --bare` writer'ı hızlandırsın diye eklendi ve
20260909.07 ile yayınlandı. Bayrak OAuth oturumunu da atlıyor: kullanıcının
her writer çağrısı "Not logged in · Please run /login" ile düştü. Hata
kullanıcının makinesinde ortaya çıktı, tek satırlık bir denemeyle önceden
görülebilirdi.

**How to apply:** Aynı disiplin çıktı filtreleri için de geçerli — içeriği
değiştirebilen bir kural (tekrar satır ayıklama, boş satır tekilleştirme)
üretilen kodu bozabiliyor. Değişikliği ölç, öncesi/sonrası rakamı ve
"kod hâlâ derleniyor mu" kontrolünü birlikte raporla.

İlgili: [[project-cem-hedef]]
