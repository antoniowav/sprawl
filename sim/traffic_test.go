package sim

import "testing"

// commuteTown: homes at the west end of a long road, factories at the east.
func commuteTown(w int) *City {
	c := flat(w, 6)
	c.Funds = 1e6
	c.Apply(ToolRoad, RectPts(Pt{0, 2}, Pt{w - 1, 2}), false)
	c.Apply(ToolZoneR, RectPts(Pt{0, 1}, Pt{5, 1}), false)
	c.Apply(ToolZoneI, RectPts(Pt{w - 4, 3}, Pt{w - 1, 3}), false)
	for x := 0; x <= 5; x++ {
		c.At(x, 1).Level = 3
	}
	for x := w - 4; x < w; x++ {
		c.At(x, 3).Level = 2
	}
	return c
}

func TestCommuteLoadsRoads(t *testing.T) {
	c := commuteTown(30)
	c.updateTraffic()
	if c.At(0, 1).Commute <= 0 || c.At(0, 1).Commute != int16(30-4) {
		t.Errorf("commute %d", c.At(0, 1).Commute)
	}
	// All six homes pass the middle of the road: 6 × 45 commuters.
	if got := c.At(15, 2).Traffic; got != 270 {
		t.Errorf("middle traffic %d", got)
	}
	if c.At(15, 2).Congestion() <= 1 {
		t.Error("expected congestion")
	}
	if c.noise[1*c.W+15] == 0 {
		t.Error("congestion should make noise next to the road")
	}
}

func TestNoJobsByRoad(t *testing.T) {
	c := commuteTown(30)
	c.Apply(ToolBulldoze, []Pt{{12, 2}}, false) // cut the road
	c.updateTraffic()
	c.Stats = Stats{IndJobs: 40}
	if c.At(0, 1).Commute != -1 || c.commutePenalty(c.At(0, 1)) != noJobsPenalty {
		t.Errorf("cut-off home: commute %d", c.At(0, 1).Commute)
	}
}

func TestBusesNeedADepot(t *testing.T) {
	c := commuteTown(30)
	c.Apply(ToolBusStop, []Pt{{3, 0}}, false)
	c.updateUtilities()
	c.updateTraffic()
	before := c.At(15, 2).Traffic
	if before != 270 {
		t.Fatalf("bus stop without a depot changed traffic: %d", before)
	}
	c.Apply(ToolBusDepot, []Pt{{10, 3}}, false)
	c.Apply(ToolPlant, []Pt{{12, 3}}, false)
	c.updateUtilities()
	c.updateTraffic()
	if got := c.At(15, 2).Traffic; got >= before {
		t.Errorf("buses didn't take commuters off the road: %d", got)
	}
}

func TestTripRoute(t *testing.T) {
	c := commuteTown(30)
	c.updateTraffic()
	r := c.TripRoute(1*c.W + 0)
	if len(r) != 27 || r[0] != 2*c.W+0 || r[len(r)-1] != 2*c.W+26 {
		t.Fatalf("route %d tiles, %v..%v", len(r), r[0], r[len(r)-1])
	}
	c.Apply(ToolBulldoze, []Pt{{12, 2}}, false)
	c.updateTraffic()
	if c.TripRoute(1*c.W+0) != nil {
		t.Error("route through a missing road")
	}
}
