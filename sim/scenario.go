package sim

// Scenario is a fixed map with a goal, a deadline and scripted events.
type Scenario struct {
	ID    string
	Name  string
	Brief string
	Map   MapType
	Size  int
	Seed  int64
	Funds float64
}

// NewCity generates the scenario's map.
func (s *Scenario) NewCity() *City {
	c := NewMap(s.Size, s.Size, s.Seed, s.Map)
	c.Funds = s.Funds
	return c
}

// Begin attaches the scenario to a freshly generated city.
func (s *Scenario) Begin(c *City) {
	c.ScenarioID = s.ID
	c.Logf(Info, "scenario: %s", s.Name)
}

// Scenarios lists the built-in scenarios.
var Scenarios []*Scenario
