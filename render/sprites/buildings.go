package sprites

import (
	"image"
	"image/color"

	"github.com/antoniowav/sprawl/theme"
)

// Buildings holds developed zone sprites, indexed [zone][level-1][variant].
type Buildings struct {
	Img [3][3][4]*image.RGBA
	// Lights are night window masks (transparent except lit windows).
	Lights [3][3][4]*image.RGBA
	// Chimney tops of high-density industry, per variant, for smoke.
	Chimneys [4][]image.Point
}

// BuildBuildings renders every zone building.
func BuildBuildings(r theme.Roles) Buildings {
	var b Buildings
	p := newPainter(r)
	for v := 0; v < 4; v++ {
		b.Img[0][0][v] = p.house(v)
		b.Img[0][1][v] = p.rowHouses(v)
		b.Img[0][2][v] = p.apartments(v)
		b.Img[1][0][v] = p.shop(v)
		b.Img[1][1][v] = p.office(v)
		b.Img[1][2][v] = p.tower(v)
		b.Img[2][0][v] = p.shed(v)
		b.Img[2][1][v] = p.warehouse(v)
		b.Img[2][2][v], b.Chimneys[v] = p.factory(v)
	}
	for z := range b.Img {
		for l := range b.Img[z] {
			for v, img := range b.Img[z][l] {
				b.Lights[z][l][v] = p.lights(img, uint64(z*100+l*10+v))
			}
		}
	}
	return b
}

// lights builds a night mask: about 70% of window pixels lit, picked by a
// hash so the pattern is stable and differs between variants.
func (p *painter) lights(img *image.RGBA, seed uint64) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(b)
	glow := p.r.Window
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := img.RGBAAt(x, y)
			if !p.isWindow(c) {
				continue
			}
			h := (uint64(x)*73856093 ^ uint64(y)*19349663 ^ seed*83492791) % 10
			if h < 7 {
				out.SetRGBA(x, y, glow)
			}
		}
	}
	return out
}

func (p *painter) isWindow(c color.RGBA) bool {
	if c == p.glass || c == dark(p.glass) {
		return true
	}
	for v := 0; v < 3; v++ { // tower glass bands
		g := theme.Mix(p.glass, p.r.ZoneC, 0.15*float64(v))
		if c == dark(g) {
			return true
		}
	}
	return false
}

type painter struct {
	r                       theme.Roles
	shadow, glass, door     color.RGBA
	pave, dirt, flat, flatD color.RGBA
}

func newPainter(r theme.Roles) *painter {
	deep := r.Void
	return &painter{
		r:      r,
		shadow: theme.Mix(r.Grass, deep, 0.45),
		glass:  theme.Mix(r.ZoneC, deep, 0.45),
		door:   theme.Mix(r.Roofs[1], deep, 0.4),
		pave:   theme.Mix(r.Sidewalk, r.Road, 0.4),
		dirt:   theme.Mix(r.Roofs[1], r.Grass, 0.55),
		flat:   theme.Mix(r.Walls[0], r.UIBorder, 0.45),
		flatD:  theme.Mix(r.Walls[0], deep, 0.5),
	}
}

func light(c color.RGBA) color.RGBA { return theme.Mix(c, color.RGBA{255, 255, 255, 255}, 0.18) }
func dark(c color.RGBA) color.RGBA  { return theme.Mix(c, color.RGBA{0, 0, 0, 255}, 0.28) }

// block draws a building seen from above and slightly in front: a roof of
// w×d, a front wall of height h below it, and a soft drop shadow.
func (p *painter) block(img *image.RGBA, x, y, w, d, h int, roof, wall color.RGBA) {
	rect(img, x+1, y+1, x+w+1, y+d+h+1, p.shadow)
	rect(img, x, y, x+w, y+d, roof)
	rect(img, x, y, x+w, y+1, light(roof))
	rect(img, x, y+d, x+w, y+d+h, wall)
	rect(img, x, y+d, x+w, y+d+1, dark(wall))
}

// windows puts glass on wall rows every `every` pixels, starting at
// (x, y) for `rows` rows spaced `gap` apart.
func (p *painter) windows(img *image.RGBA, x0, x1, y, rows, gap, every int, c color.RGBA) {
	for r := 0; r < rows; r++ {
		for x := x0; x < x1; x += every {
			img.SetRGBA(x, y+r*gap, c)
		}
	}
}

func (p *painter) bush(img *image.RGBA, x, y int) {
	rect(img, x, y, x+2, y+2, p.r.Tree)
	img.SetRGBA(x, y, p.r.TreeLight)
	img.SetRGBA(x+1, y+1, p.r.TreeDark)
}

// --- residential ---

func (p *painter) house(v int) *image.RGBA {
	img := grass(p.r, v)
	g := rng(10, v)
	roof := p.r.Roofs[v]
	wall := p.r.Walls[v%3]
	x, w := 3+g.IntN(2), 9+g.IntN(2)
	y := 3
	p.block(img, x, y, w, 6, 4, roof, wall)
	// Pitched roof: lit upper slope, ridge, shaded lower slope.
	rect(img, x, y, x+w, y+3, light(roof))
	rect(img, x, y+2, x+w, y+3, theme.Mix(roof, p.r.UIText, 0.25))
	rect(img, x, y+4, x+w, y+6, dark(roof))
	if v%2 == 1 {
		rect(img, x+w-3, y-1, x+w-2, y+2, p.flatD) // chimney
	}
	door := x + w/2
	rect(img, door, y+8, door+1, y+10, p.door)
	img.SetRGBA(x+2, y+7, p.glass)
	img.SetRGBA(x+w-3, y+7, p.glass)
	p.bush(img, []int{1, 12, 1, 12}[v], 12+g.IntN(2))
	return img
}

func (p *painter) rowHouses(v int) *image.RGBA {
	img := grass(p.r, v+1)
	x, y, w := 1, 2, 14
	p.block(img, x, y, w, 5, 6, p.r.Roofs[v], p.r.Walls[v%3])
	// Three units with alternating roof shades.
	for i := 0; i < 3; i++ {
		ux := x + 1 + i*5
		if i%2 == 1 {
			rect(img, ux-1, y, ux+4, y+5, dark(p.r.Roofs[v]))
		}
		rect(img, ux+1, y+11, ux+2, y+13, p.door)
	}
	rect(img, x, y+2, x+w, y+3, theme.Mix(p.r.Roofs[v], p.r.UIText, 0.2))
	p.windows(img, x+1, x+w-1, y+7, 2, 2, 2, p.glass)
	if v%2 == 0 {
		p.bush(img, 13, 14)
	}
	return img
}

func (p *painter) apartments(v int) *image.RGBA {
	img := grass(p.r, v)
	rect(img, 0, 13, T, T, p.pave)
	x, y, w := 1, 1, 14
	wall := p.r.Walls[v%3]
	p.block(img, x, y, w, 4, 9, p.flat, wall)
	rect(img, x+1, y+1, x+w-1, y+3, theme.Mix(p.flat, p.flatD, 0.3))
	rect(img, x+2+v, y+1, x+4+v, y+3, p.flatD) // rooftop unit
	p.windows(img, x+1, x+w-1, y+6, 4, 2, 2, p.glass)
	// Balcony accents in the roof colour on one column.
	for r := 0; r < 4; r++ {
		img.SetRGBA(x+3+2*(v%3), y+6+r*2, p.r.Roofs[v])
	}
	rect(img, x+6, y+12, x+8, y+13, p.door)
	return img
}

// --- commercial ---

func (p *painter) shop(v int) *image.RGBA {
	img := newTile()
	fill(img, p.pave)
	for i := 2; i < T; i += 4 {
		img.SetRGBA(i, 14, p.r.Sidewalk) // parking lines
	}
	x, y, w := 2, 3, 12
	p.block(img, x, y, w, 5, 4, p.flat, p.r.Walls[v%3])
	// Striped awning in the zone colour.
	for i := x; i < x+w; i++ {
		c := p.r.ZoneC
		if (i+v)%2 == 0 {
			c = light(p.r.Walls[0])
		}
		img.SetRGBA(i, y+5, c)
	}
	rect(img, x+1, y+7, x+w-1, y+8, p.glass)
	rect(img, x+w/2, y+7, x+w/2+1, y+9, p.door)
	rect(img, x+1+v*2, y+1, x+3+v*2, y+3, p.flatD)
	return img
}

func (p *painter) office(v int) *image.RGBA {
	img := newTile()
	fill(img, p.pave)
	x, y, w := 2, 1, 12
	p.block(img, x, y, w, 5, 8, p.flat, p.r.Walls[(v+1)%3])
	for r := 0; r < 3; r++ {
		rect(img, x+1, y+7+r*2, x+w-1, y+8+r*2, p.glass)
		for mx := x + 3; mx < x+w-1; mx += 3 {
			img.SetRGBA(mx, y+7+r*2, dark(p.glass))
		}
	}
	rect(img, x+2, y+1, x+w-2, y+2, p.r.ZoneC) // sign
	if v%2 == 1 {
		rect(img, x+w-4, y+2, x+w-2, y+4, p.flatD)
	}
	return img
}

func (p *painter) tower(v int) *image.RGBA {
	img := newTile()
	fill(img, p.pave)
	x, y, w := 3, 0, 10
	glass := theme.Mix(p.glass, p.r.ZoneC, 0.15*float64(v%3))
	p.block(img, x, y, w, 4, 11, theme.Mix(p.flat, p.r.ZoneC, 0.2), glass)
	for r := 0; r < 5; r++ {
		rect(img, x, y+5+r*2, x+w, y+6+r*2, dark(glass))
	}
	rect(img, x+1+v, y+5, x+2+v, y+15, light(glass)) // reflection
	rect(img, x+3, y+1, x+w-3, y+3, p.r.ZoneC)       // crown
	rect(img, x+4, y+14, x+6, y+15, p.door)
	return img
}

// --- industrial ---

func (p *painter) shed(v int) *image.RGBA {
	img := newTile()
	fill(img, p.dirt)
	g := rng(11, v)
	for i := 0; i < 12; i++ {
		img.SetRGBA(g.IntN(T), g.IntN(T), dark(p.dirt))
	}
	x, y, w := 2, 3, 9
	roof := theme.Mix(p.flat, p.r.ZoneI, 0.25)
	p.block(img, x, y, w, 5, 3, roof, p.r.Walls[0])
	for i := x; i < x+w; i += 2 {
		rect(img, i, y, i+1, y+5, dark(roof)) // corrugation
	}
	rect(img, x+2, y+6, x+5, y+8, p.door)
	// Crates.
	for i := 0; i < 2+v%2; i++ {
		cx, cy := 12+g.IntN(2), 4+i*3
		rect(img, cx, cy, cx+2, cy+2, p.r.Roofs[1])
		img.SetRGBA(cx, cy, light(p.r.Roofs[1]))
	}
	return img
}

func (p *painter) warehouse(v int) *image.RGBA {
	img := newTile()
	fill(img, p.dirt)
	x, y, w := 1, 2, 14
	roof := theme.Mix(p.flat, p.r.ZoneI, 0.2)
	p.block(img, x, y, w, 7, 4, roof, p.r.Walls[v%3])
	// Sawtooth roof.
	for i := x; i < x+w; i += 3 {
		rect(img, i, y, i+1, y+7, light(roof))
		rect(img, i+2, y, i+3, y+7, dark(roof))
	}
	for i := 0; i < 2; i++ {
		dx := x + 2 + i*6 + v%2
		rect(img, dx, y+8, dx+3, y+11, p.door)
		rect(img, dx, y+8, dx+3, y+9, dark(p.door))
	}
	return img
}

func (p *painter) factory(v int) (*image.RGBA, []image.Point) {
	img := newTile()
	fill(img, p.dirt)
	x, y, w := 1, 5, 10
	roof := theme.Mix(p.flat, p.r.ZoneI, 0.3)
	p.block(img, x, y, w, 5, 5, roof, p.r.Walls[(v+2)%3])
	p.windows(img, x+1, x+w-1, y+7, 1, 2, 2, theme.Mix(p.glass, p.r.ZoneI, 0.3))
	rect(img, x+3, y+8, x+5, y+10, p.door)
	if v%2 == 0 {
		// A storage tank next to the hall.
		disc(img, 4, 2.5, 2.2, func(dx, dy float64) color.RGBA {
			if dx+dy < -0.5 {
				return light(p.flat)
			}
			return p.flat
		})
	}
	var tops []image.Point
	stack := func(cx int) {
		rect(img, cx+1, 2, cx+3, 13, p.shadow)
		rect(img, cx, 1, cx+2, 12, p.flatD)
		rect(img, cx, 3, cx+2, 4, p.r.UIErr)
		rect(img, cx, 1, cx+2, 2, theme.Mix(p.flatD, p.r.Void, 0.5))
		tops = append(tops, image.Point{cx + 1, 1})
	}
	stack(12)
	if v >= 2 {
		stack(8)
	}
	return img, tops
}
