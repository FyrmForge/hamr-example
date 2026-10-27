package service

import "testing"

func TestPrice(t *testing.T) {
	for in, want := range map[string]int{"": 0, "0": 0, "5": 500, "7.5": 750, "7.50": 750, "19.99": 1999} {
		if got, err := ParsePrice(in); err != nil || got != want {
			t.Errorf("ParsePrice(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"-1", "five", "1.999", "20000"} {
		if _, err := ParsePrice(in); err == nil {
			t.Errorf("ParsePrice(%q) accepted", in)
		}
	}
	if FormatPrice(500) != "€5" || FormatPrice(750) != "€7.50" || FormatPrice(1999) != "€19.99" {
		t.Error("FormatPrice")
	}
}
