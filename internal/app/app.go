// Package app wires sim, render, input and theme into an ebiten.Game.
package app

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/antoniowav/sprawl/config"
	"github.com/antoniowav/sprawl/input"
	"github.com/antoniowav/sprawl/render"
	"github.com/antoniowav/sprawl/render/font"
	"github.com/antoniowav/sprawl/sim"
	"github.com/antoniowav/sprawl/sound"
	"github.com/antoniowav/sprawl/theme"
)

const (
	soundClick     = sound.Click
	soundMilestone = sound.Milestone
)

type mode int

const (
	modeNormal mode = iota
	modeCommand
	modeHelp
	modeMenu
	modeBudget
	modeDialog
	modePause
	modeStats
	modeSettings
)

// Options are the command-line settings.
type Options struct {
	Config     config.Config
	ConfigPath string
	ConfigErr  error
	Seed       int64
	Screenshot string   // write a PNG after the first frames and quit
	Actions    []string // actions to run at startup (for scripted screenshots)
	Ticks      int      // sim ticks to run after the actions (for screenshots)
	SkipTitle  bool
}

// App is the running game.
type App struct {
	cfg   config.Config
	city  *sim.City
	km    *input.Keymap
	poll  *input.Poller
	watch *theme.Watcher

	pal   theme.Palette
	roles theme.Roles
	atlas *render.Atlas
	hud   *render.HUD
	cam   render.Camera

	cx, cy int // cursor tile
	mode   mode
	prompt string
	paused bool
	speed  int

	showPanel, showLog bool

	tool            sim.Tool
	hasTool         bool
	zonePending     bool
	visual          bool // keyboard visual selection
	drag            bool // mouse drag selection
	anchor          sim.Pt
	vertFirst       bool
	paint           bool
	underground     bool
	autoUnderground bool // turned on by the pipe tool, off when it's put away
	overlay         render.Overlay
	menuSel         int
	dlg             *dialog
	showMinimap     bool
	mini            *ebiten.Image
	miniPix         []byte
	miniDirty       bool
	miniNext        time.Time
	miniDrag        bool
	chunks          *render.Chunks
	settingSel      int
	configPath      string
	snd             *sound.Player
	lastPeak        int
	lastResult      int
	showGuide       bool
	sawBudget       bool
	undos, redos    []*sim.Edit
	strokeOpen      bool // a paint stroke is being merged into one undo
	toolHover       int
	lastMX, lastMY  int
	scene           scene
	titleSel        int
	pauseSel        int
	nc              *newCityState
	saveName        string // where Ctrl+S saves; empty until first save
	unsaved         bool
	history         []string
	histIdx         int
	seed            int64

	simAcc      float64
	lastUpdate  time.Time
	dayClock    float64 // seconds into the day/night cycle (sim time)
	carClock    float64 // seconds of running sim time, for cars
	carsVisible bool
	particles   []render.Particle
	emitters    []image.Point
	smokeAcc    float64
	smokeRng    *rand.Rand
	grows       map[sim.Pt]time.Time // grow animations in progress

	msg      string
	msgLevel sim.EventLevel
	msgUntil time.Time

	animFrame   int
	animVisible bool
	nextAnim    time.Time

	w, h       int
	dirty      bool
	focused    bool
	wheel      float64
	panning    bool
	panX, panY int
	quit       bool
	shotPath   string
	shotFrames int
	pace       *pacer
}

// New builds the game state.
func New(o Options) *App {
	a := &App{
		cfg:         o.Config,
		city:        sim.New(o.Config.MapSize, o.Config.MapSize, o.Seed),
		speed:       1,
		showPanel:   true,
		showLog:     true,
		dirty:       true,
		shotPath:    o.Screenshot,
		grows:       map[sim.Pt]time.Time{},
		seed:        o.Seed,
		toolHover:   -1,
		showMinimap: true,
		chunks:      render.NewChunks(),
		configPath:  o.ConfigPath,
		showGuide:   guideFirstRun(),
		dayClock:    dayLength * 0.3, // start in the morning
		smokeRng:    rand.New(rand.NewPCG(1, 2)),
	}
	a.city.Logf(sim.Info, "%s founded · seed %d · %dx%d", a.city.Name, o.Seed, a.city.W, a.city.H)
	if o.ConfigErr != nil {
		a.city.Logf(sim.Err, "config: %v (using defaults)", o.ConfigErr)
	}
	var errs []error
	a.km, errs = input.NewKeymap(o.Config.Keys)
	for _, e := range errs {
		a.city.Logf(sim.Warn, "%v", e)
	}
	a.poll = input.NewPoller(a.km)
	vol := float64(o.Config.Volume) / 100
	if o.Screenshot != "" {
		vol = 0
	}
	a.snd = sound.New(vol)
	a.hud = &render.HUD{Face: font.New(), S: o.Config.UIScale}
	a.loadTheme(false)
	if o.Config.WatchTheme {
		a.watch = theme.NewWatcher()
	}
	if o.Config.LowPower && o.Screenshot == "" {
		a.pace = startPacer()
	}

	a.cx, a.cy = a.city.W/2, a.city.H/2
	a.cam.Zoom = o.Config.Zoom
	a.cam.Jump(float64(a.cx*render.TileSize+render.TileSize/2), float64(a.cy*render.TileSize+render.TileSize/2))
	// Scripted input: action names, or single characters looked up in the
	// keymap as if typed.
	for _, tok := range o.Actions {
		switch tok {
		case "@title":
			a.showTitle()
			continue
		case "@newcity":
			a.startNewCityForm()
			continue
		case "@pause":
			a.mode = modePause
			continue
		case "@settings":
			a.openSettings()
			continue
		case "@start":
			a.startCity()
			continue
		}
		if v, ok := strings.CutPrefix(tok, "@tool:"); ok {
			for _, b := range sim.Buildings {
				if b.Tool.String() == v {
					a.setTool(b.Tool)
				}
			}
			continue
		}
		if v, ok := strings.CutPrefix(tok, "@map:"); ok && a.nc != nil {
			a.nc.mapType, _ = strconv.Atoi(v)
			a.refreshPreview()
			continue
		}
		if v, ok := strings.CutPrefix(tok, "@mode:"); ok && a.nc != nil {
			a.nc.mode, _ = strconv.Atoi(v)
			a.refreshPreview()
			continue
		}
		if cmd, ok := strings.CutPrefix(tok, ":"); ok && cmd != "" {
			a.run(cmd)
			continue
		}
		if r := []rune(tok); len(r) == 1 {
			act, _ := a.km.Lookup(input.Binding{Rune: r[0]})
			a.do(input.Press{Action: act, Rune: r[0]})
			continue
		}
		a.do(input.Press{Action: input.Action(tok)})
	}
	a.cam.Jump(a.cam.TX, a.cam.TY)
	if o.Screenshot == "" && len(o.Actions) == 0 && !o.SkipTitle {
		a.showTitle()
	}
	for i := 0; i < o.Ticks; i++ {
		a.city.Tick()
	}
	a.dayClock += float64(o.Ticks) / float64(o.Config.TicksPerSecond)
	a.msg = ""
	return a
}

func (a *App) loadTheme(reload bool) {
	p, dir, err := theme.Load()
	if err != nil {
		a.city.Logf(sim.Warn, "theme %s: %v (using %s)", dir, err, theme.BuiltinName)
	}
	a.pal = p
	a.roles = theme.Derive(p)
	if a.atlas != nil {
		a.atlas.Dispose()
	}
	a.atlas = render.NewAtlas(a.roles)
	if a.chunks != nil {
		a.chunks.Reset()
	}
	a.miniDirty = true
	a.hud.R = a.roles
	a.hud.Invalidate()
	src := dir
	if src == "" {
		src = "built-in"
	}
	verb := "theme"
	if reload {
		verb = "theme changed:"
	}
	a.city.Logf(sim.Info, "%s %s (%s)", verb, p.Name, src)
	a.dirty = true
}

// Update runs at 60 TPS while focused.
func (a *App) Update() error {
	now := time.Now()
	if f := ebiten.IsFocused(); f != a.focused {
		a.focused, a.dirty = f, true
	}
	if a.watch != nil && a.watch.Changed(now) {
		a.loadTheme(true)
	}

	if ebiten.IsWindowBeingClosed() && a.mode != modeDialog {
		a.requestQuit()
	}
	switch {
	case a.mode == modeDialog:
		a.updateDialog()
	case a.mode == modeSettings:
		a.updateSettings()
	case a.scene == sceneTitle:
		dt := 0.0
		if !a.lastUpdate.IsZero() {
			dt = min(now.Sub(a.lastUpdate).Seconds(), 0.25)
		}
		a.updateTitle(dt)
	case a.scene == sceneNewCity:
		a.updateNewCity()
	case a.mode == modePause:
		a.updatePause()
	case a.mode == modeMenu:
		if i, ok := a.menuMouse(); ok {
			if i != a.menuSel {
				a.menuSel, a.dirty = i, true
			}
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				a.pick(i)
				break
			}
		}
		for _, p := range a.poll.Poll() {
			a.do(p)
		}
	case a.mode == modeCommand:
		a.updatePrompt()
	default:
		for _, p := range a.poll.Poll() {
			a.do(p)
		}
	}
	a.updateMouse()

	a.refreshMinimap(now)
	simRate := 0.0
	if a.scene == sceneGame {
		simRate = a.stepSim(now)
	} else {
		a.lastUpdate = now
	}

	camMoving := a.cam.Ease()
	if camMoving {
		a.dirty = true
	}
	if a.cfg.Animations && a.animVisible && !a.paused && now.After(a.nextAnim) {
		a.animFrame++
		a.nextAnim = now.Add(250 * time.Millisecond)
		a.dirty = true
	}
	if a.msg != "" && now.After(a.msgUntil) {
		a.msg, a.dirty = "", true
	}
	growing := len(a.grows) > 0
	cars := a.carsVisible && !a.paused && a.cfg.Animations && a.scene == sceneGame
	if cars {
		a.dirty = true
	}
	if growing {
		a.dirty = true
	}
	a.pace.set(camMoving, growing || cars, a.scene != sceneGame || a.cfg.Animations && a.animVisible && !a.paused, simRate)
	if a.quit {
		return ebiten.Termination
	}
	return nil
}

// newCity replaces the city with a freshly generated river map.
func (a *App) newCity(seed int64) {
	c := sim.New(a.cfg.MapSize, a.cfg.MapSize, seed)
	a.installCity(c, seed)
	c.Logf(sim.Info, "%s founded · seed %d", c.Name, seed)
}

// installCity makes c the running city with a fresh view and history.
func (a *App) installCity(c *sim.City, seed int64) {
	a.seed = seed
	a.city = c
	a.saveName, a.unsaved = "", false
	a.showGuide, a.sawBudget = guideFirstRun(), false
	a.undos, a.redos = nil, nil
	a.grows = map[sim.Pt]time.Time{}
	a.chunks.Reset()
	a.miniDirty = true
	a.lastPeak, a.lastResult = 0, c.ScenarioResult
	a.clearTool()
	a.cx, a.cy = c.Start.X, c.Start.Y
	if a.cx == 0 && a.cy == 0 {
		a.cx, a.cy = c.W/2, c.H/2
	}
	a.cam.Jump(a.cursorWorld())
	a.mode, a.dirty = modeNormal, true
}

// speedMult maps speed 1/2/3 to a tick-rate multiplier.
var speedMult = [4]float64{0, 1, 2, 4}

// growDuration is how long a new building takes to rise.
const growDuration = 300 * time.Millisecond

// stepSim runs as many fixed-rate ticks as wall time allows. It returns the
// current tick rate (0 when paused) for frame pacing.
func (a *App) stepSim(now time.Time) float64 {
	dt := 0.0
	if !a.lastUpdate.IsZero() {
		dt = min(now.Sub(a.lastUpdate).Seconds(), 0.25)
	}
	a.lastUpdate = now
	if a.paused {
		return 0
	}
	if a.cfg.TimeOfDay == "cycle" {
		a.dayClock += dt * speedMult[a.speed]
	}
	a.stepSmoke(dt * speedMult[a.speed])
	a.carClock += dt
	rate := float64(a.cfg.TicksPerSecond) * speedMult[a.speed]
	a.simAcc += dt * rate
	for n := 0; a.simAcc >= 1 && n < 16; n++ {
		a.simAcc--
		prevDay := a.city.Day
		a.unsaved = true
		changed := a.city.Tick()
		if len(changed) > 0 {
			a.miniDirty = true
		}
		for _, p := range changed {
			a.chunks.Touch(p.X, p.Y)
			if a.cfg.Animations {
				a.grows[p] = now
			}
		}
		a.autosave(prevDay)
		// The screen only needs a new frame when something visible moved:
		// a building changed or the date (and with it the HUD) advanced.
		if len(changed) > 0 || a.city.Day != prevDay {
			a.dirty = true
		}
	}
	a.simAcc = min(a.simAcc, 2) // don't try to catch up after a stall
	if r := a.city.ScenarioResult; r != a.lastResult {
		a.lastResult = r
		a.scenarioOver(r)
	}
	if a.city.PeakPop > a.lastPeak {
		for _, m := range sim.Milestones {
			if a.lastPeak < m.Pop && a.city.PeakPop >= m.Pop {
				a.snd.Play(soundMilestone)
				a.flash(sim.Info, "%s is now a %s! +$%s", a.city.Name, m.Title, commas(m.Grant))
			}
		}
		a.lastPeak = a.city.PeakPop
	}
	return rate
}

// dayLength is one day/night cycle at speed 1, in seconds (SPEC §7.3).
const dayLength = 180.0

// night is 0 by day and rises to 1 around midnight.
func (a *App) night() float64 {
	switch a.cfg.TimeOfDay {
	case "day":
		return 0
	case "night":
		return 0.85
	}
	phase := math.Mod(a.dayClock/dayLength, 1) // 0 midnight, 0.5 noon
	sun := 0.5 - 0.5*math.Cos(2*math.Pi*phase)
	return max(0, min(1, (0.4-sun)/0.4))
}

// stepSmoke spawns puffs at on-screen chimneys and moves existing ones.
func (a *App) stepSmoke(dt float64) {
	if !a.cfg.Animations {
		a.particles = a.particles[:0]
		return
	}
	live := a.particles[:0]
	for _, p := range a.particles {
		p.Age += dt
		if p.Age < p.Life {
			p.Y -= 4 * dt   // rise
			p.X += 1.5 * dt // drift with the wind
			live = append(live, p)
		}
	}
	a.particles = live
	if len(a.emitters) == 0 {
		return
	}
	a.smokeAcc += dt * float64(len(a.emitters)) * 1.2 // ~1.2 puffs per chimney per second
	for a.smokeAcc >= 1 && len(a.particles) < 64 {
		a.smokeAcc--
		e := a.emitters[a.smokeRng.IntN(len(a.emitters))]
		a.particles = append(a.particles, render.Particle{
			X: float64(e.X) + a.smokeRng.Float64() - 1, Y: float64(e.Y) - 2, Life: 2 + a.smokeRng.Float64(),
		})
	}
	a.smokeAcc = min(a.smokeAcc, 1)
}

func (a *App) growingList() []sim.Pt {
	if len(a.grows) == 0 {
		return nil
	}
	out := make([]sim.Pt, 0, len(a.grows))
	for p := range a.grows {
		out = append(out, p)
	}
	return out
}

// growProgress is the WorldView.Grow callback.
func (a *App) growProgress(x, y int) float64 {
	start, ok := a.grows[sim.Pt{X: x, Y: y}]
	if !ok {
		return 1
	}
	p := float64(time.Since(start)) / float64(growDuration)
	if p >= 1 {
		delete(a.grows, sim.Pt{X: x, Y: y})
		a.chunks.Touch(x, y) // bake the finished building into the cache
		return 1
	}
	return p
}

func (a *App) flash(level sim.EventLevel, format string, args ...any) {
	a.msg = fmt.Sprintf(format, args...)
	a.msgLevel = level
	a.msgUntil = time.Now().Add(4 * time.Second)
	a.dirty = true
}

func (a *App) move(dx, dy int) {
	a.cx = max(0, min(a.city.W-1, a.cx+dx))
	a.cy = max(0, min(a.city.H-1, a.cy+dy))
	a.follow()
	if a.paint && a.hasTool {
		a.applyTiles([]sim.Pt{{X: a.cx, Y: a.cy}}, true)
	}
}

func (a *App) cursorWorld() (float64, float64) {
	return float64(a.cx*render.TileSize + render.TileSize/2), float64(a.cy*render.TileSize + render.TileSize/2)
}

// follow scrolls the camera target when the cursor nears the view edge.
func (a *App) follow() {
	z := float64(a.cam.Zoom)
	ts := float64(render.TileSize) * z
	mx := math.Min(4*ts, float64(a.w)/4)
	my := math.Min(4*ts, float64(a.h)/4)
	wx, wy := a.cursorWorld()
	sx := float64(a.w)/2 + (wx-a.cam.TX)*z
	sy := float64(a.h)/2 + (wy-a.cam.TY)*z
	switch {
	case sx < mx:
		a.cam.TX -= (mx - sx) / z
	case sx > float64(a.w)-mx:
		a.cam.TX += (sx - (float64(a.w) - mx)) / z
	}
	switch {
	case sy < my:
		a.cam.TY -= (my - sy) / z
	case sy > float64(a.h)-my:
		a.cam.TY += (sy - (float64(a.h) - my)) / z
	}
}

func (a *App) zoom(d int) {
	z := max(1, min(4, a.cam.Zoom+d))
	if z == a.cam.Zoom {
		return
	}
	a.cam.Zoom = z
	a.follow()
	a.flash(sim.Info, "zoom %dx", z)
}

// zoomAt zooms keeping the world point under screen position (mx, my) in
// place, so the wheel zooms toward the mouse.
func (a *App) zoomAt(d, mx, my int) {
	old := float64(a.cam.Zoom)
	wx := a.cam.X + float64(mx-a.w/2)/old
	wy := a.cam.Y + float64(my-a.h/2)/old
	z := max(1, min(4, a.cam.Zoom+d))
	if z == a.cam.Zoom {
		return
	}
	a.cam.Zoom = z
	a.cam.Jump(wx-float64(mx-a.w/2)/float64(z), wy-float64(my-a.h/2)/float64(z))
	a.dirty = true
	a.flash(sim.Info, "zoom %dx", z)
}

func (a *App) updateMouse() {
	mx, my := ebiten.CursorPosition()
	if _, dy := ebiten.Wheel(); dy != 0 && a.scene == sceneGame {
		a.wheel += dy
		for a.wheel >= 1 {
			a.wheel--
			a.zoomAt(1, mx, my)
		}
		for a.wheel <= -1 {
			a.wheel++
			a.zoomAt(-1, mx, my)
		}
	}
	if a.updateMinimap(mx, my) {
		return
	}
	a.updateLeftButton(mx, my)
	pan := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)
	if pan && a.panning && (mx != a.panX || my != a.panY) {
		z := float64(a.cam.Zoom)
		a.cam.Jump(a.cam.X-float64(mx-a.panX)/z, a.cam.Y-float64(my-a.panY)/z)
		a.dirty = true
	}
	a.panning, a.panX, a.panY = pan, mx, my
}

// Draw only repaints when something changed; otherwise the previous frame
// stays on screen (SetScreenClearedEveryFrame is off) and the GPU idles.
func (a *App) Draw(screen *ebiten.Image) {
	if !a.dirty && a.shotPath == "" {
		return
	}
	a.dirty = false
	cx, cy := a.cx, a.cy
	if a.scene != sceneGame {
		cx, cy = -100, -100 // no cursor behind menus
	}
	view := render.WorldView{
		City: a.city, Atlas: a.atlas, Roles: a.roles, Cam: &a.cam,
		CursorX: cx, CursorY: cy, AnimFrame: a.animFrame, Animations: a.cfg.Animations,
		Underground: a.underground, Grow: a.growProgress, Chunks: a.chunks, Growing: a.growingList(),
		Overlay: a.overlay, Blink: !a.cfg.Animations || time.Now().UnixMilli()/500%2 == 0,
		Night: a.night(), Particles: a.particles, Emitters: &a.emitters, Time: a.carClock, CarsOut: &a.carsVisible,
	}
	if plan, sel, ok := a.pending(); ok {
		view.Preview, view.PreviewBad = plan.Tiles, plan.Err != ""
		if plan.Err != "" && len(plan.Tiles) == 0 {
			view.Preview = sel
		}
	}
	a.carsVisible = false
	a.animVisible = render.DrawWorld(screen, view) || len(a.particles) > 0 || len(a.emitters) > 0
	switch a.scene {
	case sceneTitle:
		if a.mode != modeSettings {
			a.hud.Title(screen, a.titleEntries(), a.titleSel, versionLine()+" · theme "+a.pal.Name)
		}
		if a.dlg != nil {
			a.hud.Draw(screen, render.HUDState{City: a.city, Dialog: a.dlg.view(), Only: true})
		}
	case sceneNewCity:
		a.hud.NewCity(screen, a.formView())
	default:
		st := a.hudState()
		a.hud.DrawCached(screen, st, a.hudKey(st))
		if a.mode == modePause {
			a.hud.PauseMenu(screen, a.pauseEntries(), a.pauseSel)
		}
		if a.mode == modeStats {
			a.hud.Stats(screen, a.city)
		}
	}
	if a.mode == modeSettings {
		a.hud.Settings(screen, a.settingsEntries(), a.settingSel, tildePath(a.configPath))
	}

	if a.shotPath != "" {
		a.shotFrames++
		if a.shotFrames == 3 {
			if err := savePNG(screen, a.shotPath); err != nil {
				fmt.Fprintln(os.Stderr, "screenshot:", err)
			}
			a.quit = true
		}
	}
}

func (a *App) hudState() render.HUDState {
	modeName := "NORMAL"
	switch {
	case a.mode == modeHelp:
		modeName = "HELP"
	case a.mode == modeMenu:
		modeName = "BUILD"
	case a.mode == modeBudget:
		modeName = "BUDGET"
	case a.mode == modeStats:
		modeName = "STATS"
	case a.mode == modeDialog:
		modeName = "FILE"
	case a.zonePending:
		modeName = "ZONE r/c/i"
	case a.visual || a.drag:
		modeName = "VISUAL"
	case a.paint:
		modeName = "PAINT"
	}
	tool := "inspect"
	if a.hasTool {
		tool = a.tool.String()
		if a.tool == sim.ToolBulldoze && a.underground {
			tool = "bulldoze pipes"
		}
	}
	st := render.HUDState{
		City: a.city, Paused: a.paused, Speed: a.speed, Mode: modeName, Tool: tool,
		CursorX: a.cx, CursorY: a.cy, TileInfo: a.tileInfo(),
		Message: a.msg, MessageLevel: a.msgLevel,
		ShowPanel: a.showPanel, ShowLog: a.showLog, ShowHelp: a.mode == modeHelp,
		Zoom: a.cam.Zoom, Overlay: a.overlay, ThemeName: a.pal.Name,
	}
	st.Forecast = a.city.Forecast()
	if a.mode == modeBudget {
		st.ShowBudget = true
		a.sawBudget = true
	}
	if a.mode == modeMenu {
		st.Menu = &render.Menu{Sel: a.menuSel}
		for _, b := range sim.Buildings {
			it := render.MenuItem{
				Name: b.Tool.String(), Size: fmt.Sprintf("%dx%d", b.Size, b.Size),
				Cost: "$" + commas(b.Cost), Note: b.Note,
			}
			if !a.city.Unlocked(b) {
				it.Note, it.Locked = fmt.Sprintf("unlocks at %s people", commas(float64(b.Unlock))), true
			}
			st.Menu.Items = append(st.Menu.Items, it)
		}
	}
	if a.mode == modeCommand {
		st.Prompt = &a.prompt
	}
	if a.mode == modeDialog {
		st.Dialog = a.dlg.view()
	}
	st.Toolbar, st.ToolHover = a.toolbarButtons(), a.toolHover
	if a.showMinimap && a.mini != nil {
		st.Minimap, st.MinimapView = a.mini, a.viewTiles()
	}
	st.Guide = a.guideView()
	if plan, sel, ok := a.pending(); ok {
		if plan.Err != "" {
			st.Hint, st.HintBad = plan.Err, true
		} else {
			st.Hint = fmt.Sprintf("%d tiles · $%s", len(plan.Tiles), commas(plan.Cost))
			if len(sel) == 1 {
				st.Hint = "$" + commas(plan.Cost)
			}
		}
	}
	if st.ShowHelp {
		for _, d := range a.km.Defs {
			st.Help = append(st.Help, render.HelpRow{Group: d.Group, Keys: strings.Join(d.Keys, " "), Desc: d.Help})
		}
	}
	return st
}

func (a *App) tileInfo() string {
	t := a.city.At(a.cx, a.cy)
	var parts []string
	switch {
	case t.Kind == sim.Road && t.Terrain == sim.Water:
		parts = append(parts, "bridge")
	case t.Kind == sim.Road:
		parts = append(parts, "road")
	case t.IsZone():
		name := [...]string{"residential", "commercial", "industrial"}[t.Kind-sim.ZoneR]
		if t.Level == 0 {
			name += " lot"
			if !a.city.RoadAccess(a.cx, a.cy) {
				name += " · no road"
			}
			if !t.Powered {
				name += " · no power"
			}
		} else {
			name += fmt.Sprintf(" lvl%d", t.Level)
		}
		parts = append(parts, name)
	case t.IsBuilding():
		b, _ := sim.SpecOf(t.Kind)
		parts = append(parts, b.Tool.String())
	case t.Terrain == sim.Water:
		parts = append(parts, "water")
	case t.Terrain == sim.Rock:
		parts = append(parts, "rock")
	case t.Terrain == sim.Trees:
		parts = append(parts, "trees")
	default:
		parts = append(parts, "grass")
	}
	if t.Line {
		parts = append(parts, "power line")
	}
	spec, isB := sim.SpecOf(t.Kind)
	if (t.IsZone() && t.Level > 0) || (isB && spec.PowerUse > 0) {
		if !t.Powered {
			parts = append(parts, "no power")
		}
	}
	if (t.IsZone() && t.Level > 0) || (isB && spec.WaterUse > 0) {
		if !t.Watered {
			parts = append(parts, "no water")
		}
	}
	if t.Pipe {
		parts = append(parts, "pipe")
	}
	if t.Terrain != sim.Water {
		parts = append(parts, fmt.Sprintf("LV %.2f", t.LandValue))
		switch {
		case t.Kind == sim.Road && t.Traffic > 0:
			parts = append(parts, fmt.Sprintf("traffic %d (%.0f%%)", t.Traffic, 100*t.Congestion()))
		case t.Kind == sim.ZoneR && t.Level > 0 && t.Commute < 0:
			parts = append(parts, "no road to jobs")
		case func() bool { _, ok := a.overlay.Service(); return ok }():
			si, _ := a.overlay.Service()
			parts = append(parts, fmt.Sprintf("%s %.0f%%", a.overlay, 100*t.Cover[si]))
		case t.Pollution > 0.05:
			parts = append(parts, fmt.Sprintf("smog %.0f%%", 100*t.Pollution))
		}
	}
	return strings.Join(parts, " · ")
}

// hudKey digests everything the HUD shows, so the cached HUD layer is
// redrawn exactly when it would look different.
func (a *App) hudKey(st render.HUDState) string {
	var b strings.Builder
	c := a.city
	fmt.Fprintf(&b, "%p|%d|%.0f|%d|%v|%.2f|%.2f|%.2f|%v|%v|%v|%.0f|%.0f|%d|%v|%d|%s|%d|",
		c, c.Day, c.Funds, c.PeakPop, c.Stats, c.Demand[0], c.Demand[1], c.Demand[2], c.Power, c.Water,
		c.Tax, st.Forecast.Net, c.LastMonth.Net, c.DebtMonths, c.Loan != nil, len(c.Log), c.Name, a.hud.S)
	if n := len(c.Log); n > 0 {
		b.WriteString(c.Log[n-1].Msg)
	}
	fmt.Fprintf(&b, "|%v|%d|%s|%s|%d|%d|%s|%s|%v|%s|%d|%v|%v|%v|%v|%d|%v|%s|%d|%d|%v|%v|%d|%v|%s|",
		st.Paused, st.Speed, st.Mode, st.Tool, st.CursorX, st.CursorY, st.TileInfo, st.Hint, st.HintBad,
		st.Message, st.MessageLevel, st.ShowPanel, st.ShowLog, st.ShowHelp, st.ShowBudget, st.Zoom, st.Overlay,
		st.ThemeName, st.ToolHover, a.mode, a.hasTool, a.tool, a.menuSel, a.underground, a.roles.Name)
	if st.Prompt != nil {
		b.WriteString(*st.Prompt)
	}
	fmt.Fprintf(&b, "|mm%v%v%v", st.Minimap != nil, st.MinimapView, a.miniNext)
	if st.Guide != nil {
		fmt.Fprintf(&b, "|g%d%v", st.Guide.Step, st.Guide.Done)
	}
	if st.Dialog != nil {
		fmt.Fprintf(&b, "|d%s%d", st.Dialog.Title, st.Dialog.Sel)
		if st.Dialog.Input != nil {
			b.WriteString(*st.Dialog.Input)
		}
	}
	if st.Menu != nil {
		fmt.Fprintf(&b, "|m%d", st.Menu.Sel)
	}
	return b.String()
}

// Layout maps the window to physical pixels 1:1 so integer scaling stays
// crisp under fractional desktop scaling.
func (a *App) Layout(ow, oh int) (int, int) {
	s := ebiten.Monitor().DeviceScaleFactor()
	w, h := int(math.Ceil(float64(ow)*s)), int(math.Ceil(float64(oh)*s))
	if a.shotPath != "" {
		w, h = 1280, 800 // screenshots don't depend on how the WM tiled the window
	}
	if w != a.w || h != a.h {
		first := a.w == 0
		a.w, a.h, a.dirty = w, h, true
		if first {
			a.follow()
		}
	}
	return w, h
}

func savePNG(img *ebiten.Image, path string) error {
	b := img.Bounds()
	rgba := image.NewRGBA(b)
	img.ReadPixels(rgba.Pix)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, rgba)
}
