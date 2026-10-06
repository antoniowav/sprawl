package sim

import (
	"fmt"
	"math"
)

// Terrain height (SPEC §6.11). Every tile has a height 0..MaxHeight; water
// sits at 0 and the land rises from it, so rivers and lakes lie in valleys.
// Neighbouring tiles differ by at most 2 levels after generation.

const (
	MaxHeight     = 10
	gentle        = 1  // zones and buildings: neighbours within this many levels
	roadClimb     = 2  // roads can take steps up to this; steeper is a cliff
	CostTerraform = 25 // per level per tile
	viewBonus     = 0.05
	viewRadius    = 4
)

// mapRelief is the height range each map type uses.
var mapRelief = [mapTypeCount]float64{MapRiver: 4, MapCoast: 6, MapLakes: 4, MapIslands: 6, MapHighlands: 10}

// genHeights builds the height field after water and rock are placed.
func genHeights(c *City, m MapType) {
	f := newNoise(c.rng, c.W, c.H, 30)
	n := len(c.Tiles)
	raw := make([]float64, n)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			raw[y*c.W+x] = f.fbm(x, y)
		}
	}
	lo, hi := quantile(raw, 0.02), quantile(raw, 0.98)
	// Distance from water, so land climbs out of the valleys.
	dist := make([]int, n)
	queue := make([]int, 0, n)
	for i := range dist {
		dist[i] = -1
		if c.Tiles[i].Terrain == Water {
			dist[i] = 0
			queue = append(queue, i)
		}
	}
	for k := 0; k < len(queue); k++ {
		i := queue[k]
		x, y := i%c.W, i/c.W
		for _, d := range dirs4 {
			if c.In(x+d[0], y+d[1]) {
				j := (y+d[1])*c.W + x + d[0]
				if dist[j] < 0 {
					dist[j] = dist[i] + 1
					queue = append(queue, j)
				}
			}
		}
	}
	for i := range c.Tiles {
		t := &c.Tiles[i]
		if t.Terrain == Water {
			t.Height = 0
			continue
		}
		v := (raw[i] - lo) / math.Max(hi-lo, 1e-9)
		h := 1 + math.Round(math.Max(0, math.Min(1, v))*(mapRelief[m]-1))
		if dist[i] >= 0 {
			h = math.Min(h, 1+math.Floor(float64(dist[i]-1)/2)) // banks rise gently
		}
		if t.Terrain == Rock {
			h += 2 // outcrops stand above the land around them
		}
		t.Height = uint8(math.Max(1, math.Min(MaxHeight, h)))
	}
	limitSlopes(c, roadClimb)
}

// limitSlopes lowers tiles until no neighbour is more than max levels below.
func limitSlopes(c *City, max uint8) {
	for changed := true; changed; {
		changed = false
		for i := range c.Tiles {
			x, y := i%c.W, i/c.W
			for _, d := range dirs4 {
				if !c.In(x+d[0], y+d[1]) {
					continue
				}
				nb := c.At(x+d[0], y+d[1]).Height
				if c.Tiles[i].Height > nb+max {
					c.Tiles[i].Height = nb + max
					changed = true
				}
			}
		}
	}
}

// Slope is the largest height step from (x, y) to a neighbour.
func (c *City) Slope(x, y int) int {
	h := int(c.At(x, y).Height)
	s := 0
	for _, d := range dirs4 {
		if c.In(x+d[0], y+d[1]) {
			s = max(s, abs(h-int(c.At(x+d[0], y+d[1]).Height)))
		}
	}
	return s
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// view is how far a tile stands above the land around it, 0..3.
func (c *City) view(x, y int) float64 {
	sum, n := 0, 0
	for dy := -viewRadius; dy <= viewRadius; dy += 2 {
		for dx := -viewRadius; dx <= viewRadius; dx += 2 {
			if c.In(x+dx, y+dy) {
				sum += int(c.At(x+dx, y+dy).Height)
				n++
			}
		}
	}
	return math.Max(0, math.Min(3, float64(c.At(x, y).Height)-float64(sum)/float64(n)))
}

// Terraforming tools.
const (
	TerraRaise = iota
	TerraLower
	TerraLevel
)

// TerraPlan is what a terraform action would do.
type TerraPlan struct {
	Tiles []Pt
	Cost  float64
	Err   string
}

// PlanTerraform works out raising, lowering or levelling the tiles. Level
// brings every tile to the height of the first one.
func (c *City) PlanTerraform(op int, pts []Pt) TerraPlan {
	var p TerraPlan
	if len(pts) == 0 {
		return p
	}
	target := int(c.At(pts[0].X, pts[0].Y).Height)
	for _, pt := range pts {
		if !c.In(pt.X, pt.Y) {
			continue
		}
		t := c.At(pt.X, pt.Y)
		if t.Kind != Empty || t.Line || t.Terrain == Rock {
			continue // only open ground (and water) can be reshaped
		}
		steps := 0
		switch op {
		case TerraRaise:
			if t.Height < MaxHeight {
				steps = 1
			}
		case TerraLower:
			if t.Height > 0 && t.Terrain != Water {
				steps = 1
			}
		case TerraLevel:
			steps = abs(int(t.Height) - target)
			if t.Terrain == Water && target > 0 {
				steps = target
			}
		}
		if steps == 0 {
			continue
		}
		p.Tiles = append(p.Tiles, pt)
		p.Cost += float64(steps * CostTerraform)
	}
	switch {
	case len(p.Tiles) == 0:
		p.Err = "nothing to reshape here (only open ground)"
	case c.Funds < 0:
		p.Err = "in debt: only bulldozing allowed"
	case p.Cost > c.Funds:
		p.Err = fmt.Sprintf("not enough funds ($%.0f needed)", p.Cost)
	}
	return p
}

// Terraform applies a plan made by PlanTerraform. Lowering ground next to
// water down to the water floods it; raising water makes land.
func (c *City) Terraform(op int, pts []Pt) TerraPlan {
	p := c.PlanTerraform(op, pts)
	if p.Err != "" {
		return p
	}
	target := c.At(pts[0].X, pts[0].Y).Height
	for _, pt := range p.Tiles {
		t := c.At(pt.X, pt.Y)
		switch op {
		case TerraRaise:
			if t.Terrain == Water {
				t.Terrain, t.Height = Land, 1
			} else {
				t.Height++
			}
		case TerraLower:
			t.Height--
		case TerraLevel:
			t.Height = target
			if t.Terrain == Water && target > 0 {
				t.Terrain = Land
			}
		}
		if t.Terrain == Trees {
			t.Terrain = Land
		}
		if t.Height == 0 && t.Terrain != Water && c.nextToWater(pt.X, pt.Y) {
			t.Terrain = Water
		}
		if t.Height == 0 && t.Terrain != Water {
			t.Height = 1 // dry land never sits at water level
		}
	}
	c.Funds -= p.Cost
	c.lvDirty = true
	return p
}

func (c *City) nextToWater(x, y int) bool {
	for _, d := range dirs4 {
		if c.In(x+d[0], y+d[1]) && c.At(x+d[0], y+d[1]).Terrain == Water {
			return true
		}
	}
	return false
}
