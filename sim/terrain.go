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

func generateTerrain(c *City) {
	lakes := newNoise(c.rng, c.W, c.H, 24)
	forest := newNoise(c.rng, c.W, c.H, 12)

	n := len(c.Tiles)
	lake := make([]float64, n)
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			lake[y*c.W+x] = lakes.fbm(x, y)
		}
	}
	cut := quantile(lake, 1-lakeShare)
	for i, v := range lake {
		if v >= cut {
			c.Tiles[i].Terrain = Water
		}
	}
	carveRiver(c)

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
	cut = quantile(tree, 1-treeShare)
	for i, v := range tree {
		if v >= cut && c.Tiles[i].Terrain == Land {
			c.Tiles[i].Terrain = Trees
		}
	}
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
