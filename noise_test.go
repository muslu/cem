package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestFiltreKoduBozmaz — sahada yaşandı: filtre, arka arkaya gelen iki "}"
// satırından ikincisini atıyordu ve kullanıcıya DERLENMEYEN kod gidiyordu.
func TestFiltreKoduBozmaz(t *testing.T) {
	code := strings.Join([]string{
		"```go",
		"func quicksort(arr []int) {",
		"\tfor left <= right {",
		"\t\tif left <= right {",
		"\t\t\tleft++",
		"\t\t}",
		"\t}",
		"}",
		"```",
	}, "\n") + "\n"

	var out bytes.Buffer
	nf := newNoiseFilter("claude", &out, false)
	if _, err := nf.Write([]byte(code)); err != nil {
		t.Fatal(err)
	}
	if err := nf.Close(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	if strings.Count(got, "}") != strings.Count(code, "}") {
		t.Errorf("kapanış parantezi sayısı değişti: %d → %d\n%s",
			strings.Count(code, "}"), strings.Count(got, "}"), got)
	}
	for _, line := range strings.Split(strings.TrimRight(code, "\n"), "\n") {
		if !strings.Contains(got, line) {
			t.Errorf("satır kayboldu: %q", line)
		}
	}
}

// TestFiltreBannerAtar — gürültü elemesi çalışmaya devam etmeli.
func TestFiltreBannerAtar(t *testing.T) {
	in := strings.Join([]string{
		"OpenAI Codex v0.153.4",
		"--------",
		"workdir: /home/muslu",
		"model: gpt-5.6-terra",
		"--------",
		"user",
		"asıl cevap burada",
	}, "\n") + "\n"

	var out bytes.Buffer
	nf := newNoiseFilter("gpt", &out, false)
	nf.Write([]byte(in))
	nf.Close()

	got := strings.TrimSpace(out.String())
	if got != "asıl cevap burada" {
		t.Errorf("banner tam elenmedi:\n%q", got)
	}
}

// TestRawFiltrelemez — --raw her şeyi geçirmeli.
func TestRawFiltrelemez(t *testing.T) {
	in := "workdir: /x\nOpenAI Codex v1\ncevap\n"
	var out bytes.Buffer
	nf := newNoiseFilter("gpt", &out, true)
	nf.Write([]byte(in))
	nf.Close()
	if out.String() != in {
		t.Errorf("--raw çıktıyı değiştirdi:\n%q", out.String())
	}
}

// TestPythonBosSatirlariKorunur — üst düzey tanımlar arasındaki iki boş satır
// PEP8 gereği; teke indirmek üretilen kodu stil olarak bozuyordu.
func TestPythonBosSatirlariKorunur(t *testing.T) {
	code := "def a():\n    return 1\n\n\ndef b():\n    return 2\n"
	var out bytes.Buffer
	nf := newNoiseFilter("claude", &out, false)
	nf.Write([]byte(code))
	nf.Close()
	if got := out.String(); got != code {
		t.Errorf("boş satırlar değişti:\n%q\nbeklenen:\n%q", got, code)
	}
}
