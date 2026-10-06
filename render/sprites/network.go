package sprites

import (
	"image"
	"image/color"

	"github.com/antoniowav/sprawl/theme"
)

// Network holds autotiled sprites (indexed by N/E/S/W neighbour mask) and
// zone lots.
type Network struct {
	Road [16]*image.RGBA // opaque, grass underneath
	Line [16]*image.RGBA // transparent overlay
	Pipe [16]*image.RGBA // transparent overlay, underground view
	Lot  [3]*image.RGBA  // empty R, C, I lots
}

// BuildNetwork renders roads, lines, pipes and lots.
func BuildNetwork(r theme.Roles) Network {
	var n Network
	for m := 0; m < 16; m++ {
		n.Road[m] = road(r, m)
		n.Line[m] = powerLine(r, m)
		n.Pipe[m] = pipe(r, m)
	}
	for z, c := range []color.RGBA{r.ZoneR, r.ZoneC, r.ZoneI} {
		n.Lot[z] = lot(r, c, z)
	}
	return n
}

func rect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := max(0, y0); y < min(T, y1); y++ {
		for x := max(0, x0); x < min(T, x1); x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

// arms returns, for a mask, the rectangles that extend a centre block of
// [lo, hi) to each connected edge.
func arms(mask, lo, hi int) [][4]int {
	out := [][4]int{{lo, lo, hi, hi}}
	if mask&N != 0 {
		out = append(out, [4]int{lo, 0, hi, lo})
	}
	if mask&S != 0 {
		out = append(out, [4]int{lo, hi, hi, T})
	}
	if mask&W != 0 {
		out = append(out, [4]int{0, lo, lo, hi})
	}
	if mask&E != 0 {
		out = append(out, [4]int{hi, lo, T, hi})
	}
	return out
}

// road: grass verge, a sidewalk ring, asphalt, and a dashed centre line on
// straight runs and dead ends.
func road(r theme.Roles, mask int) *image.RGBA {
	img := grass(r, 0)
	for _, a := range arms(mask, 2, 14) {
		rect(img, a[0], a[1], a[2], a[3], r.Sidewalk)
	}
	for _, a := range arms(mask, 3, 13) {
		rect(img, a[0], a[1], a[2], a[3], r.Road)
	}
	// Speckle the asphalt a little so long roads aren't flat.
	g := rng(5, mask)
	for i := 0; i < 6; i++ {
		x, y := 3+g.IntN(10), 3+g.IntN(10)
		if img.RGBAAt(x, y) == r.Road {
			img.SetRGBA(x, y, r.RoadEdge)
		}
	}
	vertical := mask&(N|S) != 0 && mask&(E|W) == 0
	horizontal := mask&(E|W) != 0 && mask&(N|S) == 0
	for i := 1; i < T; i += 5 {
		if vertical {
			rect(img, 7, i, 9, i+3, r.RoadMark)
		}
		if horizontal {
			rect(img, i, 7, i+3, 9, r.RoadMark)
		}
	}
	return img
}

// powerLine: a pole in the middle with wires to connected sides, each wire
// with a soft shadow so it reads over grass and asphalt alike.
func powerLine(r theme.Roles, mask int) *image.RGBA {
	img := newTile()
	shadow := color.RGBA{0, 0, 0, 0x50}
	if mask == 0 {
		mask = E | W // a lone segment still shows a wire
	}
	for _, a := range arms(mask, 7, 8) {
		rect(img, a[0]+1, a[1]+1, a[2]+1, a[3]+1, shadow)
	}
	for _, a := range arms(mask, 7, 8) {
		rect(img, a[0], a[1], a[2], a[3], r.Power)
	}
	rect(img, 7, 6, 10, 10, shadow)
	rect(img, 6, 5, 9, 9, r.Pole)
	img.SetRGBA(6, 5, r.Power)
	img.SetRGBA(8, 5, r.Power)
	return img
}

// pipe: a 4 px pipe with darker walls and a highlight, joints at the centre.
func pipe(r theme.Roles, mask int) *image.RGBA {
	img := newTile()
	if mask == 0 {
		mask = E | W
	}
	for _, a := range arms(mask, 5, 11) {
		rect(img, a[0], a[1], a[2], a[3], r.PipeDark)
	}
	for _, a := range arms(mask, 6, 10) {
		rect(img, a[0], a[1], a[2], a[3], r.Pipe)
	}
	hi := theme.Mix(r.Pipe, r.Foam, 0.5)
	for _, a := range arms(mask, 6, 7) {
		rect(img, a[0], a[1], a[2], a[3], hi)
	}
	// Flanged joint only where the pipe turns, branches or ends.
	straight := mask == N|S || mask == E|W
	if !straight {
		rect(img, 4, 4, 12, 12, r.PipeDark)
		rect(img, 5, 5, 11, 11, r.Pipe)
	}
	return img
}

// lot: tinted ground with a dotted border in the zone colour.
func lot(r theme.Roles, c color.RGBA, z int) *image.RGBA {
	img := newTile()
	fill(img, theme.Mix(r.Grass, c, 0.12))
	g := rng(6, z)
	for i := 0; i < 10; i++ {
		img.SetRGBA(1+g.IntN(T-2), 1+g.IntN(T-2), r.GrassDark)
	}
	dot := theme.Mix(c, r.Grass, 0.25)
	for i := 1; i < T-1; i += 2 {
		img.SetRGBA(i, 1, dot)
		img.SetRGBA(i, T-2, dot)
		img.SetRGBA(1, i, dot)
		img.SetRGBA(T-2, i, dot)
	}
	return img
}
