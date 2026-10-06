package font

import (
	"strings"
	"testing"
)

func TestGlyphShapes(t *testing.T) {
	for r, s := range glyphs {
		rows := strings.Fields(s)
		if len(rows) != 7 && len(rows) != 8 {
			t.Errorf("%q: %d rows", r, len(rows))
		}
		for _, row := range rows {
			if len(row) != GlyphW || strings.Trim(row, "01") != "" {
				t.Errorf("%q: bad row %q", r, row)
			}
		}
	}
}

func TestPrintableASCIICovered(t *testing.T) {
	for r := rune(32); r < 127; r++ {
		if !Has(r) {
			t.Errorf("missing %q", r)
		}
	}
}

func TestBits(t *testing.T) {
	b := Bits('L')
	if b[0] != 0b10000 || b[6] != 0b11111 || b[7] != 0 {
		t.Fatalf("L = %v", b)
	}
	if Bits('\x01') != Bits('?') {
		t.Fatal("unknown rune should render as ?")
	}
}
