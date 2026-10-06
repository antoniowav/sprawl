package render

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/render/sprites"
	"github.com/antoniowav/sprawl/sim"
	"github.com/antoniowav/sprawl/theme"
)

// TileSize is the base tile size in world pixels.
const TileSize = sprites.T

// WorldView is what the world pass needs to know.
type WorldView struct {
	City       *sim.City
	Atlas      *Atlas
	Roles      theme.Roles
	Cam        *Camera
	CursorX    int
	CursorY    int
	AnimFrame  int
	Animations bool

	Grow    func(x, y int) float64 // grow-animation progress 0..1, 1 = done
	Growing []sim.Pt               // tiles with a grow animation running
	Chunks  *Chunks                // static-layer cache; nil draws directly

	Overlay   Overlay
	Blink     bool    // icon blink phase
	Night     float64 // 0 day .. 1 deep night
	Particles []Particle

	// Filled by DrawWorld: smoke sources on screen, in world pixels.
	Emitters *[]image.Point

	Underground bool     // dim the surface and show pipes
	Preview     []sim.Pt // tiles the pending action will change
	PreviewBad  bool     // the pending action would be refused

	dst *ebiten.Image
}

// DrawWorld draws the visible tiles and the cursor over the whole screen;
// the HUD is drawn on top afterwards. It reports whether anything animated
// is visible, so the caller can skip animation frames when nothing is.
func DrawWorld(dst *ebiten.Image, v WorldView) (animated bool) {
	v.dst = dst
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	dst.Fill(v.Roles.Void)
	c := v.City
	z := v.Cam.Zoom
	ts := TileSize * z
	ox, oy := v.Cam.Origin(w, h)

	x0, y0 := max(0, floorDiv(-ox, ts)), max(0, floorDiv(-oy, ts))
	x1, y1 := min(c.W-1, floorDiv(w-ox, ts)), min(c.H-1, floorDiv(h-oy, ts))

	var op ebiten.DrawImageOptions
	draw := func(img *ebiten.Image, tx, ty int) {
		op.GeoM.Reset()
		op.GeoM.Scale(float64(z), float64(z))
		op.GeoM.Translate(float64(ox+tx*ts), float64(oy+ty*ts))
		dst.DrawImage(img, &op)
	}

	if v.Chunks != nil {
		v.Chunks.draw(dst, v, ox, oy, x0, y0, x1, y1)
	} else {
		drawStatic(draw, v.Atlas, c, x0, y0, x1, y1, v.isGrowing)
	}

	// Per-frame layers on top of the static map.
	for ty := y0; ty <= y1; ty++ {
		for tx := x0; tx <= x1; tx++ {
			t := c.At(tx, ty)
			if t.Terrain == sim.Water && v.Animations {
				if hsh := c.Hash(tx, ty); hsh%3 == 0 {
					animated = true
					draw(v.Atlas.Shimmer[(v.AnimFrame+int(hsh>>8))%4], tx, ty)
				}
			}
		}
	}
	for _, p := range v.Growing {
		if p.X < x0 || p.X > x1 || p.Y < y0 || p.Y > y1 {
			continue
		}
		t := c.At(p.X, p.Y)
		if !t.IsZone() || t.Level == 0 {
			continue
		}
		// Rise from the ground: show the bottom rows only.
		img := v.Atlas.Bldg[t.Kind-sim.ZoneR][t.Level-1][t.Variant%4]
		rows := max(1, int(v.Grow(p.X, p.Y)*TileSize+0.999))
		drawAt(v, img.SubImage(image.Rect(0, TileSize-rows, TileSize, TileSize)).(*ebiten.Image), p.X, p.Y, TileSize-rows)
		if t.Line {
			draw(v.Atlas.Line[mask(c, p.X, p.Y, func(t *sim.Tile) bool { return t.Line })], p.X, p.Y)
		}
	}

	if v.Emitters != nil {
		*v.Emitters = (*v.Emitters)[:0]
		for ty := max(0, y0-2); ty <= y1; ty++ {
			for tx := max(0, x0-2); tx <= x1; tx++ {
				t := c.At(tx, ty)
				switch {
				case t.Kind == sim.ZoneI && t.Level == 3:
					for _, p := range v.Atlas.Chimney[t.Variant%4] {
						*v.Emitters = append(*v.Emitters, image.Pt(tx*TileSize+p.X, ty*TileSize+p.Y))
					}
				case t.Kind == sim.PowerPlant && int(t.Anchor) == ty*c.W+tx:
					for _, p := range v.Atlas.Stacks {
						*v.Emitters = append(*v.Emitters, image.Pt(tx*TileSize+p.X, ty*TileSize+p.Y))
					}
				}
			}
		}
	}
	drawParticles(dst, v, ox, oy)

	if v.Night > 0 && v.Overlay == OverlayNone && !v.Underground {
		drawNight(dst, v, x0, y0, x1, y1, draw)
	}

	if v.Overlay != OverlayNone {
		drawOverlay(dst, v, x0, y0, x1, y1, ox, oy, ts, draw)
	} else if v.Underground {
		Rect(dst, 0, 0, w, h, color.RGBA{0, 0, 0, 0xa0})
		for ty := y0; ty <= y1; ty++ {
			for tx := x0; tx <= x1; tx++ {
				if c.At(tx, ty).Pipe {
					draw(v.Atlas.Pipe[mask(c, tx, ty, func(t *sim.Tile) bool { return t.Pipe })], tx, ty)
				}
			}
		}
	}

	if len(v.Preview) > 0 {
		tint := v.Roles.UIAccent
		if v.PreviewBad {
			tint = v.Roles.UIErr
		}
		tint.A = 0x60
		tint = premul(tint)
		for _, p := range v.Preview {
			Rect(dst, ox+p.X*ts, oy+p.Y*ts, ts, ts, tint)
		}
	}

	if drawIcons(dst, v, x0, y0, x1, y1, ox, oy, ts) {
		animated = true
	}

	drawCursor(dst, ox+v.CursorX*ts, oy+v.CursorY*ts, ts, z, v.Roles.UIAccent)
	return animated
}

// Particle is one puff of smoke, in world pixels.
type Particle struct {
	X, Y, Age, Life float64
}

func drawParticles(dst *ebiten.Image, v WorldView, ox, oy int) {
	z := v.Cam.Zoom
	base := theme.Mix(v.Roles.Walls[0], v.Roles.UIDim, 0.35)
	for _, p := range v.Particles {
		f := p.Age / p.Life
		col := base
		col.A = uint8(180 * (1 - f))
		size := 2 + int(f*2)
		Rect(dst, ox+int(p.X*float64(z)), oy+int(p.Y*float64(z)), size*z, size*z, premul(col))
	}
}

// drawNight darkens the world and lights windows on top.
func drawNight(dst *ebiten.Image, v WorldView, x0, y0, x1, y1 int, draw func(*ebiten.Image, int, int)) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	tint := v.Roles.Void
	tint.A = uint8(165 * v.Night)
	Rect(dst, 0, 0, w, h, premul(tint))
	if v.Night < 0.25 {
		return
	}
	c := v.City
	for ty := max(0, y0-2); ty <= y1; ty++ {
		for tx := max(0, x0-2); tx <= x1; tx++ {
			t := c.At(tx, ty)
			switch {
			case t.IsZone() && t.Level > 0 && t.Powered && tx >= x0 && ty >= y0:
				if v.Grow != nil && v.Grow(tx, ty) < 1 {
					continue
				}
				draw(v.Atlas.Lights[t.Kind-sim.ZoneR][t.Level-1][t.Variant%4], tx, ty)
			case t.IsBuilding() && t.Powered && int(t.Anchor) == ty*c.W+tx:
				draw(v.Atlas.CivicLt[t.Kind], tx, ty)
			}
		}
	}
}

// drawIcons marks developed tiles and buildings that lack power (bolt) or,
// from medium density up, water (droplet). Icons blink, so it reports
// whether any are on screen.
func drawIcons(dst *ebiten.Image, v WorldView, x0, y0, x1, y1, ox, oy, ts int) bool {
	c := v.City
	any := false
	z := v.Cam.Zoom
	var op ebiten.DrawImageOptions
	for ty := y0; ty <= y1; ty++ {
		for tx := x0; tx <= x1; tx++ {
			t := c.At(tx, ty)
			var img *ebiten.Image
			switch {
			case t.IsZone() && t.Level > 0 && !t.Powered,
				t.IsBuilding() && int(t.Anchor) == ty*c.W+tx && !t.Powered:
				img = v.Atlas.NoPower
			case t.IsZone() && t.Level >= 2 && !t.Watered:
				img = v.Atlas.NoWater
			}
			if img == nil {
				continue
			}
			any = true
			if !v.Blink {
				continue
			}
			op.GeoM.Reset()
			op.GeoM.Scale(float64(z), float64(z))
			op.GeoM.Translate(float64(ox+tx*ts+ts-10*z), float64(oy+ty*ts+z))
			dst.DrawImage(img, &op)
		}
	}
	return any
}

// drawStatic draws everything that only changes when tiles change:
// ground, roads, zones and their buildings, multi-tile buildings and power
// lines. growing reports tiles whose building is still rising (drawn later,
// per frame).
func drawStatic(draw func(*ebiten.Image, int, int), at *Atlas, c *sim.City, x0, y0, x1, y1 int, growing func(x, y int) bool) {
	for ty := y0; ty <= y1; ty++ {
		for tx := x0; tx <= x1; tx++ {
			t := c.At(tx, ty)
			hsh := c.Hash(tx, ty)
			switch {
			case t.Terrain == sim.Water:
				draw(at.Water[shoreMask(c, tx, ty)], tx, ty)
			case t.Terrain == sim.Trees:
				draw(at.Trees[hsh%3], tx, ty)
			case t.Kind == sim.Road:
				draw(at.Road[mask(c, tx, ty, func(t *sim.Tile) bool { return t.Kind == sim.Road })], tx, ty)
			case t.IsZone():
				z := t.Kind - sim.ZoneR
				draw(at.Lot[z], tx, ty)
				if t.Level > 0 && !growing(tx, ty) {
					draw(at.Bldg[z][t.Level-1][t.Variant%4], tx, ty)
				}
			case nearTrees(c, tx, ty):
				draw(at.Lush[hsh%2], tx, ty)
			default:
				draw(at.Grass[hsh%4], tx, ty)
			}
		}
	}
	// Multi-tile buildings, drawn from their anchor. Look two tiles up and
	// left so a building whose corner is in range still shows.
	for ty := max(0, y0-2); ty <= y1; ty++ {
		for tx := max(0, x0-2); tx <= x1; tx++ {
			t := c.At(tx, ty)
			if t.IsBuilding() && int(t.Anchor) == ty*c.W+tx {
				draw(at.Civic[t.Kind], tx, ty)
			}
		}
	}
	// Overhead lines go above everything on the surface.
	for ty := y0; ty <= y1; ty++ {
		for tx := x0; tx <= x1; tx++ {
			if t := c.At(tx, ty); t.Line && !growing(tx, ty) {
				draw(at.Line[mask(c, tx, ty, func(t *sim.Tile) bool { return t.Line })], tx, ty)
			}
		}
	}
}

func (v WorldView) isGrowing(x, y int) bool { return v.Grow != nil && v.Grow(x, y) < 1 }

// drawAt draws img at tile (x, y) shifted down by dy world pixels.
func drawAt(v WorldView, img *ebiten.Image, x, y, dy int) {
	z := v.Cam.Zoom
	ts := TileSize * z
	w, h := v.dst.Bounds().Dx(), v.dst.Bounds().Dy()
	ox, oy := v.Cam.Origin(w, h)
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(float64(z), float64(z))
	op.GeoM.Translate(float64(ox+x*ts), float64(oy+y*ts+dy*z))
	v.dst.DrawImage(img, &op)
}

// mask builds the N/E/S/W neighbour mask of tiles matching f.
func mask(c *sim.City, x, y int, f func(*sim.Tile) bool) int {
	m := 0
	for i, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		if c.In(x+d[0], y+d[1]) && f(c.At(x+d[0], y+d[1])) {
			m |= 1 << i
		}
	}
	return m
}

// premul converts a straight-alpha colour to the premultiplied form
// ColorScale expects.
func premul(c color.RGBA) color.RGBA {
	a := uint16(c.A)
	return color.RGBA{uint8(uint16(c.R) * a / 255), uint8(uint16(c.G) * a / 255), uint8(uint16(c.B) * a / 255), c.A}
}

// nearTrees is true when a 4-neighbour is forest; those tiles get shaded
// grass so forests fade into fields instead of ending hard.
func nearTrees(c *sim.City, x, y int) bool {
	for _, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		if c.In(x+d[0], y+d[1]) && c.At(x+d[0], y+d[1]).Terrain == sim.Trees {
			return true
		}
	}
	return false
}

func shoreMask(c *sim.City, x, y int) int {
	m := 0
	land := func(x, y int) bool { return c.In(x, y) && c.At(x, y).Terrain != sim.Water }
	if land(x, y-1) {
		m |= sprites.N
	}
	if land(x+1, y) {
		m |= sprites.E
	}
	if land(x, y+1) {
		m |= sprites.S
	}
	if land(x-1, y) {
		m |= sprites.W
	}
	return m
}

// drawCursor draws corner brackets, one world pixel thick, with a dark
// outline so it reads on any terrain.
func drawCursor(dst *ebiten.Image, x, y, ts, z int, c color.RGBA) {
	arm := 5 * z
	shade := color.RGBA{0, 0, 0, 0x70}
	for _, pass := range []struct {
		d int
		c color.RGBA
	}{{z, shade}, {0, c}} {
		d := pass.d
		for _, corner := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
			cx := x + corner[0]*(ts-z)
			cy := y + corner[1]*(ts-z)
			hx := cx
			if corner[0] == 1 {
				hx = cx - arm + z
			}
			vy := cy
			if corner[1] == 1 {
				vy = cy - arm + z
			}
			Rect(dst, hx+d, cy+d, arm, z, pass.c)
			Rect(dst, cx+d, vy+d, z, arm, pass.c)
		}
	}
}
