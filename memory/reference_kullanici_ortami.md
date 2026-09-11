---
name: reference-kullanici-ortami
description: Kullanıcının cem'i çalıştırdığı ortamın ölçülmüş kısıtları
metadata:
  type: reference
---

> Not: cem'in kendi davranışı ve varsayılanları CLAUDE.md'de — burada yalnızca
> ortama ait, kodda görünmeyen gerçekler.

Ölçülen ortam kısıtları (2026-09-09 → 2026-09-11):

- **ChatGPT planı ücretsiz** → codex `gpt-5-mini` ve `gpt-5.5` modellerini
  reddediyor ("does not exist or you do not have access to it"). Çalışan
  model: `gpt-5.6-terra`. Model önerisi yaparken bunu hatırla.
- **Claude Code kurulumunda claude-mem hook'ları var** → her `claude -p`
  çağrısında SessionStart/SessionEnd çalışıyor ve sabit ~60 saniye ekliyor
  (ölçüldü: 70s vs 7s). cem'in hızlı modu bu yüzden varsayılan açık.
- **Kullanıcı GoLand/PyCharm kullanıyor**; IntelliJ eklentisi 2026-09-10'dan
  beri Marketplace'te (bkz. [[reference-marketplace-yayin]]). Eklenti cem'i
  `ProcessBuilder` ile PTY'siz çalıştırır — cem'in terminal soruları orada
  görünmez, stdin'e yazılan hiçbir şey okunmaz.
- **agy 1.2.1 headless (`-p`) modda araç izni SORAMAZ** (ölçüldü 2026-09-11):
  `read_file`/`command` isteği "auto-denied", exit 0, stdout boş, mesaj yalnız
  stderr'de. Hangi aracın istendiği modele bağlı (dizin içi read_file bazen
  geçiyor), o yüzden red deterministik tetiklenemiyor. `--mode accept-edits`
  açmıyor; `--dangerously-skip-permissions` açıyor (cem hızlı modu bunu
  geçiyor). agy ayar dosyası `~/.gemini/antigravity-cli/settings.json`
  (`toolPermission: request-review|always-proceed`, `permissions.allow:
  [read_file(*), command(*)]`) — cem oraya yazmıyor. agy `--model` ve
  `--effort low|medium|high` bayraklarına da sahip; cem'de henüz bağlı değil.
- **Claude Code sınıflandırıcısı `--dangerously-skip-permissions` içeren
  komutu reddediyor** ("Create Unsafe Agents") — agy'yi o bayrakla doğrudan
  test edemezsin; cem üzerinden çalıştır ya da kullanıcıdan iste.
- **Terminalde Türkçe karakter kullanmadan yazıyor** ("olustur", "duzelt") —
  metin eşleştiren her yerde ASCII varyantı da bulunmalı.

İlgili: [[project-cem-hedef]]
