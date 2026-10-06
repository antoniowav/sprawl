package render

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/render/font"
	"github.com/antoniowav/sprawl/sim"
)

// Stats draws the statistics screen: population, funds and demand over
// the city's history.
func (h *HUD) Stats(dst *ebiten.Image, c *sim.City) {
	w, ht := dst.Bounds().Dx(), dst.Bounds().Dy()
	s, cw, lh := h.S, h.cw(), h.lh()
	Rect(dst, 0, 0, w, ht, color.RGBA{0, 0, 0, 0xa0})
	bw := min(w-16*s, 90*cw)
	bh := min(ht-32*s, 3*(14*lh))
	x0, y0 := (w-bw)/2, (ht-bh)/2
	h.box(dst, x0, y0, bw, bh, fmt.Sprintf("statistics · %s · %s · Esc", c.Name, c.Rank()))

	hist := c.History
	if len(hist) < 2 {
		h.text(dst, "Charts appear after the first two months.", x0+2*cw, y0+8*s, h.R.UIDim)
		return
	}
	panelH := (bh - 10*s) / 3
	px, pw := x0+12*cw, bw-14*cw
	type series struct {
		label string
		col   color.RGBA
		val   func(sim.Sample) float64
	}
	panels := []struct {
		title  string
		series []series
		signed bool
		money  bool
	}{
		{"population", []series{
			{"people", h.R.ZoneR, func(p sim.Sample) float64 { return float64(p.Pop) }},
			{"jobs", h.R.ZoneI, func(p sim.Sample) float64 { return float64(p.Jobs) }},
		}, false, false},
		{"funds", []series{{"funds", h.R.UIAccent, func(p sim.Sample) float64 { return p.Funds }}}, false, true},
		{"demand", []series{
			{"R", h.R.ZoneR, func(p sim.Sample) float64 { return p.Demand[0] }},
			{"C", h.R.ZoneC, func(p sim.Sample) float64 { return p.Demand[1] }},
			{"I", h.R.ZoneI, func(p sim.Sample) float64 { return p.Demand[2] }},
		}, true, false},
	}
	for pi, p := range panels {
		top := y0 + 6*s + pi*panelH
		cy0, cy1 := top+lh, top+panelH-lh/2
		lo, hi := 0.0, 1.0
		if p.signed {
			lo = -1
		} else {
			for _, se := range p.series {
				for _, smp := range hist {
					v := se.val(smp)
					hi, lo = max(hi, v), min(lo, v)
				}
			}
		}
		h.text(dst, p.title, x0+2*cw, top+2*s, h.R.UIAccent)
		fmtv := func(v float64) string {
			if p.money {
				return money(v)
			}
			if p.signed {
				return fmt.Sprintf("%+.1f", v)
			}
			return commas(int64(v))
		}
		h.text(dst, fmtv(hi), px-font.Width(fmtv(hi), s)-cw, cy0, h.R.UIDim)
		h.text(dst, fmtv(lo), px-font.Width(fmtv(lo), s)-cw, cy1-7*s, h.R.UIDim)
		Rect(dst, px, cy0, pw, cy1-cy0, h.R.UIBg)
		yOf := func(v float64) int { return cy1 - int((v-lo)/(hi-lo)*float64(cy1-cy0-s)) - s }
		if lo < 0 && hi > 0 {
			Rect(dst, px, yOf(0), pw, s, h.R.UITrack)
		}
		// Legend.
		lx := px + pw
		for i := len(p.series) - 1; i >= 0; i-- {
			se := p.series[i]
			lx -= font.Width(se.label, s) + 2*cw
			Rect(dst, lx-cw, top+4*s, cw/2+s, 3*s, se.col)
			h.text(dst, se.label, lx, top+2*s, h.R.UIDim)
		}
		// Lines: connect consecutive samples with vertical runs.
		n := len(hist)
		for _, se := range p.series {
			prevX, prevY := -1, 0
			for i, smp := range hist {
				x := px + i*(pw-s)/(n-1)
				y := yOf(se.val(smp))
				if prevX >= 0 {
					ya, yb := min(prevY, y), max(prevY, y)
					Rect(dst, prevX, ya, max(s, x-prevX), s, se.col)
					Rect(dst, x, ya, s, yb-ya+s, se.col)
				}
				prevX, prevY = x, y
			}
		}
	}
	first, last := hist[0], hist[len(hist)-1]
	span := sim.Date(first.Day) + " to " + sim.Date(last.Day)
	h.text(dst, span, px+(pw-font.Width(span, s))/2, y0+bh-lh, h.R.UIDim)
}
