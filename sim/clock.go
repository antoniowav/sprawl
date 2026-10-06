package sim

// Stats are city-wide aggregates, recomputed daily.
type Stats struct {
	Residents int
	CommJobs  int
	IndJobs   int
}

// Workforce is the share of residents who work.
func (s Stats) Workforce() float64 { return workforceShare * float64(s.Residents) }

// Tick advances the simulation by one tick. It returns the tiles whose
// level changed, so the renderer can animate them.
func (c *City) Tick() []Pt {
	if c.Bankrupt {
		return nil
	}
	if c.Ticks%TicksPerDay == 0 {
		if c.Day > 0 && c.Day%DaysPerMonth == 0 {
			c.monthly()
		}
		c.daily()
	}
	changed := c.growStripe(int(c.Ticks % TicksPerDay))
	c.Ticks++
	c.Day = int(c.Ticks / TicksPerDay)
	return changed
}

// daily runs once at the start of every day.
func (c *City) daily() {
	c.updateUtilities()
	if c.Day%landValueEveryNd == 0 || c.lvDirty {
		c.updateTraffic()
		c.updateLandValue()
		c.lvDirty = false
	}
	c.updateStats()
	c.updateDemand()
	c.scenarioDaily()
}

func (c *City) updateStats() {
	var s Stats
	for i := range c.Tiles {
		t := &c.Tiles[i]
		if !t.IsZone() || t.Level == 0 {
			continue
		}
		n := 0
		switch t.Kind {
		case ZoneR:
			n = resCap[t.Level]
		case ZoneC:
			n = comCap[t.Level]
		case ZoneI:
			n = indCap[t.Level]
		}
		if !t.Powered {
			n /= 2
		}
		switch t.Kind {
		case ZoneR:
			s.Residents += n
		case ZoneC:
			s.CommJobs += n
		case ZoneI:
			s.IndJobs += n
		}
	}
	c.Stats = s
	c.reachMilestones()
}
