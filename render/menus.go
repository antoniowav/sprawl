package render

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/internal/meta"
	"github.com/antoniowav/sprawl/render/font"
)

// MenuEntry is one line of a vertical menu.
type MenuEntry struct {
	Label    string
	Detail   string // dim text on the right
	Disabled bool
}

// Title draws the title screen: the logo and a menu, over whatever the
// caller drew behind it.
func (h *HUD) Title(dst *ebiten.Image, entries []MenuEntry, sel int, footer string) {
	w, ht := dst.Bounds().Dx(), dst.Bounds().Dy()
	s := h.S
	Rect(dst, 0, 0, w, ht, color.RGBA{0, 0, 0, 0x70})
	h.Hits = h.Hits[:0]

	logo := meta.Name
	big := 5 * s
	lw := font.Width(logo, big)
	ly := ht/2 - 26*h.lh()/4
	// Logo plate with a drop shadow, then the letters.
	pad := 6 * s
	Rect(dst, (w-lw)/2-pad+2*s, ly-pad+2*s, lw+2*pad, 7*big+2*pad, color.RGBA{0, 0, 0, 0x90})
	Rect(dst, (w-lw)/2-pad, ly-pad, lw+2*pad, 7*big+2*pad, h.R.UIAccent)
	h.Face.Draw(dst, logo, (w-lw)/2, ly, big, h.R.UIAccentText)
	sub := "a small city builder"
	h.text(dst, sub, (w-font.Width(sub, s))/2, ly+7*big+pad+6*s, h.R.UIText)

	h.list(dst, w, ly+7*big+pad+20*s, 30, "", entries, sel)
	if footer != "" {
		h.text(dst, footer, (w-font.Width(footer, s))/2, ht-12*s, h.R.UIDim)
	}
}

// PauseMenu draws the in-game menu.
func (h *HUD) PauseMenu(dst *ebiten.Image, entries []MenuEntry, sel int) {
	w, ht := dst.Bounds().Dx(), dst.Bounds().Dy()
	Rect(dst, 0, 0, w, ht, color.RGBA{0, 0, 0, 0x80})
	h.Hits = h.Hits[:0]
	h.list(dst, w, (ht-(len(entries)*h.lh()+10*h.S))/2, 30, "menu · Esc resumes", entries, sel)
}

// list draws a boxed menu centred horizontally at top y and records hit boxes.
func (h *HUD) list(dst *ebiten.Image, w, y, cols int, title string, entries []MenuEntry, sel int) {
	s, cw, lh := h.S, h.cw(), h.lh()
	bw := cols * cw
	bh := len(entries)*lh + 10*s
	x0 := (w - bw) / 2
	h.box(dst, x0, y, bw, bh, title)
	for i, e := range entries {
		ry := y + 7*s + i*lh
		r := image.Rect(x0+s, ry-2*s, x0+bw-s, ry-2*s+lh)
		h.Hits = append(h.Hits, r)
		fg := h.R.UIText
		switch {
		case e.Disabled:
			fg = h.R.UIDim
		case i == sel:
			Rect(dst, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), h.R.UIAccent)
			fg = h.R.UIAccentText
		}
		h.text(dst, e.Label, x0+2*cw, ry, fg)
		if e.Detail != "" {
			dc := h.R.UIDim
			if i == sel && !e.Disabled {
				dc = fg
			}
			h.text(dst, e.Detail, x0+bw-2*cw-font.Width(e.Detail, s), ry, dc)
		}
	}
}

// FormField is one row of the new-city form.
type FormField struct {
	Label, Value string
	Text         bool // typed text (shows a caret); otherwise ←→ choices
	Disabled     bool
}

// NewCityForm is the new-city screen: fields, then Start and Back.
type NewCityForm struct {
	Fields  []FormField
	Focus   int // field index, then len(Fields) = Start, +1 = Back
	Note    []string
	Preview *ebiten.Image
}

// NewCity draws the new-city form with a terrain preview.
func (h *HUD) NewCity(dst *ebiten.Image, f *NewCityForm) {
	w, ht := dst.Bounds().Dx(), dst.Bounds().Dy()
	s, cw, lh := h.S, h.cw(), h.lh()
	Rect(dst, 0, 0, w, ht, color.RGBA{0, 0, 0, 0x90})
	h.Hits = h.Hits[:0]

	pv := 0
	if f.Preview != nil {
		pv = f.Preview.Bounds().Dx()
	}
	pvs := 128 * s
	scale := float64(pvs) / float64(max(1, pv))
	leftW := 34 * cw
	bw := leftW + pvs + 4*cw
	rows := len(f.Fields)*3/2 + 3 + len(f.Note)
	bh := max(pvs+10*s, rows*lh+10*s)
	x0, y0 := (w-bw)/2, (ht-bh)/2
	h.box(dst, x0, y0, bw, bh, "new city · ↑↓ field · ←→ change · Enter")

	x, y := x0+2*cw, y0+8*s
	for i, fd := range f.Fields {
		r := image.Rect(x0+s, y-2*s, x0+leftW, y-2*s+lh)
		h.Hits = append(h.Hits, r)
		if f.Focus == i && !fd.Disabled {
			Rect(dst, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), h.R.UISel)
		}
		vc := h.R.UIText
		if fd.Disabled {
			vc = h.R.UIDim
		}
		h.text(dst, fd.Label, x, y, h.R.UIDim)
		val := fd.Value
		if !fd.Text && !fd.Disabled {
			val = "< " + val + " >"
		}
		end := h.text(dst, val, x+7*cw, y, vc)
		if f.Focus == i && fd.Text {
			h.text(dst, "█", end, y, h.R.UIAccent)
		}
		y += lh + lh/2
	}
	y += lh / 2
	for i, label := range []string{" Start ", " Back "} {
		bwid := font.Width(label, s) + 2*s
		bx := x + i*(bwid+2*cw)
		r := image.Rect(bx, y-2*s, bx+bwid, y-2*s+lh)
		h.Hits = append(h.Hits, r)
		bg, fg := h.R.UITrack, h.R.UIText
		if f.Focus == len(f.Fields)+i {
			bg, fg = h.R.UIAccent, h.R.UIAccentText
		}
		Rect(dst, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), bg)
		h.text(dst, label, bx+s, y, fg)
	}
	y += 2 * lh
	for _, n := range f.Note {
		h.text(dst, n, x, y, h.R.UIDim)
		y += lh
	}

	if f.Preview != nil {
		var op ebiten.DrawImageOptions
		op.GeoM.Scale(scale, scale)
		px, py := x0+bw-pvs-2*cw, y0+(bh-pvs)/2
		Frame(dst, px-s, py-s, pvs+2*s, pvs+2*s, s, h.R.UIBorder)
		op.GeoM.Translate(float64(px), float64(py))
		dst.DrawImage(f.Preview, &op)
	}
}

// SizeLabel names a map size.
func SizeLabel(n int) string {
	switch {
	case n <= 96:
		return "small 96×96"
	case n <= 128:
		return "medium 128×128"
	case n <= 192:
		return "large 192×192"
	}
	return "huge 256×256"
}

// Settings draws the settings list.
func (h *HUD) Settings(dst *ebiten.Image, entries []MenuEntry, sel int, path string) {
	w, ht := dst.Bounds().Dx(), dst.Bounds().Dy()
	Rect(dst, 0, 0, w, ht, color.RGBA{0, 0, 0, 0x90})
	h.Hits = h.Hits[:0]
	y := (ht - (len(entries)*h.lh() + 10*h.S)) / 2
	h.list(dst, w, y, 44, "settings · ←→ change · Esc saves", entries, sel)
	note := "Keys: edit [keys] in " + path
	h.text(dst, note, (w-font.Width(note, h.S))/2, y+len(entries)*h.lh()+16*h.S, h.R.UIDim)
}
