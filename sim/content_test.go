package sim

import "testing"

func TestWindAndTower(t *testing.T) {
	c := flat(12, 6)
	c.Apply(ToolWind, []Pt{{0, 0}}, false)
	c.Apply(ToolZoneR, RectPts(Pt{1, 0}, Pt{5, 0}), false)
	for x := 1; x <= 5; x++ {
		c.At(x, 0).Level = 3
	}
	c.Apply(ToolTower, []Pt{{0, 2}}, false) // no water terrain anywhere
	c.Apply(ToolLine, []Pt{{0, 1}}, false)
	c.Apply(ToolPipe, RectPts(Pt{0, 1}, Pt{5, 1}), false)
	c.updateUtilities()
	if c.Power.Capacity != 25 || !c.At(1, 0).Powered {
		t.Fatalf("wind power %+v", c.Power)
	}
	if !c.At(0, 2).Powered || c.Water.Capacity != 40 || !c.At(3, 0).Watered {
		t.Errorf("water tower: powered %v water %+v", c.At(0, 2).Powered, c.Water)
	}
}

func TestParkRaisesLandValue(t *testing.T) {
	c := flat(20, 20)
	c.updateLandValue()
	before := c.At(10, 10).LandValue
	c.Apply(ToolPark, []Pt{{10, 11}}, false)
	c.updateLandValue()
	if c.At(10, 10).LandValue <= before+0.05 {
		t.Errorf("park: %v -> %v", before, c.At(10, 10).LandValue)
	}
}

func TestUnlocksAndCityHall(t *testing.T) {
	c := flat(20, 20)
	if p := c.Apply(ToolCityHall, []Pt{{2, 2}}, false); p.Err == "" {
		t.Fatal("city hall built before unlock")
	}
	c.PeakPop = 1000
	if p := c.Apply(ToolCityHall, []Pt{{2, 2}}, false); p.Err != "" {
		t.Fatal(p.Err)
	}
	c.Stats = Stats{Residents: 1000}
	if l := c.Forecast(); !near(l.Income[R], 1.05*0.1*1000*9) {
		t.Errorf("city hall tax bonus: %v", l.Income[R])
	}
}

func TestHistoryThins(t *testing.T) {
	c := flat(4, 4)
	for i := 0; i < 1000; i++ {
		c.Day = i * DaysPerMonth
		c.record()
	}
	if len(c.History) > maxHistory || c.History[0].Day != 0 || c.History[len(c.History)-1].Day != 999*DaysPerMonth {
		t.Errorf("history %d entries, first %d last %d", len(c.History), c.History[0].Day, c.History[len(c.History)-1].Day)
	}
}
