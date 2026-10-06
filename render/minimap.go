package render

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/sim"
	"github.com/antoniowav/sprawl/theme"
)

// MinimapPixels fills pix (RGBA, W×H) with one pixel per tile.
func MinimapPixels(c *sim.City, r theme.Roles, pix []byte) {
	zone := [3]color.RGBA{r.ZoneR, r.ZoneC, r.ZoneI}
	for i := range c.Tiles {
		t := &c.Tiles[i]
		col := r.Grass
		switch {
		case t.Terrain == sim.Water && t.Kind != sim.Road:
			col = r.Water
		case t.Kind == sim.Road:
			col = theme.Mix(r.Road, r.UIText, 0.25)
		case t.IsZone():
			z := zone[t.Kind-sim.ZoneR]
			col = theme.Mix(r.Grass, z, 0.35+0.2*float64(t.Level))
		case t.IsBuilding():
			col = r.Walls[0]
		case t.Terrain == sim.Trees:
			col = r.Tree
		case t.Terrain == sim.Rock:
			col = r.Rock
		}
		pix[i*4], pix[i*4+1], pix[i*4+2], pix[i*4+3] = col.R, col.G, col.B, 0xff
	}
}

// Minimap draws the map thumbnail in the bottom-right corner above bottom,
// with the visible area outlined, and returns where it went.
func (h *HUD) Minimap(dst, img *ebiten.Image, view image.Rectangle, bottom int) image.Rectangle {
	s := h.S
	w := dst.Bounds().Dx()
	mw, mh := img.Bounds().Dx(), img.Bounds().Dy()
	side := 80 * s
	scale := float64(side) / float64(max(mw, mh))
	bw, bh := int(float64(mw)*scale), int(float64(mh)*scale)
	x0, y0 := w-bw-6*s, bottom-bh-6*s
	Rect(dst, x0-2*s, y0-2*s, bw+4*s, bh+4*s, h.R.UIPanel)
	Frame(dst, x0-2*s, y0-2*s, bw+4*s, bh+4*s, s, h.R.UIBorder)
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x0), float64(y0))
	dst.DrawImage(img, &op)
	// Visible area, clipped to the map.
	v := view.Intersect(image.Rect(0, 0, mw, mh))
	if !v.Empty() {
		vx, vy := x0+int(float64(v.Min.X)*scale), y0+int(float64(v.Min.Y)*scale)
		vw, vh := max(s, int(float64(v.Dx())*scale)), max(s, int(float64(v.Dy())*scale))
		Frame(dst, vx, vy, vw, vh, s, h.R.UIAccent)
	}
	r := image.Rect(x0, y0, x0+bw, y0+bh)
	h.Blocks = append(h.Blocks, r.Inset(-2*s))
	return r
}
