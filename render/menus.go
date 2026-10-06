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

// NewCityForm is the state of the new-city screen.
type NewCityForm struct {
	Name    string
	Size    int
	Seed    string
	Field   int // 0 name, 1 size, 2 seed, 3 start, 4 back
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
	scale := max(1, (128*s)/max(1, pv))
	pvs := pv * scale
	bw := 30*cw + pvs + 4*cw
	bh := max(pvs+10*s, 8*lh)
	x0, y0 := (w-bw)/2, (ht-bh)/2
	h.box(dst, x0, y0, bw, bh, "new city · ↑↓ field · ←→ change · Enter")

	x, y := x0+2*cw, y0+8*s
	field := func(i int, label, value, hint string) {
		r := image.Rect(x0+s, y-2*s, x0+30*cw, y-2*s+lh)
		h.Hits = append(h.Hits, r)
		if f.Field == i {
			Rect(dst, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), h.R.UISel)
		}
		h.text(dst, label, x, y, h.R.UIDim)
		end := h.text(dst, value, x+7*cw, y, h.R.UIText)
		if f.Field == i && hint == "" {
			h.text(dst, "█", end, y, h.R.UIAccent)
		}
		if hint != "" && f.Field == i {
			h.text(dst, hint, end+cw, y, h.R.UIDim)
		}
		y += lh + lh/2
	}
	field(0, "name", f.Name, "")
	field(1, "size", sizeLabel(f.Size), "←→")
	field(2, "seed", f.Seed, "")
	y += lh / 2
	for i, label := range []string{" Start ", " Back "} {
		bwid := font.Width(label, s) + 2*s
		bx := x + i*(bwid+2*cw)
		r := image.Rect(bx, y-2*s, bx+bwid, y-2*s+lh)
		h.Hits = append(h.Hits, r)
		bg, fg := h.R.UITrack, h.R.UIText
		if f.Field == 3+i {
			bg, fg = h.R.UIAccent, h.R.UIAccentText
		}
		Rect(dst, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), bg)
		h.text(dst, label, bx+s, y, fg)
	}
	h.text(dst, "←→ on seed rolls a new map", x, y+2*lh, h.R.UIDim)

	if f.Preview != nil {
		var op ebiten.DrawImageOptions
		op.GeoM.Scale(float64(scale), float64(scale))
		px, py := x0+bw-pvs-2*cw, y0+(bh-pvs)/2
		Frame(dst, px-s, py-s, pvs+2*s, pvs+2*s, s, h.R.UIBorder)
		op.GeoM.Translate(float64(px), float64(py))
		dst.DrawImage(f.Preview, &op)
	}
}

func sizeLabel(n int) string {
	switch {
	case n <= 96:
		return "small 96×96"
	case n <= 128:
		return "medium 128×128"
	}
	return "large 192×192"
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
