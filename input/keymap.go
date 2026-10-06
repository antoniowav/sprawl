// Package input maps keys to actions. Bindings come from defaults plus the
// [keys] table in config.toml, so every key is rebindable.
package input

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Action is a named thing the player can do.
type Action string

const (
	MoveLeft      Action = "move_left"
	MoveDown      Action = "move_down"
	MoveUp        Action = "move_up"
	MoveRight     Action = "move_right"
	MoveLeftFast  Action = "move_left_fast"
	MoveDownFast  Action = "move_down_fast"
	MoveUpFast    Action = "move_up_fast"
	MoveRightFast Action = "move_right_fast"
	Center        Action = "center"
	Road          Action = "road"
	PowerLine     Action = "power_line"
	WaterPipe     Action = "water_pipe"
	Bulldoze      Action = "bulldoze"
	ZonePrefix    Action = "zone_prefix"
	BuildMenu     Action = "build_menu"
	Apply         Action = "apply"
	Paint         Action = "paint"
	Visual        Action = "visual"
	Overlay       Action = "overlay"
	Underground   Action = "underground"
	ZoomIn        Action = "zoom_in"
	ZoomOut       Action = "zoom_out"
	Pause         Action = "pause"
	Speed1        Action = "speed_1"
	Speed2        Action = "speed_2"
	Speed3        Action = "speed_3"
	TogglePanel   Action = "toggle_panel"
	ToggleLog     Action = "toggle_log"
	Command       Action = "command"
	Help          Action = "help"
	Cancel        Action = "cancel"
	Save          Action = "save"
	SaveAs        Action = "save_as"
	Open          Action = "open"
	NewCity       Action = "new_city"
	Quit          Action = "quit"
	Undo          Action = "undo"
	Redo          Action = "redo"
	Budget        Action = "budget"
	Guide         Action = "guide"
	Stats         Action = "stats"
	TimeOfDay     Action = "time_of_day"
	Minimap       Action = "minimap"
	Photo         Action = "photo"
	Achievements  Action = "achievements"
	Inspect       Action = "inspect"
)

// Def is a default binding with its help text.
type Def struct {
	Action Action
	Keys   []string
	Help   string
	Group  string
}

// Defaults is the default keymap, in help-overlay order.
var Defaults = []Def{
	{MoveLeft, []string{"h", "Left"}, "cursor left", "move"},
	{MoveDown, []string{"j", "Down"}, "cursor down", "move"},
	{MoveUp, []string{"k", "Up"}, "cursor up", "move"},
	{MoveRight, []string{"l", "Right"}, "cursor right", "move"},
	{MoveLeftFast, []string{"H", "Shift+Left"}, "left 8", "move"},
	{MoveDownFast, []string{"J", "Shift+Down"}, "down 8", "move"},
	{MoveUpFast, []string{"K", "Shift+Up"}, "up 8", "move"},
	{MoveRightFast, []string{"L", "Shift+Right"}, "right 8", "move"},
	{Center, []string{"c"}, "centre camera", "move"},
	{ZoomIn, []string{"+", "="}, "zoom in", "view"},
	{ZoomOut, []string{"-"}, "zoom out", "view"},
	{Overlay, []string{"o"}, "cycle overlay", "view"},
	{Underground, []string{"u"}, "underground view", "view"},
	{TogglePanel, []string{"Tab"}, "side panel", "view"},
	{Budget, []string{"Ctrl+b", "F3"}, "budget", "view"},
	{Stats, []string{"Ctrl+g", "F4"}, "statistics", "view"},
	{Guide, []string{"F1"}, "guide", "view"},
	{TimeOfDay, []string{"n"}, "day/night mode", "view"},
	{Minimap, []string{"m"}, "minimap", "view"},
	{Inspect, []string{"i"}, "inspect the tile", "view"},
	{Photo, []string{"F12"}, "photo (PNG)", "view"},
	{Achievements, []string{"F5"}, "achievements", "view"},
	{ToggleLog, []string{"e", "F2"}, "event log", "view"},
	{Road, []string{"r"}, "road", "build"},
	{PowerLine, []string{"p"}, "power line", "build"},
	{WaterPipe, []string{"w"}, "water pipe", "build"},
	{Bulldoze, []string{"d"}, "bulldoze", "build"},
	{ZonePrefix, []string{"z"}, "zone: then r/c/i", "build"},
	{BuildMenu, []string{"b"}, "buildings menu", "build"},
	{Apply, []string{"Enter"}, "apply tool", "build"},
	{Paint, []string{"Shift+Enter"}, "paint mode", "build"},
	{Visual, []string{"v"}, "visual select", "build"},
	{Pause, []string{"Space"}, "pause", "time"},
	{Speed1, []string{"1"}, "speed 1", "time"},
	{Speed2, []string{"2"}, "speed 2", "time"},
	{Speed3, []string{"3"}, "speed 3", "time"},
	{Save, []string{"Ctrl+s"}, "save", "file"},
	{SaveAs, []string{"Ctrl+Shift+s"}, "save as…", "file"},
	{Open, []string{"Ctrl+o"}, "open a save", "file"},
	{NewCity, []string{"Ctrl+n"}, "new city", "file"},
	{Quit, []string{"Ctrl+q"}, "quit", "file"},
	{Undo, []string{"Ctrl+z"}, "undo", "file"},
	{Redo, []string{"Ctrl+Shift+z", "Ctrl+y"}, "redo", "file"},
	{Command, []string{":"}, "command palette", "other"},
	{Help, []string{"?"}, "this help", "other"},
	{Cancel, []string{"Esc"}, "back / clear tool", "other"},
}

// Binding is one physical key: either a typed character (layout-aware,
// shift already applied) or a named key with an optional Shift.
type Binding struct {
	Rune  rune
	Key   ebiten.Key
	Shift bool
	Ctrl  bool
}

var namedKeys = map[string]ebiten.Key{
	"left": ebiten.KeyArrowLeft, "right": ebiten.KeyArrowRight,
	"up": ebiten.KeyArrowUp, "down": ebiten.KeyArrowDown,
	"enter": ebiten.KeyEnter, "esc": ebiten.KeyEscape, "escape": ebiten.KeyEscape,
	"tab": ebiten.KeyTab, "backspace": ebiten.KeyBackspace,
	"pageup": ebiten.KeyPageUp, "pagedown": ebiten.KeyPageDown,
	"home": ebiten.KeyHome, "end": ebiten.KeyEnd,
	"f1": ebiten.KeyF1, "f2": ebiten.KeyF2, "f3": ebiten.KeyF3, "f4": ebiten.KeyF4,
	"f5": ebiten.KeyF5, "f6": ebiten.KeyF6, "f7": ebiten.KeyF7, "f8": ebiten.KeyF8,
	"f9": ebiten.KeyF9, "f10": ebiten.KeyF10, "f11": ebiten.KeyF11, "f12": ebiten.KeyF12,
}

// KeyName is the binding name of a physical key ("a", "7", "Left", "F2"),
// or "" for keys that can't be bound (modifiers).
func KeyName(k ebiten.Key) string {
	switch {
	case k >= ebiten.KeyA && k <= ebiten.KeyZ:
		return string(rune('a' + (k - ebiten.KeyA)))
	case k >= ebiten.KeyDigit0 && k <= ebiten.KeyDigit9:
		return string(rune('0' + (k - ebiten.KeyDigit0)))
	}
	for name, key := range namedKeys {
		if key == k && name != "escape" {
			if len(name) > 1 && name[0] == 'f' && name[1] >= '0' && name[1] <= '9' {
				return "F" + name[1:]
			}
			return strings.ToUpper(name[:1]) + name[1:]
		}
	}
	return ""
}

// ParseKey parses "h", "H", ":", "Space", "Left", "Shift+Enter", "F2",
// "Ctrl+s", "Ctrl+Shift+s".
func ParseKey(s string) (Binding, error) {
	if r := []rune(s); len(r) == 1 {
		return Binding{Rune: r[0]}, nil
	}
	var shift, ctrl bool
	name := strings.ToLower(s)
	for {
		if rest, ok := strings.CutPrefix(name, "shift+"); ok {
			shift, name = true, rest
		} else if rest, ok := strings.CutPrefix(name, "ctrl+"); ok {
			ctrl, name = true, rest
		} else {
			break
		}
	}
	// Ctrl+letter/digit: typed characters don't arrive while Ctrl is held,
	// so these are matched as physical keys.
	if r := []rune(name); len(r) == 1 && ctrl {
		switch {
		case r[0] >= 'a' && r[0] <= 'z':
			return Binding{Key: ebiten.KeyA + ebiten.Key(r[0]-'a'), Shift: shift, Ctrl: true}, nil
		case r[0] >= '0' && r[0] <= '9':
			return Binding{Key: ebiten.KeyDigit0 + ebiten.Key(r[0]-'0'), Shift: shift, Ctrl: true}, nil
		}
		return Binding{}, fmt.Errorf("key %q: Ctrl works with letters and digits", s)
	}
	if name == "space" {
		if shift {
			return Binding{}, fmt.Errorf("key %q: Shift+Space not supported", s)
		}
		return Binding{Rune: ' '}, nil
	}
	k, ok := namedKeys[name]
	if !ok {
		return Binding{}, fmt.Errorf("unknown key %q", s)
	}
	return Binding{Key: k, Shift: shift, Ctrl: ctrl}, nil
}

// Keymap resolves bindings to actions.
type Keymap struct {
	byBinding map[Binding]Action
	named     []Binding // bindings that need key polling, not text input
	Defs      []Def     // effective bindings, for the help overlay
}

// NewKeymap applies overrides (action -> keys) on top of Defaults. Bad
// entries are skipped and returned as errors; the rest still loads.
func NewKeymap(overrides map[string][]string) (*Keymap, []error) {
	var errs []error
	known := map[Action]bool{}
	for _, d := range Defaults {
		known[d.Action] = true
	}
	names := make([]string, 0, len(overrides))
	for a := range overrides {
		names = append(names, a)
	}
	sort.Strings(names)
	for _, a := range names {
		if !known[Action(a)] {
			errs = append(errs, fmt.Errorf("keys: unknown action %q", a))
		}
	}

	km := &Keymap{byBinding: map[Binding]Action{}}
	for _, d := range Defaults {
		if keys, ok := overrides[string(d.Action)]; ok {
			d.Keys = keys
		}
		var kept []string
		for _, k := range d.Keys {
			b, err := ParseKey(k)
			if err != nil {
				errs = append(errs, fmt.Errorf("keys.%s: %w", d.Action, err))
				continue
			}
			if prev, dup := km.byBinding[b]; dup {
				errs = append(errs, fmt.Errorf("keys.%s: %q already bound to %s", d.Action, k, prev))
				continue
			}
			km.byBinding[b] = d.Action
			if b.Rune == 0 {
				km.named = append(km.named, b)
			}
			kept = append(kept, k)
		}
		d.Keys = kept
		km.Defs = append(km.Defs, d)
	}
	return km, errs
}

// Lookup returns the action bound to b.
func (k *Keymap) Lookup(b Binding) (Action, bool) {
	a, ok := k.byBinding[b]
	return a, ok
}

// KeysFor returns the display keys for an action.
func (k *Keymap) KeysFor(a Action) []string {
	for _, d := range k.Defs {
		if d.Action == a {
			return d.Keys
		}
	}
	return nil
}
