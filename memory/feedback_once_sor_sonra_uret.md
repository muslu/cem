---
name: feedback-once-sor-sonra-uret
description: Kullanıcının kararını değiştirecek bilgi (önbellek, süre, maliyet) koşudan SONRA değil ÖNCE gösterilir
metadata:
  type: feedback
---

> Not: `askUseCache`, `isInteractiveStdin`/`term.IsTerminal` ve ortak
> `stdinReader` gerekçeleri CLAUDE.md "Runtime Gotchas"ta — burada tekrar yok.

Kullanıcı bir seçimi etkileyecek bilgiyi iş bittikten sonra öğrenmek
istemiyor. 2026-09-11: önbellekten gelen cevabın altına "♻ önbellekten ·
--no-cache" notu düşülüyordu; kullanıcı taze cevap istediğini ancak o zaman
fark edip komutu yeniden yazıyordu. Aynı gün: `cem -w`'de süre hiç
basılmıyor, pair'de üç süre tek gri satırdı.

**Why:** Kullanıcının sözleri: "--no-cache kullanımını daha önceden söylemek
için soru veya cevap zaten varsa no-cache kullanmak ister misin diye sorup
duruma göre üretim yapılmalı, şu an sonradan fark ediliyor." Sonradan
öğrenilen bilgi = bir çağrı daha = fatura.

**How to apply:** Koşudan önce bilinen ve kararı değiştirebilecek her şey
(saklı cevap var, hangi model/effort, kaç dosya yazılacak) TTY varsa **soru**
olarak öne alınır; Enter = ucuz/güvenli seçenek. TTY yoksa (eklenti
ProcessBuilder ile PTY'siz çalıştırıyor, pipe, CI) soru sorulmaz, eski
davranış korunur. Adım süreleri her modda, rolün rengiyle, adımın hemen
altında basılır — "gözükmeli" dendi, soluk gri sayılmıyor.

İlgili: [[project-cem-hedef]], [[feedback-olcerek-gonder]]
