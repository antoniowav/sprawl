// Package sprites generates every tile image procedurally from theme roles.
// It works on image.RGBA only (no GPU), so it is testable and fast.
package sprites

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/antoniowav/sprawl/theme"
)

// T is the base tile size in pixels.
const T = 16

// Water neighbour mask bits: set when that side touches land.
const (
	N = 1 << iota
	E
	S
	W
)

// Terrain holds the ground sprites.
type Terrain struct {
	Grass   [4]*image.RGBA
	Lush    [2]*image.RGBA // darker grass at forest edges
	Trees   [3]*image.RGBA
	Water   [16]*image.RGBA // indexed by land-side mask
	Shimmer [4]*image.RGBA  // transparent overlays, one per animation frame
	Rock    [3]*image.RGBA
}

// BuildTerrain renders all terrain sprites for the given roles.
func BuildTerrain(r theme.Roles) Terrain {
	var t Terrain
	for i := range t.Grass {
		t.Grass[i] = grass(r, i)
	}
	for i := range t.Lush {
		t.Lush[i] = lush(r, i)
	}
	for i := range t.Trees {
		t.Trees[i] = trees(r, i)
	}
	for m := range t.Water {
		t.Water[m] = water(r, m)
	}
	for f := range t.Shimmer {
		t.Shimmer[f] = shimmer(r, f)
	}
	for v := range t.Rock {
		t.Rock[v] = rock(r, v)
	}
	return t
}

func rng(kind, v int) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(kind)*7919+uint64(v), 0x5157))
}

func newTile() *image.RGBA { return image.NewRGBA(image.Rect(0, 0, T, T)) }

func fill(img *image.RGBA, c color.RGBA) {
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
	}
}

func grass(r theme.Roles, v int) *image.RGBA {
	img := newTile()
	fill(img, r.Grass)
	g := rng(1, v)
	for i := 0; i < 14; i++ {
		img.SetRGBA(g.IntN(T), g.IntN(T), r.GrassDark)
	}
	// Short light blades: two pixels, the top one offset half the time.
	for i := 0; i < 4; i++ {
		x, y := g.IntN(T-1), 1+g.IntN(T-1)
		img.SetRGBA(x, y, r.GrassLight)
		img.SetRGBA(x+g.IntN(2), y-1, r.GrassLight)
	}
	return img
}

func lush(r theme.Roles, v int) *image.RGBA {
	img := newTile()
	fill(img, r.Grass)
	g := rng(4, v)
	// A dither of dark speckles: reads as shade under nearby trees without
	// drawing a hard edge where it stops.
	for i := 0; i < 48; i++ {
		img.SetRGBA(g.IntN(T), g.IntN(T), r.GrassDark)
	}
	for i := 0; i < 8; i++ {
		img.SetRGBA(g.IntN(T), g.IntN(T), r.Shadow)
	}
	return img
}

type canopy struct{ x, y, rad float64 }

var treeLayouts = [3][]canopy{
	{{8, 7, 5.2}},
	{{5, 5.5, 3.6}, {11, 10, 4.2}},
	{{4.5, 4.5, 3.1}, {11.5, 5, 3.1}, {7.5, 11, 3.6}},
}

func trees(r theme.Roles, v int) *image.RGBA {
	img := grass(r, v)
	g := rng(2, v)
	cs := treeLayouts[v]
	// Shadows first so canopies overlap them.
	for _, c := range cs {
		disc(img, c.x+1.5, c.y+2, c.rad, func(dx, dy float64) color.RGBA { return r.Shadow })
	}
	for _, c := range cs {
		rad := c.rad
		disc(img, c.x, c.y, rad, func(dx, dy float64) color.RGBA {
			lit := (dx + dy) / rad // light from top-left
			switch {
			case lit > 0.55:
				return r.TreeDark
			case lit < -0.6:
				return r.TreeLight
			}
			if g.IntN(9) == 0 {
				return r.TreeDark
			}
			return r.Tree
		})
	}
	return img
}

// disc paints pixels whose centres lie within rad of (cx, cy).
func disc(img *image.RGBA, cx, cy, rad float64, col func(dx, dy float64) color.RGBA) {
	for y := 0; y < T; y++ {
		for x := 0; x < T; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			if math.Hypot(dx, dy) <= rad {
				img.SetRGBA(x, y, col(dx, dy))
			}
		}
	}
}

func water(r theme.Roles, mask int) *image.RGBA {
	img := newTile()
	fill(img, r.Water)
	g := rng(3, 0)
	for i := 0; i < 10; i++ {
		x, y := g.IntN(T-2), g.IntN(T)
		img.SetRGBA(x, y, r.WaterDeep)
		img.SetRGBA(x+1, y, r.WaterDeep)
	}
	// Shore: foam on the outer pixel, shallow water on the next.
	edge := func(side int, at func(i, d int) (int, int)) {
		if mask&side == 0 {
			return
		}
		for i := 0; i < T; i++ {
			x, y := at(i, 1)
			img.SetRGBA(x, y, r.WaterLight)
		}
		for i := 0; i < T; i++ {
			x, y := at(i, 0)
			img.SetRGBA(x, y, r.Foam)
		}
	}
	edge(N, func(i, d int) (int, int) { return i, d })
	edge(S, func(i, d int) (int, int) { return i, T - 1 - d })
	edge(W, func(i, d int) (int, int) { return d, i })
	edge(E, func(i, d int) (int, int) { return T - 1 - d, i })
	return img
}

// shimmer draws short highlight dashes that drift one pixel per frame and
// fade in and out over the four frames.
func shimmer(r theme.Roles, f int) *image.RGBA {
	img := newTile()
	alpha := [4]float64{0.35, 0.8, 0.8, 0.35}[f]
	c := r.WaterLight
	c = color.RGBA{uint8(float64(c.R) * alpha), uint8(float64(c.G) * alpha), uint8(float64(c.B) * alpha), uint8(255 * alpha)}
	for _, d := range [][3]int{{3, 4, 3}, {9, 9, 2}, {5, 12, 3}} {
		for i := 0; i < d[2]; i++ {
			img.SetRGBA((d[0]+i+f)%T, d[1], c)
		}
	}
	return img
}

// rock: grass with grey boulders lit from the top-left.
func rock(r theme.Roles, v int) *image.RGBA {
	img := grass(r, v)
	g := rng(7, v)
	n := 2 + v
	for i := 0; i < n; i++ {
		cx, cy := 3+g.Float64()*10, 3+g.Float64()*10
		rad := 2.2 + g.Float64()*2.3
		disc(img, cx+1, cy+1.5, rad, func(dx, dy float64) color.RGBA { return r.Shadow })
		disc(img, cx, cy, rad, func(dx, dy float64) color.RGBA {
			lit := (dx + dy) / rad
			switch {
			case lit < -0.6:
				return r.RockLight
			case lit > 0.5:
				return r.RockDark
			}
			return r.Rock
		})
	}
	return img
}

// Hill shading: an overlay per (light, edges). Light is -2..2 (facing away
// from or toward the north-west sun); edge bits mark which neighbours are
// lower (S, E: shadowed step below) or higher (N, W: lit lip above).
const (
	EdgeS = 1 << iota
	EdgeE
	EdgeN
	EdgeW
)

// ShadeKey indexes the shade overlays.
func ShadeKey(light, edges int) int { return (light+2)*16 + edges }

// BuildShades renders the 5×16 hill-shade overlays.
func BuildShades() [80]*image.RGBA {
	var out [80]*image.RGBA
	for light := -2; light <= 2; light++ {
		for e := 0; e < 16; e++ {
			img := newTile()
			var base color.RGBA
			switch {
			case light > 0:
				a := uint8(0x09 * light)
				base = color.RGBA{a, a, a, a}
			case light < 0:
				base = color.RGBA{0, 0, 0, uint8(0x10 * -light)}
			}
			fill(img, base)
			dark := color.RGBA{0, 0, 0, 0x50}
			lip := color.RGBA{0x22, 0x22, 0x22, 0x22}
			if e&EdgeN != 0 {
				rect(img, 0, 0, T, 1, lip)
			}
			if e&EdgeW != 0 {
				rect(img, 0, 0, 1, T, lip)
			}
			if e&EdgeS != 0 {
				rect(img, 0, T-2, T, T, dark)
			}
			if e&EdgeE != 0 {
				rect(img, T-1, 0, T, T, color.RGBA{0, 0, 0, 0x38})
			}
			out[ShadeKey(light, e)] = img
		}
	}
	return out
}

// BuildHeightTints lightens ground a little per level, so plateaus read as
// one surface and high ground stands out.
func BuildHeightTints() [11]*image.RGBA {
	var out [11]*image.RGBA
	for h := range out {
		img := newTile()
		a := uint8(4 * h)
		fill(img, color.RGBA{a, a, a, a})
		out[h] = img
	}
	return out
}

// mountain is the terrain tool icon.
func mountain(r theme.Roles) *image.RGBA {
	return glyphIcon([]string{
		"................",
		"................",
		".......#........",
		"......###.......",
		".....#####..#...",
		"....#######.##..",
		"...###########..",
		"..#############.",
		".###############",
	}, r.UIText)
}
