package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/antoniowav/sprawl/internal/meta"
	"github.com/antoniowav/sprawl/sim"
)

const saveExt = ".city"

// saveName turns a city or user name into a safe file stem.
func saveName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "city"
	}
	return b.String()
}

func savePath(name string) string { return filepath.Join(meta.DataDir(), saveName(name)+saveExt) }

// writeSave writes atomically: a temp file renamed over the old save.
func (a *App) writeSave(name string) error {
	path := savePath(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".save-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := a.city.Save(tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (a *App) save(name string) bool {
	if name == "" {
		name = a.city.Name
	}
	if err := a.writeSave(name); err != nil {
		a.flash(sim.Err, "save failed: %v", err)
		return false
	}
	a.saveName, a.unsaved = saveName(name), false
	a.city.Logf(sim.Info, "saved %s", savePath(name))
	a.flash(sim.Info, "saved %s", saveName(name))
	return true
}

func (a *App) load(name string) {
	if name == "" {
		a.flash(sim.Info, "saves: %s", strings.Join(listSaves(), " "))
		return
	}
	f, err := os.Open(savePath(name))
	if err != nil {
		a.flash(sim.Err, "load: no save named %s (try :e with no name)", saveName(name))
		return
	}
	defer f.Close()
	c, err := sim.Load(f)
	if err != nil {
		a.flash(sim.Err, "load %s: %v", saveName(name), err)
		return
	}
	a.city = c
	a.lastPeak, a.lastResult = c.PeakPop, c.ScenarioResult
	a.saveName, a.unsaved = saveName(name), false
	if a.saveName == "autosave" {
		a.saveName = "" // don't let Ctrl+S overwrite the autosave
	}
	a.city.Logf(sim.Info, "loaded %s", saveName(name)+saveExt)
	a.grows = map[sim.Pt]time.Time{}
	a.chunks.Reset()
	a.miniDirty = true
	a.undos, a.redos = nil, nil
	a.clearTool()
	a.cx, a.cy = c.W/2, c.H/2
	a.cam.Jump(a.cursorWorld())
	a.mode, a.dirty = modeNormal, true
	a.scene = sceneGame
}

// saveInfo is one save for the open dialog.
type saveInfo struct{ name, when string }

// listSaveInfo returns saves newest first, with a readable date.
func listSaveInfo() []saveInfo {
	paths, _ := filepath.Glob(filepath.Join(meta.DataDir(), "*"+saveExt))
	type entry struct {
		info saveInfo
		mod  time.Time
	}
	var es []entry
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		es = append(es, entry{saveInfo{strings.TrimSuffix(filepath.Base(p), saveExt), fi.ModTime().Format("Jan 2 15:04")}, fi.ModTime()})
	}
	sort.Slice(es, func(i, j int) bool { return es[i].mod.After(es[j].mod) })
	out := make([]saveInfo, len(es))
	for i, e := range es {
		out[i] = e.info
	}
	return out
}

// listSaves returns save names, newest first.
func listSaves() []string {
	paths, _ := filepath.Glob(filepath.Join(meta.DataDir(), "*"+saveExt))
	type entry struct {
		name string
		mod  time.Time
	}
	var es []entry
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		es = append(es, entry{strings.TrimSuffix(filepath.Base(p), saveExt), fi.ModTime()})
	}
	sort.Slice(es, func(i, j int) bool { return es[i].mod.After(es[j].mod) })
	names := make([]string, len(es))
	for i, e := range es {
		names[i] = e.name
	}
	if len(names) == 0 {
		return []string{"(none yet · :w to save)"}
	}
	return names
}

// autosave runs at the start of every Nth month.
func (a *App) autosave(prevDay int) {
	n := a.cfg.AutosaveMonths
	if n == 0 || a.city.Bankrupt {
		return
	}
	month := a.city.Day / sim.DaysPerMonth
	if month == prevDay/sim.DaysPerMonth || month%n != 0 {
		return
	}
	if err := a.writeSave("autosave"); err != nil {
		a.city.Logf(sim.Err, "autosave failed: %v", err)
	}
}
