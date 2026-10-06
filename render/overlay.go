package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/sim"
	"github.com/antoniowav/sprawl/theme"
)

// Overlay is a data view drawn over the map.
type Overlay int

const (
	OverlayNone Overlay = iota
	OverlayPower
	OverlayWater
	OverlayTraffic
	OverlayPolice
	OverlayFire
	OverlaySchool
	OverlayHealth
	OverlayLandValue
	OverlayPollution
	OverlayHeight
	overlayCount
)

var overlayNames = [...]string{"none", "power", "water", "traffic", "police", "fire", "school", "health",
	"land value", "pollution", "height"}

// Heat reports whether the overlay is a 0..1 heat map (it has a legend).
func (o Overlay) Heat() bool { return o >= OverlayTraffic }

// Service reports the Tile.Cover index a coverage overlay shows.
func (o Overlay) Service() (int, bool) {
	if o >= OverlayPolice && o <= OverlayHealth {
		return int(o - OverlayPolice), true
	}
	return 0, false
}

// heatValue is the 0..1 value a heat overlay shows for a tile.
func heatValue(o Overlay, t *sim.Tile) float64 {
	if s, ok := o.Service(); ok {
		return float64(t.Cover[s])
	}
	switch o {
	case OverlayTraffic:
		if t.Kind == sim.Road {
			return min(1, t.Congestion())
		}
	case OverlayLandValue:
		return float64(t.LandValue)
	case OverlayPollution:
		return float64(t.Pollution)
	case OverlayHeight:
		return float64(t.Height) / sim.MaxHeight
	}
	return 0
}

// HeatColor is the tint for value v of overlay o (premultiplied).
func HeatColor(r theme.Roles, o Overlay, v float64) color.RGBA {
	c := r.UIAccent
	if o == OverlayPollution || o == OverlayTraffic {
		c = r.UIWarn
	}
	c.A = uint8(0x10 + v*0xc0)
	return premul(c)
}

func (o Overlay) String() string { return overlayNames[o] }

// Next cycles to the following overlay.
func (o Overlay) Next() Overlay { return (o + 1) % overlayCount }

// drawOverlay dims the map and tints the tiles the overlay is about:
// served in the ok colour, unserved in the error colour.
func drawOverlay(dst *ebiten.Image, v WorldView, x0, y0, x1, y1, ox, oy, ts int, draw func(*ebiten.Image, int, int)) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	c := v.City
	Rect(dst, 0, 0, w, h, color.RGBA{0, 0, 0, 0x90})
	ok, bad := v.Roles.UIOk, v.Roles.UIErr
	ok.A, bad.A = 0x80, 0x90
	ok, bad = premul(ok), premul(bad)

	if v.Overlay.Heat() {
		for ty := y0; ty <= y1; ty++ {
			for tx := x0; tx <= x1; tx++ {
				if hv := heatValue(v.Overlay, c.At(tx, ty)); hv > 0.02 {
					Rect(dst, ox+tx*ts, oy+ty*ts, ts, ts, HeatColor(v.Roles, v.Overlay, hv))
				}
			}
		}
		return
	}

	for ty := y0; ty <= y1; ty++ {
		for tx := x0; tx <= x1; tx++ {
			t := c.At(tx, ty)
			var served, relevant bool
			switch v.Overlay {
			case OverlayPower:
				b, isB := sim.SpecOf(t.Kind)
				relevant = t.IsZone() || (isB && (b.PowerUse > 0 || b.PowerCap > 0)) || t.Line || t.Kind == sim.Road
				served = t.Powered
			case OverlayWater:
				if t.Pipe {
					draw(v.Atlas.Pipe[mask(c, tx, ty, func(t *sim.Tile) bool { return t.Pipe })], tx, ty)
				}
				b, isB := sim.SpecOf(t.Kind)
				relevant = (t.IsZone() && t.Level > 0) || (isB && (b.WaterUse > 0 || b.WaterCap > 0))
				served = t.Watered || (isB && b.WaterCap > 0)
			}
			if !relevant {
				continue
			}
			if served {
				Rect(dst, ox+tx*ts, oy+ty*ts, ts, ts, ok)
				continue
			}
			// Unserved: hatched, so it reads even when the theme's red and
			// green are close.
			var op ebiten.DrawImageOptions
			op.GeoM.Scale(float64(ts)/TileSize, float64(ts)/TileSize)
			op.GeoM.Translate(float64(ox+tx*ts), float64(oy+ty*ts))
			op.ColorScale.ScaleWithColor(bad)
			dst.DrawImage(v.Atlas.Hatch, &op)
		}
	}
}
