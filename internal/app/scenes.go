package app

import (
	"fmt"
	"image"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

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
	a.city = sim.New(a.cfg.MapSize, a.cfg.MapSize, time.Now().UnixNano()%1_000_000)
	a.saveName, a.unsaved = "", false
	a.chunks.Reset()
	a.clearTool()
	a.mode, a.dlg = modeNormal, nil
	a.cam.Jump(float64(a.city.W*render.TileSize)/2, float64(a.city.H*render.TileSize)/2)
	a.titleSel = 0
	if a.titleEntries()[0].Disabled {
		a.titleSel = 1
	}
	a.dirty = true
}

func (a *App) updateTitle(dt float64) {
	// Slow drift over the backdrop map.
	a.cam.X += dt * 6
	a.cam.Y += dt * 2
	maxX := float64(a.city.W * render.TileSize)
	if a.cam.X > maxX*0.8 {
		a.cam.X = maxX * 0.2
	}
	a.cam.TX, a.cam.TY = a.cam.X, a.cam.Y
	a.dirty = true

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

var mapSizes = []int{96, 128, 192}

func (a *App) startNewCityForm() {
	seed := rand.Int64N(1_000_000)
	a.form = &render.NewCityForm{Size: a.cfg.MapSize, Seed: strconv.FormatInt(seed, 10)}
	a.form.Name = sim.New(32, 32, seed).Name
	a.scene, a.mode, a.dlg = sceneNewCity, modeNormal, nil
	a.refreshPreview()
}

func (a *App) formSeed() int64 {
	n, err := strconv.ParseInt(a.form.Seed, 10, 64)
	if err != nil {
		return 1
	}
	return n
}

// refreshPreview regenerates the terrain thumbnail (one pixel per tile).
func (a *App) refreshPreview() {
	c := sim.New(a.form.Size, a.form.Size, a.formSeed())
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			col := a.roles.Grass
			switch c.At(x, y).Terrain {
			case sim.Water:
				col = a.roles.Water
			case sim.Trees:
				col = a.roles.Tree
			}
			img.SetRGBA(x, y, col)
		}
	}
	if a.form.Preview != nil {
		a.form.Preview.Deallocate()
	}
	a.form.Preview = ebiten.NewImageFromImage(img)
	a.dirty = true
}

func (a *App) updateNewCity() {
	f := a.form
	chars := a.poll.Chars()
	if len(chars) > 0 {
		a.dirty = true
	}
	switch f.Field {
	case 0:
		for _, r := range chars {
			if len(f.Name) < 24 && (r == ' ' || r == '-' || r == '\'' || ('a' <= r|32 && r|32 <= 'z') || ('0' <= r && r <= '9')) {
				f.Name += string(r)
			}
		}
		if input.Repeated(ebiten.KeyBackspace) && f.Name != "" {
			f.Name, a.dirty = f.Name[:len(f.Name)-1], true
		}
	case 2:
		changed := false
		for _, r := range chars {
			if r >= '0' && r <= '9' && len(f.Seed) < 9 {
				f.Seed += string(r)
				changed = true
			}
		}
		if input.Repeated(ebiten.KeyBackspace) && f.Seed != "" {
			f.Seed, changed = f.Seed[:len(f.Seed)-1], true
		}
		if input.Repeated(ebiten.KeyArrowLeft) || input.Repeated(ebiten.KeyArrowRight) {
			f.Seed, changed = strconv.FormatInt(rand.Int64N(1_000_000), 10), true
		}
		if changed && f.Seed != "" {
			a.refreshPreview()
		}
	case 1:
		i := 0
		for j, n := range mapSizes {
			if n == f.Size {
				i = j
			}
		}
		switch {
		case input.Repeated(ebiten.KeyArrowLeft):
			f.Size = mapSizes[(i+len(mapSizes)-1)%len(mapSizes)]
			a.refreshPreview()
		case input.Repeated(ebiten.KeyArrowRight):
			f.Size = mapSizes[(i+1)%len(mapSizes)]
			a.refreshPreview()
		}
	case 3, 4:
		if input.Repeated(ebiten.KeyArrowLeft) || input.Repeated(ebiten.KeyArrowRight) {
			f.Field, a.dirty = 7-f.Field, true
		}
	}
	switch {
	case input.Repeated(ebiten.KeyArrowDown) || input.Repeated(ebiten.KeyTab):
		f.Field, a.dirty = (f.Field+1)%5, true
	case input.Repeated(ebiten.KeyArrowUp):
		f.Field, a.dirty = (f.Field+4)%5, true
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.scene, a.dirty = sceneTitle, true
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		if f.Field == 4 {
			a.scene, a.dirty = sceneTitle, true
		} else {
			a.startCity()
		}
	}
	if i, ok := a.menuMouse(); ok && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		switch i {
		case 3:
			a.startCity()
		case 4:
			a.scene, a.dirty = sceneTitle, true
		default:
			f.Field, a.dirty = i, true
		}
	}
}

func (a *App) startCity() {
	f := a.form
	a.cfg.MapSize = f.Size
	a.newCity(a.formSeed())
	if name := strings.TrimSpace(f.Name); name != "" {
		a.city.Name = name
	}
	a.city.Log[len(a.city.Log)-1].Msg = fmt.Sprintf("%s founded · seed %d · %dx%d", a.city.Name, a.seed, f.Size, f.Size)
	a.scene = sceneGame
	a.paused = false
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
