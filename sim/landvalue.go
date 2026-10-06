package sim

import "math"

// updateLandValue recomputes service coverage, pollution and land value
// for every tile (SPEC §6.6).
func (c *City) updateLandValue() {
	for i := range c.Tiles {
		c.Tiles[i].Pollution = 0
		c.Tiles[i].Cover = [4]float32{}
	}
	addPollution := func(t *Tile, v float64) { t.Pollution = float32(min(1, float64(t.Pollution)+v)) }
	if len(c.lvBonus) != len(c.Tiles) {
		c.lvBonus = make([]float32, len(c.Tiles))
	}
	clear(c.lvBonus)
	for i := range c.Tiles {
		t := &c.Tiles[i]
		x, y := i%c.W, i/c.W
		switch {
		case t.Kind == ZoneI && t.Level > 0:
			c.spread(x, y, pollIndustryRad, pollIndustry*float64(t.Level), addPollution)
		case t.IsBuilding() && int(t.Anchor) == i:
			b := spec(t.Kind)
			// Measure from the building's centre.
			cx, cy := float64(x)+float64(b.Size)/2, float64(y)+float64(b.Size)/2
			if b.Smog > 0 {
				c.spreadFrom(cx, cy, b.SmogRad, b.Smog, addPollution)
			}
			if b.LVBonus > 0 {
				c.spreadIndex(cx, cy, b.LVRadius, b.LVBonus, func(j int, v float64) {
					c.lvBonus[j] = float32(min(0.3, float64(c.lvBonus[j])+v))
				})
			}
			if s := b.Service; s >= 0 {
				strength := 1.0
				if !t.Powered {
					strength = 0.5
				}
				// Full strength near the building: 1.25·(1 − d/r), capped at 1.
				rad := serviceRadius[s]
				if b.ServiceRadius > 0 {
					rad = b.ServiceRadius
				}
				c.spreadFrom(cx, cy, rad, coverPlateau*strength, func(t *Tile, v float64) {
					t.Cover[s] = float32(min(1, float64(t.Cover[s])+v))
				})
			}
		}
	}
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			t := c.At(x, y)
			lv := lvBase +
				lvWater*b2f(c.near(x, y, lvWaterRadius, func(t *Tile) bool { return t.Terrain == Water })) +
				lvNeighbourhood*c.neighbourhood(x, y) -
				lvPollution*float64(t.Pollution)
			lv += 0.10*float64(t.Cover[0]) + 0.10*float64(t.Cover[1]) + 0.15*float64(t.Cover[2]) + 0.10*float64(t.Cover[3])
			if len(c.noise) == len(c.Tiles) {
				lv -= float64(c.noise[y*c.W+x])
			}
			lv += float64(c.lvBonus[y*c.W+x])
			t.LandValue = float32(clamp(lv, 0, 1))
		}
	}
}

// spread adds v·(1 − d/r) to tiles within radius r of tile (x, y).
func (c *City) spread(x, y, r int, v float64, add func(*Tile, float64)) {
	c.spreadFrom(float64(x)+0.5, float64(y)+0.5, r, v, add)
}

// spreadFrom is spread from a point in tile units (tile centres are at +0.5).
func (c *City) spreadFrom(px, py float64, r int, v float64, add func(*Tile, float64)) {
	c.spreadIndex(px, py, r, v, func(i int, v float64) { add(&c.Tiles[i], v) })
}

// spreadIndex is spreadFrom with tile indexes.
func (c *City) spreadIndex(px, py float64, r int, v float64, add func(int, float64)) {
	x0, y0 := int(px)-r-1, int(py)-r-1
	for y := y0; y <= y0+2*r+2; y++ {
		for x := x0; x <= x0+2*r+2; x++ {
			if !c.In(x, y) {
				continue
			}
			d := math.Hypot(float64(x)+0.5-px, float64(y)+0.5-py)
			if d < float64(r) {
				add(y*c.W+x, v*(1-d/float64(r)))
			}
		}
	}
}

func (c *City) near(x, y, r int, f func(*Tile) bool) bool {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if c.In(x+dx, y+dy) && f(c.At(x+dx, y+dy)) {
				return true
			}
		}
	}
	return false
}

// neighbourhood is the average developed level/3 of R and C zones within
// 2 tiles. Industry is left out: it lowers value through pollution instead.
func (c *City) neighbourhood(x, y int) float64 {
	sum, n := 0.0, 0
	r := lvNeighbourRad
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if (dx != 0 || dy != 0) && c.In(x+dx, y+dy) {
				if t := c.At(x+dx, y+dy); t.Kind == ZoneR || t.Kind == ZoneC {
					sum += float64(t.Level) / 3
					n++
				}
			}
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
