package sprites

import (
	"image"
	"image/draw"

	"github.com/antoniowav/sprawl/theme"
)

// Icon composes the app icon from game sprites: a 2×2-tile scene (tower,
// apartments, house, road) on a rounded tile, scaled by an integer factor
// with no smoothing. It always uses the built-in palette so the icon
// doesn't change with the desktop theme.
func Icon(scale int) *image.RGBA {
	r := theme.Derive(theme.Builtin())
	b := BuildBuildings(r)
	n := BuildNetwork(r)
	scene := image.NewRGBA(image.Rect(0, 0, 2*T, 2*T))
	put := func(src *image.RGBA, x, y int) {
		draw.Draw(scene, image.Rect(x*T, y*T, x*T+T, y*T+T), src, image.Point{}, draw.Over)
	}
	put(b.Img[1][2][0], 0, 0) // tower
	put(b.Img[0][2][1], 1, 0) // apartments
	put(n.Road[E|W], 0, 1)
	put(b.Img[0][0][2], 1, 1) // house
	// Round the corners by clearing a 2-pixel notch.
	for _, p := range []image.Point{{0, 0}, {1, 0}, {0, 1}, {2*T - 1, 0}, {2*T - 2, 0}, {2*T - 1, 1},
		{0, 2*T - 1}, {1, 2*T - 1}, {0, 2*T - 2}, {2*T - 1, 2*T - 1}, {2*T - 2, 2*T - 1}, {2*T - 1, 2*T - 2}} {
		scene.Set(p.X, p.Y, image.Transparent)
	}
	out := image.NewRGBA(image.Rect(0, 0, 2*T*scale, 2*T*scale))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			out.Set(x, y, scene.At(x/scale, y/scale))
		}
	}
	return out
}
