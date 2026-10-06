package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/antoniowav/sprawl/config"
	"github.com/antoniowav/sprawl/input"
	"github.com/antoniowav/sprawl/render"
	"github.com/antoniowav/sprawl/sim"
)

// setting is one adjustable option: show renders its value, step moves it
// by d (±1) and applies it at once.
type setting struct {
	label string
	show  func(c *config.Config) string
	step  func(a *App, d int)
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func cycle(v, d int, opts []int) int {
	i := 0
	for j, o := range opts {
		if o == v {
			i = j
		}
	}
	return opts[(i+d+len(opts))%len(opts)]
}

var settings = []setting{
	{"Interface size", func(c *config.Config) string { return fmt.Sprintf("%dx", c.UIScale) },
		func(a *App, d int) {
			a.cfg.UIScale = cycle(a.cfg.UIScale, d, []int{1, 2, 3, 4})
			a.hud.S = a.cfg.UIScale
		}},
	{"Starting zoom", func(c *config.Config) string { return fmt.Sprintf("%dx", c.Zoom) },
		func(a *App, d int) { a.cfg.Zoom = cycle(a.cfg.Zoom, d, []int{1, 2, 3, 4}) }},
	{"Sound volume", func(c *config.Config) string {
		if c.Volume == 0 {
			return "off"
		}
		return fmt.Sprintf("%d%%", c.Volume)
	}, func(a *App, d int) {
		a.cfg.Volume = max(0, min(100, a.cfg.Volume+10*d))
		a.snd.SetVolume(float64(a.cfg.Volume) / 100)
	}},
	{"Advisor tips", func(c *config.Config) string { return onOff(c.Tips) },
		func(a *App, d int) { a.cfg.Tips = !a.cfg.Tips }},
	{"Animations", func(c *config.Config) string { return onOff(c.Animations) },
		func(a *App, d int) { a.cfg.Animations = !a.cfg.Animations }},
	{"Day and night", func(c *config.Config) string { return timeLabel(c.TimeOfDay) },
		func(a *App, d int) { a.cycleTimeOfDay(d) }},
	{"Autosave", func(c *config.Config) string {
		if c.AutosaveMonths == 0 {
			return "off"
		}
		return fmt.Sprintf("every %d months", c.AutosaveMonths)
	}, func(a *App, d int) { a.cfg.AutosaveMonths = cycle(a.cfg.AutosaveMonths, d, []int{0, 3, 6, 12}) }},
	{"Pause when unfocused", func(c *config.Config) string { return onOff(c.PauseUnfocused) },
		func(a *App, d int) {
			a.cfg.PauseUnfocused = !a.cfg.PauseUnfocused
			ebiten.SetRunnableOnUnfocused(!a.cfg.PauseUnfocused)
		}},
	{"Follow desktop theme", func(c *config.Config) string { return onOff(c.WatchTheme) },
		func(a *App, d int) { a.cfg.WatchTheme = !a.cfg.WatchTheme }},
	{"Low-power rendering", func(c *config.Config) string { return onOff(c.LowPower) + " (restart)" },
		func(a *App, d int) { a.cfg.LowPower = !a.cfg.LowPower }},
}

func (a *App) openSettings() {
	a.mode, a.settingSel, a.dirty = modeSettings, 0, true
}

func (a *App) closeSettings() {
	a.mode, a.dirty = modeNormal, true
	if a.configPath != "" {
		if err := config.Save(a.configPath, a.cfg); err != nil {
			a.flash(sim.Err, "settings not saved: %v", err)
			return
		}
	}
	a.flash(sim.Info, "settings saved")
}

func (a *App) settingsEntries() []render.MenuEntry {
	es := make([]render.MenuEntry, len(settings)+2)
	for i, s := range settings {
		es[i] = render.MenuEntry{Label: s.label, Detail: "< " + s.show(&a.cfg) + " >"}
	}
	es[len(settings)] = render.MenuEntry{Label: "Keys…", Detail: "Enter"}
	es[len(settings)+1] = render.MenuEntry{Label: "Done", Detail: "Esc"}
	return es
}

func (a *App) updateSettings() {
	n := len(settings) + 2
	keysRow := len(settings)
	done := len(settings) + 1
	step := func(d int) {
		if a.settingSel < len(settings) {
			settings[a.settingSel].step(a, d)
			a.snd.Play(soundClick)
		}
	}
	for _, p := range a.poll.Poll() {
		a.dirty = true
		switch p.Action {
		case input.MoveDown:
			a.settingSel = (a.settingSel + 1) % n
		case input.MoveUp:
			a.settingSel = (a.settingSel + n - 1) % n
		case input.MoveLeft:
			step(-1)
		case input.MoveRight:
			step(1)
		case input.Apply:
			switch a.settingSel {
			case done:
				a.closeSettings()
				return
			case keysRow:
				a.mode, a.keySel, a.keyCapture = modeKeys, 0, false
				return
			}
			step(1)
		case input.Cancel:
			a.closeSettings()
			return
		}
	}
	if i, ok := a.menuMouse(); ok {
		if i != a.settingSel {
			a.settingSel, a.dirty = i, true
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			switch i {
			case done:
				a.closeSettings()
				return
			case keysRow:
				a.mode, a.keySel, a.keyCapture = modeKeys, 0, false
				return
			}
			step(1)
			a.dirty = true
		}
	}
}

// tildePath shortens a path under $HOME to ~/....
func tildePath(p string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if rest, ok := strings.CutPrefix(p, home); ok {
			return "~" + rest
		}
	}
	return p
}

func timeLabel(m string) string {
	return map[string]string{"cycle": "cycle", "day": "always day", "night": "always night", "frozen": "paused"}[m]
}

// cycleTimeOfDay steps through cycle, day, night and frozen.
func (a *App) cycleTimeOfDay(d int) {
	ms := config.TimesOfDay
	i := 0
	for j, m := range ms {
		if m == a.cfg.TimeOfDay {
			i = j
		}
	}
	a.cfg.TimeOfDay = ms[(i+d+len(ms))%len(ms)]
	a.dirty = true
}
