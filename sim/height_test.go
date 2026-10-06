package sim

import "testing"

func TestGeneratedHeights(t *testing.T) {
	for _, m := range MapTypes {
		c := NewMap(96, 96, 5, m)
		top := uint8(0)
		for y := 0; y < c.H; y++ {
			for x := 0; x < c.W; x++ {
				tl := c.At(x, y)
				if tl.Terrain == Water && tl.Height != 0 {
					t.Fatalf("%s: water at height %d", m, tl.Height)
				}
				if tl.Terrain != Water && tl.Height == 0 {
					t.Fatalf("%s: land at water level", m)
				}
				if c.Slope(x, y) > roadClimb {
					t.Fatalf("%s: cliff of %d at %d,%d", m, c.Slope(x, y), x, y)
				}
				top = max(top, tl.Height)
			}
		}
		if top < 3 {
			t.Errorf("%s: flat map, top %d", m, top)
		}
		if s := c.At(c.Start.X, c.Start.Y); s.Terrain != Land {
			t.Errorf("%s: start not on land", m)
		}
	}
}

func TestSlopeRules(t *testing.T) {
	c := flat(10, 10)
	for i := range c.Tiles {
		c.Tiles[i].Height = 1
	}
	c.At(5, 5).Height = 3 // a spike two levels up
	if p := c.Apply(ToolZoneR, []Pt{{5, 5}}, false); p.Err == "" {
		t.Error("zoned a steep tile")
	}
	p := c.Apply(ToolRoad, []Pt{{5, 5}}, false)
	if p.Err != "" || p.Cost != 3*CostRoad {
		t.Errorf("steep road %+v", p)
	}
	c.At(2, 2).Height = 4 // three levels: a cliff
	if p := c.Apply(ToolRoad, []Pt{{2, 2}}, false); p.Err == "" {
		t.Error("road up a cliff")
	}
	c.At(7, 7).Height = 2
	c.At(8, 8).Height = 0
	if p := c.Apply(ToolSchool, []Pt{{7, 7}}, false); p.Err == "" {
		t.Error("building on uneven ground")
	}
}

func TestTerraform(t *testing.T) {
	c := flat(10, 10)
	for i := range c.Tiles {
		c.Tiles[i].Height = 1
	}
	c.At(0, 5).Terrain, c.At(0, 5).Height = Water, 0
	f := c.Funds
	c.Terraform(TerraRaise, RectPts(Pt{3, 3}, Pt{4, 4}))
	if c.At(3, 3).Height != 2 || f-c.Funds != 4*CostTerraform {
		t.Fatalf("raise: h %d spent %v", c.At(3, 3).Height, f-c.Funds)
	}
	c.Terraform(TerraLower, []Pt{{1, 5}})
	if c.At(1, 5).Terrain != Water {
		t.Error("lowering next to water should flood")
	}
	c.Terraform(TerraLower, []Pt{{8, 8}})
	if c.At(8, 8).Terrain != Land || c.At(8, 8).Height != 1 {
		t.Error("dry ground can't go below water level")
	}
	c.Terraform(TerraRaise, []Pt{{0, 5}})
	if c.At(0, 5).Terrain != Land {
		t.Error("raising water should make land")
	}
	c.Terraform(TerraLevel, []Pt{{3, 3}, {5, 5}, {6, 6}})
	if c.At(5, 5).Height != 2 || c.At(6, 6).Height != 2 {
		t.Error("level to the first tile's height")
	}
	c.Apply(ToolRoad, []Pt{{9, 9}}, false)
	if p := c.PlanTerraform(TerraRaise, []Pt{{9, 9}}); p.Err == "" {
		t.Error("terraformed under a road")
	}
}

func TestViewRaisesLandValue(t *testing.T) {
	c := flat(20, 20)
	for i := range c.Tiles {
		c.Tiles[i].Height = 1
	}
	c.updateLandValue()
	low := c.At(10, 10).LandValue
	for _, p := range RectPts(Pt{9, 9}, Pt{11, 11}) {
		c.At(p.X, p.Y).Height = 3
	}
	c.updateLandValue()
	if c.At(10, 10).LandValue <= low {
		t.Errorf("hilltop %v vs flat %v", c.At(10, 10).LandValue, low)
	}
}

func TestTerraformToolAndUndo(t *testing.T) {
	c := flat(10, 10)
	for i := range c.Tiles {
		c.Tiles[i].Height = 2
	}
	c.At(5, 5).Height = 4
	_, e := c.ApplyEdit(ToolLevel, Selection(ToolLevel, Pt{5, 5}, Pt{6, 6}, false), false)
	if e == nil || c.At(6, 6).Height != 4 {
		t.Fatalf("level from the anchor: %d", c.At(6, 6).Height)
	}
	c.Undo(e)
	if c.At(6, 6).Height != 2 {
		t.Error("undo didn't restore the height")
	}
}
