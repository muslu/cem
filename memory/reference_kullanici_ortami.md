---
name: reference-kullanici-ortami
description: Kullanıcının cem'i çalıştırdığı ortamın ölçülmüş kısıtları
metadata:
  type: reference
---

> Not: cem'in kendi davranışı ve varsayılanları CLAUDE.md'de — burada yalnızca
> ortama ait, kodda görünmeyen gerçekler.

2026-09-09'da ölçülen ortam kısıtları:

- **ChatGPT planı ücretsiz** → codex `gpt-5-mini` ve `gpt-5.5` modellerini
  reddediyor ("does not exist or you do not have access to it"). Çalışan
  model: `gpt-5.6-terra`. Model önerisi yaparken bunu hatırla.
- **Claude Code kurulumunda claude-mem hook'ları var** → her `claude -p`
  çağrısında SessionStart/SessionEnd çalışıyor ve sabit ~60 saniye ekliyor
  (ölçüldü: 70s vs 7s). cem'in hızlı modu bu yüzden varsayılan açık.
- **Kullanıcı GoLand/PyCharm kullanıyor**; IntelliJ eklentisi Marketplace'te
  değil, `updatePlugins.xml` deposu üzerinden güncelleniyor.
- **Terminalde Türkçe karakter kullanmadan yazıyor** ("olustur", "duzelt") —
  metin eşleştiren her yerde ASCII varyantı da bulunmalı.

İlgili: [[project-cem-hedef]]
