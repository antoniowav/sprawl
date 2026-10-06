package sim

import (
	"fmt"
	"os"
	"testing"
)

// TestBalanceTrace prints a growth curve; run with BALANCE=1.
func TestBalanceTrace(t *testing.T) {
	if os.Getenv("BALANCE") == "" {
		t.Skip()
	}
	c := flat(60, 60)
	// A grid: roads every 3 rows, zones between (R left, C middle, I right).
	for y := 0; y < 60; y += 3 {
		c.Apply(ToolRoad, RectPts(Pt{0, y}, Pt{59, y}), false)
	}
	c.Funds = 1e9
	c.Apply(ToolZoneR, RectPts(Pt{0, 0}, Pt{29, 59}), false)
	c.Apply(ToolZoneC, RectPts(Pt{30, 0}, Pt{39, 59}), false)
	c.Apply(ToolZoneI, RectPts(Pt{40, 0}, Pt{59, 59}), false)
	for d := 0; d <= 2000; d++ {
		for i := 0; i < TicksPerDay; i++ {
			c.Tick()
		}
		if d%200 == 0 {
			lv := [4]int{}
			for _, tl := range c.Tiles {
				if tl.IsZone() {
					lv[tl.Level]++
				}
			}
			fmt.Printf("day %4d pop %5d C %4d I %4d  D %+.2f %+.2f %+.2f  levels %v\n", d, c.Stats.Residents, c.Stats.CommJobs, c.Stats.IndJobs, c.Demand[0], c.Demand[1], c.Demand[2], lv)
		}
	}
}
