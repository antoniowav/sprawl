package theme

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Candidates lists theme directories in search order.
func Candidates() []string {
	var dirs []string
	if v := os.Getenv("SPRAWL_THEME_DIR"); v != "" {
		dirs = append(dirs, v)
	}
	home, _ := os.UserHomeDir()
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		state = filepath.Join(home, ".local", "state")
	}
	// Cuore (an Omarchy-based distro) keeps its theme under its own name.
	return append(dirs,
		filepath.Join(state, "cuore", "current", "theme"),
		filepath.Join(home, ".config", "omarchy", "current", "theme"),
		filepath.Join(state, "omarchy", "current", "theme"))
}

// FindDir returns the first candidate holding a parseable palette file, or "".
func FindDir() string {
	for _, d := range Candidates() {
		if exists(filepath.Join(d, "colors.toml")) || exists(filepath.Join(d, "alacritty.toml")) {
			return d
		}
	}
	return ""
}

// Load returns the active palette and the directory it came from. Errors are
// returned alongside the builtin palette so the game can always start.
func Load() (Palette, string, error) {
	dir := FindDir()
	if dir == "" {
		return Builtin(), "", nil
	}
	p, err := LoadDir(dir)
	if err != nil {
		return Builtin(), dir, err
	}
	return p, dir, nil
}

// LoadDir parses colors.toml, falling back to alacritty.toml.
func LoadDir(dir string) (Palette, error) {
	var (
		p   Palette
		err error
	)
	if path := filepath.Join(dir, "colors.toml"); exists(path) {
		p, err = parseColors(path)
	} else if path := filepath.Join(dir, "alacritty.toml"); exists(path) {
		p, err = parseAlacritty(path)
	} else {
		return Builtin(), errors.New("no colors.toml or alacritty.toml in " + dir)
	}
	if err != nil {
		return Builtin(), err
	}
	p.Name = themeName(dir)
	return p, nil
}

func parseColors(path string) (Palette, error) {
	var raw map[string]any
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return Palette{}, err
	}
	vals := map[string]string{}
	for k, v := range raw {
		if s, ok := v.(string); ok {
			vals[k] = s
		}
	}
	return build(vals)
}

func parseAlacritty(path string) (Palette, error) {
	var a struct {
		Colors struct {
			Primary   map[string]string `toml:"primary"`
			Normal    map[string]string `toml:"normal"`
			Bright    map[string]string `toml:"bright"`
			Selection map[string]string `toml:"selection"`
		} `toml:"colors"`
	}
	if _, err := toml.DecodeFile(path, &a); err != nil {
		return Palette{}, err
	}
	c := a.Colors
	vals := map[string]string{
		"background": c.Primary["background"],
		"foreground": c.Primary["foreground"],
		"muted":      c.Bright["black"],
		"accent":     c.Normal["blue"],
		"selection":  c.Selection["background"],
	}
	for _, n := range []string{"red", "yellow", "green", "cyan", "blue", "magenta"} {
		vals[n] = c.Normal[n]
		vals["bright_"+n] = c.Bright[n]
	}
	return build(vals)
}

// build overlays the given hex values on the builtin palette and derives
// the rest from what was set.
func build(vals map[string]string) (Palette, error) {
	p := Builtin()
	set := map[string]bool{}
	for k, ptr := range p.fields() {
		s, ok := vals[k]
		if !ok || s == "" {
			continue
		}
		c, err := ParseHex(s)
		if err != nil {
			return Palette{}, errors.New(k + ": " + err.Error())
		}
		*ptr = c
		set[k] = true
	}
	if !set["background"] || !set["foreground"] {
		return Palette{}, errors.New("theme has no background/foreground")
	}
	switch strings.ToLower(vals["mode"]) {
	case "light":
		p.Light = true
	case "dark":
		p.Light = false
	default:
		p.Light = Luminance(p.Background) > 0.4
	}
	fillDerived(&p, set)
	return p, nil
}

// themeName prefers Omarchy's theme.name next to the theme dir.
func themeName(dir string) string {
	if b, err := os.ReadFile(filepath.Join(dir, "..", "theme.name")); err == nil {
		if n := strings.TrimSpace(string(b)); n != "" {
			return n
		}
	}
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	return filepath.Base(dir)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
