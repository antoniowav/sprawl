package sim

import (
	"bytes"
	"testing"
)

func TestScenarioEventsAndLoss(t *testing.T) {
	s := ScenarioByID("island")
	c := s.NewCity()
	s.Begin(c)
	if c.ScenarioEvents != 1 || c.Funds != 12000 || c.Map != MapIslands {
		t.Fatalf("begin: events %b funds %v", c.ScenarioEvents, c.Funds)
	}
	c.Day = year(3)
	c.scenarioDaily()
	if c.demandBoost()[C] != 0.35 {
		t.Errorf("tourism boost %v", c.demandBoost())
	}
	c.scenarioDaily()
	if c.ScenarioEvents != 0b11 {
		t.Errorf("events fired twice or missed: %b", c.ScenarioEvents)
	}
	c.Day = year(5)
	if c.demandBoost()[C] != 0 {
		t.Error("boost should expire")
	}
	c.Day = s.Deadline
	c.scenarioDaily()
	if c.ScenarioResult != ScenarioLost {
		t.Error("deadline passed without a loss")
	}
}

func TestScenarioWin(t *testing.T) {
	s := ScenarioByID("boomtown")
	c := s.NewCity()
	s.Begin(c)
	if c.demandBoost()[I] != 0.3 {
		t.Fatal("factory boost missing")
	}
	c.Day = 400
	c.Stats = Stats{Residents: 4000}
	c.LastMonth.Net = 10
	c.scenarioDaily()
	if c.ScenarioResult != ScenarioWon {
		t.Fatalf("result %d", c.ScenarioResult)
	}
	if g := c.Goal(); g != "Boomtown complete" {
		t.Errorf("goal line %q", g)
	}
}

func TestScenarioSaved(t *testing.T) {
	s := ScenarioByID("boomtown")
	c := s.NewCity()
	s.Begin(c)
	var buf bytes.Buffer
	if err := c.Save(&buf); err != nil {
		t.Fatal(err)
	}
	d, err := Load(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if d.ScenarioID != "boomtown" || len(d.Boosts) != 1 || d.ScenarioEvents != 1 {
		t.Errorf("scenario state lost: %q %v %b", d.ScenarioID, d.Boosts, d.ScenarioEvents)
	}
}
