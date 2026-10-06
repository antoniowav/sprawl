// Package theme loads the active Omarchy theme palette and maps it to game
// colours. Without Omarchy it falls back to the built-in "Sprawl Dusk".
package theme

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
)

// Palette mirrors the keys of an Omarchy colors.toml.
type Palette struct {
	Name  string
	Light bool

	Accent, Selection, Muted                                        color.RGBA
	Background, DarkBackground, DarkerBackground, LighterBackground color.RGBA
	Foreground, DarkForeground, LightForeground, BrightForeground   color.RGBA

	Red, Yellow, Orange, Green, Cyan, Blue, Magenta, Brown color.RGBA

	BrightRed, BrightYellow, BrightGreen, BrightCyan, BrightBlue, BrightMagenta color.RGBA
}

func (p *Palette) fields() map[string]*color.RGBA {
	return map[string]*color.RGBA{
		"accent": &p.Accent, "selection": &p.Selection, "muted": &p.Muted,
		"background": &p.Background, "dark_background": &p.DarkBackground,
		"darker_background": &p.DarkerBackground, "lighter_background": &p.LighterBackground,
		"foreground": &p.Foreground, "dark_foreground": &p.DarkForeground,
		"light_foreground": &p.LightForeground, "bright_foreground": &p.BrightForeground,
		"red": &p.Red, "yellow": &p.Yellow, "orange": &p.Orange, "green": &p.Green,
		"cyan": &p.Cyan, "blue": &p.Blue, "magenta": &p.Magenta, "brown": &p.Brown,
		"bright_red": &p.BrightRed, "bright_yellow": &p.BrightYellow, "bright_green": &p.BrightGreen,
		"bright_cyan": &p.BrightCyan, "bright_blue": &p.BrightBlue, "bright_magenta": &p.BrightMagenta,
	}
}

// BuiltinName is the name of the fallback palette.
const BuiltinName = "sprawl-dusk"

// Builtin is the fallback palette, used when no Omarchy theme is found and
// to fill keys a theme leaves out.
func Builtin() Palette {
	h := MustHex
	return Palette{
		Name:              BuiltinName,
		Accent:            h("#e0a458"),
		Selection:         h("#3a4152"),
		Muted:             h("#4a505e"),
		Background:        h("#1c1f26"),
		DarkBackground:    h("#16181e"),
		DarkerBackground:  h("#101217"),
		LighterBackground: h("#2a2e38"),
		Foreground:        h("#d8d4c8"),
		DarkForeground:    h("#7d8290"),
		LightForeground:   h("#e6e2d8"),
		BrightForeground:  h("#f2efe6"),
		Red:               h("#d9655b"),
		Yellow:            h("#e3c16f"),
		Orange:            h("#e08e4f"),
		Green:             h("#8fb573"),
		Cyan:              h("#7fbfb3"),
		Blue:              h("#6f9fd8"),
		Magenta:           h("#c48bb8"),
		Brown:             h("#8a6248"),
		BrightRed:         h("#ec8f86"),
		BrightYellow:      h("#f4d996"),
		BrightGreen:       h("#b2d39a"),
		BrightCyan:        h("#a5d9cf"),
		BrightBlue:        h("#9cc0ec"),
		BrightMagenta:     h("#dcb0d2"),
	}
}

// ParseHex accepts #rrggbb, rrggbb, 0xrrggbb and #rrggbbaa.
func ParseHex(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "#"), "0x")
	if len(s) != 6 && len(s) != 8 {
		return color.RGBA{}, fmt.Errorf("bad colour %q", s)
	}
	v, err := strconv.ParseUint(s[:6], 16, 32)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("bad colour %q", s)
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}, nil
}

// MustHex is ParseHex for constants.
func MustHex(s string) color.RGBA {
	c, err := ParseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

// Mix blends a toward b by t (0 = a, 1 = b).
func Mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(math.Round(float64(x)*(1-t) + float64(y)*t)) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 0xff}
}

// Luminance is WCAG relative luminance.
func Luminance(c color.RGBA) float64 {
	ch := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c.R) + 0.7152*ch(c.G) + 0.0722*ch(c.B)
}

// Contrast is the WCAG contrast ratio between two colours (1..21).
func Contrast(a, b color.RGBA) float64 {
	la, lb := Luminance(a), Luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

var (
	black = color.RGBA{0, 0, 0, 0xff}
	white = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

// fillDerived computes keys the source didn't set from the ones it did,
// so a sparse theme still looks like itself rather than like the builtin.
func fillDerived(p *Palette, set map[string]bool) {
	d := func(key string, dst *color.RGBA, v color.RGBA) {
		if !set[key] {
			*dst = v
		}
	}
	shade := black
	if p.Light {
		shade = white
	}
	d("dark_background", &p.DarkBackground, Mix(p.Background, shade, 0.15))
	d("darker_background", &p.DarkerBackground, Mix(p.Background, shade, 0.3))
	d("lighter_background", &p.LighterBackground, Mix(p.Background, p.Foreground, 0.12))
	d("dark_foreground", &p.DarkForeground, Mix(p.Foreground, p.Background, 0.45))
	d("light_foreground", &p.LightForeground, Mix(p.Foreground, p.Background, 0.1))
	d("bright_foreground", &p.BrightForeground, p.Foreground)
	d("muted", &p.Muted, Mix(p.Foreground, p.Background, 0.6))
	d("orange", &p.Orange, Mix(p.Red, p.Yellow, 0.5))
	d("brown", &p.Brown, Mix(p.Orange, black, 0.45))
	d("accent", &p.Accent, p.Blue)
	d("selection", &p.Selection, Mix(p.Accent, p.Background, 0.65))
	d("bright_red", &p.BrightRed, Mix(p.Red, white, 0.25))
	d("bright_yellow", &p.BrightYellow, Mix(p.Yellow, white, 0.25))
	d("bright_green", &p.BrightGreen, Mix(p.Green, white, 0.25))
	d("bright_cyan", &p.BrightCyan, Mix(p.Cyan, white, 0.25))
	d("bright_blue", &p.BrightBlue, Mix(p.Blue, white, 0.25))
	d("bright_magenta", &p.BrightMagenta, Mix(p.Magenta, white, 0.25))
}
