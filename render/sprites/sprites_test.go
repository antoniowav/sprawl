package sprites

import (
	"bytes"
	"testing"

	"github.com/antoniowav/sprawl/theme"
)

func TestTerrainDeterministicAndOpaque(t *testing.T) {
	r := theme.Derive(theme.Builtin())
	a, b := BuildTerrain(r), BuildTerrain(r)
	for i := range a.Grass {
		if !bytes.Equal(a.Grass[i].Pix, b.Grass[i].Pix) {
			t.Fatal("grass not deterministic")
		}
	}
	for m, img := range a.Water {
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] != 0xff {
				t.Fatalf("water mask %d has transparent pixels", m)
			}
		}
	}
	if bytes.Equal(a.Grass[0].Pix, a.Grass[1].Pix) {
		t.Error("grass variants identical")
	}
	if bytes.Equal(a.Water[0].Pix, a.Water[N|E].Pix) {
		t.Error("shore mask has no effect")
	}
}

func TestThemeChangesSprites(t *testing.T) {
	p := theme.Builtin()
	a := BuildTerrain(theme.Derive(p))
	p.Green = theme.MustHex("#ff0000")
	b := BuildTerrain(theme.Derive(p))
	if bytes.Equal(a.Grass[0].Pix, b.Grass[0].Pix) {
		t.Error("grass ignores palette")
	}
}

func TestNetworkMasksDiffer(t *testing.T) {
	n := BuildNetwork(theme.Derive(theme.Builtin()))
	seen := map[string]int{}
	for m, img := range n.Road {
		k := string(img.Pix)
		if prev, dup := seen[k]; dup {
			t.Errorf("road masks %d and %d identical", prev, m)
		}
		seen[k] = m
	}
	if bytes.Equal(n.Lot[0].Pix, n.Lot[1].Pix) {
		t.Error("R and C lots identical")
	}
}

func TestBuildingsAllDistinct(t *testing.T) {
	b := BuildBuildings(theme.Derive(theme.Builtin()))
	seen := map[string]bool{}
	for z := range b.Img {
		for l := range b.Img[z] {
			for v, img := range b.Img[z][l] {
				k := string(img.Pix)
				if seen[k] {
					t.Errorf("zone %d level %d variant %d duplicates another sprite", z, l+1, v)
				}
				seen[k] = true
			}
		}
	}
	for v, c := range b.Chimneys {
		if len(c) == 0 {
			t.Errorf("factory variant %d has no chimney", v)
		}
	}
}

func TestCivicSizes(t *testing.T) {
	c := BuildCivic(theme.Derive(theme.Builtin()))
	want := map[string]int{"plant": 3, "wind": 1, "pump": 2, "tower": 1, "police": 2, "fire": 2,
		"school": 2, "park": 1, "hall": 3, "stadium": 3}
	for name, n := range want {
		img := c.Img[name]
		if img == nil || img.Bounds().Dx() != n*T || img.Bounds().Dy() != n*T {
			t.Errorf("civic %s missing or wrong size", name)
		}
	}
	ic := BuildIcons(theme.Derive(theme.Builtin()))
	if ic.NoPower.Bounds().Dx() != 9 {
		t.Error("icon size")
	}
}
