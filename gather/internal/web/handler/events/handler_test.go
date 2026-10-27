package events

import (
	"testing"
	"time"
)

func TestRules(t *testing.T) {
	for v, want := range map[string]bool{"1": true, "40": true, "0": false, "-3": false, "ten": false, "": false} {
		if got := positiveInt(v) == ""; got != want {
			t.Errorf("positiveInt(%q) ok=%v, want %v", v, got, want)
		}
	}
	for v, want := range map[string]bool{"": true, "5": true, "7.50": true, "-2": false, "free": false} {
		if got := price(v) == ""; got != want {
			t.Errorf("price(%q) ok=%v, want %v", v, got, want)
		}
	}
	future := time.Now().Add(48 * time.Hour).Format("2006-01-02T15:04")
	for v, want := range map[string]bool{future: true, "2020-01-01T10:00": false, "soon": false} {
		if got := inTheFuture(v) == ""; got != want {
			t.Errorf("inTheFuture(%q) ok=%v, want %v", v, got, want)
		}
	}
}
