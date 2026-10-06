package font

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Face draws text from pre-rendered white glyph images, tinted per call.
type Face struct {
	img map[rune]*ebiten.Image
}

// New renders every glyph once.
func New() *Face {
	f := &Face{img: map[rune]*ebiten.Image{}}
	for r := range glyphs {
		f.img[r] = render(r)
	}
	return f
}

func render(r rune) *ebiten.Image {
	rgba := image.NewRGBA(image.Rect(0, 0, GlyphW, GlyphH))
	bits := Bits(r)
	for y, row := range bits {
		for x := 0; x < GlyphW; x++ {
			if row&(1<<(GlyphW-1-x)) != 0 {
				rgba.Set(x, y, color.White)
			}
		}
	}
	return ebiten.NewImageFromImage(rgba)
}

// Draw writes s at (x, y) (top-left of the cell) with integer scale.
func (f *Face) Draw(dst *ebiten.Image, s string, x, y, scale int, c color.Color) {
	var op ebiten.DrawImageOptions
	op.ColorScale.ScaleWithColor(c)
	cx := x
	for _, r := range s {
		g, ok := f.img[r]
		if !ok {
			g = f.img['?']
		}
		if r != ' ' {
			op.GeoM.Reset()
			op.GeoM.Scale(float64(scale), float64(scale))
			op.GeoM.Translate(float64(cx), float64(y))
			dst.DrawImage(g, &op)
		}
		cx += Advance * scale
	}
}

// Width is the drawn width of s in screen pixels.
func Width(s string, scale int) int {
	n := 0
	for range s {
		n++
	}
	if n == 0 {
		return 0
	}
	return (n*Advance - 1) * scale
}
