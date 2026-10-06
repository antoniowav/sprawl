package sim

// growStripe evaluates the zone tiles whose index falls in this tick's
// quarter of the map, so each tile is evaluated once a day and the work is
// spread evenly over ticks.
func (c *City) growStripe(stripe int) []Pt {
	var changed []Pt
	for i := stripe; i < len(c.Tiles); i += TicksPerDay {
		t := &c.Tiles[i]
		if !t.IsZone() {
			continue
		}
		x, y := i%c.W, i/c.W
		if d := c.growTile(t, x, y); d != 0 {
			t.Level = uint8(int(t.Level) + d)
			t.Variant = uint8(c.rng.IntN(4))
			c.lvDirty = true
			changed = append(changed, Pt{x, y})
		}
	}
	return changed
}

// MaxLevel is the highest density land value allows.
func MaxLevel(lv float32) uint8 {
	switch {
	case lv < lvLowMax:
		return 1
	case lv < lvMidMax:
		return 2
	}
	return 3
}

// Score is the growth score of a zone tile (SPEC §6.3).
func (c *City) Score(t *Tile) float64 {
	z := int(t.Kind - ZoneR)
	cover := float64(t.Cover[0]+t.Cover[1]+t.Cover[2]) / 3
	return c.Demand[z] + lvWeight*(float64(t.LandValue)-0.5) + coverWeight*(cover-0.5)
}

// growTile returns +1, -1 or 0.
func (c *City) growTile(t *Tile, x, y int) int {
	access := c.RoadAccess(x, y)
	if !access {
		if t.Level > 0 && c.rng.Float64() < noAccessDecay {
			return -1
		}
		return 0
	}
	s := c.Score(t)
	maxLvl := MaxLevel(t.LandValue)
	canGrow := t.Powered && (t.Level == 0 || t.Watered) && t.Level < maxLvl
	// Demand alone fills empty lots; land value and services decide whether
	// a building upgrades.
	gs := s
	if t.Level == 0 {
		gs = c.Demand[t.Kind-ZoneR]
	}
	if canGrow && gs > 0 && c.rng.Float64() < min(growMax, growScale*gs) {
		return 1
	}
	if t.Level > 0 && t.UnpoweredDays >= unpoweredDecayDays && c.rng.Float64() < noAccessDecay {
		return -1
	}
	if t.Level > 0 && (s < decayFloor || t.Level > maxLvl) {
		if c.rng.Float64() < min(decayMax, decayScale*max(-s, 0.2)) {
			return -1
		}
	}
	return 0
}

// RoadAccess: a road on one of the 4 neighbours (SPEC §12: adjacent only).
func (c *City) RoadAccess(x, y int) bool {
	for _, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		if c.In(x+d[0], y+d[1]) && c.At(x+d[0], y+d[1]).Kind == Road {
			return true
		}
	}
	return false
}
