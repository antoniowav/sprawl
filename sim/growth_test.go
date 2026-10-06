package sim

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestRawDemand(t *testing.T) {
	d := RawDemand(Stats{})
	if !near(d[R], 1) || !near(d[C], 0) || !near(d[I], 1) {
		t.Errorf("empty city %v", d)
	}
	// 100 residents, no jobs: too many workers, shops wanted, factories wanted.
	d = RawDemand(Stats{Residents: 100})
	if !near(d[R], -1) || !near(d[C], 1) || !near(d[I], 1) {
		t.Errorf("bedroom town %v", d)
	}
	// Balanced: 100 residents (50 workers), 20 shop jobs, 55 factory jobs.
	d = RawDemand(Stats{Residents: 100, CommJobs: 20, IndJobs: 55})
	if !near(d[R], (75+20-50)/95.0) || !near(d[C], 0) || !near(d[I], 0) {
		t.Errorf("balanced %v", d)
	}
}

func TestTaxMod(t *testing.T) {
	if TaxMod(9) != 0 || !near(TaxMod(20), 0.44) || !near(TaxMod(0), -0.36) {
		t.Error("tax curve")
	}
}

func TestDemandSmoothsAndTaxes(t *testing.T) {
	c := flat(8, 8)
	c.updateDemand()
	if !near(c.Demand[R], demandSmooth) {
		t.Fatalf("first step %v", c.Demand[R])
	}
	c.Tax[R] = 20
	for i := 0; i < 200; i++ {
		c.updateDemand()
	}
	if !near(c.Demand[R], 1-TaxMod(20)) {
		t.Errorf("converged to %v", c.Demand[R])
	}
}

// strip builds a road along y=0 with an R zone row at y=1 and an R row
// at y=3 with no road; a power line on y=2 and a plant below power both.
func strip(w int) *City {
	c := flat(w, 8)
	c.Apply(ToolRoad, RectPts(Pt{0, 0}, Pt{w - 1, 0}), false)
	c.Apply(ToolZoneR, RectPts(Pt{0, 1}, Pt{w - 1, 1}), false)
	c.Apply(ToolLine, RectPts(Pt{0, 2}, Pt{w - 1, 2}), false)
	c.Apply(ToolZoneR, RectPts(Pt{0, 3}, Pt{w - 1, 3}), false)
	c.Apply(ToolPlant, []Pt{{0, 4}}, false)
	return c
}

func TestGrowthNeedsRoad(t *testing.T) {
	c := strip(10)
	for i := 0; i < 400*TicksPerDay; i++ {
		c.Tick()
	}
	grown := 0
	for x := 0; x < 10; x++ {
		if c.At(x, 1).Level > 0 {
			grown++
		}
		if c.At(x, 3).Level > 0 {
			t.Fatal("lot without road access grew")
		}
	}
	if grown == 0 {
		t.Fatal("nothing grew next to the road")
	}
	if c.Stats.Residents == 0 || c.Day != 400 {
		t.Errorf("stats %+v day %d", c.Stats, c.Day)
	}
}

func TestLandValueCapsDensity(t *testing.T) {
	c := strip(10)
	c.Demand = [3]float64{1, 1, 1}
	for i := range c.Tiles {
		c.Tiles[i].LandValue = 0.2
	}
	for i := 0; i < 60; i++ {
		c.growStripe(i % TicksPerDay)
	}
	for x := 0; x < 10; x++ {
		if c.At(x, 1).Level > 1 {
			t.Fatal("grew past the land-value cap")
		}
	}
	if MaxLevel(0.2) != 1 || MaxLevel(0.5) != 2 || MaxLevel(0.61) != 3 {
		t.Error("MaxLevel thresholds")
	}
}

func TestDeclineWithoutRoad(t *testing.T) {
	c := strip(4)
	c.At(0, 3).Level = 2
	for i := 0; i < 200*TicksPerDay; i++ {
		c.Tick()
	}
	if c.At(0, 3).Level != 0 {
		t.Errorf("cut-off building still level %d", c.At(0, 3).Level)
	}
}

func TestDeterministic(t *testing.T) {
	a, b := strip(12), strip(12)
	for i := 0; i < 1000; i++ {
		a.Tick()
		b.Tick()
	}
	for i := range a.Tiles {
		if a.Tiles[i] != b.Tiles[i] {
			t.Fatal("runs diverged")
		}
	}
}

func TestPollutionLowersLandValue(t *testing.T) {
	c := flat(20, 20)
	c.updateLandValue()
	base := c.At(10, 10).LandValue
	c.Apply(ToolZoneI, []Pt{{10, 11}}, false)
	c.At(10, 11).Level = 3
	c.updateLandValue()
	if c.At(10, 10).Pollution == 0 || c.At(10, 10).LandValue >= base {
		t.Errorf("pollution %v lv %v (was %v)", c.At(10, 10).Pollution, c.At(10, 10).LandValue, base)
	}
}

func TestMilestoneEvent(t *testing.T) {
	c := flat(10, 2)
	c.Apply(ToolZoneR, RectPts(Pt{0, 0}, Pt{9, 0}), false)
	for x := 0; x < 10; x++ {
		c.At(x, 0).Level = 2
	}
	for i := range c.Tiles {
		c.Tiles[i].Powered = true
	}
	c.updateStats()
	if c.Stats.Residents != 300 {
		t.Fatalf("residents %d", c.Stats.Residents)
	}
	if c.PeakPop != 300 || c.Funds != StartingFunds-10*CostZone+500+1000 {
		t.Errorf("peak %d funds %v", c.PeakPop, c.Funds)
	}
	if c.Rank() != "village" || c.Goal() != "goal: 1000 people → city hall" {
		t.Errorf("rank %q goal %q", c.Rank(), c.Goal())
	}
	// Grants are paid once.
	c.Stats.Residents = 0
	c.updateStats()
	for x := 0; x < 10; x++ {
		c.At(x, 0).Level = 2
	}
	f := c.Funds
	c.updateStats()
	if c.Funds != f {
		t.Error("milestone grant paid twice")
	}
}

func TestServiceCoverage(t *testing.T) {
	c := flat(40, 40)
	c.Apply(ToolSchool, []Pt{{10, 10}}, false)
	c.updateUtilities()
	c.updateLandValue()
	near, edge, far := c.At(10, 10).Cover[2], c.At(10+10, 11).Cover[2], c.At(30, 30).Cover[2]
	if far != 0 || near < 0.45 || edge <= 0 || edge >= near {
		t.Fatalf("unpowered school coverage near %v edge %v far %v", near, edge, far)
	}
	base := c.At(10, 10).LandValue
	c.Apply(ToolPlant, []Pt{{12, 10}}, false)
	c.updateUtilities()
	c.updateLandValue()
	if c.At(10, 10).Cover[2] <= near {
		t.Error("powered school should cover more")
	}
	if c.At(13, 11).Pollution == 0 {
		t.Error("power plant should pollute")
	}
	_ = base
}

func TestServicesRaiseDensityCap(t *testing.T) {
	c := flat(30, 30)
	c.Apply(ToolPolice, []Pt{{4, 4}}, false)
	c.Apply(ToolFire, []Pt{{8, 4}}, false)
	c.Apply(ToolSchool, []Pt{{12, 4}}, false)
	c.Apply(ToolPlant, []Pt{{14, 4}}, false)
	// Zones touching all three buildings carry power to them.
	c.Apply(ToolZoneR, RectPts(Pt{4, 6}, Pt{13, 7}), false)
	for _, p := range RectPts(Pt{4, 6}, Pt{13, 7}) {
		c.At(p.X, p.Y).Level = 2
	}
	c.updateUtilities()
	c.updateLandValue()
	if !c.At(4, 4).Powered {
		t.Fatal("police not powered")
	}
	if lv := c.At(8, 7).LandValue; MaxLevel(lv) < 3 {
		t.Errorf("full services + neighbourhood should allow high density, LV %.2f", lv)
	}
}
