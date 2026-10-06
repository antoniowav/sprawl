// Package config loads config.toml, creating a commented default on first run.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is the user configuration.
type Config struct {
	UIScale        int                 `toml:"ui_scale"`
	Zoom           int                 `toml:"zoom"`
	TicksPerSecond int                 `toml:"ticks_per_second"`
	PauseUnfocused bool                `toml:"pause_unfocused"`
	LowPower       bool                `toml:"low_power"`
	Animations     bool                `toml:"animations"`
	DayNight       bool                `toml:"day_night"`
	WatchTheme     bool                `toml:"watch_theme"`
	AutosaveMonths int                 `toml:"autosave_months"`
	MapSize        int                 `toml:"map_size"`
	Volume         int                 `toml:"volume"`
	Keys           map[string][]string `toml:"keys"`
}

// Default returns the built-in defaults.
func Default() Config {
	return Config{
		UIScale:        2,
		Zoom:           3,
		TicksPerSecond: 4,
		PauseUnfocused: true,
		LowPower:       true,
		Animations:     true,
		DayNight:       true,
		WatchTheme:     true,
		AutosaveMonths: 6,
		MapSize:        128,
		Volume:         60,
	}
}

const template = `# Sprawl configuration. Delete a line to use the default.

ui_scale = 2            # HUD pixel scale (1..4)
zoom = 3                # starting world zoom (1..4)
ticks_per_second = 4    # simulation rate at speed 1
pause_unfocused = true  # pause when the window loses focus
low_power = true        # only render when something changes (false = plain vsync loop)
animations = true       # water shimmer, smoke, grow animation
day_night = true
watch_theme = true      # follow Omarchy theme changes live
autosave_months = 6     # 0 = off
map_size = 128
volume = 60             # sound effects, 0..100 (0 = off)

# Key overrides: action = ["key", ...]. See ? in game for action names.
[keys]
# road = ["r"]
# move_left = ["h", "Left"]
`

// Load reads path. A missing file is created from the template and the
// defaults are returned.
func Load(path string) (Config, error) {
	c := Default()
	_, err := toml.DecodeFile(path, &c)
	if errors.Is(err, fs.ErrNotExist) {
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr == nil {
			_ = os.WriteFile(path, []byte(template), 0o644)
		}
		return c, nil
	}
	if err != nil {
		return Default(), err
	}
	c.clamp()
	return c, nil
}

func (c *Config) clamp() {
	c.UIScale = clampInt(c.UIScale, 1, 4)
	c.Zoom = clampInt(c.Zoom, 1, 4)
	c.TicksPerSecond = clampInt(c.TicksPerSecond, 1, 60)
	c.AutosaveMonths = clampInt(c.AutosaveMonths, 0, 120)
	c.MapSize = clampInt(c.MapSize, 32, 512)
	c.Volume = clampInt(c.Volume, 0, 100)
}

// Save writes c back to path in the same commented layout as the default
// file. Key overrides are kept.
func Save(path string, c Config) error {
	var b strings.Builder
	fmt.Fprintf(&b, `# Sprawl configuration (also editable from Settings in the game).

ui_scale = %d            # HUD pixel scale (1..4)
zoom = %d                # starting world zoom (1..4)
ticks_per_second = %d    # simulation rate at speed 1
pause_unfocused = %t  # pause when the window loses focus
low_power = %t        # only render when something changes
animations = %t       # water shimmer, smoke, grow animation
day_night = %t
watch_theme = %t      # follow Omarchy theme changes live
autosave_months = %d     # 0 = off
map_size = %d
volume = %d             # sound effects, 0..100 (0 = off)

# Key overrides: action = ["key", ...]. See ? in game for action names.
[keys]
`, c.UIScale, c.Zoom, c.TicksPerSecond, c.PauseUnfocused, c.LowPower, c.Animations, c.DayNight,
		c.WatchTheme, c.AutosaveMonths, c.MapSize, c.Volume)
	names := make([]string, 0, len(c.Keys))
	for k := range c.Keys {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		q := make([]string, len(c.Keys[k]))
		for i, v := range c.Keys[k] {
			q[i] = strconv.Quote(v)
		}
		fmt.Fprintf(&b, "%s = [%s]\n", k, strings.Join(q, ", "))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func clampInt(v, lo, hi int) int {
	return max(lo, min(hi, v))
}
