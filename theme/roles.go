package theme

import "image/color"

// Roles are the colours the game actually draws with, derived from a Palette.
type Roles struct {
	Name  string
	Light bool

	// HUD
	UIBg, UIPanel, UIBorder, UIText, UIDim, UIAccent, UIAccentText, UISel color.RGBA
	UIOk, UIWarn, UIErr, UITrack                                          color.RGBA

	// Terrain
	Void                         color.RGBA
	Grass, GrassDark, GrassLight color.RGBA
	Tree, TreeDark, TreeLight    color.RGBA
	Shadow                       color.RGBA
	Water, WaterDeep, WaterLight color.RGBA
	Foam                         color.RGBA
	Rock, RockDark, RockLight    color.RGBA

	// Built things
	Road, RoadEdge, RoadMark color.RGBA
	Sidewalk, Pole, PipeDark color.RGBA
	ZoneR, ZoneC, ZoneI      color.RGBA
	Power, Pipe, Window      color.RGBA
	Roofs                    [4]color.RGBA
	Walls                    [3]color.RGBA
}

// Derive maps a palette to game roles.
func Derive(p Palette) Roles {
	r := Roles{Name: p.Name, Light: p.Light}

	r.UIBg = p.DarkBackground
	r.UIPanel = p.Background
	r.UIBorder = p.Muted
	r.UIText = readable(p, p.Foreground, p.Background)
	r.UIDim = p.DarkForeground
	if Contrast(r.UIDim, r.UIPanel) < 2.5 {
		r.UIDim = Mix(r.UIText, r.UIPanel, 0.35)
	}
	r.UIAccent = p.Accent
	r.UIAccentText = readable(p, p.Background, p.Accent)
	r.UISel = p.Selection
	r.UIOk, r.UIWarn, r.UIErr = p.Green, p.Orange, p.Red
	r.UITrack = Mix(p.Background, p.Muted, 0.5)

	// Terrain is pulled toward the background so the map sits calmly under
	// the HUD instead of being neon. Light themes pull toward the foreground
	// instead, or everything washes out.
	base := p.Background
	deep := p.DarkerBackground
	if p.Light {
		base = Mix(p.Foreground, p.Background, 0.35)
		deep = p.Foreground
	}
	r.Void = p.DarkerBackground
	r.Grass = Mix(p.Green, base, 0.55)
	r.GrassDark = Mix(p.Green, base, 0.63)
	r.GrassLight = Mix(p.Green, base, 0.42)
	leaf := Mix(p.Green, p.Cyan, 0.25)
	r.Tree = Mix(leaf, deep, 0.58)
	r.TreeDark = Mix(leaf, deep, 0.72)
	r.TreeLight = Mix(leaf, deep, 0.42)
	r.Shadow = Mix(r.Grass, deep, 0.35)
	r.Water = Mix(p.Blue, deep, 0.5)
	r.WaterDeep = Mix(p.Blue, deep, 0.6)
	r.WaterLight = Mix(p.Blue, deep, 0.3)
	r.Foam = Mix(p.BrightBlue, p.Foreground, 0.45)
	stone := Mix(p.Muted, p.Brown, 0.25)
	r.Rock = Mix(stone, base, 0.25)
	r.RockDark = Mix(stone, deep, 0.45)
	r.RockLight = Mix(stone, p.Foreground, 0.25)

	r.Road = Mix(p.LighterBackground, p.Muted, 0.35)
	r.RoadEdge = Mix(r.Road, deep, 0.35)
	r.RoadMark = Mix(p.Yellow, r.Road, 0.35)
	r.Sidewalk = Mix(r.Road, p.Foreground, 0.22)
	r.Pole = Mix(p.Brown, deep, 0.3)
	r.PipeDark = Mix(p.Cyan, deep, 0.5)
	r.ZoneR, r.ZoneC, r.ZoneI = p.Green, p.Blue, p.Yellow
	r.Power, r.Pipe, r.Window = p.Yellow, p.Cyan, p.BrightYellow
	r.Roofs = [4]color.RGBA{
		Mix(p.Red, p.Muted, 0.35), Mix(p.Brown, p.Muted, 0.2),
		Mix(p.Orange, p.Muted, 0.4), Mix(p.Magenta, p.Muted, 0.45),
	}
	r.Walls = [3]color.RGBA{
		Mix(p.Foreground, p.Muted, 0.35), Mix(p.LightForeground, p.Brown, 0.3), Mix(p.Foreground, p.Blue, 0.3),
	}
	return r
}

// readable returns fg if it reads on bg (4.5:1), else the best alternative.
func readable(p Palette, fg, bg color.RGBA) color.RGBA {
	if Contrast(fg, bg) >= 4.5 {
		return fg
	}
	best := fg
	for _, c := range []color.RGBA{p.BrightForeground, p.Foreground, p.Background, p.DarkerBackground, black, white} {
		if Contrast(c, bg) > Contrast(best, bg) {
			best = c
		}
	}
	return best
}
