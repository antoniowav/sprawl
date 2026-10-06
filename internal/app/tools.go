package app

import (
	"fmt"
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/antoniowav/sprawl/input"
	"github.com/antoniowav/sprawl/render"
	"github.com/antoniowav/sprawl/render/sprites"
	"github.com/antoniowav/sprawl/sim"
	"github.com/antoniowav/sprawl/sound"
)

func (a *App) do(p input.Press) {
	a.dirty = true
	act := p.Action
	// File shortcuts work from anywhere except text entry.
	switch act {
	case input.Save:
		a.mode = modeNormal
		a.quickSave(nil)
		return
	case input.SaveAs:
		a.mode = modeNormal
		a.saveAs(nil)
		return
	case input.Open:
		a.mode = modeNormal
		a.openSaves()
		return
	case input.NewCity:
		a.mode = modeNormal
		a.guard("Start a new city?", a.startNewCityForm)
		return
	case input.Quit:
		a.requestQuit()
		return
	case input.Undo:
		a.undo()
		return
	case input.Redo:
		a.redo()
		return
	}
	if a.mode == modeHelp || a.mode == modeBudget || a.mode == modeStats {
		if act == input.Help || act == input.Cancel || act == input.Apply {
			a.mode = modeNormal
		}
		return
	}
	if a.city.Bankrupt && act != input.Command && act != input.Help {
		a.flash(sim.Err, "bankrupt: :new for a new city or :load a save")
		return
	}
	if a.mode == modeMenu {
		a.doMenu(p)
		return
	}
	if a.zonePending {
		a.zonePending = false
		switch p.Rune {
		case 'r':
			a.setTool(sim.ToolZoneR)
		case 'c':
			a.setTool(sim.ToolZoneC)
		case 'i':
			a.setTool(sim.ToolZoneI)
		default:
			a.flash(sim.Info, "zone: press r, c or i after z")
		}
		return
	}
	switch act {
	case input.MoveLeft:
		a.move(-1, 0)
	case input.MoveRight:
		a.move(1, 0)
	case input.MoveUp:
		a.move(0, -1)
	case input.MoveDown:
		a.move(0, 1)
	case input.MoveLeftFast:
		a.move(-8, 0)
	case input.MoveRightFast:
		a.move(8, 0)
	case input.MoveUpFast:
		a.move(0, -8)
	case input.MoveDownFast:
		a.move(0, 8)
	case input.Center:
		a.cam.TX, a.cam.TY = a.cursorWorld()
	case input.ZoomIn:
		a.zoom(1)
	case input.ZoomOut:
		a.zoom(-1)
	case input.Pause:
		a.paused = !a.paused
	case input.Speed1, input.Speed2, input.Speed3:
		a.speed = int(act[len(act)-1] - '0')
		a.paused = false
	case input.TogglePanel:
		a.showPanel = !a.showPanel
	case input.Budget:
		a.mode = modeBudget
	case input.Stats:
		a.mode = modeStats
	case input.TimeOfDay:
		a.cycleTimeOfDay(1)
		a.flash(sim.Info, "day and night: %s", timeLabel(a.cfg.TimeOfDay))
	case input.Minimap:
		a.showMinimap = !a.showMinimap
	case input.Guide:
		a.showGuide = !a.showGuide
	case input.ToggleLog:
		a.showLog = !a.showLog
	case input.Help:
		a.mode = modeHelp
	case input.Command:
		a.mode, a.prompt = modeCommand, ""

	case input.Road:
		a.setTool(sim.ToolRoad)
	case input.PowerLine:
		a.setTool(sim.ToolLine)
	case input.WaterPipe:
		a.setTool(sim.ToolPipe)
	case input.Bulldoze:
		a.setTool(sim.ToolBulldoze)
	case input.ZonePrefix:
		a.zonePending = true
	case input.Apply:
		a.apply()
	case input.BuildMenu:
		a.mode = modeMenu
	case input.Paint:
		if !a.needTool() || a.isBuilding() {
			return
		}
		a.paint, a.visual, a.strokeOpen = !a.paint, false, false
		if a.paint {
			a.applyTiles([]sim.Pt{{X: a.cx, Y: a.cy}}, true)
		}
	case input.Visual:
		if !a.needTool() || a.isBuilding() {
			return
		}
		a.visual, a.paint = !a.visual, false
		a.anchor = sim.Pt{X: a.cx, Y: a.cy}
	case input.Overlay:
		if a.visual {
			a.vertFirst = !a.vertFirst // vim's o: swap the corner
			return
		}
		a.overlay = a.overlay.Next()
		a.flash(sim.Info, "overlay: %s", a.overlay)
	case input.Underground:
		a.underground, a.autoUnderground = !a.underground, false
	case input.Cancel:
		switch {
		case a.visual:
			a.visual = false
		case a.paint:
			a.paint = false
		case a.hasTool:
			a.clearTool()
		default:
			a.msg = ""
			a.mode, a.pauseSel = modePause, 0
		}
	case "":
		// unbound key
	default:
		a.flash(sim.Info, "%s: not built yet (see PLAN.md)", strings.ReplaceAll(string(act), "_", " "))
	}
}

func (a *App) doMenu(p input.Press) {
	n := len(sim.Buildings)
	switch {
	case p.Rune >= '1' && p.Rune <= '9' && int(p.Rune-'1') < n:
		a.pick(int(p.Rune - '1'))
	case p.Rune == '0' && n >= 10:
		a.pick(9)
	case p.Action == input.MoveDown:
		a.menuSel = (a.menuSel + 1) % n
	case p.Action == input.MoveUp:
		a.menuSel = (a.menuSel + n - 1) % n
	case p.Action == input.Apply:
		a.pick(a.menuSel)
	case p.Action == input.Cancel, p.Action == input.BuildMenu:
		a.mode = modeNormal
	}
}

func (a *App) pick(i int) {
	a.menuSel = i
	if b := sim.Buildings[i]; !a.city.Unlocked(b) {
		a.flash(sim.Info, "%s unlocks at %s people", b.Tool, commas(float64(b.Unlock)))
		return
	}
	a.mode = modeNormal
	a.setTool(sim.Buildings[i].Tool)
	a.flash(sim.Info, "%s: Enter places it with its top-left corner at the cursor", sim.Buildings[i].Tool)
}

func (a *App) isBuilding() bool {
	_, ok := a.tool.Building()
	if ok {
		a.flash(sim.Info, "buildings are placed one at a time with Enter")
	}
	return ok
}

const maxUndo = 100

func (a *App) touchEdit(e *sim.Edit) {
	a.miniDirty = true
	for _, ch := range e.Changes {
		a.chunks.Touch(ch.I%a.city.W, ch.I/a.city.W)
	}
}

func (a *App) undo() {
	if len(a.undos) == 0 {
		a.flash(sim.Info, "nothing to undo")
		return
	}
	e := a.undos[len(a.undos)-1]
	a.undos = a.undos[:len(a.undos)-1]
	a.city.Undo(e)
	a.touchEdit(e)
	a.redos = append(a.redos, e)
	a.strokeOpen, a.unsaved, a.dirty = false, true, true
	a.flash(sim.Info, "undid %s (+$%s)", e.Tool, commas(e.Cost))
}

func (a *App) redo() {
	if len(a.redos) == 0 {
		a.flash(sim.Info, "nothing to redo")
		return
	}
	e := a.redos[len(a.redos)-1]
	if err := a.city.Redo(e); err != nil {
		a.flash(sim.Warn, "%v", err)
		a.redos = a.redos[:0]
		return
	}
	a.touchEdit(e)
	a.redos = a.redos[:len(a.redos)-1]
	a.undos = append(a.undos, e)
	a.unsaved, a.dirty = true, true
	a.flash(sim.Info, "redid %s", e.Tool)
}

// Toolbar entries, top to bottom.
var toolbarTools = []struct {
	icon int
	tool sim.Tool
	act  input.Action
	tip  string
}{
	{sprites.IconInspect, 0, input.Cancel, "Inspect"},
	{sprites.IconRoad, sim.ToolRoad, input.Road, "Road"},
	{sprites.IconLine, sim.ToolLine, input.PowerLine, "Power line"},
	{sprites.IconPipe, sim.ToolPipe, input.WaterPipe, "Water pipe"},
	{sprites.IconBulldoze, sim.ToolBulldoze, input.Bulldoze, "Bulldoze"},
	{sprites.IconZoneR, sim.ToolZoneR, "", "Residential zone"},
	{sprites.IconZoneC, sim.ToolZoneC, "", "Commercial zone"},
	{sprites.IconZoneI, sim.ToolZoneI, "", "Industrial zone"},
	{sprites.IconBuild, 0, input.BuildMenu, "Buildings"},
	{sprites.IconOverlay, 0, input.Overlay, "Overlay"},
	{sprites.IconPause, 0, input.Pause, "Pause"},
}

func (a *App) toolbarButtons() []render.ToolButton {
	out := make([]render.ToolButton, len(toolbarTools))
	for i, t := range toolbarTools {
		b := render.ToolButton{Icon: a.atlas.Tools[t.icon], Tip: t.tip}
		keys := ""
		switch {
		case t.act != "" && len(a.km.KeysFor(t.act)) > 0:
			keys = a.km.KeysFor(t.act)[0]
		case t.tool >= sim.ToolZoneR && t.tool <= sim.ToolZoneI:
			keys = "z " + string("rci"[t.tool-sim.ToolZoneR])
		}
		if keys != "" {
			b.Tip += " · " + keys
		}
		switch {
		case i == 0:
			b.Active = !a.hasTool
		case t.act == input.Overlay:
			b.Active = a.overlay != render.OverlayNone
		case t.act == input.Pause:
			if a.paused {
				b.Icon, b.Tip, b.Active = a.atlas.Tools[sprites.IconPlay], "Resume · Space", true
			}
		case t.act == input.BuildMenu:
			_, b.Active = a.tool.Building()
			b.Active = b.Active && a.hasTool
		default:
			b.Active = a.hasTool && a.tool == t.tool
		}
		out[i] = b
	}
	return out
}

func toolSound(t sim.Tool) sound.Effect {
	switch {
	case t == sim.ToolBulldoze:
		return sound.Bulldoze
	case t >= sim.ToolZoneR && t <= sim.ToolZoneI:
		return sound.Zone
	}
	if _, ok := t.Building(); ok {
		return sound.Building
	}
	return sound.Place
}

// clickToolbar handles a click on toolbar button i.
func (a *App) clickToolbar(i int) {
	a.snd.Play(sound.Click)
	t := toolbarTools[i]
	switch {
	case i == 0:
		a.clearTool()
		a.dirty = true
	case t.tool >= sim.ToolZoneR && t.tool <= sim.ToolZoneI:
		a.setTool(t.tool)
		a.dirty = true
	default:
		a.do(input.Press{Action: t.act})
	}
}

func (a *App) needTool() bool {
	if !a.hasTool {
		a.flash(sim.Info, "pick a tool first: r p w d or z")
	}
	return a.hasTool
}

func (a *App) setTool(t sim.Tool) {
	a.tool, a.hasTool = t, true
	a.paint = false
	switch {
	case t == sim.ToolPipe && !a.underground:
		a.underground, a.autoUnderground = true, true
	case t != sim.ToolPipe && t != sim.ToolBulldoze && a.autoUnderground:
		a.underground, a.autoUnderground = false, false
	}
}

func (a *App) clearTool() {
	a.hasTool, a.paint, a.visual = false, false, false
	if a.autoUnderground {
		a.underground, a.autoUnderground = false, false
	}
}

// pending returns the plan the next apply would run, and the selected
// tiles, when a tool is active.
func (a *App) pending() (sim.Plan, []sim.Pt, bool) {
	if !a.hasTool || a.mode != modeNormal {
		return sim.Plan{}, nil, false
	}
	cur := sim.Pt{X: a.cx, Y: a.cy}
	sel := []sim.Pt{cur}
	switch {
	case a.visual:
		sel = sim.Selection(a.tool, a.anchor, cur, a.vertFirst)
	case a.drag:
		if _, b := a.tool.Building(); !b {
			sel = sim.Selection(a.tool, a.anchor, cur, false)
		}
	}
	return a.city.PlanTool(a.tool, sel, a.pipesOnly()), sel, true
}

func (a *App) pipesOnly() bool { return a.underground && a.tool == sim.ToolBulldoze }

func (a *App) apply() {
	if !a.needTool() {
		return
	}
	_, sel, _ := a.pending()
	a.applyTiles(sel, false)
	a.visual = false
}

// applyTiles runs the tool. In paint mode, tiles the tool can't change are
// skipped silently so painting across existing roads doesn't nag.
func (a *App) applyTiles(sel []sim.Pt, quiet bool) {
	p, edit := a.city.ApplyEdit(a.tool, sel, a.pipesOnly())
	if edit != nil {
		a.touchEdit(edit)
		a.unsaved = true
		// Paint mode makes one edit per tile; merge a paint stroke into one undo.
		if quiet && a.paint && len(a.undos) > 0 && a.strokeOpen {
			last := a.undos[len(a.undos)-1]
			last.Changes = append(last.Changes, edit.Changes...)
			last.Cost += edit.Cost
		} else {
			a.undos = append(a.undos, edit)
			if len(a.undos) > maxUndo {
				a.undos = a.undos[1:]
			}
		}
		a.strokeOpen = a.paint
		a.redos = a.redos[:0]
	}
	switch {
	case p.Err != "" && quiet && len(p.Tiles) == 0 && p.Skipped > 0:
	case p.Err != "":
		a.flash(sim.Warn, "%s", p.Err)
		a.snd.Play(sound.Error)
	default:
		a.snd.Play(toolSound(a.tool))
		a.flash(sim.Info, "%s ×%d  -$%s", a.tool, len(p.Tiles), commas(p.Cost))
	}
	a.dirty = true
}

// updateLeftButton: click moves the cursor; with a tool, click applies to
// one tile and drag selects like visual mode, applied on release.
func (a *App) updateLeftButton(mx, my int) {
	if a.mode != modeNormal {
		return
	}
	pt := image.Pt(mx, my)
	hover := -1
	for i, r := range a.hud.ToolHits {
		if pt.In(r) {
			hover = i
		}
	}
	if hover != a.toolHover {
		a.toolHover, a.dirty = hover, true
	}
	overUI := false
	for _, r := range a.hud.Blocks {
		if pt.In(r) {
			overUI = true
		}
	}
	tx, ty := a.cam.TileAt(mx, my, a.w, a.h)
	tx, ty = max(0, min(a.city.W-1, tx)), max(0, min(a.city.H-1, ty))
	// The cursor follows the mouse while it moves over the map.
	if (mx != a.lastMX || my != a.lastMY) && !overUI && !a.visual && a.city.In(a.cam.TileAt(mx, my, a.w, a.h)) {
		if tx != a.cx || ty != a.cy {
			a.cx, a.cy, a.dirty = tx, ty, true
			if a.paint && a.hasTool && !a.drag {
				a.applyTiles([]sim.Pt{{X: tx, Y: ty}}, true)
			}
		}
	}
	a.lastMX, a.lastMY = mx, my
	switch {
	case inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft):
		if hover >= 0 {
			a.clickToolbar(hover)
			return
		}
		if overUI || !a.city.In(a.cam.TileAt(mx, my, a.w, a.h)) {
			return
		}
		a.cx, a.cy, a.dirty = tx, ty, true
		if a.hasTool {
			a.drag, a.visual, a.paint = true, false, false
			a.anchor = sim.Pt{X: tx, Y: ty}
		}
	case a.drag && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft):
		if tx != a.cx || ty != a.cy {
			a.cx, a.cy, a.dirty = tx, ty, true
		}
	case a.drag:
		_, sel, _ := a.pending()
		a.drag = false
		a.applyTiles(sel, false)
	}
}

func commas(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}
