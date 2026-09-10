# Değişiklik Günlüğü

**cem** ve IntelliJ Platform eklentisindeki önemli değişiklikler.
Sürümler `YYYYMMDD.MINOR` (takvim sürümlemesi) biçimindedir; doğruluk kaynağı
[github.com/muslu/cem](https://github.com/muslu/cem/releases) üzerindeki tag'lerdir.
İngilizce sürüm: [CHANGELOG.md](CHANGELOG.md).

## 20260910.06

- `cem uninstall`'a `--yes`, `--config`, `--plugin` ve `--all` eklendi; IDE
  eklentilerini de bulup siliyor. JetBrains her ürün ve sürüm için ayrı kopya
  tutuyor ve geride kalan eklenti her IDE açılışında yüklenip cem'in
  bulunmadığını bildiriyordu. Terminal yoksa onaysız silmiyor.

## 20260910.05

- Kurulum terminalsiz yapılabiliyor: `cem setup --thinker X --writer Y`
  (ayrıca `--model-*`, `--effort-*`, `--endpoint-*`, `--lang`) ve
  `cem status --json`. Kurulumu zorunlu yapmak, eklentiyi ve CI'ı
  yapılandırma yolsuz bırakmıştı.
- **Eklenti:** Settings → Tools → cem artık kurulum sayfası. `cem status
  --json` okuyor, Apply `cem setup` çalıştırıyor; seçimleri cem doğruluyor ve
  `~/.cem/config.yaml`'i yazan tek yer cem kalıyor. HTTP araçlarda sunucu
  adresi alanı var; kurulum eksikken düşen çalıştırma bu sayfayı açmayı
  öneriyor.

## 20260910.04

- Yerel ve kendi sunucundaki modeller: `ollama`, `lmstudio` ve `unsloth` artık
  araç. cem onlarla HTTP konuşuyor (OpenAI uyumlu `/v1/chat/completions` ve
  ollama `/api/chat`) ve cevabı akıtıyor. Adresi `cem endpoint <araç>
  <ip:port>` veriyor; model adı sunucuya soruluyor, tahmin edilmiyor.
- Kurulum zorunlu: cem yapılandırılmadan çalışmayı reddediyor — aksi halde
  kullanıcının seçmediği araca istek gidiyordu.
- OAuth kodu artık araca ulaşıyor. Prompt'u argümanla alan araçlarda alt
  sürecin stdin'i yoktu, interaktif oturum açma istemi okuyacak bir şey
  bulamıyordu: Windows'ta yapıştırılan kod PowerShell'e gidiyor ve
  `4/0ATs…` komut sanılıyordu.
- **Eklenti:** çalıştırma sekmesinde sohbete devam (önceki turlar kırpılmış
  bağlam olarak gidiyor), proje kökünde komut çalıştıran Terminal sekmesi ve
  `＋` ile yeni terminal sekmeleri.

## 20260910.03

- Eklenti JetBrains Marketplace'te listelendi (id `dev.cempw.cem`). İlk
  yükleme, plugin ID'sinin `intellij` kelimesini içermesi yüzünden
  reddedilmişti.
- Marketplace doğrulayıcısının iki uyumluluk uyarısı düzeltildi:
  `SimpleListCellRenderer.create(String, Function)` kaldırılmak üzereydi,
  `doWhenFocusSettlesDown(Runnable)` deprecated'di.
- Eklenti 21 yerine Java 17 için derleniyor: `sinceBuild=233` IDE'leri JBR 17
  ile çalışıyor, 21 bayt kodu orada hiç yüklenemiyordu.

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
