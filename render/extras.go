package render

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/render/font"
	"github.com/antoniowav/sprawl/sim"
)

// Toast draws a short message under the top bar (tips, achievements).
func (h *HUD) Toast(dst *ebiten.Image, msg string, highlight bool) {
	if msg == "" {
		return
	}
	s, cw, lh := h.S, h.cw(), h.lh()
	w := dst.Bounds().Dx()
	lines := wrap(msg, 56)
	bw := 0
	for _, l := range lines {
		bw = max(bw, len([]rune(l)))
	}
	bw = (bw + 4) * cw
	bh := len(lines)*lh + 6*s
	x0, y0 := (w-bw)/2, h.barH()+6*s
	border := h.R.UIBorder
	if highlight {
		border = h.R.UIAccent
	}
	Rect(dst, x0, y0, bw, bh, h.R.UIPanel)
	Frame(dst, x0, y0, bw, bh, s, border)
	for i, l := range lines {
		h.text(dst, l, x0+2*cw, y0+4*s+i*lh, h.R.UIText)
	}
	h.Blocks = append(h.Blocks, image.Rect(x0, y0, x0+bw, y0+bh))
}

// Inspector draws the building inspector in the bottom-left corner.
func (h *HUD) Inspector(dst *ebiten.Image, in sim.Inspection, bottom int) {
	s, cw, lh := h.S, h.cw(), h.lh()
	var rows []struct {
		s string
		c color.RGBA
	}
	add := func(t string, c color.RGBA) {
		for _, l := range wrap(t, 44) {
			rows = append(rows, struct {
				s string
				c color.RGBA
			}{l, c})
		}
	}
	for _, f := range in.Facts {
		add(f, h.R.UIText)
	}
	if len(in.Holding) > 0 {
		add("held back by:", h.R.UIWarn)
		for _, r := range in.Holding {
			add("· "+r, h.R.UIWarn)
		}
	}
	bw := 48 * cw
	bh := len(rows)*lh + 10*s
	x0, y0 := 34*s, bottom-bh-6*s
	h.box(dst, x0, y0, bw, bh, in.Title+" · Esc closes")
	for i, r := range rows {
		h.text(dst, r.s, x0+2*cw, y0+7*s+i*lh, r.c)
	}
	h.Blocks = append(h.Blocks, image.Rect(x0, y0, x0+bw, y0+bh))
}

// AchievementRow is one line of the achievements list.
type AchievementRow struct {
	Name, Desc, Date string
}

// Achievements draws the list of achievements.
func (h *HUD) Achievements(dst *ebiten.Image, rows []AchievementRow) {
	w, ht := dst.Bounds().Dx(), dst.Bounds().Dy()
	s, cw, lh := h.S, h.cw(), h.lh()
	Rect(dst, 0, 0, w, ht, color.RGBA{0, 0, 0, 0x90})
	done := 0
	for _, r := range rows {
		if r.Date != "" {
			done++
		}
	}
	bw := 64 * cw
	bh := len(rows)*lh*2 + 10*s
	x0, y0 := (w-bw)/2, (ht-bh)/2
	h.box(dst, x0, y0, bw, bh, "achievements · "+itoa(done)+"/"+itoa(len(rows))+" · Esc")
	for i, r := range rows {
		y := y0 + 7*s + i*2*lh
		mark, col := "·", h.R.UIDim
		if r.Date != "" {
			mark, col = "✓", h.R.UIOk
		}
		h.text(dst, mark, x0+2*cw, y, col)
		name := h.R.UIText
		if r.Date == "" {
			name = h.R.UIDim
		}
		h.text(dst, r.Name, x0+4*cw, y, name)
		h.text(dst, r.Date, x0+bw-2*cw-font.Width(r.Date, s), y, h.R.UIDim)
		h.text(dst, r.Desc, x0+4*cw, y+lh-2*s, h.R.UIDim)
	}
}

// KeyRow is one action in the key editor.
type KeyRow struct {
	Action, Help, Keys string
}

// KeyEditor draws the key-binding editor; capturing names the action
// waiting for a key, if any.
func (h *HUD) KeyEditor(dst *ebiten.Image, rows []KeyRow, sel int, capturing string) {
	w, ht := dst.Bounds().Dx(), dst.Bounds().Dy()
	s, cw, lh := h.S, h.cw(), h.lh()
	Rect(dst, 0, 0, w, ht, color.RGBA{0, 0, 0, 0x90})
	h.Hits = h.Hits[:0]
	visible := max(5, (ht-100*s)/lh)
	visible = min(visible, len(rows))
	first := max(0, min(sel-visible/2, len(rows)-visible))
	bw := 60 * cw
	bh := (visible+2)*lh + 10*s
	x0, y0 := (w-bw)/2, (ht-bh)/2
	h.box(dst, x0, y0, bw, bh, "keys · Enter set · Backspace default · Esc")
	for i := 0; i < visible; i++ {
		r := rows[first+i]
		y := y0 + 7*s + i*lh
		rect := image.Rect(x0+s, y-2*s, x0+bw-s, y-2*s+lh)
		h.Hits = append(h.Hits, rect)
		if first+i == sel {
			Rect(dst, rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy(), h.R.UISel)
		}
		h.text(dst, r.Help, x0+2*cw, y, h.R.UIText)
		h.text(dst, r.Keys, x0+30*cw, y, h.R.UIAccent)
	}
	msg := "↑↓ choose · changes save when you leave Settings"
	col := h.R.UIDim
	if capturing != "" {
		msg, col = "press the new key for "+capturing+" (Esc cancels)", h.R.UIWarn
	}
	h.text(dst, msg, x0+2*cw, y0+bh-lh-2*s, col)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
