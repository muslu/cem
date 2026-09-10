# Değişiklik Günlüğü

**cem** ve IntelliJ Platform eklentisindeki önemli değişiklikler.
Sürümler `YYYYMMDD.MINOR` (takvim sürümlemesi) biçimindedir; doğruluk kaynağı
[github.com/muslu/cem](https://github.com/muslu/cem/releases) üzerindeki tag'lerdir.
İngilizce sürüm: [CHANGELOG.md](CHANGELOG.md).

## 20260910.02

- **Eklenti:** `-p` / `-w` istemi modal dialog yerine cem araç penceresinin
  altındaki kutuya yazılıyor. Dört aksiyon yolu da (düşün / yaz / ikili / sor)
  bu kutudan geçiyor.

## 20260910.01

- İstek bir soruysa yazan rol artık dosya oluşturmuyor. "lua'da hello world nasıl
  yazılır?" sorusu çalışma dizinine `hello.lua` bırakıyor ve faturaya ikinci bir
  AI çağrısı yazıyordu.

## 20260909.20

- İkili modda spinner, yazan rolün cevabının ilk satırlarının üzerine biniyordu.

## 20260909.19

- **Eklenti:** editörde seçim yokken açık dosyanın tamamı görev olarak
  gönderilmiyor; imleç giriş kutusuna gidiyor.

## 20260909.18

- Yazan rol de önbelleğe alınabiliyor: önbellek isabetinde üretilen dosyalar geri
  yazılıyor, arada değişmiş bir dosya asla ezilmiyor — çakışma raporlanıyor.

## 20260909.09

- Hızlı mod (varsayılan açık): AI CLI'nin kullanıcı ayarlarını — hook, izin
  kuralları, MCP — atlar ve düzenlemeleri otomatik onaylar. Aynı görevde
  claude 2.1.266 ile ölçüldü: **124s → 8s**.
- Yazan aşamasında canlı spinner, aşama başına süre, dile özgü diyagramlar.

## Öncesi

`20260909.09` öncesi için
[sürüm listesine](https://github.com/muslu/cem/releases) bakın.
