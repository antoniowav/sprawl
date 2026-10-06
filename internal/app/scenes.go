package app

import (
	"image"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/antoniowav/sprawl/input"
	"github.com/antoniowav/sprawl/internal/meta"
	"github.com/antoniowav/sprawl/render"
	"github.com/antoniowav/sprawl/sim"
)

type scene int

const (
	sceneGame scene = iota
	sceneTitle
	sceneNewCity
)

// --- title screen ---

func (a *App) titleEntries() []render.MenuEntry {
	cont := render.MenuEntry{Label: "Continue", Disabled: true, Detail: "no saves yet"}
	if saves := listSaveInfo(); len(saves) > 0 {
		cont = render.MenuEntry{Label: "Continue", Detail: saves[0].name}
	}
	return []render.MenuEntry{
		cont,
		{Label: "New city", Detail: "Ctrl+N"},
		{Label: "Open…", Detail: "Ctrl+O"},
		{Label: "Settings"},
		{Label: "Quit", Detail: "Ctrl+Q"},
	}
}

// showTitle switches to the title screen with a fresh backdrop map.
func (a *App) showTitle() {
	a.scene = sceneTitle
	a.saveName, a.unsaved = "", false
	a.clearTool()
	a.mode, a.dlg = modeNormal, nil
	a.newTimelapse()
	a.titleSel = 0
	if a.titleEntries()[0].Disabled {
		a.titleSel = 1
	}
	a.dirty = true
}

func (a *App) updateTitle(dt float64) {
	a.updateTimelapse(dt)

	entries := a.titleEntries()
	pick := -1
	for _, p := range a.poll.Poll() {
		switch p.Action {
		case input.MoveDown:
			a.titleSel = nextEnabled(entries, a.titleSel, 1)
		case input.MoveUp:
			a.titleSel = nextEnabled(entries, a.titleSel, -1)
		case input.Apply:
			pick = a.titleSel
		case input.NewCity:
			pick = 1
		case input.Open:
			pick = 2
		case input.Quit:
			pick = 4
		}
	}
	if i, ok := a.menuMouse(); ok && !entries[i].Disabled {
		a.titleSel = i
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			pick = i
		}
	}
	if pick < 0 || entries[pick].Disabled {
		return
	}
	a.snd.Play(soundClick)
	switch pick {
	case 0:
		if saves := listSaveInfo(); len(saves) > 0 {
			a.load(saves[0].name)
		}
	case 1:
		a.startNewCityForm()
	case 2:
		a.openSaves()
	case 3:
		a.openSettings()
	case 4:
		a.quit = true
	}
}

// menuMouse returns the hit box under the mouse, if any.
func (a *App) menuMouse() (int, bool) {
	x, y := ebiten.CursorPosition()
	for i, r := range a.hud.Hits {
		if image.Pt(x, y).In(r) {
			return i, true
		}
	}
	return 0, false
}

func nextEnabled(es []render.MenuEntry, i, d int) int {
	for range es {
		i = (i + d + len(es)) % len(es)
		if !es[i].Disabled {
			return i
		}
	}
	return i
}

// --- new city form ---

var mapSizes = []int{96, 128, 192, 256}

// newCityState backs the new-city form.
type newCityState struct {
	mode    int // index into gameModes
	name    string
	mapType int // index into sim.MapTypes
	size    int
	seed    string
	focus   int
	preview *ebiten.Image
}

const (
	fMode = iota
	fName
	fMap
	fSize
	fSeed
	fieldCount
	fStart = fieldCount
	fBack  = fieldCount + 1
)

func (a *App) startNewCityForm() {
	seed := rand.Int64N(1_000_000)
	a.nc = &newCityState{size: a.cfg.MapSize, seed: strconv.FormatInt(seed, 10), focus: fName}
	a.nc.name = sim.New(32, 32, seed).Name
	a.scene, a.mode, a.dlg = sceneNewCity, modeNormal, nil
	a.refreshPreview()
}

func (a *App) formSeed() int64 {
	n, err := strconv.ParseInt(a.nc.seed, 10, 64)
	if err != nil {
		return 1
	}
	return n
}

// formCity generates the city the form describes (scenarios fix the map).
func (a *App) formCity() *sim.City {
	nc := a.nc
	if sc := gameModes[nc.mode].scenario; sc != nil {
		return sc.NewCity()
	}
	return sim.NewMap(nc.size, nc.size, a.formSeed(), sim.MapTypes[nc.mapType])
}

// refreshPreview regenerates the terrain thumbnail (one pixel per tile).
func (a *App) refreshPreview() {
	c := a.formCity()
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	render.MinimapPixels(c, a.roles, img.Pix)
	if a.nc.preview != nil {
		a.nc.preview.Deallocate()
	}
	a.nc.preview = ebiten.NewImageFromImage(img)
	a.dirty = true
}

// formView builds what the HUD draws.
func (a *App) formView() *render.NewCityForm {
	nc := a.nc
	gm := gameModes[nc.mode]
	fixed := gm.scenario != nil
	f := &render.NewCityForm{Focus: nc.focus, Preview: nc.preview}
	f.Fields = []render.FormField{
		{Label: "mode", Value: gm.name},
		{Label: "name", Value: nc.name, Text: true},
		{Label: "map", Value: sim.MapTypes[nc.mapType].String(), Disabled: fixed},
		{Label: "size", Value: render.SizeLabel(nc.size), Disabled: fixed},
		{Label: "seed", Value: nc.seed, Text: !fixed, Disabled: fixed},
	}
	if fixed {
		sc := gm.scenario
		f.Fields[fMap].Value = sc.Map.String()
		f.Fields[fSize].Value = render.SizeLabel(sc.Size)
		f.Fields[fSeed].Value = strconv.FormatInt(sc.Seed, 10)
	}
	if fixed {
		f.Note = wrapText(gm.scenario.Brief, 32)
	} else {
		f.Note = []string{"←→ on seed rolls a new map"}
	}
	return f
}

func (a *App) updateNewCity() {
	nc := a.nc
	fixed := gameModes[nc.mode].scenario != nil
	chars := a.poll.Chars()
	if len(chars) > 0 {
		a.dirty = true
	}
	left, right := input.Repeated(ebiten.KeyArrowLeft), input.Repeated(ebiten.KeyArrowRight)
	d := 0
	if left {
		d = -1
	} else if right {
		d = 1
	}
	switch nc.focus {
	case fMode:
		if d != 0 {
			nc.mode = (nc.mode + d + len(gameModes)) % len(gameModes)
			a.refreshPreview()
		}
	case fName:
		for _, r := range chars {
			if len(nc.name) < 24 && (r == ' ' || r == '-' || r == '\'' || ('a' <= r|32 && r|32 <= 'z') || ('0' <= r && r <= '9')) {
				nc.name += string(r)
			}
		}
		if input.Repeated(ebiten.KeyBackspace) && nc.name != "" {
			nc.name, a.dirty = nc.name[:len(nc.name)-1], true
		}
	case fMap:
		if d != 0 && !fixed {
			nc.mapType = (nc.mapType + d + len(sim.MapTypes)) % len(sim.MapTypes)
			a.refreshPreview()
		}
	case fSize:
		if d != 0 && !fixed {
			nc.size = cycle(nc.size, d, mapSizes)
			a.refreshPreview()
		}
	case fSeed:
		if fixed {
			break
		}
		changed := false
		for _, r := range chars {
			if r >= '0' && r <= '9' && len(nc.seed) < 9 {
				nc.seed += string(r)
				changed = true
			}
		}
		if input.Repeated(ebiten.KeyBackspace) && nc.seed != "" {
			nc.seed, changed = nc.seed[:len(nc.seed)-1], true
		}
		if d != 0 {
			nc.seed, changed = strconv.FormatInt(rand.Int64N(1_000_000), 10), true
		}
		if changed && nc.seed != "" {
			a.refreshPreview()
		}
	case fStart, fBack:
		if d != 0 {
			nc.focus, a.dirty = fStart+fBack-nc.focus, true
		}
	}
	n := fieldCount + 2
	step := func(dir int) {
		for {
			nc.focus = (nc.focus + dir + n) % n
			if !fixed || (nc.focus != fMap && nc.focus != fSize && nc.focus != fSeed) {
				break
			}
		}
		a.dirty = true
	}
	switch {
	case input.Repeated(ebiten.KeyArrowDown) || input.Repeated(ebiten.KeyTab):
		step(1)
	case input.Repeated(ebiten.KeyArrowUp):
		step(-1)
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.scene, a.dirty = sceneTitle, true
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		if nc.focus == fBack {
			a.scene, a.dirty = sceneTitle, true
		} else {
			a.startCity()
		}
	}
	if i, ok := a.menuMouse(); ok && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		switch i {
		case fStart:
			a.startCity()
		case fBack:
			a.scene, a.dirty = sceneTitle, true
		default:
			nc.focus, a.dirty = i, true
		}
	}
}

func (a *App) startCity() {
	nc := a.nc
	c := a.formCity()
	a.installCity(c, a.formSeed())
	if name := strings.TrimSpace(nc.name); name != "" {
		c.Name = name
	}
	c.Log = nil
	c.Logf(sim.Info, "%s founded · %s · %dx%d", c.Name, c.Map, c.W, c.H)
	if sc := gameModes[nc.mode].scenario; sc != nil {
		sc.Begin(c)
	}
	a.scene = sceneGame
	a.paused = false
}

// wrapText breaks s into lines of at most n characters.
func wrapText(s string, n int) []string {
	var out []string
	line := ""
	for _, w := range strings.Fields(s) {
		if line != "" && len(line)+1+len(w) > n {
			out = append(out, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += w
	}
	return append(out, line)
}

// --- pause menu ---

var pauseItems = []string{"Resume", "Save", "Save as…", "Open…", "New city…", "Budget", "Statistics", "Settings", "Quit to title", "Quit"}

func (a *App) pauseEntries() []render.MenuEntry {
	keys := []string{"Esc", "Ctrl+S", "Ctrl+Shift+S", "Ctrl+O", "Ctrl+N", "Ctrl+B", "Ctrl+G", "", "", "Ctrl+Q"}
	es := make([]render.MenuEntry, len(pauseItems))
	for i, l := range pauseItems {
		es[i] = render.MenuEntry{Label: l, Detail: keys[i]}
	}
	return es
}

func (a *App) updatePause() {
	pick := -1
	for _, p := range a.poll.Poll() {
		switch p.Action {
		case input.MoveDown:
			a.pauseSel = (a.pauseSel + 1) % len(pauseItems)
		case input.MoveUp:
			a.pauseSel = (a.pauseSel + len(pauseItems) - 1) % len(pauseItems)
		case input.Apply:
			pick = a.pauseSel
		case input.Cancel:
			pick = 0
		default:
			if p.Action == input.Save || p.Action == input.Open || p.Action == input.Quit ||
				p.Action == input.SaveAs || p.Action == input.NewCity {
				a.mode = modeNormal
				a.do(p)
				return
			}
		}
		a.dirty = true
	}
	if i, ok := a.menuMouse(); ok {
		if i != a.pauseSel {
			a.pauseSel, a.dirty = i, true
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			pick = i
		}
	}
	if pick < 0 {
		return
	}
	a.snd.Play(soundClick)
	a.mode, a.dirty = modeNormal, true
	switch pick {
	case 1:
		a.quickSave(nil)
	case 2:
		a.saveAs(nil)
	case 3:
		a.openSaves()
	case 4:
		a.requestNew()
	case 5:
		a.mode = modeBudget
	case 6:
		a.mode = modeStats
	case 7:
		a.openSettings()
	case 8:
		a.guard("Quit to the title screen?", a.showTitle)
	case 9:
		a.requestQuit()
	}
}

func versionLine() string { return meta.Name + " " + meta.Version }
