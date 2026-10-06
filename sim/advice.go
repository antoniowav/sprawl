package sim

import "fmt"

// Inspect describes a tile for the building inspector: a title, facts,
// and the reasons it isn't growing (empty when nothing holds it back).
type Inspection struct {
	Title   string
	Facts   []string
	Holding []string
}

// Inspect builds the inspector panel for tile (x, y).
func (c *City) Inspect(x, y int) Inspection {
	t := c.At(x, y)
	var in Inspection
	cover := func() {
		in.Facts = append(in.Facts, fmt.Sprintf("police %.0f%%  fire %.0f%%  school %.0f%%  health %.0f%%",
			100*t.Cover[0], 100*t.Cover[1], 100*t.Cover[2], 100*t.Cover[3]))
	}
	defer func() {
		in.Facts = append(in.Facts, fmt.Sprintf("height %d, slope %d", t.Height, c.Slope(x, y)))
	}()
	switch {
	case t.IsZone():
		z := t.Kind - ZoneR
		names := [3]string{"residential", "commercial", "industrial"}
		levels := [4]string{"empty lot", "low density", "medium density", "high density"}
		in.Title = fmt.Sprintf("%s · %s", names[z], levels[t.Level])
		if t.IsBig() {
			in.Title = [3]string{"residential tower", "big commercial", "industrial complex"}[z] + " (2×2)"
		}
		switch t.Kind {
		case ZoneR:
			in.Facts = append(in.Facts, fmt.Sprintf("%d residents", c.capacity(t)))
			if t.Level > 0 {
				switch {
				case t.Commute < 0:
					in.Facts = append(in.Facts, "commute: no road to any job")
				case t.Commute > 0:
					in.Facts = append(in.Facts, fmt.Sprintf("commute: %d tiles", t.Commute))
				}
			}
		case ZoneC:
			in.Facts = append(in.Facts, fmt.Sprintf("%d jobs", c.capacity(t)))
		case ZoneI:
			in.Facts = append(in.Facts, fmt.Sprintf("%d jobs", c.capacity(t)))
		}
		in.Facts = append(in.Facts, fmt.Sprintf("land value %.2f  smog %.0f%%", t.LandValue, 100*t.Pollution))
		cover()
		in.Holding = c.holding(t, x, y)
	case t.IsBuilding():
		b := spec(t.Kind)
		in.Title = b.Tool.String()
		if b.Note != "" {
			in.Facts = append(in.Facts, b.Note)
		}
		in.Facts = append(in.Facts, fmt.Sprintf("upkeep $%.0f/month", b.Upkeep))
		if b.PowerUse > 0 && !c.Tiles[t.Anchor].Powered {
			in.Holding = append(in.Holding, "no power: connect it with a road or power line")
		}
		if b.WaterUse > 0 && !c.Tiles[t.Anchor].Watered {
			in.Holding = append(in.Holding, "no water: lay a pipe next to it")
		}
	case t.Kind == Road:
		in.Title = "road"
		if t.Terrain == Water {
			in.Title = "bridge"
		}
		in.Facts = append(in.Facts, fmt.Sprintf("%d commuters a day, %.0f%% of capacity", t.Traffic, 100*t.Congestion()))
	default:
		in.Title = [...]string{"grass", "water", "trees", "rock"}[t.Terrain]
		in.Facts = append(in.Facts, fmt.Sprintf("land value %.2f", t.LandValue))
		cover()
	}
	return in
}

// holding lists what keeps a zone tile from growing, in plain words.
func (c *City) holding(t *Tile, x, y int) []string {
	var h []string
	if !c.RoadAccess(x, y) {
		return []string{"no road next to it"}
	}
	if !t.Powered {
		h = append(h, "no power")
	}
	if t.Level >= 1 && !t.Watered && t.Level < 3 {
		h = append(h, "needs water to grow taller (a pipe beside it)")
	}
	z := t.Kind - ZoneR
	if c.Demand[z] <= 0 {
		h = append(h, fmt.Sprintf("low demand for %s", [3]string{"homes", "shops", "industry"}[z]))
	}
	if t.Level > 0 && t.Level >= MaxLevel(t.LandValue) && t.Level < 3 {
		h = append(h, "land value too low to grow taller: add parks or services")
	}
	if c.commutePenalty(t) > 0 {
		if t.Commute < 0 {
			h = append(h, "no road leads to any job")
		} else {
			h = append(h, "long commute: bring jobs closer")
		}
	}
	return h
}

// Tip is advice for the player, keyed so it isn't repeated too often.
type Tip struct {
	Key string
	Msg string
}

// Advice checks the city for common problems, most urgent first.
func (c *City) Advice() []Tip {
	var tips []Tip
	if c.Power.Demand > c.Power.Capacity && c.Power.Capacity > 0 {
		tips = append(tips, Tip{"power", "Brownout: demand is over capacity. Build another power plant, wind turbines or solar."})
	}
	if c.Water.Demand > c.Water.Capacity && c.Water.Capacity > 0 {
		tips = append(tips, Tip{"water", "Water shortage: add a pump by the water or a water tower."})
	}
	if c.LastMonth.Net < 0 && c.Day > 3*DaysPerMonth && c.Funds < 5000 {
		tips = append(tips, Tip{"money", "Money is running low and the city loses money each month. Raise taxes a little (:tax 11) or cut services you don't need."})
	}
	noJobs, unpowered, dry := 0, 0, 0
	maxCong := 0.0
	for i := range c.Tiles {
		t := &c.Tiles[i]
		switch {
		case t.Kind == ZoneR && t.Level > 0 && t.Commute < 0:
			noJobs++
		case t.Kind == Road:
			maxCong = max(maxCong, t.Congestion())
		}
		if t.IsZone() && t.Level == 0 && !t.Powered && c.RoadAccess(i%c.W, i/c.W) {
			unpowered++
		}
		if t.IsZone() && t.Level == 1 && !t.Watered {
			dry++
		}
	}
	if unpowered >= 10 {
		tips = append(tips, Tip{"lots", fmt.Sprintf("%d lots by a road have no power, so nothing can be built on them.", unpowered)})
	}
	if noJobs >= 5 && c.Stats.CommJobs+c.Stats.IndJobs > 0 {
		tips = append(tips, Tip{"commute", fmt.Sprintf("%d homes can't reach any job by road. Connect their streets to your shops and industry.", noJobs)})
	}
	if maxCong > 1.5 {
		tips = append(tips, Tip{"traffic", "Roads are jammed. Add a parallel road, or bus stops with a bus depot."})
	}
	if dry >= 15 && c.Stats.Residents > 300 {
		tips = append(tips, Tip{"dry", "Many low-density buildings have no water, so they can't grow taller. Lay pipes beside them."})
	}
	if c.Stats.Residents > 600 && !c.has(School) && !c.has(University) {
		tips = append(tips, Tip{"school", "No school yet: education raises land value, and taller buildings need it."})
	}
	if c.Demand[R] > 0.6 && c.Stats.Residents > 50 {
		tips = append(tips, Tip{"demandR", "People want to move in: zone more homes."})
	}
	if c.Demand[I] > 0.6 && c.Day > DaysPerMonth {
		tips = append(tips, Tip{"demandI", "Industry wants to grow: zone more industrial land."})
	}
	if c.Demand[C] > 0.6 && c.Stats.Residents > 100 {
		tips = append(tips, Tip{"demandC", "Residents want shops: zone commercial land."})
	}
	return tips
}

// capacity is what one zone tile holds (people or jobs), big-building bonus
// included; a big building's tiles are summed.
func (c *City) capacity(t *Tile) int {
	caps := map[Kind][4]int{ZoneR: resCap, ZoneC: comCap, ZoneI: indCap}[t.Kind]
	n := caps[t.Level]
	if t.IsBig() {
		return 4 * int(float64(n)*bigBonus)
	}
	return n
}
