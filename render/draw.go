package render

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var pixel = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	// Use the centre pixel so filtering never samples the image edge.
	return img.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}()

// Rect fills a rectangle in screen pixels.
func Rect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(float64(w), float64(h))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(pixel, &op)
}

// Frame draws a rectangle outline of thickness t.
func Frame(dst *ebiten.Image, x, y, w, h, t int, c color.Color) {
	Rect(dst, x, y, w, t, c)
	Rect(dst, x, y+h-t, w, t, c)
	Rect(dst, x, y, t, h, c)
	Rect(dst, x+w-t, y, t, h, c)
}
