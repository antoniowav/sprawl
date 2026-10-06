package sim

import (
	"math"
	"math/rand/v2"
	"sort"
)

// Target shares of the map; the generator picks thresholds to hit them.
const (
	lakeShare = 0.05
	treeShare = 0.15
)

// MapType selects the terrain generator.
type MapType uint8

const (
	MapRiver MapType = iota
	MapCoast
	MapLakes
	MapIslands
	MapHighlands
	mapTypeCount
)

// MapTypes in menu order.
var MapTypes = []MapType{MapRiver, MapCoast, MapLakes, MapIslands, MapHighlands}

func (m MapType) String() string {
	return [...]string{"river valley", "coast", "lakes", "islands", "highlands"}[m]
}

func generateTerrain(c *City, m MapType) {
	switch m {
	case MapCoast:
		genCoast(c)
	case MapLakes:
		genLakes(c, 0.14, false)
	case MapIslands:
		genIslands(c)
	case MapHighlands:
		genLakes(c, 0.04, true)
		genRock(c)
	default:
		genLakes(c, lakeShare, true)
	}
	plantForest(c)
	genHeights(c, m)
	c.Start = findStart(c)
}

// genLakes floods the top share of a noise field and optionally adds a river.
func genLakes(c *City, share float64, river bool) {
	lakes := newNoise(c.rng, c.W, c.H, 24)

	n := len(c.Tiles)
	lake := make([]float64, n)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			lake[y*c.W+x] = lakes.fbm(x, y)
		}
	}
	cut := quantile(lake, 1-share)
	for i, v := range lake {
		if v >= cut {
			c.Tiles[i].Terrain = Water
		}
	}
	if river {
		carveRiver(c)
	}
}

// genCoast puts the sea along one edge with a ragged shoreline.
func genCoast(c *City) {
	shore := newNoise(c.rng, c.W, c.H, 16)
	edge := c.rng.IntN(4)
	depth := float64(min(c.W, c.H)) * 0.22
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			d := [4]float64{float64(y), float64(c.W - 1 - x), float64(c.H - 1 - y), float64(x)}[edge]
			if d+(shore.fbm(x, y)-0.875)*18 < depth {
				c.At(x, y).Terrain = Water
			}
		}
	}
	genLakes(c, 0.02, false)
}

// genIslands keeps the top 40% of a noise field as land: a scatter of
// islands of different sizes, with open sea around the map edge.
func genIslands(c *City) {
	f := newNoise(c.rng, c.W, c.H, 11)
	n := len(c.Tiles)
	v := make([]float64, n)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			ex := math.Min(float64(x), float64(c.W-1-x)) / float64(c.W) * 4
			ey := math.Min(float64(y), float64(c.H-1-y)) / float64(c.H) * 4
			v[y*c.W+x] = f.fbm(x, y) - 0.6*math.Max(0, 1-math.Min(ex, ey))
		}
	}
	cut := quantile(v, 0.6)
	for i, val := range v {
		if val < cut {
			c.Tiles[i].Terrain = Water
		}
	}
}

// genRock raises rocky outcrops nothing can be built on.
func genRock(c *City) {
	f := newNoise(c.rng, c.W, c.H, 14)
	n := len(c.Tiles)
	v := make([]float64, n)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			v[y*c.W+x] = f.fbm(x, y) + 0.15*c.rng.Float64()
		}
	}
	cut := quantile(v, 0.82)
	for i, val := range v {
		if val >= cut && c.Tiles[i].Terrain == Land {
			c.Tiles[i].Terrain = Rock
		}
	}
}

// plantForest adds tree cover on open land.
func plantForest(c *City) {
	forest := newNoise(c.rng, c.W, c.H, 12)
	n := len(c.Tiles)
	// Forest noise, damped near the centre so the start area is open.
	tree := make([]float64, n)
	cx, cy := float64(c.W)/2, float64(c.H)/2
	rad := float64(min(c.W, c.H)) / 5
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / rad
			// Per-tile jitter breaks up blob edges and scatters lone trees.
			tree[y*c.W+x] = forest.fbm(x, y) - 0.25*math.Max(0, 1-d) + 0.22*c.rng.Float64()
		}
	}
	cut := quantile(tree, 1-treeShare)
	for i, v := range tree {
		if v >= cut && c.Tiles[i].Terrain == Land {
			c.Tiles[i].Terrain = Trees
		}
	}
}

// findStart picks where the camera opens: the open land closest to the
// centre with the most buildable ground around it.
func findStart(c *City) Pt {
	best, bestScore := Pt{c.W / 2, c.H / 2}, -1e9
	open := func(x, y int) bool { return c.In(x, y) && c.At(x, y).Terrain == Land }
	for y := 4; y < c.H-4; y += 3 {
		for x := 4; x < c.W-4; x += 3 {
			if !open(x, y) {
				continue
			}
			n := 0
			for dy := -6; dy <= 6; dy += 2 {
				for dx := -6; dx <= 6; dx += 2 {
					if open(x+dx, y+dy) {
						n++
					}
				}
			}
			score := float64(n) - 0.15*math.Hypot(float64(x-c.W/2), float64(y-c.H/2))
			if score > bestScore {
				best, bestScore = Pt{x, y}, score
			}
		}
	}
	return best
}

// carveRiver runs a meandering river from one edge to the opposite edge:
// two sine waves for the big bends plus slow noise, width swelling gently.
func carveRiver(c *City) {
	r := c.rng
	vertical := r.IntN(2) == 0
	length, across := c.H, c.W
	if !vertical {
		length, across = c.W, c.H
	}
	// Centre line in an outer third so the river doesn't split the start area.
	base := float64(across) * (0.2 + 0.12*r.Float64())
	if r.IntN(2) == 0 {
		base = float64(across) - base
	}
	amp1, amp2 := 4+5*r.Float64(), 1.5+2*r.Float64()
	len1, len2 := 28+20*r.Float64(), 9+6*r.Float64()
	ph1, ph2 := r.Float64()*2*math.Pi, r.Float64()*2*math.Pi
	wobble := newNoise(r, length, 4, 16)
	for i := 0; i < length; i++ {
		fi := float64(i)
		pos := base + amp1*math.Sin(fi/len1+ph1) + amp2*math.Sin(fi/len2+ph2) + 6*(wobble.fbm(i, 0)-0.875)
		half := 1.2 + 0.8*wobble.fbm(i, 2)
		for a := int(math.Floor(pos - half)); a <= int(math.Ceil(pos+half)); a++ {
			if math.Abs(float64(a)+0.5-pos) > half+0.5 {
				continue
			}
			x, y := a, i
			if !vertical {
				x, y = i, a
			}
			if c.In(x, y) {
				c.At(x, y).Terrain = Water
			}
		}
	}
}

func quantile(vals []float64, q float64) float64 {
	s := append([]float64(nil), vals...)
	sort.Float64s(s)
	return s[int(q*float64(len(s)-1))]
}

// noise is seeded value noise on a lattice with smooth interpolation.
type noise struct {
	gw, gh int
	period int
	v      []float64
}

func newNoise(r *rand.Rand, w, h, period int) *noise {
	n := &noise{gw: w/period*4 + 3, gh: h/period*4 + 3, period: period}
	n.v = make([]float64, n.gw*n.gh)
	for i := range n.v {
		n.v[i] = r.Float64()
	}
	return n
}

func (n *noise) at(x, y float64) float64 {
	x0, y0 := int(x), int(y)
	fx, fy := smooth(x-float64(x0)), smooth(y-float64(y0))
	g := func(i, j int) float64 { return n.v[(j%n.gh)*n.gw+i%n.gw] }
	top := g(x0, y0)*(1-fx) + g(x0+1, y0)*fx
	bot := g(x0, y0+1)*(1-fx) + g(x0+1, y0+1)*fx
	return top*(1-fy) + bot*fy
}

// fbm sums three octaves.
func (n *noise) fbm(x, y int) float64 {
	p := float64(n.period)
	return n.at(float64(x)/p, float64(y)/p) +
		0.5*n.at(float64(x)*2/p, float64(y)*2/p) +
		0.25*n.at(float64(x)*4/p, float64(y)*4/p)
}

func smooth(t float64) float64 { return t * t * (3 - 2*t) }
