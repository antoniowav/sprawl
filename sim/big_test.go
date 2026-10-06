package sim

import "testing"

// bigBlock: a 2×2 high-density block, powered, watered, high land value.
func bigBlock() *City {
	c := flat(6, 6)
	c.Apply(ToolRoad, RectPts(Pt{0, 0}, Pt{5, 0}), false)
	c.Apply(ToolRoad, RectPts(Pt{0, 3}, Pt{5, 3}), false)
	c.Apply(ToolZoneR, RectPts(Pt{1, 1}, Pt{2, 2}), false)
	for _, p := range RectPts(Pt{1, 1}, Pt{2, 2}) {
		t := c.At(p.X, p.Y)
		t.Level, t.LandValue, t.Powered, t.Watered = 3, 0.9, true, true
	}
	c.Demand[R] = 0.8
	return c
}

func TestMergeSplit(t *testing.T) {
	c := bigBlock()
	if !c.canMerge(1, 1) || c.canMerge(2, 1) {
		t.Fatal("merge check")
	}
	c.merge(1, 1)
	a := int32(1*c.W + 1)
	for _, p := range RectPts(Pt{1, 1}, Pt{2, 2}) {
		if c.At(p.X, p.Y).Anchor != a || !c.At(p.X, p.Y).IsBig() {
			t.Fatal("merge didn't mark all four tiles")
		}
	}
	c.updateStats()
	if per := float64(90) * bigBonus; c.Stats.Residents != 4*int(per) {
		t.Errorf("residents %d", c.Stats.Residents)
	}
	// Bulldozing one tile clears the whole block.
	p := c.Apply(ToolBulldoze, []Pt{{2, 2}}, false)
	if len(p.Tiles) != 4 || c.At(1, 1).Kind != Empty || c.At(1, 1).Anchor != -1 {
		t.Fatalf("bulldoze %+v", p)
	}
}

func TestBigSplitsOnDecline(t *testing.T) {
	c := bigBlock()
	c.merge(1, 1)
	for _, p := range RectPts(Pt{1, 1}, Pt{2, 2}) {
		c.At(p.X, p.Y).LandValue = 0.1 // way too low for high density
	}
	for i := 0; i < 400 && c.At(1, 1).IsBig(); i++ {
		c.growStripe(i % TicksPerDay)
	}
	if c.At(1, 1).IsBig() || c.At(2, 2).IsBig() {
		t.Fatal("block should split when a tile declines")
	}
}

func TestMergesOverTime(t *testing.T) {
	c := bigBlock()
	merged := false
	for i := 0; i < 2000 && !merged; i++ {
		c.growStripe(i % TicksPerDay)
		merged = c.At(1, 1).IsBig()
	}
	if !merged {
		t.Fatal("qualifying block never merged")
	}
}
