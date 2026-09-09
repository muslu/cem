---
name: project-cem-hedef
description: cem'in varlık sebebi — pahalı model karar verir, ucuz model yazar; ölçüt token israfı
metadata:
  type: project
---

> Not: mimari, kod kuralları, çalışma-zamanı tuzakları ve varsayılanların
> gerekçeleri CLAUDE.md'de tutuluyor — burada tekrar yok.

cem'de her tasarım kararı **token israfını önleme** ölçütüyle veriliyor:
düşünen rol pahalı/derin model, yazan rol ucuz/hızlı model; düşünen plan
çıkarır, yazan yalnızca uygular.

**Why:** Kullanıcının bu araçtan beklentisi hız değil, aynı işi iki kez
faturalamamak. 2026-09-09'da sahada görüldü ki talimatsız bırakıldığında iki
model de görevi baştan sona çözüyor ve pahalı olan da kodu yazıyor.

**How to apply:** Yeni bir varsayılan önerirken önce "bu neyi ucuzlatıyor"
sorusunu yanıtla. Kullanıcı deneyimli olmayabilir — kurulum hiç
dokunulmadan da israfsız olmalı. Ölçüm olmadan performans iddiası yazma;
CLAUDE.md'deki rakamlar (124s→8s, 1m38s→1m06s) gerçek koşulardan.

İlgili: [[feedback-olcerek-gonder]], [[reference-kullanici-ortami]]
