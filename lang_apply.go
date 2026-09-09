package main

// Bu dosya, kullanıcıya görünen KOMUT metinlerinin (cobra Short/Long)
// Türkçe karşılıklarını tutar. Ayrı durmasının sebebi teknik: cobra komutları
// paket-seviyesi var olarak kuruluyor ve o an config henüz okunmadığı için
// L() doğru dili bilemiyor. applyLang() Execute'dan hemen önce çalışır.
//
// Çalışma-zamanı mesajları (hata, ipucu, tablo) için bu dosyaya bir şey
// eklemeye gerek yok — onlar fonksiyon içinde L(...) ile yazılır.

// preloadLang — config'teki dili aktif hâle getirir. Hata durumunda ortam
// tahmini (CEM_LANG > LANG) geçerli kalır.
func preloadLang() {
	cfg, err := loadGlobalConfig()
	if err != nil || cfg == nil {
		return
	}
	setLang(cfg.Lang)
}

// applyLang — dil Türkçe ise komut açıklamalarını Türkçeye çevirir.
// İngilizce, kodda yazılı hâliyle zaten varsayılan.
func applyLang() {
	if Lang() != LangTR {
		return
	}

	rootCmd.Short = "⚡ Compose · Execute · Multiplex — tek komut, çok AI"

	rolesCmd.Short = "Rolleri göster / değiştir (thinker / writer)"
	rolesCmd.Long = `  cem roles                    → mevcut rolleri göster
  cem roles claude agy         → global olarak ayarla
  cem roles claude             → sadece thinker
  cem roles --here claude agy  → sadece bu proje (.cem.yaml)`

	setupCmd.Short = "Kurulum sihirbazını çalıştır"
	initCmd.Short = "Bu proje için .cem.yaml oluştur"
	initCmd.Long = `  cem init                 → soru-cevap sihirbazı
  cem init claude agy      → doğrudan oluştur`
	statusCmd.Short = "Yapılandırma ve araç durumunu göster"

	effortCmd.Short = "Düşünme seviyesini göster / değiştir"
	effortCmd.Long = `  cem effort                    → mevcut seviyeleri göster
  cem effort gpt high           → global ayarla
  cem effort claude xhigh       → global ayarla
  cem effort --here gpt xhigh   → sadece bu proje (.cem.yaml)
  cem effort gpt default        → seçimi kaldır (CLI kendi seçer)`

	modelCmd.Short = "Araç başına modeli göster / değiştir"
	modelCmd.Long = `  cem model                       → mevcut modelleri göster
  cem model gpt                   → o aracın bilinen modellerini listele
  cem model gpt gpt-5.6-terra     → global ayarla
  cem model --here claude sonnet  → sadece bu proje (.cem.yaml)
  cem model gpt default           → seçimi kaldır (CLI kendi seçer)`

	langCmd.Short = "Arayüz dilini göster / değiştir"
	langCmd.Long = `  cem lang        → mevcut dili göster
  cem lang tr     → Türkçe
  cem lang en     → English

  Öncelik: CEM_LANG ortam değişkeni > config > sistem dili (LANG)`

	doctorCmd.Short = "Tanılama raporu (araçlar, config, PATH, auth)"
	historyCmd.Short = "Çalıştırma geçmişini göster"
	updateCmd.Short = "cem'i son sürüme güncelle"
	uninstallCmd.Short = "cem / cemi / cemir ikililerini kaldır"
	authCmd.Short = "Bir AI aracında oturum aç"
	keysCmd.Short = "API anahtarlarını yönet — rate limit'te otomatik döner"
	keysAddCmd.Short = "Bir provider için yeni API anahtarı ekle (etkileşimli)"
	keysListCmd.Short = "Kayıtlı anahtarları listele (maskelenmiş)"
	keysRemoveCmd.Short = "Belirli bir anahtarı sil (listedeki sıra numarası)"
	slashCmd.Short = "/cem slash komutunu desteklenen AI CLI'larına kur"

	// Bayrak açıklamaları
	if f := rootCmd.Flags().Lookup("write"); f != nil {
		f.Usage = "Writer AI kullan"
	}
	if f := rootCmd.Flags().Lookup("pair"); f != nil {
		f.Usage = "Pair: önce düşünen, sonra yazan"
	}
	if f := rootCmd.Flags().Lookup("file"); f != nil {
		f.Usage = "Dosya içeriğini gönder"
	}
	if f := rootCmd.PersistentFlags().Lookup("raw"); f != nil {
		f.Usage = "AI CLI çıktısını filtresiz göster (banner, ara adımlar, diff'ler)"
	}
	if f := rolesCmd.Flags().Lookup("here"); f != nil {
		f.Usage = "sadece bu proje için (.cem.yaml)"
	}
	if f := effortCmd.Flags().Lookup("here"); f != nil {
		f.Usage = "sadece bu proje için (.cem.yaml)"
	}
	if f := modelCmd.Flags().Lookup("here"); f != nil {
		f.Usage = "sadece bu proje için (.cem.yaml)"
	}
}
