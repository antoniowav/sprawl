package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.toml")
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Zoom != 3 || c.MapSize != 128 {
		t.Fatalf("defaults not applied: %+v", c)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("template not written: %v", err)
	}
	// The template itself must parse back to the defaults.
	again, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.UIScale != c.UIScale || again.TicksPerSecond != c.TicksPerSecond {
		t.Fatalf("template differs from defaults: %+v", again)
	}
}

func TestLoadOverridesAndClamps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("zoom = 9\nanimations = false\n[keys]\nroad = [\"R\"]\n"), 0o644)
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Zoom != 4 {
		t.Errorf("zoom not clamped: %d", c.Zoom)
	}
	if c.Animations {
		t.Error("animations override ignored")
	}
	if c.UIScale != 2 {
		t.Errorf("unset key lost its default: %d", c.UIScale)
	}
	if got := c.Keys["road"]; len(got) != 1 || got[0] != "R" {
		t.Errorf("keys: %v", c.Keys)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	c := Default()
	c.UIScale, c.Volume, c.Animations = 3, 25, false
	c.Keys = map[string][]string{"road": {"R", "Ctrl+r"}}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.UIScale != 3 || got.Volume != 25 || got.Animations || len(got.Keys["road"]) != 2 {
		t.Errorf("round trip: %+v", got)
	}
}

func TestOldDayNightSetting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("day_night = false\n"), 0o644)
	c, _ := Load(path)
	if c.TimeOfDay != "day" {
		t.Errorf("day_night=false should mean always day, got %q", c.TimeOfDay)
	}
}
