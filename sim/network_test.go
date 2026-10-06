package sim

import (
	"strings"
	"testing"
)

func TestPowerConnectivity(t *testing.T) {
	c := flat(20, 10)
	c.Apply(ToolPlant, []Pt{{0, 0}}, false)
	// A zone block touching the plant, and one reached only through a line.
	c.Apply(ToolZoneR, RectPts(Pt{3, 0}, Pt{6, 1}), false)
	c.Apply(ToolZoneR, RectPts(Pt{12, 0}, Pt{14, 1}), false)
	c.Apply(ToolZoneR, RectPts(Pt{12, 6}, Pt{14, 7}), false) // isolated
	for _, p := range RectPts(Pt{3, 0}, Pt{14, 7}) {
		if t := c.At(p.X, p.Y); t.IsZone() {
			t.Level = 1
		}
	}
	c.updatePower()
	if !c.At(6, 1).Powered {
		t.Error("zone block should conduct from the plant")
	}
	if c.At(12, 0).Powered {
		t.Error("gap in the network still powered")
	}
	c.Apply(ToolLine, LPath(Pt{7, 0}, Pt{11, 0}, false), false)
	c.updatePower()
	if !c.At(14, 1).Powered {
		t.Error("line didn't connect the far block")
	}
	if c.At(12, 6).Powered {
		t.Error("isolated block powered")
	}
	if c.Power.Capacity != PlantCapacity || c.Power.Served != 14 {
		t.Errorf("totals %+v", c.Power)
	}
}

func TestRoadsCarryPower(t *testing.T) {
	c := flat(20, 6)
	c.Apply(ToolPlant, []Pt{{0, 0}}, false)
	c.Apply(ToolRoad, LPath(Pt{3, 1}, Pt{15, 1}, false), false)
	c.Apply(ToolZoneR, []Pt{{15, 2}}, false)
	c.At(15, 2).Level = 1
	c.Apply(ToolZoneR, []Pt{{18, 4}}, false) // not touching anything
	c.At(18, 4).Level = 1
	c.updatePower()
	if !c.At(15, 2).Powered {
		t.Error("zone at the end of a road from the plant has no power")
	}
	if c.At(18, 4).Powered {
		t.Error("isolated zone powered")
	}
}

func TestBrownoutNearestFirst(t *testing.T) {
	c := flat(40, 4)
	c.Apply(ToolPlant, []Pt{{0, 0}}, false)
	c.Funds = 1e6
	c.Apply(ToolZoneI, RectPts(Pt{3, 0}, Pt{39, 0}), false)
	for x := 3; x < 40; x++ {
		c.At(x, 0).Level = 3 // 9 units each, 37 tiles = 333 > 200
	}
	c.updatePower()
	if !c.At(3, 0).Powered || c.At(39, 0).Powered {
		t.Fatal("brownout should hit the far end")
	}
	powered := 0
	for x := 3; x < 40; x++ {
		if c.At(x, 0).Powered {
			powered++
		}
	}
	if powered != PlantCapacity/9 {
		t.Errorf("powered %d tiles", powered)
	}
	if !strings.HasPrefix(c.Log[len(c.Log)-1].Msg, "brownout") {
		t.Error("no brownout event")
	}
	c.updatePower()
	if len(c.Log) != 1 {
		t.Errorf("brownout logged %d times", len(c.Log))
	}
}

func TestWater(t *testing.T) {
	c := flat(20, 8)
	for x := 0; x < 20; x++ {
		c.At(x, 0).Terrain = Water
	}
	if p := c.Apply(ToolPump, []Pt{{5, 4}}, false); p.Err == "" {
		t.Fatal("pump away from water accepted")
	}
	c.Apply(ToolPump, []Pt{{0, 1}}, false)
	c.Apply(ToolZoneR, RectPts(Pt{2, 3}, Pt{10, 3}), false)
	for x := 2; x <= 10; x++ {
		c.At(x, 3).Level = 2
	}
	c.Apply(ToolPipe, LPath(Pt{1, 2}, Pt{8, 2}, false), false)

	c.updateUtilities()
	if c.At(3, 3).Watered {
		t.Fatal("unpowered pump supplied water")
	}
	c.Apply(ToolPlant, []Pt{{11, 3}}, false)
	c.Apply(ToolLine, LPath(Pt{2, 1}, Pt{2, 2}, false), false)
	c.updateUtilities()
	if !c.At(0, 1).Powered {
		t.Fatal("pump not powered by the zone row next to the plant")
	}
	if !c.At(3, 3).Watered || !c.At(8, 3).Watered {
		t.Error("tiles beside the pipe should be watered")
	}
	if c.At(10, 3).Watered {
		t.Error("tile two away from the pipe watered")
	}
	if c.Water.Capacity != PumpCapacity {
		t.Errorf("water %+v", c.Water)
	}
}

func TestBuildingPlaceAndBulldoze(t *testing.T) {
	c := flat(10, 10)
	c.At(1, 1).Terrain = Trees
	p := c.Apply(ToolPlant, []Pt{{0, 0}}, false)
	if p.Err != "" || len(p.Tiles) != 9 || c.Funds != StartingFunds-3000 {
		t.Fatalf("place %+v", p)
	}
	if c.At(2, 2).Kind != PowerPlant || c.At(2, 2).Anchor != 0 || c.At(1, 1).Terrain != Land {
		t.Fatal("footprint wrong")
	}
	if p := c.Apply(ToolSchool, []Pt{{2, 2}}, false); p.Err == "" {
		t.Fatal("overlapping building accepted")
	}
	if p := c.Apply(ToolSchool, []Pt{{9, 9}}, false); p.Err == "" {
		t.Fatal("building off the map edge accepted")
	}
	p = c.Apply(ToolBulldoze, RectPts(Pt{1, 1}, Pt{2, 2}), false)
	if len(p.Tiles) != 9 || p.Cost != CostBulldoze+CostBulldozeBuilt {
		t.Fatalf("bulldoze %+v", p)
	}
	for _, pt := range RectPts(Pt{0, 0}, Pt{2, 2}) {
		if tl := c.At(pt.X, pt.Y); tl.Kind != Empty || tl.Anchor != -1 {
			t.Fatal("building not fully removed")
		}
	}
}

func TestUnpoweredDecline(t *testing.T) {
	c := flat(6, 3)
	c.Apply(ToolRoad, RectPts(Pt{0, 0}, Pt{5, 0}), false)
	c.Apply(ToolZoneR, []Pt{{0, 1}}, false)
	c.At(0, 1).Level = 1
	c.Demand = [3]float64{1, 1, 1}
	for i := 0; i < 400*TicksPerDay; i++ {
		c.Tick()
	}
	if c.At(0, 1).Level != 0 {
		t.Error("building without power for a year should decline")
	}
}
