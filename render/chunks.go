package render

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/sim"
)

// chunkTiles is the side of a cached chunk, in tiles.
const chunkTiles = 32

// Chunks caches the static map layers in chunkTiles² images at 1× scale.
// A chunk is redrawn only after Touch marks one of its tiles, so a running
// city costs a few blits per frame instead of thousands of tile draws.
type Chunks struct {
	imgs  map[[2]int]*ebiten.Image
	dirty map[[2]int]bool
}

// NewChunks returns an empty cache.
func NewChunks() *Chunks {
	return &Chunks{imgs: map[[2]int]*ebiten.Image{}, dirty: map[[2]int]bool{}}
}

// Reset marks everything for redraw (new city, load, theme change).
func (k *Chunks) Reset() {
	for key := range k.imgs {
		k.dirty[key] = true
	}
}

// Touch marks the chunks of tile (x, y) and its neighbours: autotiles
// (roads, shores, lines) and building overhangs depend on them.
func (k *Chunks) Touch(x, y int) {
	for dy := -2; dy <= 1; dy++ {
		for dx := -2; dx <= 1; dx++ {
			k.dirty[[2]int{floorDiv(x+dx, chunkTiles), floorDiv(y+dy, chunkTiles)}] = true
		}
	}
}

func (k *Chunks) draw(dst *ebiten.Image, v WorldView, ox, oy, x0, y0, x1, y1 int) {
	z := v.Cam.Zoom
	cs := chunkTiles * TileSize
	var op ebiten.DrawImageOptions
	for cy := y0 / chunkTiles; cy <= y1/chunkTiles; cy++ {
		for cx := x0 / chunkTiles; cx <= x1/chunkTiles; cx++ {
			img := k.chunk(v, cx, cy)
			op.GeoM.Reset()
			op.GeoM.Scale(float64(z), float64(z))
			op.GeoM.Translate(float64(ox+cx*cs*z), float64(oy+cy*cs*z))
			dst.DrawImage(img, &op)
		}
	}
}

func (k *Chunks) chunk(v WorldView, cx, cy int) *ebiten.Image {
	key := [2]int{cx, cy}
	img, ok := k.imgs[key]
	if ok && !k.dirty[key] {
		return img
	}
	cs := chunkTiles * TileSize
	if !ok {
		img = ebiten.NewImage(cs, cs)
		k.imgs[key] = img
	}
	img.Fill(v.Roles.Void)
	var op ebiten.DrawImageOptions
	draw := func(src *ebiten.Image, tx, ty int) {
		op.GeoM.Reset()
		op.GeoM.Translate(float64((tx-cx*chunkTiles)*TileSize), float64((ty-cy*chunkTiles)*TileSize))
		img.DrawImage(src, &op)
	}
	c := v.City
	x0, y0 := cx*chunkTiles, cy*chunkTiles
	x1, y1 := min(c.W-1, x0+chunkTiles-1), min(c.H-1, y0+chunkTiles-1)
	if x0 <= x1 && y0 <= y1 {
		drawStatic(draw, v.Atlas, c, x0, y0, x1, y1, v.isGrowing)
	}
	delete(k.dirty, key)
	return img
}

// TouchAll marks the tiles of a plan or edit.
func (k *Chunks) TouchAll(pts []sim.Pt) {
	for _, p := range pts {
		k.Touch(p.X, p.Y)
	}
}
