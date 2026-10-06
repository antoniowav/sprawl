package app

import (
	"bufio"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/antoniowav/sprawl/input"
	"github.com/antoniowav/sprawl/render"
	"github.com/antoniowav/sprawl/sim"
)

// --- toast: advisor tips and achievements ---

func (a *App) toast(msg string, highlight bool) {
	a.toastMsg, a.toastHi = msg, highlight
	a.toastUntil = time.Now().Add(10 * time.Second)
	a.dirty = true
}

// monthly runs once at the start of each month while playing.
func (a *App) monthly() {
	a.checkAchievements()
	if !a.cfg.Tips || a.tl != nil {
		return
	}
	for _, t := range a.city.Advice() {
		if last, ok := a.tipShown[t.Key]; ok && a.city.Day-last < 6*sim.DaysPerMonth {
			continue
		}
		a.tipShown[t.Key] = a.city.Day
		a.toast("Advisor: "+t.Msg, false)
		return
	}
}

// --- photo mode ---

// picturesDir follows XDG user-dirs, falling back to ~/Pictures.
func picturesDir() string {
	home, _ := os.UserHomeDir()
	if f, err := os.Open(filepath.Join(home, ".config", "user-dirs.dirs")); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if v, ok := strings.CutPrefix(sc.Text(), "XDG_PICTURES_DIR="); ok {
				v = strings.Trim(v, `"`)
				return strings.Replace(v, "$HOME", home, 1)
			}
		}
	}
	return filepath.Join(home, "Pictures")
}

// takePhoto renders the map without HUD or cursor and saves it as a PNG.
func (a *App) takePhoto(view render.WorldView) {
	b := image.Rect(0, 0, a.w, a.h)
	img := ebiten.NewImage(b.Dx(), b.Dy())
	defer img.Deallocate()
	view.CursorX, view.CursorY, view.Preview = -100, -100, nil
	view.Emitters, view.CarsOut = nil, nil
	render.DrawWorld(img, view)
	rgba := image.NewRGBA(b)
	img.ReadPixels(rgba.Pix)
	dir := filepath.Join(picturesDir(), "Sprawl")
	name := fmt.Sprintf("%s-%s.png", saveName(a.city.Name), time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, name)
	go func() { // encoding a large PNG takes a moment; don't stall the frame
		err := os.MkdirAll(dir, 0o755)
		if err == nil {
			var f *os.File
			if f, err = os.Create(path); err == nil {
				err = png.Encode(f, rgba)
				f.Close()
			}
		}
		a.photoDone <- photoResult{path, err}
	}()
}

type photoResult struct {
	path string
	err  error
}

func (a *App) pollPhoto() {
	select {
	case r := <-a.photoDone:
		if r.err != nil {
			a.flash(sim.Err, "photo failed: %v", r.err)
		} else {
			a.flash(sim.Info, "photo saved: %s", tildePath(r.path))
		}
	default:
	}
}

// --- key editor ---

func (a *App) keyRows() []render.KeyRow {
	rows := make([]render.KeyRow, len(a.km.Defs))
	for i, d := range a.km.Defs {
		rows[i] = render.KeyRow{Action: string(d.Action), Help: d.Help, Keys: strings.Join(d.Keys, " ")}
	}
	return rows
}

// rebuildKeymap applies cfg.Keys; on a conflict it restores old and
// reports the problem.
func (a *App) rebuildKeymap(old []string, action string) bool {
	km, errs := input.NewKeymap(a.cfg.Keys)
	if len(errs) > 0 {
		if old == nil {
			delete(a.cfg.Keys, action)
		} else {
			a.cfg.Keys[action] = old
		}
		a.flash(sim.Warn, "%v", errs[0])
		return false
	}
	a.km = km
	a.poll = input.NewPoller(km)
	return true
}

func (a *App) updateKeys() {
	defs := a.km.Defs
	if a.keyCapture {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			a.keyCapture, a.dirty = false, true
			return
		}
		token := ""
		ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
		shift := ebiten.IsKeyPressed(ebiten.KeyShift)
		for _, r := range a.poll.Chars() {
			if !ctrl {
				token = string(r)
				if r == ' ' {
					token = "Space"
				}
			}
		}
		if token == "" {
			for _, k := range inpututil.AppendJustPressedKeys(nil) {
				name := input.KeyName(k)
				if name == "" || (!ctrl && len(name) == 1) {
					continue // plain letters arrive as typed characters
				}
				if shift {
					name = "Shift+" + name
				}
				if ctrl {
					name = "Ctrl+" + name
				}
				token = name
			}
		}
		if token == "" {
			return
		}
		act := string(defs[a.keySel].Action)
		if a.cfg.Keys == nil {
			a.cfg.Keys = map[string][]string{}
		}
		old, had := a.cfg.Keys[act]
		if !had {
			old = nil
		}
		a.cfg.Keys[act] = []string{token}
		if a.rebuildKeymap(old, act) {
			a.flash(sim.Info, "%s: %s", defs[a.keySel].Help, token)
		}
		a.keyCapture, a.dirty = false, true
		return
	}
	n := len(defs)
	switch {
	case input.Repeated(ebiten.KeyArrowDown) || hasRune(a.poll.Chars(), 'j'):
		a.keySel = (a.keySel + 1) % n
	case input.Repeated(ebiten.KeyArrowUp):
		a.keySel = (a.keySel + n - 1) % n
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		a.keyCapture = true
	case inpututil.IsKeyJustPressed(ebiten.KeyBackspace):
		act := string(defs[a.keySel].Action)
		old := a.cfg.Keys[act]
		delete(a.cfg.Keys, act)
		if a.rebuildKeymap(old, act) {
			a.flash(sim.Info, "%s: back to default", defs[a.keySel].Help)
		}
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.mode = modeSettings
	default:
		if i, ok := a.menuMouse(); ok && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			visible := len(a.hud.Hits)
			first := max(0, min(a.keySel-visible/2, n-visible))
			a.keySel = first + i
			a.keyCapture = true
		} else {
			return
		}
	}
	a.dirty = true
}

func (a *App) achievementRows() []render.AchievementRow {
	rows := make([]render.AchievementRow, len(achievements))
	for i, ac := range achievements {
		rows[i] = render.AchievementRow{Name: ac.name, Desc: ac.desc, Date: a.unlocked[ac.id]}
	}
	return rows
}
