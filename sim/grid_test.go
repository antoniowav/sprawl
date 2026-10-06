package sim

import "testing"

func TestNewIsDeterministic(t *testing.T) {
	a, b := New(96, 80, 42), New(96, 80, 42)
	if a.Name != b.Name {
		t.Fatalf("names differ: %s %s", a.Name, b.Name)
	}
	for i := range a.Tiles {
		if a.Tiles[i] != b.Tiles[i] {
			t.Fatalf("tile %d differs", i)
		}
	}
	c := New(96, 80, 43)
	same := 0
	for i := range a.Tiles {
		if a.Tiles[i] == c.Tiles[i] {
			same++
		}
	}
	if same == len(a.Tiles) {
		t.Fatal("different seeds produced the same map")
	}
}

func TestTerrainShares(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		c := New(128, 128, seed)
		var water, trees int
		for _, tl := range c.Tiles {
			switch tl.Terrain {
			case Water:
				water++
			case Trees:
				trees++
			}
			if tl.Anchor != -1 || tl.Kind != Empty {
				t.Fatal("new map has buildings")
			}
		}
		n := float64(len(c.Tiles))
		if w := float64(water) / n; w < 0.05 || w > 0.2 {
			t.Errorf("seed %d: water share %.3f", seed, w)
		}
		if tr := float64(trees) / n; tr < 0.08 || tr > 0.2 {
			t.Errorf("seed %d: tree share %.3f", seed, tr)
		}
	}
}

func TestDate(t *testing.T) {
	cases := map[int]string{0: "Jan 1, Y1", 29: "Jan 30, Y1", 30: "Feb 1, Y1", 360: "Jan 1, Y2", 433: "Mar 14, Y2"}
	for day, want := range cases {
		if got := Date(day); got != want {
			t.Errorf("Date(%d) = %q, want %q", day, got, want)
		}
	}
}

func TestLogRing(t *testing.T) {
	c := New(32, 32, 1)
	for i := 0; i < MaxEvents+10; i++ {
		c.Logf(Info, "e%d", i)
	}
	if len(c.Log) != MaxEvents || c.Log[0].Msg != "e10" {
		t.Fatalf("ring wrong: len %d first %q", len(c.Log), c.Log[0].Msg)
	}
}

func TestMapTypes(t *testing.T) {
	for _, m := range MapTypes {
		for seed := int64(1); seed <= 5; seed++ {
			c := NewMap(128, 128, seed, m)
			var water, rock int
			for _, tl := range c.Tiles {
				switch tl.Terrain {
				case Water:
					water++
				case Rock:
					rock++
				}
			}
			n := float64(len(c.Tiles))
			lo, hi := 0.04, 0.3
			if m == MapIslands {
				lo, hi = 0.5, 0.7
			}
			if w := float64(water) / n; w < lo || w > hi {
				t.Errorf("%s seed %d: water %.2f", m, seed, w)
			}
			if (m == MapHighlands) != (rock > 0) {
				t.Errorf("%s seed %d: rock %d", m, seed, rock)
			}
			if c.At(c.Start.X, c.Start.Y).Terrain != Land {
				t.Errorf("%s seed %d: start %v not on land", m, seed, c.Start)
			}
		}
	}
	a, b := NewMap(64, 64, 3, MapIslands), NewMap(64, 64, 3, MapIslands)
	for i := range a.Tiles {
		if a.Tiles[i] != b.Tiles[i] {
			t.Fatal("islands not deterministic")
		}
	}
}
