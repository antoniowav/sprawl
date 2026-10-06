package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestParseKey(t *testing.T) {
	cases := map[string]Binding{
		"h":            {Rune: 'h'},
		"H":            {Rune: 'H'},
		":":            {Rune: ':'},
		"Space":        {Rune: ' '},
		"Left":         {Key: ebiten.KeyArrowLeft},
		"shift+left":   {Key: ebiten.KeyArrowLeft, Shift: true},
		"Shift+Enter":  {Key: ebiten.KeyEnter, Shift: true},
		"F2":           {Key: ebiten.KeyF2},
		"Ctrl+s":       {Key: ebiten.KeyS, Ctrl: true},
		"ctrl+shift+S": {Key: ebiten.KeyS, Ctrl: true, Shift: true},
		"Ctrl+Enter":   {Key: ebiten.KeyEnter, Ctrl: true},
	}
	for s, want := range cases {
		got, err := ParseKey(s)
		if err != nil || got != want {
			t.Errorf("ParseKey(%q) = %+v, %v", s, got, err)
		}
	}
	for _, bad := range []string{"Hyper", "Shift+Space", "", "Ctrl+;"} {
		if _, err := ParseKey(bad); err == nil {
			t.Errorf("ParseKey(%q) accepted", bad)
		}
	}
}

func TestDefaultsHaveNoConflicts(t *testing.T) {
	_, errs := NewKeymap(nil)
	for _, e := range errs {
		t.Error(e)
	}
}

func TestOverrides(t *testing.T) {
	km, errs := NewKeymap(map[string][]string{
		"road":     {"R"},
		"bogus":    {"x"},
		"pause":    {"Nope", "P"},
		"bulldoze": {"R"}, // conflicts with the new road binding
	})
	if len(errs) != 3 {
		t.Fatalf("want 3 errors, got %v", errs)
	}
	if a, _ := km.Lookup(Binding{Rune: 'R'}); a != Road {
		t.Errorf("R -> %s", a)
	}
	if _, ok := km.Lookup(Binding{Rune: 'r'}); ok {
		t.Error("old road key still bound")
	}
	if a, _ := km.Lookup(Binding{Rune: 'P'}); a != Pause {
		t.Error("valid key after a bad one dropped")
	}
	if got := km.KeysFor(Pause); len(got) != 1 || got[0] != "P" {
		t.Errorf("help keys %v", got)
	}
}

func TestKeyNameRoundTrip(t *testing.T) {
	for _, k := range []ebiten.Key{ebiten.KeyA, ebiten.KeyZ, ebiten.KeyDigit3, ebiten.KeyArrowLeft, ebiten.KeyF12, ebiten.KeyEnter} {
		name := KeyName(k)
		b, err := ParseKey("Ctrl+" + name)
		if err != nil || b.Key != k {
			t.Errorf("%v -> %q -> %+v %v", k, name, b, err)
		}
	}
	if KeyName(ebiten.KeyShift) != "" {
		t.Error("modifier got a name")
	}
}
