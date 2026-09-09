package main

import "testing"

func TestSemverLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v0.1.29", "v0.1.30", true},
		{"v0.1.30", "v0.1.29", false},
		{"v0.1.31", "v0.1.33", true},
		{"v0.1.33", "v0.1.31", false}, // bu user'ın sahip olduğu durumdu
		{"v0.1.33", "v0.1.33", false}, // eşitlik → less değil
		{"0.1.33", "v0.1.33", false},  // v prefix opsiyonel
		{"v0.1.9", "v0.1.10", true},   // string sıralamayla yanlış olurdu; sayı sıralamayla doğru
		{"v0.2.0", "v0.1.99", false},
		{"v1.0.0", "v0.99.0", false},
		{"dev", "v0.1.30", true}, // dev → 0.0.0 → her tag'den küçük
	}
	for _, c := range cases {
		got := semverLess(c.a, c.b)
		if got != c.want {
			t.Errorf("semverLess(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// "20260909.19-dirty" sürümü .18'den küçük sanılıyordu: git describe eki
// Atoi'yi düşürüp bileşeni 0 yapıyordu, her yerel build sahte bir güncelleme
// bildirimi basıyordu.
func TestSemverGitDescribeEkiniYoksayar(t *testing.T) {
	if semverLess("20260909.19-dirty", "20260909.18") {
		t.Fatal("dirty ek yüzünden .19 < .18 sanıldı")
	}
	if !semverLess("20260909.18", "20260909.19-dirty") {
		t.Fatal(".18 < .19-dirty olmalı")
	}
	if semverLess("20260909.19-2-gabc123", "20260909.19") {
		t.Fatal("commit sayısı eki sürümü küçültmemeli")
	}
}
