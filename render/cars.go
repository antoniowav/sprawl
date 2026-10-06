package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Car is a car to draw, centred at (X, Y) in world pixels.
type Car struct {
	X, Y       float64
	Horizontal bool
	Color      int
}

// drawCars draws the cars of the trip simulation; it reports whether any
// are on screen (they move, so frames are needed).
func drawCars(dst *ebiten.Image, v WorldView, ox, oy int) bool {
	if !v.Animations || len(v.Cars) == 0 {
		return false
	}
	z := float64(v.Cam.Zoom)
	colors := []color.RGBA{v.Roles.Roofs[0], v.Roles.ZoneC, v.Roles.Walls[0], v.Roles.Roofs[3], v.Roles.ZoneI}
	shadow := premul(color.RGBA{0, 0, 0, 0x50})
	for _, c := range v.Cars {
		w, h := 3.0, 2.0
		if !c.Horizontal {
			w, h = 2, 3
		}
		x := ox + int(math.Round((c.X-w/2)*z))
		y := oy + int(math.Round((c.Y-h/2)*z))
		Rect(dst, x+int(z), y+int(z), int(w*z), int(h*z), shadow)
		Rect(dst, x, y, int(w*z), int(h*z), colors[c.Color%len(colors)])
	}
	return true
}
