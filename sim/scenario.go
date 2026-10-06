package sim

import "fmt"

// Scenario is a fixed map with goals, a deadline and scripted events.
type Scenario struct {
	ID       string
	Name     string
	Brief    string
	Map      MapType
	Size     int
	Seed     int64
	Funds    float64
	Deadline int // day by which every goal must be met
	Goals    []Goal
	Events   []ScenarioEvent
}

// Goal is one condition a scenario needs, with progress for the HUD.
type Goal struct {
	Label    string
	Target   float64
	Progress func(c *City) float64
}

// Met reports whether the goal is reached.
func (g Goal) Met(c *City) bool { return g.Progress(c) >= g.Target }

// ScenarioEvent fires once on its day.
type ScenarioEvent struct {
	Day   int
	Msg   string
	Apply func(c *City)
}

// Scenario results stored on the city.
const (
	ScenarioRunning = iota
	ScenarioWon
	ScenarioLost
)

// Boost raises a zone's demand until a day (0 = for good).
type Boost struct {
	Amount [3]float64
	Until  int
}

func year(y int) int { return (y - 1) * DaysPerMonth * MonthsPerYear }

// Scenarios lists the built-in scenarios.
var Scenarios = []*Scenario{
	{
		ID: "boomtown", Name: "Boomtown", Map: MapRiver, Size: 128, Seed: 4242, Funds: 25000,
		Brief: "Northfield Motors is building a factory here. Have 2,000 workers and a budget " +
			"in the black by January of year 5.",
		Deadline: year(5),
		Goals: []Goal{
			{"workers", 2000, func(c *City) float64 { return c.Stats.Workforce() }},
			{"monthly budget", 0, func(c *City) float64 {
				if c.Day < DaysPerMonth {
					return -1
				}
				return c.LastMonth.Net
			}},
		},
		Events: []ScenarioEvent{
			{0, "Northfield Motors announces a factory: industry wants to move in", func(c *City) {
				c.Boosts = append(c.Boosts, Boost{Amount: [3]float64{0, 0, 0.3}})
			}},
			{year(2) + 6*DaysPerMonth, "The factory opens early: everyone wants homes and workers now", func(c *City) {
				c.Boosts = append(c.Boosts, Boost{Amount: [3]float64{0.25, 0.1, 0.2}})
			}},
			{year(4), "One year left: 2,000 workers and a positive budget by January", nil},
		},
	},
	{
		ID: "island", Name: "Island", Map: MapIslands, Size: 128, Seed: 77, Funds: 12000,
		Brief: "Little flat land, little money. Reach 3,000 people by January of year 11. " +
			"Bridges cost $60 a tile.",
		Deadline: year(11),
		Goals: []Goal{
			{"people", 3000, func(c *City) float64 { return float64(c.Stats.Residents) }},
		},
		Events: []ScenarioEvent{
			{0, "Welcome to the islands: build bridges to reach more land", nil},
			{year(3), "Tourism boom: for two years visitors want shops everywhere", func(c *City) {
				c.Boosts = append(c.Boosts, Boost{Amount: [3]float64{0.1, 0.35, 0}, Until: year(5)})
			}},
			{year(5), "The tourists have moved on", nil},
		},
	},
}

// ScenarioByID finds a built-in scenario.
func ScenarioByID(id string) *Scenario {
	for _, s := range Scenarios {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// NewCity generates the scenario's map.
func (s *Scenario) NewCity() *City {
	c := NewMap(s.Size, s.Size, s.Seed, s.Map)
	c.Funds = s.Funds
	return c
}

// Begin attaches the scenario to a freshly generated city and fires its
// opening events.
func (s *Scenario) Begin(c *City) {
	c.ScenarioID = s.ID
	c.Logf(Info, "scenario %s: %s", s.Name, s.Brief)
	c.scenarioDaily()
}

// scenarioDaily fires due events and decides win or loss.
func (c *City) scenarioDaily() {
	s := ScenarioByID(c.ScenarioID)
	if s == nil || c.ScenarioResult != ScenarioRunning {
		return
	}
	for i, e := range s.Events {
		if c.ScenarioEvents&(1<<i) == 0 && c.Day >= e.Day {
			c.ScenarioEvents |= 1 << i
			c.Logf(Warn, "%s", e.Msg)
			if e.Apply != nil {
				e.Apply(c)
			}
		}
	}
	won := true
	for _, g := range s.Goals {
		won = won && g.Met(c)
	}
	switch {
	case won && c.Day > 0:
		c.ScenarioResult = ScenarioWon
		c.Logf(Info, "scenario complete: %s", s.Name)
	case c.Day >= s.Deadline:
		c.ScenarioResult = ScenarioLost
		c.Logf(Err, "time's up: %s not completed", s.Name)
	}
}

// scenarioGoal is the top-bar line while a scenario runs.
func (c *City) scenarioGoal() (string, bool) {
	s := ScenarioByID(c.ScenarioID)
	if s == nil {
		return "", false
	}
	switch c.ScenarioResult {
	case ScenarioWon:
		return s.Name + " complete", true
	case ScenarioLost:
		return s.Name + ": time's up", true
	}
	g := s.Goals[0]
	left := (s.Deadline - c.Day) / DaysPerMonth
	return fmt.Sprintf("%s %s/%s · %d months left", g.Label, commasInt(int(g.Progress(c))), commasInt(int(g.Target)), left), true
}

func commasInt(n int) string {
	s := fmt.Sprint(n)
	out := ""
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 && s[i-1] != '-' {
			out += ","
		}
		out += string(r)
	}
	return out
}

// demandBoost sums active scenario boosts.
func (c *City) demandBoost() [3]float64 {
	var b [3]float64
	for _, x := range c.Boosts {
		if x.Until == 0 || c.Day < x.Until {
			for z := range b {
				b[z] += x.Amount[z]
			}
		}
	}
	return b
}
