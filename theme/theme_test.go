package theme

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const gruvbox = `mode = "dark"
accent = "#7daea3"
background = "#282828"
foreground = "#d4be98"
green = "#a9b665"
blue = "#7daea3"
red = "#ea6962"
yellow = "#d8a657"
`

const alacritty = `[colors.primary]
background = "#fafafa"
foreground = "#383a42"
[colors.normal]
red = "#e45649"
green = "#50a14f"
yellow = "#c18401"
blue = "#4078f2"
magenta = "#a626a4"
cyan = "#0184bc"
[colors.bright]
black = "#a0a1a7"
`

func writeTheme(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "theme")
	os.MkdirAll(dir, 0o755)
	for name, body := range files {
		os.WriteFile(filepath.Join(root, name), []byte(body), 0o644)
	}
	return dir
}

func TestParseHex(t *testing.T) {
	for _, s := range []string{"#7daea3", "7daea3", "0x7daea3", "#7daea3ff"} {
		c, err := ParseHex(s)
		if err != nil || c.R != 0x7d || c.G != 0xae || c.B != 0xa3 {
			t.Errorf("%s -> %v %v", s, c, err)
		}
	}
	if _, err := ParseHex("#fff"); err == nil {
		t.Error("short hex accepted")
	}
}

func TestColorsToml(t *testing.T) {
	dir := writeTheme(t, map[string]string{"theme/colors.toml": gruvbox, "theme.name": "gruvbox\n"})
	p, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "gruvbox" || p.Light {
		t.Errorf("name %q light %v", p.Name, p.Light)
	}
	if p.Background != MustHex("#282828") || p.Green != MustHex("#a9b665") {
		t.Error("keys not read")
	}
	// Unset keys derive from the theme, not from the builtin.
	if p.DarkBackground == Builtin().DarkBackground {
		t.Error("dark_background came from builtin")
	}
	// Unset hues fall back to builtin.
	if p.Magenta != Builtin().Magenta {
		t.Error("magenta should fall back")
	}
}

func TestAlacrittyFallbackAndLightMode(t *testing.T) {
	dir := writeTheme(t, map[string]string{"theme/alacritty.toml": alacritty})
	p, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Light {
		t.Error("light background not detected")
	}
	if p.Blue != MustHex("#4078f2") || p.Muted != MustHex("#a0a1a7") {
		t.Error("alacritty colours not mapped")
	}
	if p.Name != "theme" {
		t.Errorf("name %q", p.Name)
	}
	r := Derive(p)
	if Contrast(r.UIText, r.UIPanel) < 4.5 {
		t.Errorf("text contrast %.2f", Contrast(r.UIText, r.UIPanel))
	}
}

func TestBadThemeFallsBack(t *testing.T) {
	dir := writeTheme(t, map[string]string{"theme/colors.toml": "background = \"nope\"\n"})
	if _, err := LoadDir(dir); err == nil {
		t.Fatal("expected error")
	}
	t.Setenv("SPRAWL_THEME_DIR", dir)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	p, got, err := Load()
	if err == nil || got != dir || p.Name != BuiltinName {
		t.Fatalf("Load = %q %q %v", p.Name, got, err)
	}
}

func TestNoThemeUsesBuiltin(t *testing.T) {
	t.Setenv("SPRAWL_THEME_DIR", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	p, dir, err := Load()
	if err != nil || dir != "" || p.Name != BuiltinName {
		t.Fatalf("got %q %q %v", p.Name, dir, err)
	}
}

func TestReadableAllRoles(t *testing.T) {
	r := Derive(Builtin())
	if Contrast(r.UIText, r.UIPanel) < 4.5 || Contrast(r.UIAccentText, r.UIAccent) < 4.5 {
		t.Error("builtin palette unreadable")
	}
}

func TestWatcher(t *testing.T) {
	dir := writeTheme(t, map[string]string{"theme/colors.toml": gruvbox})
	t.Setenv("SPRAWL_THEME_DIR", dir)
	w := NewWatcher()
	now := time.Now()
	if w.Changed(now) {
		t.Fatal("changed without edit")
	}
	os.WriteFile(filepath.Join(dir, "colors.toml"), []byte(gruvbox+"cyan = \"#89b482\"\n"), 0o644)
	if w.Changed(now.Add(time.Second)) {
		t.Fatal("polled before interval")
	}
	if !w.Changed(now.Add(3 * time.Second)) {
		t.Fatal("edit not seen")
	}
	if w.Changed(now.Add(6 * time.Second)) {
		t.Fatal("change reported twice")
	}
}

func TestCuoreThemeFirst(t *testing.T) {
	state := t.TempDir()
	for _, d := range []string{"cuore", "omarchy"} {
		dir := filepath.Join(state, d, "current", "theme")
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "colors.toml"), []byte(gruvbox), 0o644)
	}
	t.Setenv("SPRAWL_THEME_DIR", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", state)
	if got := FindDir(); got != filepath.Join(state, "cuore", "current", "theme") {
		t.Errorf("FindDir = %s", got)
	}
}
