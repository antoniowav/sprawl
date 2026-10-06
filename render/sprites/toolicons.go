package sprites

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/antoniowav/sprawl/theme"
)

// Toolbar icon order.
const (
	IconInspect = iota
	IconRoad
	IconLine
	IconPipe
	IconBulldoze
	IconZoneR
	IconZoneC
	IconZoneI
	IconBuild
	IconOverlay
	IconPause
	IconPlay
	IconTerrain
	IconCount
)

// ToolIcons renders the 16×16 toolbar icons.
func ToolIcons(r theme.Roles) [IconCount]*image.RGBA {
	var ic [IconCount]*image.RGBA
	b := BuildBuildings(r)
	n := BuildNetwork(r)
	cv := BuildCivic(r)
	over := func(base, top *image.RGBA) *image.RGBA {
		out := newTile()
		draw.Draw(out, out.Bounds(), base, image.Point{}, draw.Src)
		draw.Draw(out, out.Bounds(), top, image.Point{}, draw.Over)
		return out
	}
	solid := func(c color.RGBA) *image.RGBA { img := newTile(); fill(img, c); return img }

	ic[IconInspect] = glyphIcon([]string{
		"................",
		"....######......",
		"...#......#.....",
		"..#........#....",
		"..#........#....",
		"..#........#....",
		"..#........#....",
		"...#......#.....",
		"....######.#....",
		"............#...",
		".............#..",
		"..............#.",
	}, r.UIText)
	ic[IconRoad] = n.Road[E|W]
	ic[IconLine] = over(grass(r, 0), n.Line[E|W])
	ic[IconPipe] = over(solid(r.Void), n.Pipe[E|W])
	ic[IconBulldoze] = bulldozer(r)
	for z := 0; z < 3; z++ {
		ic[IconZoneR+z] = over(n.Lot[z], b.Img[z][0][0])
	}
	// The power plant, sampled down from 48 to 16 pixels.
	plant := newTile()
	for y := 0; y < T; y++ {
		for x := 0; x < T; x++ {
			plant.Set(x, y, cv.Img["plant"].At(x*3+1, y*3+1))
		}
	}
	ic[IconBuild] = plant
	ic[IconOverlay] = glyphIcon([]string{
		"................",
		"................",
		".......##.......",
		".....##..##.....",
		"...##......##...",
		"...##......##...",
		".....##..##.....",
		"...#...##...#...",
		"...##......##...",
		".....##..##.....",
		".......##.......",
	}, r.UIAccent)
	ic[IconPause] = glyphIcon([]string{
		"................",
		"................",
		"....###..###....",
		"....###..###....",
		"....###..###....",
		"....###..###....",
		"....###..###....",
		"....###..###....",
		"....###..###....",
		"....###..###....",
	}, r.UIWarn)
	ic[IconPlay] = glyphIcon([]string{
		"................",
		"................",
		".....#..........",
		".....###........",
		".....#####......",
		".....#######....",
		".....#######....",
		".....#####......",
		".....###........",
		".....#..........",
	}, r.UIOk)
	ic[IconTerrain] = mountain(r)
	return ic
}

func glyphIcon(rows []string, c color.RGBA) *image.RGBA {
	img := newTile()
	off := (T - len(rows)) / 2
	for y, row := range rows {
		for x, ch := range row {
			if ch == '#' {
				img.SetRGBA(x, y+off, c)
			}
		}
	}
	return img
}

func bulldozer(r theme.Roles) *image.RGBA {
	img := newTile()
	body := r.ZoneI
	dark := theme.Mix(body, r.Void, 0.5)
	rect(img, 5, 4, 12, 9, body) // body
	rect(img, 9, 2, 12, 5, dark) // cab
	rect(img, 10, 3, 11, 4, r.Window)
	rect(img, 2, 5, 4, 12, theme.Mix(r.Walls[0], r.Void, 0.3)) // blade
	rect(img, 4, 7, 6, 8, dark)                                // arm
	rect(img, 5, 9, 14, 12, r.Void)                            // tracks
	for x := 6; x < 14; x += 2 {
		img.SetRGBA(x, 10, theme.Mix(r.Walls[0], r.Void, 0.5))
	}
	return img
}
