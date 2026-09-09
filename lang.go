package main

import (
	"os"
	"strings"
)

// Desteklenen diller. Yeni dil eklemek isteyen için: L()/Lf() imzası iki
// metin alıyor (tr, en). Üçüncü bir dil eklenecekse burası bir map'e
// dönüşmeli — o zamana kadar iki dil için en yalın çözüm bu.
const (
	LangTR = "tr"
	LangEN = "en"
)

// activeLang — süreç boyunca sabit. LoadConfig sırasında setLang ile
// belirlenir; belirlenmezse detectLang() devreye girer.
var activeLang = ""

// Lang — aktif dil kodu ("tr" | "en").
func Lang() string {
	if activeLang == "" {
		activeLang = detectLang()
	}
	return activeLang
}

// setLang — config'den okunan dili uygular. Geçersiz/boş değer yok sayılır.
func setLang(code string) {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case LangTR:
		activeLang = LangTR
	case LangEN:
		activeLang = LangEN
	}
}

// detectLang — config'de dil yoksa: CEM_LANG > LC_ALL/LC_MESSAGES/LANG > en.
// Ortam Türkçe ise Türkçe, aksi halde İngilizce; ilk kurulumda wizard zaten
// kullanıcıya soruyor.
func detectLang() string {
	if v := os.Getenv("CEM_LANG"); v != "" {
		switch strings.ToLower(v) {
		case LangTR, "tr_tr", "turkish", "türkçe":
			return LangTR
		case LangEN, "en_us", "en_gb", "english":
			return LangEN
		}
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := strings.ToLower(os.Getenv(key)); strings.HasPrefix(v, "tr") {
			return LangTR
		}
	}
	return LangEN
}

// L — dile göre metin seçer. Kaynak metinler kodun içinde yan yana durur,
// böylece bir mesaj değiştiğinde çevirisinin unutulması zorlaşır.
//
//	fmt.Println(L("✗ Dosya okunamadı", "✗ Cannot read file"))
func L(tr, en string) string {
	if Lang() == LangEN {
		return en
	}
	return tr
}
