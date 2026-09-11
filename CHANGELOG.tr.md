# Değişiklik Günlüğü

**cem** ve IntelliJ Platform eklentisindeki önemli değişiklikler.
Sürümler `YYYYMMDD.MINOR` (takvim sürümlemesi) biçimindedir; doğruluk kaynağı
[github.com/muslu/cem](https://github.com/muslu/cem/releases) üzerindeki tag'lerdir.
İngilizce sürüm: [CHANGELOG.md](CHANGELOG.md).

## 20260911.02

- **Eklenti: Terminal sekmesinde Tab tamamlama.** `python3 fe⇥` yazınca kutuya
  sekme karakteri giriyordu — sekme her komutu yeni bir `sh -c` ile
  çalıştırdığından kabuğun kendi tamamlaması hiç devreye girmiyordu. Tab artık
  kabuk gibi tamamlıyor: komut sözcüğü `PATH`'teki çalıştırılabilirlere, diğer
  sözcükler proje kökündeki dosya ve dizinlere göre; tek aday bitirilir (dizine
  `/`, dosyaya boşluk), çok aday ortak öneke ilerler; eklenecek bir şey
  kalmayınca adaylar kutunun üstünde seçilebilir bir listede açılır — ↑/↓ +
  Enter ya da tıklama seçileni kutuya yazar, yazmaya devam edince liste
  daralır, Esc kapatır. (İlk sürüm adayları çıktı alanına basıyordu:
  `python⇥` her basışta 30 adı bir daha döküyor, kutu ise değişmiyordu. İkinci
  sürüm Tab'ı yalnız Swing `InputMap`'ine bağlıyordu ve IDE'de oraya hiç
  ulaşmıyordu: tuş olayı önce `IdeKeyEventDispatcher`'dan geçer, Tab orada odak
  gezinmesi olarak yutuluyordu — Tab artık kutuya `CustomShortcutSet` ile IDE
  action'ı olarak kayıtlı; dispatcher keymap'ten önce ona bakar.)

- **agy headless modda artık kör çalışmıyor.** Antigravity `-p` altında araç
  izni soramıyor: ihtiyaç duyduğu ilk dosya okuma ya da komut otomatik
  reddediliyor ve exit 0 + boş stdout ile dönüyor. cem bu boş cevabı plan
  sanıp yazana veriyor, yazan da arkasında plan olmadan kendi başına dosya
  üretiyordu (sahada görüldü 2026-09-11 — düşünenin tek çıktısı "no output
  produced … auto-denied" idi). agy'nin hızlı modu (varsayılan açık) artık
  `--dangerously-skip-permissions` geçiyor — print modunda araç kullanımını
  açan tek bayrak bu (`--mode accept-edits` ölçüldü, açmıyor) — ve `cem fast`
  bunu söylüyor. Red yine olursa (hızlı mod kapalı ya da eski agy) cem bunu
  adıyla bildiriyor, `cem fast agy on`'u gösteriyor ve ikinci çağrıyı
  ödemek yerine yazanı atlıyor.
- Boş cevap döndüren her düşünen artık pair koşusunu uyarıyla durduruyor,
  yazan başlatılmıyor.
- **Geçen süre her adımda, adımın rengiyle gösteriliyor.** `cem -w` hiç süre
  basmıyordu; pair'de düşünme/yazma/toplam satırları uzun cevabın altında tek
  gri blok hâlindeydi. Her adım artık rolün rengiyle (düşünen mavi, yazan
  yeşil) kendi satırıyla bitiyor; pair toplamı kalın, parçalar renkli.
- **Saklı cevap koşudan ÖNCE teklif ediliyor, sonra açıklanmıyor.**
  Önbellekte cevap varken cem onu basıp altına "önbellekten · --no-cache"
  notu düşüyordu — kullanıcı geç fark edip komutu yeniden yazıyordu.
  Terminalde artık önce soruyor (`Kullanılsın mı? [E/h]`; Enter bedava cevabı
  kullanır, `h` taze üretir ve onu saklar). Terminal yoksa (eklenti, pipe, CI)
  soru sorulmaz, saklı cevap eskisi gibi kullanılır.
- `cem … </dev/null` artık terminal sayılmıyor: `/dev/null` karakter aygıtı
  olduğu için TTY kontrolü "etkileşimli" diyor, soru soruyor ve EOF'u Enter
  okuyordu. Tüm sorular artık tek stdin okuyucusunu paylaşıyor — her biri
  kendi tamponlu okuyucusunu açıyor, ilki sonrakine ait satırları yutuyordu.

## 20260911.01

- **Eklenti:** girdi kutusundaki ↑/↓ geçmişi artık kalıcı ve ortak. Tüm sohbet
  kutuları (Interactive ve her çalıştırma sekmesinin devam kutusu) tek listeyi,
  tüm Terminal kutuları başka bir listeyi gezer; ikisi de IDE kapanınca durur.
  Geçmiş eskiden kutunun kendi içindeydi: yeni açılan her sekme boş başlıyor,
  dünkü prompt geri çağrılamıyordu. cem'in kendi `~/.cem/history.log` dosyası
  bilerek kullanılmıyor — girdiyi 80 karakterde kırpar ve geri yüklenen prompt
  sessizce yarım gider.
- **Eklenti:** ↑/↓ çok satırlı bir girdide artık takılı kalmıyor. Eski kural
  "metinde satır sonu varsa imleci oynat" idi: ↑ ile çağrılan üç satırlık
  prompt satır sonu içerdiğinden sonraki ↑/↓ imlece gidiyor, kullanıcı o
  girdiden çıkamıyordu. Şimdi ↑ ilk satırdan, ↓ son satırdan geçmişi gezer
  (aradaki satırlarda imleç hareket eder); geçmişten gelen ve henüz
  düzenlenmemiş girdide her zaman geçmiş gezilir.

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
