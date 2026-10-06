package sim

import "testing"

// flat returns a city with all-land terrain so tests control every tile.
func flat(w, h int) *City {
	c := New(w, h, 1)
	for i := range c.Tiles {
		c.Tiles[i] = Tile{Anchor: -1, Powered: true, Watered: true}
	}
	return c
}

func TestRoadRules(t *testing.T) {
	c := flat(8, 8)
	c.At(1, 0).Terrain = Water
	c.At(2, 0).Terrain = Trees
	p := c.Apply(ToolRoad, LPath(Pt{0, 0}, Pt{3, 0}, false), false)
	if p.Err != "" || len(p.Tiles) != 4 || p.Cost != 3*CostRoad+CostBridge {
		t.Fatalf("plan %+v", p)
	}
	if c.Funds != StartingFunds-3*CostRoad-CostBridge {
		t.Errorf("funds %v", c.Funds)
	}
	if c.At(2, 0).Terrain != Land || c.At(2, 0).Kind != Road {
		t.Error("trees not cleared under road")
	}
	if tl := c.At(1, 0); tl.Kind != Road || tl.Terrain != Water {
		t.Error("road over water should be a bridge")
	}
	c.At(5, 5).Terrain = Rock
	if p := c.Apply(ToolRoad, []Pt{{5, 5}}, false); p.Err == "" {
		t.Error("road on rock")
	}
	if p := c.Apply(ToolZoneR, []Pt{{1, 0}}, false); p.Err == "" {
		t.Error("zone on a bridge")
	}
	// Building again over the same road costs nothing and is refused.
	if p := c.Apply(ToolRoad, []Pt{{0, 0}}, false); p.Err == "" {
		t.Error("duplicate road accepted")
	}
}

func TestLineSharesRoad(t *testing.T) {
	c := flat(4, 4)
	c.Apply(ToolRoad, []Pt{{0, 0}}, false)
	if p := c.Apply(ToolLine, []Pt{{0, 0}, {1, 0}}, false); len(p.Tiles) != 2 {
		t.Fatalf("plan %+v", p)
	}
	if tl := c.At(0, 0); tl.Kind != Road || !tl.Line {
		t.Error("line should sit on road")
	}
}

func TestZoning(t *testing.T) {
	c := flat(4, 4)
	c.Apply(ToolLine, []Pt{{0, 0}}, false)
	p := c.Apply(ToolZoneR, RectPts(Pt{0, 0}, Pt{1, 1}), false)
	if len(p.Tiles) != 4 || p.Cost != 4*CostZone {
		t.Fatalf("plan %+v", p)
	}
	if c.At(0, 0).Line {
		t.Error("zone should replace the line")
	}
	// Rezoning an empty lot works; a developed lot is skipped.
	c.At(1, 1).Level = 2
	p = c.Apply(ToolZoneC, RectPts(Pt{0, 0}, Pt{1, 1}), false)
	if len(p.Tiles) != 3 || c.At(1, 1).Kind != ZoneR || c.At(0, 0).Kind != ZoneC {
		t.Fatalf("rezone plan %+v", p)
	}
	// Roads may replace empty lots but not buildings.
	p = c.Apply(ToolRoad, []Pt{{0, 1}, {1, 1}}, false)
	if len(p.Tiles) != 1 || c.At(0, 1).Kind != Road {
		t.Fatalf("road over lots %+v", p)
	}
}

func TestPipesAndUndergroundBulldoze(t *testing.T) {
	c := flat(4, 4)
	c.At(0, 0).Terrain = Trees
	c.Apply(ToolRoad, []Pt{{1, 0}}, false)
	c.Apply(ToolPipe, []Pt{{0, 0}, {1, 0}}, false)
	if c.At(0, 0).Terrain != Trees || !c.At(0, 0).Pipe || !c.At(1, 0).Pipe {
		t.Fatal("pipes go under trees and roads without clearing them")
	}
	c.Apply(ToolBulldoze, []Pt{{1, 0}}, true)
	if c.At(1, 0).Pipe || c.At(1, 0).Kind != Road {
		t.Error("underground bulldoze must only remove the pipe")
	}
	c.Apply(ToolBulldoze, []Pt{{0, 0}}, false)
	if c.At(0, 0).Terrain != Land || !c.At(0, 0).Pipe {
		t.Error("surface bulldoze clears trees and keeps pipes")
	}
}

func TestBulldozeCost(t *testing.T) {
	c := flat(4, 4)
	c.Apply(ToolZoneI, []Pt{{0, 0}, {1, 0}}, false)
	c.At(1, 0).Level = 1
	f := c.Funds
	p := c.Apply(ToolBulldoze, []Pt{{0, 0}, {1, 0}, {2, 0}}, false)
	if p.Skipped != 1 || f-c.Funds != 2*CostBulldoze+CostBulldozeBuilt {
		t.Fatalf("plan %+v, spent %v", p, f-c.Funds)
	}
}

func TestFundsAndDebt(t *testing.T) {
	c := flat(8, 8)
	c.Funds = 15
	p := c.Apply(ToolRoad, LPath(Pt{0, 0}, Pt{1, 0}, false), false)
	if p.Err == "" || c.Funds != 15 || c.At(0, 0).Kind != Empty {
		t.Fatal("over-budget action must change nothing")
	}
	c.Apply(ToolRoad, []Pt{{0, 0}}, false)
	c.Funds = -100
	if p := c.Apply(ToolRoad, []Pt{{1, 0}}, false); p.Err == "" {
		t.Error("building while in debt")
	}
	if p := c.Apply(ToolBulldoze, []Pt{{0, 0}}, false); p.Err != "" {
		t.Errorf("bulldoze in debt refused: %s", p.Err)
	}
}

func TestGeometry(t *testing.T) {
	got := LPath(Pt{0, 0}, Pt{2, 1}, false)
	want := []Pt{{0, 0}, {1, 0}, {2, 0}, {2, 1}}
	if len(got) != len(want) {
		t.Fatalf("LPath %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("LPath %v", got)
		}
	}
	if v := LPath(Pt{2, 1}, Pt{0, 0}, true); v[1] != (Pt{2, 0}) || len(v) != 4 {
		t.Errorf("vertical-first %v", v)
	}
	if n := len(RectPts(Pt{3, 3}, Pt{1, 2})); n != 6 {
		t.Errorf("rect %d", n)
	}
	if n := len(LPath(Pt{1, 1}, Pt{1, 1}, false)); n != 1 {
		t.Errorf("single point path %d", n)
	}
}
