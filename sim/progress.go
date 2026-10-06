package sim

import "fmt"

// Milestone is a population goal with a reward.
type Milestone struct {
	Pop   int
	Grant float64
	Title string // what the settlement is called from here on
}

// Milestones in order. Buildings with a matching Unlock open up with them.
var Milestones = []Milestone{
	{100, 500, "hamlet"},
	{250, 1000, "village"},
	{1000, 5000, "town"},
	{2500, 8000, "city"},
	{5000, 15000, "big city"},
	{10000, 25000, "metropolis"},
}

// NextMilestone is the next goal, or false when all are reached.
func (c *City) NextMilestone() (Milestone, bool) {
	for _, m := range Milestones {
		if c.PeakPop < m.Pop {
			return m, true
		}
	}
	return Milestone{}, false
}

// Rank is the settlement's current title.
func (c *City) Rank() string {
	r := "settlement"
	for _, m := range Milestones {
		if c.PeakPop >= m.Pop {
			r = m.Title
		}
	}
	return r
}

// Unlocks lists buildings that open at population pop.
func Unlocks(pop int) []BuildingSpec {
	var out []BuildingSpec
	for _, b := range Buildings {
		if b.Unlock == pop {
			out = append(out, b)
		}
	}
	return out
}

// reachMilestones pays grants and announces unlocks the first time the
// population passes each milestone.
func (c *City) reachMilestones() {
	p := c.Stats.Residents
	if p <= c.PeakPop {
		return
	}
	for _, m := range Milestones {
		if c.PeakPop < m.Pop && p >= m.Pop {
			c.Funds += m.Grant
			c.Logf(Info, "%s is now a %s: %d people, +$%.0f grant", c.Name, m.Title, m.Pop, m.Grant)
			for _, b := range Unlocks(m.Pop) {
				c.Logf(Info, "unlocked: %s", b.Tool)
			}
		}
	}
	c.PeakPop = p
}

// Sample is one month of history for the charts.
type Sample struct {
	Day       int
	Pop, Jobs int
	Funds     float64
	Demand    [3]float64
}

const maxHistory = 480

// record adds this month to the history, thinning old entries when full so
// the whole life of the city still fits.
func (c *City) record() {
	c.History = append(c.History, Sample{c.Day, c.Stats.Residents, c.Stats.CommJobs + c.Stats.IndJobs, c.Funds, c.Demand})
	if len(c.History) > maxHistory {
		half := len(c.History) / 2
		var thin []Sample
		for i := 0; i < half; i += 2 {
			thin = append(thin, c.History[i])
		}
		c.History = append(thin, c.History[half:]...)
	}
}

// Goal is the line shown in the top bar.
func (c *City) Goal() string {
	if g, ok := c.scenarioGoal(); ok {
		return g
	}
	m, ok := c.NextMilestone()
	if !ok {
		return "every milestone reached"
	}
	g := fmt.Sprintf("goal: %d people", m.Pop)
	if u := Unlocks(m.Pop); len(u) > 0 {
		g += " → " + u[0].Tool.String()
	}
	return g
}
