package app

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/antoniowav/sprawl/render"
)

// refreshMinimap rebuilds the thumbnail when the map changed, at most once
// a second (it's one pixel per tile, but no need to do it every tick).
func (a *App) refreshMinimap(now time.Time) {
	c := a.city
	if a.mini == nil || a.mini.Bounds().Dx() != c.W || a.mini.Bounds().Dy() != c.H {
		if a.mini != nil {
			a.mini.Deallocate()
		}
		a.mini = ebiten.NewImage(c.W, c.H)
		a.miniPix = make([]byte, c.W*c.H*4)
		a.miniDirty = true
	}
	if !a.miniDirty || now.Before(a.miniNext) {
		return
	}
	render.MinimapPixels(c, a.roles, a.miniPix)
	a.mini.WritePixels(a.miniPix)
	a.miniDirty, a.miniNext = false, now.Add(time.Second)
	a.dirty = true
}

// viewTiles is the visible area in tiles.
func (a *App) viewTiles() image.Rectangle {
	ts := render.TileSize * a.cam.Zoom
	ox, oy := a.cam.Origin(a.w, a.h)
	return image.Rect(-ox/ts, -oy/ts, (a.w-ox)/ts+1, (a.h-oy)/ts+1)
}

// updateMinimap pans the camera when the minimap is clicked or dragged.
// It reports whether it used the mouse.
func (a *App) updateMinimap(mx, my int) bool {
	r := a.hud.MiniRect
	if !a.showMinimap || a.scene != sceneGame || a.mode != modeNormal || r.Empty() {
		a.miniDrag = false
		return false
	}
	pt := image.Pt(mx, my)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && pt.In(r) {
		a.miniDrag = true
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		a.miniDrag = false
	}
	if !a.miniDrag {
		return false
	}
	fx := float64(mx-r.Min.X) / float64(r.Dx())
	fy := float64(my-r.Min.Y) / float64(r.Dy())
	fx, fy = max(0, min(1, fx)), max(0, min(1, fy))
	a.cam.Jump(fx*float64(a.city.W*render.TileSize), fy*float64(a.city.H*render.TileSize))
	a.dirty = true
	return true
}
