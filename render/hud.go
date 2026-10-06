package render

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/internal/meta"
	"github.com/antoniowav/sprawl/render/font"
	"github.com/antoniowav/sprawl/sim"
	"github.com/antoniowav/sprawl/theme"
)

// HelpRow is one line of the keybinding overlay.
type HelpRow struct {
	Group, Keys, Desc string
}

// HUDState is everything the HUD shows. The app fills it each frame.
type HUDState struct {
	City             *sim.City
	Paused           bool
	Speed            int
	Mode             string
	Tool             string
	CursorX, CursorY int
	TileInfo         string
	Hint             string // cost or refusal for the pending action
	HintBad          bool
	Message          string
	MessageLevel     sim.EventLevel
	Prompt           *string // non-nil while the command prompt is open
	ShowPanel        bool
	ShowLog          bool
	ShowHelp         bool
	Help             []HelpRow
	Menu             *Menu // build menu, when open
	Dialog           *Dialog
	Toolbar          []ToolButton
	Minimap          *ebiten.Image
	MinimapView      image.Rectangle // visible tiles
	Guide            *Guide
	ToolHover        int  // -1 when the mouse isn't over the toolbar
	Only             bool // draw only the dialog (over the title screen)
	Forecast         sim.Ledger
	ShowBudget       bool
	Zoom             int
	Overlay          Overlay
	ThemeName        string
}

// Menu is the build menu's content.
type Menu struct {
	Items []MenuItem
	Sel   int
}

// MenuItem is one row of the build menu.
type MenuItem struct {
	Name, Size, Cost, Note string
	Locked                 bool
}

// HUD draws the dashboard chrome.
type HUD struct {
	Face *font.Face
	S    int // UI pixel scale
	R    theme.Roles

	// Hits are the clickable rows or buttons of the open dialog, in screen
	// pixels, refreshed every time it is drawn.
	Hits []image.Rectangle
	// ToolHits are the toolbar buttons; Blocks are all opaque HUD areas,
	// where mouse clicks must not reach the map.
	ToolHits []image.Rectangle
	Blocks   []image.Rectangle

	layer      *ebiten.Image
	lastKey    string
	baseBlocks int

	MiniRect image.Rectangle // where the minimap was drawn
}

// DrawCached draws the HUD through an offscreen layer that is only redrawn
// when key (a digest of everything the HUD shows) changes. Most frames
// while the city runs change the map, not the HUD, so this saves redrawing
// hundreds of glyphs.
func (h *HUD) DrawCached(dst *ebiten.Image, st HUDState, key string) {
	b := dst.Bounds()
	if h.layer == nil || h.layer.Bounds() != b {
		if h.layer != nil {
			h.layer.Deallocate()
		}
		h.layer = ebiten.NewImage(b.Dx(), b.Dy())
		h.lastKey = ""
	}
	if key != h.lastKey {
		h.layer.Clear()
		h.Draw(h.layer, st)
		h.lastKey = key
		h.baseBlocks = len(h.Blocks)
	}
	// Panels drawn outside the cache (toast, inspector) add their blocks
	// every frame; drop last frame's.
	h.Blocks = h.Blocks[:min(h.baseBlocks, len(h.Blocks))]
	dst.DrawImage(h.layer, nil)
}

// BottomBars is the height of the status line plus the log pane if shown.
func (h *HUD) BottomBars(log bool) int {
	n := h.barH()
	if log {
		n += logLines*h.lh() + 8*h.S
	}
	return n
}

// Invalidate forces the next DrawCached to redraw (theme or scale change).
func (h *HUD) Invalidate() { h.lastKey = "" }

// ToolButton is one toolbar entry.
type ToolButton struct {
	Icon   *ebiten.Image
	Active bool
	Tip    string
}

// Dialog is a modal: a text field, a list, or buttons.
type Dialog struct {
	Title   string
	Text    []string
	Input   *string
	List    [][2]string // name, date
	Buttons []string
	Sel     int
	Hint    string
}

func (h *HUD) cw() int   { return font.Advance * h.S }
func (h *HUD) lh() int   { return font.LineH * h.S }
func (h *HUD) barH() int { return 13 * h.S }

func (h *HUD) text(dst *ebiten.Image, s string, x, y int, c color.Color) int {
	h.Face.Draw(dst, s, x, y, h.S, c)
	return x + len([]rune(s))*h.cw()
}

// Draw renders all HUD layers onto dst.
func (h *HUD) Draw(dst *ebiten.Image, st HUDState) {
	w, ht := dst.Bounds().Dx(), dst.Bounds().Dy()
	if st.Only {
		h.Hits = h.Hits[:0]
		h.dialog(dst, w, ht, st.Dialog)
		return
	}
	h.Blocks = h.Blocks[:0]
	h.Blocks = append(h.Blocks, image.Rect(0, 0, w, h.barH()), image.Rect(0, ht-h.barH(), w, ht))
	h.topBar(dst, w, st)
	bottom := ht - h.barH()
	if st.ShowLog {
		bottom = h.logPane(dst, w, bottom, st)
	}
	if st.ShowPanel {
		h.panel(dst, w, h.barH(), bottom, st)
	}
	h.toolbar(dst, bottom, st)
	h.MiniRect = image.Rectangle{}
	if st.Minimap != nil {
		h.MiniRect = h.Minimap(dst, st.Minimap, st.MinimapView, bottom)
	}
	if st.Guide != nil {
		h.guide(dst, st.Guide)
	}
	h.statusLine(dst, w, ht, st)
	if st.Menu != nil {
		h.menu(dst, w, ht, st.Menu)
	}
	h.Hits = h.Hits[:0]
	if st.Dialog != nil {
		h.dialog(dst, w, ht, st.Dialog)
	}
	if st.ShowBudget {
		h.budget(dst, w, ht, st)
	}
	if st.City.Bankrupt {
		h.bankrupt(dst, w, ht, st)
	}
	if st.ShowHelp {
		h.help(dst, w, ht, st.Help)
	}
}

func (h *HUD) topBar(dst *ebiten.Image, w int, st HUDState) {
	s, bh := h.S, h.barH()
	Rect(dst, 0, 0, w, bh, h.R.UIBg)
	Rect(dst, 0, bh-s, w, s, h.R.UIBorder)
	y := 3 * s
	x := h.tag(dst, 0, 0, " "+strings.ToUpper(meta.Name)+" ", h.R.UIAccent, h.R.UIAccentText)
	x = h.text(dst, "  "+st.City.Name, x, y, h.R.UIText)
	leftEnd := h.text(dst, " · "+st.City.Rank(), x, y, h.R.UIDim)

	// Right side, laid out right to left.
	pop := commas(int64(st.City.Stats.Residents))
	funds := money(st.City.Funds)
	fundsCol := h.R.UIText
	if st.City.Funds < 0 {
		fundsCol = h.R.UIErr
	}
	speed, speedCol := strings.Repeat("▶", st.Speed), h.R.UIAccent
	if st.Paused {
		speed, speedCol = "▮▮ paused", h.R.UIWarn
	}
	parts := []struct {
		s string
		c color.RGBA
	}{
		{speed, speedCol}, {"   ", h.R.UIDim}, {funds, fundsCol}, {"   pop ", h.R.UIDim}, {pop, h.R.UIText}, {" ", h.R.UIDim},
	}
	total := 0
	for _, p := range parts {
		total += len([]rune(p.s))
	}
	x = w - total*h.cw() - s
	rightStart := x
	for _, p := range parts {
		x = h.text(dst, p.s, x, y, p.c)
	}

	// Date and goal in the middle, pushed right or shortened so they never
	// run into the name on the left or the money on the right.
	date := sim.Date(st.City.Day)
	goal := "  " + st.City.Goal()
	gap := 2 * h.cw()
	room := rightStart - leftEnd - 2*gap
	if font.Width(date+goal, s) > room {
		goal = ""
	}
	dx := max(leftEnd+gap, (w-font.Width(date+goal, s))/2)
	if font.Width(date, s) <= room {
		end := h.text(dst, date, dx, y, h.R.UIText)
		h.text(dst, goal, end, y, h.R.UIDim)
	}
}

// tag draws a filled label the full height of a bar and returns its right edge.
func (h *HUD) tag(dst *ebiten.Image, x, y int, s string, bg, fg color.RGBA) int {
	wd := len([]rune(s)) * h.cw()
	Rect(dst, x, y, wd, h.barH()-h.S, bg)
	h.text(dst, s, x, y+3*h.S, fg)
	return x + wd
}

func (h *HUD) statusLine(dst *ebiten.Image, w, ht int, st HUDState) {
	s, bh := h.S, h.barH()
	y0 := ht - bh
	Rect(dst, 0, y0, w, bh, h.R.UIBg)
	Rect(dst, 0, y0, w, s, h.R.UIBorder)
	y := y0 + 4*s
	if st.Prompt != nil {
		x := h.tag(dst, 0, y0+s, " COMMAND ", h.R.UIOk, h.R.UIAccentText)
		x = h.text(dst, " :"+*st.Prompt, x, y, h.R.UIText)
		h.text(dst, "█", x, y, h.R.UIAccent)
		return
	}
	right, col := fmt.Sprintf("zoom %dx  Ctrl+S save  ? help ", st.Zoom), h.R.UIDim
	if st.Message != "" {
		right = st.Message + " "
		col = [...]color.RGBA{h.R.UIText, h.R.UIWarn, h.R.UIErr}[st.MessageLevel]
	}
	rx := w - len([]rune(right))*h.cw()
	h.text(dst, right, rx, y, col)

	tagBg := h.R.UIAccent
	switch {
	case strings.HasPrefix(st.Mode, "VISUAL"), strings.HasPrefix(st.Mode, "ZONE"):
		tagBg = h.R.UIWarn
	case st.Mode == "PAINT":
		tagBg = h.R.UIOk
	}
	x := h.tag(dst, 0, y0+s, " "+st.Mode+" ", tagBg, h.R.UIAccentText)
	hintCol := h.R.UIOk
	if st.HintBad {
		hintCol = h.R.UIErr
	}
	// Left side segments in priority order; stop before running into the
	// right-hand message.
	limit := rx - 2*h.cw()
	for _, seg := range []struct {
		s string
		c color.RGBA
	}{
		{"  " + st.Tool, h.R.UIText},
		{"  " + st.Hint, hintCol},
		{fmt.Sprintf("  (%d,%d)", st.CursorX, st.CursorY), h.R.UIDim},
		{"  " + st.TileInfo, h.R.UIText},
	} {
		if strings.TrimSpace(seg.s) == "" {
			continue
		}
		if x+len([]rune(seg.s))*h.cw() > limit {
			break
		}
		x = h.text(dst, seg.s, x, y, seg.c)
	}
}

// box draws a bordered panel with a title set into the top border.
func (h *HUD) box(dst *ebiten.Image, x, y, w, ht int, title string) {
	s := h.S
	Rect(dst, x, y, w, ht, h.R.UIPanel)
	Frame(dst, x, y, w, ht, s, h.R.UIBorder)
	if title != "" {
		h.divider(dst, x, y, w, title)
	}
}

// divider draws "├ title ───┤" across a box at row y.
func (h *HUD) divider(dst *ebiten.Image, x, y, w int, title string) {
	s := h.S
	Rect(dst, x, y, w, s, h.R.UIBorder)
	tw := font.Width(" "+title+" ", s)
	tx := x + 3*h.cw()/2
	Rect(dst, tx, y-3*s, tw+2*s, 7*s, h.R.UIPanel)
	h.text(dst, " "+title, tx, y-3*s, h.R.UIDim)
}

const (
	panelCols  = 26
	sectionPad = 11 // header gap above a section's rows, in UI pixels
)

func (h *HUD) panel(dst *ebiten.Image, w, top, bottom int, st HUDState) {
	s, cw, lh := h.S, h.cw(), h.lh()
	pw := panelCols*cw + 4*cw/2
	x0 := w - pw - 4*s
	y := top + 8*s
	ix := x0 + cw
	iw := pw - 2*cw

	type section struct {
		title string
		rows  int
		draw  func(y int)
	}
	c := st.City
	secs := []section{
		{"demand", 3, func(y int) {
			for i, z := range []struct {
				l string
				c color.RGBA
			}{{"R", h.R.ZoneR}, {"C", h.R.ZoneC}, {"I", h.R.ZoneI}} {
				ry := y + i*lh
				h.text(dst, z.l, ix, ry, z.c)
				h.signedBar(dst, ix+2*cw, ry+s, iw-2*cw, c.Demand[i], z.c)
			}
		}},
		{"power", 1, func(y int) { h.meter(dst, ix, y, iw, c.Power, h.R.Power) }},
		{"water", 1, func(y int) { h.meter(dst, ix, y, iw, c.Water, h.R.Pipe) }},
		{"jobs", 2, func(y int) {
			h.kv(dst, ix, y, iw, "workers", commas(int64(c.Stats.Workforce())), h.R.UIText)
			h.kv(dst, ix, y+lh, iw, "jobs c/i", fmt.Sprintf("%s / %s", commas(int64(c.Stats.CommJobs)), commas(int64(c.Stats.IndJobs))), h.R.UIText)
		}},
		{"budget", 2, func(y int) {
			h.kv(dst, ix, y, iw, "tax r/c/i", fmt.Sprintf("%d/%d/%d", c.Tax[0], c.Tax[1], c.Tax[2]), h.R.UIText)
			net, col := st.Forecast.Net, h.R.UIOk
			if net < 0 {
				col = h.R.UIErr
			}
			h.kv(dst, ix, y+lh, iw, "net", fmt.Sprintf("%s/mo", signed(net)), col)
		}},
		{"view", 3, func(y int) {
			h.kv(dst, ix, y, iw, "overlay", st.Overlay.String(), h.R.UIText)
			if st.Overlay.Heat() {
				// Legend: low → high ramp.
				h.text(dst, "low", ix, y+lh, h.R.UIDim)
				rx, rw := ix+4*cw, iw-9*cw
				Rect(dst, rx, y+lh+s, rw, 5*s, h.R.UITrack)
				for i := 0; i < rw; i += s {
					Rect(dst, rx+i, y+lh+s, s, 5*s, HeatColor(h.R, st.Overlay, float64(i)/float64(rw)))
				}
				h.text(dst, "high", ix+iw-font.Width("high", s), y+lh, h.R.UIDim)
			}
			h.kv(dst, ix, y+2*lh, iw, "theme", st.ThemeName, h.R.UIText)
		}},
	}
	total := 0
	for _, sec := range secs {
		total += sectionPad*s + sec.rows*lh
	}
	ht := total + s
	if y+ht > bottom-4*s {
		return // window too small; the status line still shows the essentials
	}
	h.box(dst, x0, y, pw, ht, "")
	h.Blocks = append(h.Blocks, image.Rect(x0, y, x0+pw, y+ht))
	yy := y
	for _, sec := range secs {
		h.divider(dst, x0, yy, pw, sec.title)
		sec.draw(yy + 8*s)
		yy += sectionPad*s + sec.rows*lh
	}
}

// toolbar draws the tool buttons down the left edge, with a tooltip for
// the hovered one.
func (h *HUD) toolbar(dst *ebiten.Image, bottom int, st HUDState) {
	h.ToolHits = h.ToolHits[:0]
	if len(st.Toolbar) == 0 {
		return
	}
	s := h.S
	size := 20 * s
	gap := 2 * s
	x0, y0 := 4*s, h.barH()+6*s
	n := len(st.Toolbar)
	// Shrink to fit short windows: one column, scale icons down to 1× if needed.
	total := n*(size+gap) + 2*s
	if y0+total > bottom-4*s {
		return
	}
	Rect(dst, x0, y0, size+4*s, total+2*s, h.R.UIPanel)
	Frame(dst, x0, y0, size+4*s, total+2*s, s, h.R.UIBorder)
	h.Blocks = append(h.Blocks, image.Rect(x0, y0, x0+size+4*s, y0+total+2*s))
	for i, b := range st.Toolbar {
		bx, by := x0+2*s, y0+2*s+i*(size+gap)
		r := image.Rect(bx, by, bx+size, by+size)
		h.ToolHits = append(h.ToolHits, r)
		switch {
		case b.Active:
			Rect(dst, bx, by, size, size, h.R.UIAccent)
		case i == st.ToolHover:
			Rect(dst, bx, by, size, size, h.R.UISel)
		}
		var op ebiten.DrawImageOptions
		op.GeoM.Scale(float64(s), float64(s))
		op.GeoM.Translate(float64(bx+2*s), float64(by+2*s))
		dst.DrawImage(b.Icon, &op)
	}
	if i := st.ToolHover; i >= 0 && i < n {
		tip := st.Toolbar[i].Tip
		r := h.ToolHits[i]
		tw := font.Width(" "+tip+" ", s) + 2*s
		tx, ty := r.Max.X+6*s, r.Min.Y+(r.Dy()-h.lh())/2
		Rect(dst, tx, ty, tw, h.lh(), h.R.UIBg)
		Frame(dst, tx, ty, tw, h.lh(), s, h.R.UIBorder)
		h.text(dst, " "+tip, tx+s, ty+2*s, h.R.UIText)
	}
}

// Guide is the getting-started panel.
type Guide struct {
	Step, Total int
	Text        string
	Done        bool
}

func (h *HUD) guide(dst *ebiten.Image, g *Guide) {
	s, cw, lh := h.S, h.cw(), h.lh()
	cols := 36
	lines := wrap(g.Text, cols-4)
	bw := cols * cw
	bh := (len(lines)+1)*lh + 10*s
	x0, y0 := 34*s, h.barH()+8*s
	title := fmt.Sprintf("getting started %d/%d · F1 hides", g.Step, g.Total)
	if g.Done {
		title = "getting started · done · F1 hides"
	}
	h.box(dst, x0, y0, bw, bh, title)
	h.Blocks = append(h.Blocks, image.Rect(x0, y0, x0+bw, y0+bh))
	// Progress pips.
	for i := 0; i < g.Total; i++ {
		c := h.R.UITrack
		if i < g.Step-1 || g.Done {
			c = h.R.UIOk
		} else if i == g.Step-1 {
			c = h.R.UIAccent
		}
		Rect(dst, x0+2*cw+i*4*s, y0+7*s, 3*s, 3*s, c)
	}
	for i, l := range lines {
		h.text(dst, l, x0+2*cw, y0+7*s+(i+1)*lh, h.R.UIText)
	}
}

// wrap breaks text into lines of at most n characters, on spaces.
func wrap(text string, n int) []string {
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		line := ""
		for _, w := range strings.Fields(para) {
			if line != "" && len([]rune(line))+1+len([]rune(w)) > n {
				lines = append(lines, line)
				line = ""
			}
			if line != "" {
				line += " "
			}
			line += w
		}
		lines = append(lines, line)
	}
	return lines
}

func (h *HUD) kv(dst *ebiten.Image, x, y, w int, k, v string, vc color.RGBA) {
	h.text(dst, k, x, y, h.R.UIDim)
	h.text(dst, v, x+w-font.Width(v, h.S), y, vc)
}

// signedBar draws a centred bar for v in -1..1.
func (h *HUD) signedBar(dst *ebiten.Image, x, y, w int, v float64, c color.RGBA) {
	s := h.S
	bh := 5 * s
	Rect(dst, x, y, w, bh, h.R.UITrack)
	mid := x + w/2
	Rect(dst, mid, y-s, s, bh+2*s, h.R.UIDim)
	n := int(v * float64(w/2))
	if n > 0 {
		Rect(dst, mid+s, y, n, bh, c)
	} else if n < 0 {
		Rect(dst, mid+n, y, -n, bh, h.R.UIErr)
	}
}

// meter draws "demand / capacity" with a fill bar; red when demand is
// more than the network can serve.
func (h *HUD) meter(dst *ebiten.Image, x, y, w int, u sim.Utility, c color.RGBA) {
	s := h.S
	label := fmt.Sprintf("%s / %s", commas(int64(u.Demand)), commas(int64(u.Capacity)))
	col, tcol := c, h.R.UIText
	if u.Demand > u.Capacity {
		col, tcol = h.R.UIErr, h.R.UIErr
	}
	bw := w - font.Width(label, s) - 2*h.cw()
	Rect(dst, x, y+s, bw, 5*s, h.R.UITrack)
	if u.Capacity > 0 {
		f := min(1, float64(u.Demand)/float64(u.Capacity))
		Rect(dst, x, y+s, int(f*float64(bw)), 5*s, col)
	}
	h.text(dst, label, x+w-font.Width(label, s), y, tcol)
}

func (h *HUD) menu(dst *ebiten.Image, w, ht int, m *Menu) {
	s, cw, lh := h.S, h.cw(), h.lh()
	cols := 60
	bw := cols * cw
	bh := len(m.Items)*lh + 10*s
	x0, y0 := (w-bw)/2, (ht-bh)/2
	h.box(dst, x0, y0, bw, bh, "buildings · 1-9 0 or ↑↓ Enter · Esc")
	h.Hits = h.Hits[:0]
	for i, it := range m.Items {
		y := y0 + 7*s + i*lh
		x := x0 + cw
		r := image.Rect(x0+s, y-2*s, x0+bw-s, y-2*s+lh)
		h.Hits = append(h.Hits, r)
		if i == m.Sel {
			Rect(dst, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), h.R.UISel)
		}
		name, cost := h.R.UIText, h.R.UIText
		if it.Locked {
			name, cost = h.R.UIDim, h.R.UIDim
		}
		x = h.text(dst, fmt.Sprintf("%d ", (i+1)%10), x, y, h.R.UIAccent)
		h.text(dst, it.Name, x, y, name)
		h.text(dst, it.Size, x0+19*cw, y, h.R.UIDim)
		h.text(dst, it.Cost, x0+25*cw, y, cost)
		h.text(dst, it.Note, x0+33*cw, y, h.R.UIDim)
	}
}

const logLines = 6

func (h *HUD) logPane(dst *ebiten.Image, w, bottom int, st HUDState) int {
	s, lh := h.S, h.lh()
	ht := logLines*lh + 8*s
	y0 := bottom - ht
	h.box(dst, 0, y0, w, ht+s, "log")
	h.Blocks = append(h.Blocks, image.Rect(0, y0, w, y0+ht+s))
	ev := st.City.Log
	if len(ev) > logLines {
		ev = ev[len(ev)-logLines:]
	}
	for i, e := range ev {
		y := y0 + 5*s + i*lh
		x := h.text(dst, fmt.Sprintf("[day %d] ", e.Day), h.cw(), y, h.R.UIDim)
		msg := []rune(e.Msg)
		if fit := (w-x)/h.cw() - 1; len(msg) > fit && fit > 1 {
			msg = append(msg[:fit-1], '…')
		}
		h.text(dst, string(msg), x, y, [...]color.RGBA{h.R.UIText, h.R.UIWarn, h.R.UIErr}[e.Level])
	}
	return y0
}

func (h *HUD) help(dst *ebiten.Image, w, ht int, rows []HelpRow) {
	s, cw, lh := h.S, h.cw(), h.lh()
	Rect(dst, 0, 0, w, ht, color.RGBA{0, 0, 0, 0x90})

	// Two columns, split at a group boundary near the middle.
	type line struct {
		head       bool
		keys, desc string
	}
	var lines []line
	last := ""
	for _, r := range rows {
		if r.Group != last {
			if last != "" {
				lines = append(lines, line{})
			}
			lines = append(lines, line{head: true, keys: r.Group})
			last = r.Group
		}
		lines = append(lines, line{keys: r.Keys, desc: r.Desc})
	}
	// Split at the group boundary that balances the two columns best.
	split, best := len(lines), len(lines)
	for i, l := range lines {
		if l.head && i > 0 {
			if worst := max(i, len(lines)-i); worst < best {
				split, best = i, worst
			}
		}
	}
	cols := [][]line{lines[:split], lines[split:]}
	nrows := max(len(cols[0]), len(cols[1]))

	const keyW, descW = 20, 18
	colW := (keyW + descW + 2) * cw
	bw := 2*colW + 2*cw
	bh := nrows*lh + 12*s
	x0, y0 := (w-bw)/2, (ht-bh)/2
	h.box(dst, x0, y0, bw, bh, "keys · ? or Esc closes")
	for ci, col := range cols {
		for ri, l := range col {
			x := x0 + 2*cw + ci*colW
			y := y0 + 7*s + ri*lh
			if l.head {
				h.text(dst, strings.ToUpper(l.keys), x, y, h.R.UIAccent)
				continue
			}
			h.text(dst, l.keys, x, y, h.R.UIText)
			h.text(dst, l.desc, x+keyW*cw, y, h.R.UIDim)
		}
	}
}

// money formats dollars with the sign in front: -$7,000.
func money(v float64) string {
	if v < 0 {
		return "-$" + commas(int64(-v))
	}
	return "$" + commas(int64(v))
}

func signed(v float64) string {
	if v < 0 {
		return "-$" + commas(int64(-v))
	}
	return "+$" + commas(int64(v))
}

// budget is the :budget breakdown, using this month's forecast.
func (h *HUD) budget(dst *ebiten.Image, w, ht int, st HUDState) {
	s, cw, lh := h.S, h.cw(), h.lh()
	l, c := st.Forecast, st.City
	type row struct {
		k, v string
		col  color.RGBA
	}
	money := func(v float64) string { return "$" + commas(int64(v)) }
	rows := []row{
		{"income", "", h.R.UIAccent},
		{fmt.Sprintf("  residential  %d%%", c.Tax[0]), money(l.Income[0]), h.R.UIText},
		{fmt.Sprintf("  commercial   %d%%", c.Tax[1]), money(l.Income[1]), h.R.UIText},
		{fmt.Sprintf("  industrial   %d%%", c.Tax[2]), money(l.Income[2]), h.R.UIText},
		{"", "", h.R.UIText},
		{"upkeep", "", h.R.UIAccent},
		{"  roads", money(l.Roads), h.R.UIText},
		{"  power lines", money(l.Lines), h.R.UIText},
		{"  pipes", money(l.Pipes), h.R.UIText},
	}
	for _, b := range sim.Buildings {
		if v := l.Buildings[b.Kind]; v > 0 {
			rows = append(rows, row{"  " + b.Tool.String() + "s", money(v), h.R.UIText})
		}
	}
	if l.LoanPaid > 0 {
		rows = append(rows, row{"  loan", money(l.LoanPaid), h.R.UIText})
	}
	netCol := h.R.UIOk
	if l.Net < 0 {
		netCol = h.R.UIErr
	}
	rows = append(rows, row{"", "", h.R.UIText}, row{"net per month", signed(l.Net), netCol},
		row{"last month", signed(c.LastMonth.Net), h.R.UIDim})
	if c.Loan != nil {
		rows = append(rows, row{"loan left", fmt.Sprintf("$%s · %d mo", commas(int64(c.Loan.Remaining)), c.Loan.MonthsLeft), h.R.UIWarn})
	}
	if c.DebtMonths > 0 {
		rows = append(rows, row{"months in debt", fmt.Sprintf("%d of 12", c.DebtMonths), h.R.UIErr})
	}
	bw := 44 * cw
	bh := len(rows)*lh + 10*s
	x0, y0 := (w-bw)/2, (ht-bh)/2
	h.box(dst, x0, y0, bw, bh, "budget · :tax :loan :repay · Esc")
	for i, r := range rows {
		y := y0 + 7*s + i*lh
		h.text(dst, r.k, x0+2*cw, y, r.col)
		h.text(dst, r.v, x0+bw-2*cw-font.Width(r.v, s), y, r.col)
	}
}

func (h *HUD) dialog(dst *ebiten.Image, w, ht int, d *Dialog) {
	s, cw, lh := h.S, h.cw(), h.lh()
	Rect(dst, 0, 0, w, ht, color.RGBA{0, 0, 0, 0x80})
	rows := len(d.Text) + len(d.List)
	if d.Input != nil {
		rows += 2
	}
	if len(d.Buttons) > 0 {
		rows += 2
	}
	if d.Hint != "" {
		rows += 2
	}
	bw := 52 * cw
	bh := rows*lh + 10*s
	x0, y0 := (w-bw)/2, (ht-bh)/2
	h.box(dst, x0, y0, bw, bh, d.Title)
	x, y := x0+2*cw, y0+7*s
	for _, t := range d.Text {
		h.text(dst, t, x, y, h.R.UIText)
		y += lh
	}
	if d.Input != nil {
		y += lh / 2
		Rect(dst, x-s, y-2*s, bw-4*cw+2*s, lh, h.R.UITrack)
		end := h.text(dst, *d.Input, x, y, h.R.UIText)
		h.text(dst, "█", end, y, h.R.UIAccent)
		y += lh + lh/2
	}
	for i, it := range d.List {
		r := image.Rect(x0+s, y-2*s, x0+bw-s, y-2*s+lh)
		if i == d.Sel {
			Rect(dst, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), h.R.UISel)
		}
		h.Hits = append(h.Hits, r)
		h.text(dst, it[0], x, y, h.R.UIText)
		h.text(dst, it[1], x0+bw-2*cw-font.Width(it[1], s), y, h.R.UIDim)
		y += lh
	}
	if len(d.Buttons) > 0 {
		y += lh / 2
		bx := x
		for i, b := range d.Buttons {
			bwid := font.Width(" "+b+" ", s) + 2*s
			r := image.Rect(bx, y-2*s, bx+bwid, y-2*s+lh)
			bg, fg := h.R.UITrack, h.R.UIText
			if i == d.Sel {
				bg, fg = h.R.UIAccent, h.R.UIAccentText
			}
			Rect(dst, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), bg)
			h.text(dst, " "+b, bx+s, y, fg)
			h.Hits = append(h.Hits, r)
			bx += bwid + 2*cw
		}
		y += lh + lh/2
	}
	if d.Hint != "" {
		y += lh / 2
		h.text(dst, d.Hint, x, y, h.R.UIDim)
	}
}

func (h *HUD) bankrupt(dst *ebiten.Image, w, ht int, st HUDState) {
	s, cw, lh := h.S, h.cw(), h.lh()
	Rect(dst, 0, 0, w, ht, color.RGBA{0, 0, 0, 0xa0})
	c := st.City
	lines := []string{
		c.Name + " went bankrupt on " + sim.Date(c.Day) + ".",
		"",
		fmt.Sprintf("population  %s", commas(int64(c.Stats.Residents))),
		fmt.Sprintf("jobs        %s", commas(int64(c.Stats.CommJobs+c.Stats.IndJobs))),
		fmt.Sprintf("funds       -$%s", commas(int64(-c.Funds))),
		"",
		":new for a fresh map, :load to restore a save, :q to quit",
	}
	bw := 64 * cw
	bh := len(lines)*lh + 10*s
	x0, y0 := (w-bw)/2, (ht-bh)/2
	h.box(dst, x0, y0, bw, bh, "bankrupt")
	for i, l := range lines {
		col := h.R.UIText
		if i == 0 {
			col = h.R.UIErr
		}
		h.text(dst, l, x0+2*cw, y0+7*s+i*lh, col)
	}
}

func commas(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}
