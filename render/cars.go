package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/render/sprites"
	"github.com/antoniowav/sprawl/sim"
)

// drawCars puts little cars on straight road tiles that carry traffic.
// Cars follow continuous lanes in world space, so they flow from tile to
// tile; spacing shrinks and speed drops as a road gets busier. It reports
// whether any car is on screen (they move, so frames are needed).
func drawCars(dst *ebiten.Image, v WorldView, x0, y0, x1, y1, ox, oy int) bool {
	if !v.Animations {
		return false
	}
	c := v.City
	z := float64(v.Cam.Zoom)
	any := false
	colors := []color.RGBA{v.Roles.Roofs[0], v.Roles.ZoneC, v.Roles.Walls[0], v.Roles.Roofs[3], v.Roles.ZoneI}
	for ty := y0; ty <= y1; ty++ {
		for tx := x0; tx <= x1; tx++ {
			t := c.At(tx, ty)
			if t.Kind != sim.Road || t.Traffic < 5 {
				continue
			}
			m := mask(c, tx, ty, func(t *sim.Tile) bool { return t.Kind == sim.Road })
			horizontal := m&(sprites.E|sprites.W) != 0 && m&(sprites.N|sprites.S) == 0
			vertical := m&(sprites.N|sprites.S) != 0 && m&(sprites.E|sprites.W) == 0
			if !horizontal && !vertical {
				continue // keep junctions clear
			}
			cong := t.Congestion()
			spacing := 64.0
			switch {
			case cong > 1:
				spacing = 14
			case cong > 0.5:
				spacing = 22
			case t.Traffic > 40:
				spacing = 36
			}
			speed := 12 / (1 + 2*max(0, cong-0.5))
			any = true
			for lane := 0; lane < 2; lane++ {
				dir := 1.0
				if lane == 1 {
					dir = -1
				}
				// Lane offset: right-hand traffic.
				var across float64
				line := ty
				if horizontal {
					across = 9
					if lane == 1 {
						across = 5
					}
				} else {
					across = 5
					if lane == 1 {
						across = 9
					}
					line = tx
				}
				seed := float64((line*7919+lane*104729)%97) * 1.7
				start := float64(tx * TileSize)
				if vertical {
					start = float64(ty * TileSize)
				}
				base := seed + dir*speed*v.Time
				k := math.Ceil((start - base) / spacing)
				for p := base + k*spacing; p < start+TileSize-2; p += spacing {
					idx := int(math.Abs(math.Floor((p-seed)/spacing))) + lane
					col := colors[idx%len(colors)]
					var wx, wy, w, h float64
					if horizontal {
						wx, wy, w, h = p, float64(ty*TileSize)+across, 3, 2
					} else {
						wx, wy, w, h = float64(tx*TileSize)+across, p, 2, 3
					}
					Rect(dst, ox+int(math.Round(wx*z)), oy+int(math.Round(wy*z)), int(w*z), int(h*z), col)
				}
			}
		}
	}
	return any
}
