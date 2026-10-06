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
			if t.Anchor >= 0 {
				changed = append(changed, c.split(t.Anchor)...)
			}
			t.Level = uint8(int(t.Level) + d)
			t.Variant = uint8(c.rng.IntN(4))
			c.lvDirty = true
			changed = append(changed, Pt{x, y})
			continue
		}
		if t.Level == 3 && t.Anchor < 0 && c.canMerge(x, y) && c.rng.Float64() < mergeChance {
			changed = append(changed, c.merge(x, y)...)
		}
	}
	return changed
}

// Big buildings: a 2×2 square of high-density lots of one zone merges into
// a single large building (SPEC §6.10).
const (
	mergeChance = 0.03 // per day, once a square qualifies
	mergeLV     = 0.75
	mergeDemand = 0.2
	bigBonus    = 1.25 // capacity of each tile in a big building
)

// canMerge reports whether the 2×2 square with top-left (x, y) qualifies.
func (c *City) canMerge(x, y int) bool {
	if !c.In(x+1, y+1) {
		return false
	}
	t0 := c.At(x, y)
	lv := 0.0
	for _, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		t := c.At(x+d[0], y+d[1])
		if t.Kind != t0.Kind || t.Level != 3 || t.Anchor >= 0 || !t.Powered || !t.Watered {
			return false
		}
		lv += float64(t.LandValue)
	}
	return lv/4 >= mergeLV && c.Demand[t0.Kind-ZoneR] > mergeDemand
}

func (c *City) merge(x, y int) []Pt {
	a := int32(y*c.W + x)
	v := uint8(c.rng.IntN(2))
	var pts []Pt
	for _, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		t := c.At(x+d[0], y+d[1])
		t.Anchor, t.Variant = a, v
		pts = append(pts, Pt{x + d[0], y + d[1]})
	}
	c.lvDirty = true
	return pts
}

// split turns a big building back into four separate lots.
func (c *City) split(a int32) []Pt {
	pts := c.footprint(a)
	for _, p := range pts {
		c.At(p.X, p.Y).Anchor = -1
	}
	return pts
}

// IsBig reports whether a zone tile is part of a 2×2 building.
func (t *Tile) IsBig() bool { return t.IsZone() && t.Anchor >= 0 }

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
	cover := float64(t.Cover[0]+t.Cover[1]+t.Cover[2]+t.Cover[3]) / 4
	return c.Demand[z] + lvWeight*(float64(t.LandValue)-0.5) + coverWeight*(cover-0.5) - c.commutePenalty(t)
}

// growTile returns +1, -1 or 0.
func (c *City) growTile(t *Tile, x, y int) int {
	access := c.RoadAccess(x, y)
	if t.Anchor >= 0 && !access { // a big building is reached through any of its tiles
		for _, p := range c.footprint(t.Anchor) {
			access = access || c.RoadAccess(p.X, p.Y)
		}
	}
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
		gs = c.Demand[t.Kind-ZoneR] - c.commutePenalty(t)
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

// ForceBig makes the 2×2 square at (x, y) a big building if it holds four
// lots of one zone (any level becomes high density). For demos and tools.
func (c *City) ForceBig(x, y int) bool {
	if !c.In(x+1, y+1) {
		return false
	}
	k := c.At(x, y).Kind
	for _, p := range RectPts(Pt{x, y}, Pt{x + 1, y + 1}) {
		if t := c.At(p.X, p.Y); t.Kind != k || !t.IsZone() || t.Anchor >= 0 {
			return false
		}
	}
	for _, p := range RectPts(Pt{x, y}, Pt{x + 1, y + 1}) {
		c.At(p.X, p.Y).Level = 3
	}
	c.merge(x, y)
	return true
}
