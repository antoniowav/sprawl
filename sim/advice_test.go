package sim

import (
	"strings"
	"testing"
)

func TestInspectHolding(t *testing.T) {
	c := flat(10, 10)
	c.Apply(ToolZoneR, []Pt{{5, 5}}, false)
	c.updateUtilities()
	in := c.Inspect(5, 5)
	if !strings.HasPrefix(in.Title, "residential") || len(in.Holding) != 1 || in.Holding[0] != "no road next to it" {
		t.Fatalf("%+v", in)
	}
	c.Apply(ToolRoad, []Pt{{5, 4}}, false)
	c.updateUtilities()
	in = c.Inspect(5, 5)
	if !contains(in.Holding, "no power") {
		t.Errorf("holding %v", in.Holding)
	}
}

func TestAdvice(t *testing.T) {
	c := flat(10, 10)
	c.Power = Utility{Capacity: 100, Demand: 150}
	tips := c.Advice()
	if len(tips) == 0 || tips[0].Key != "power" {
		t.Errorf("tips %v", tips)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
