package sprites

import (
	"image"
	"image/color"

	"github.com/antoniowav/sprawl/theme"
)

// Civic holds the utility and service buildings by name ("plant", "wind",
// "pump", "tower", "police", "fire", "school", "park", "hall", "stadium").
type Civic struct {
	Img    map[string]*image.RGBA
	Lights map[string]*image.RGBA
	// Smoke sources on the power plant, in sprite pixels.
	PlantStacks []image.Point
}

// Icons are 9×9 status markers drawn over tiles, plus a white diagonal
// hatch tile that overlays tint to mark "not served" without relying on
// hue (some themes make red and green nearly the same).
type Icons struct {
	NoPower, NoWater *image.RGBA
	Hatch            *image.RGBA
}

// BuildCivic renders the utility and service buildings.
func BuildCivic(r theme.Roles) Civic {
	p := newPainter(r)
	c := Civic{Img: map[string]*image.RGBA{}, Lights: map[string]*image.RGBA{}}
	c.Img["plant"], c.PlantStacks = p.plant()
	c.Img["wind"] = p.wind()
	c.Img["pump"] = p.pump()
	c.Img["tower"] = p.tower1()
	c.Img["police"] = p.police()
	c.Img["fire"] = p.fire()
	c.Img["school"] = p.school()
	c.Img["park"] = p.park()
	c.Img["hall"] = p.cityHall()
	c.Img["stadium"] = p.stadium()
	c.Img["busstop"] = p.busStop()
	c.Img["depot"] = p.busDepot()
	c.Img["hospital"] = p.hospital()
	c.Img["solar"] = p.solar()
	c.Img["university"] = p.university()
	c.Img["nuclear"] = p.nuclear()
	c.Img["monument"] = p.monument()
	i := 0
	for name, img := range c.Img {
		c.Lights[name] = p.lights(img, uint64(900+len(name)*31+i))
		i++
	}
	return c
}

func big(n int) *image.RGBA { return image.NewRGBA(image.Rect(0, 0, n*T, n*T)) }

func rectB(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	b := img.Bounds()
	for y := max(0, y0); y < min(b.Dy(), y1); y++ {
		for x := max(0, x0); x < min(b.Dx(), x1); x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func discB(img *image.RGBA, cx, cy, rad float64, col func(dx, dy float64) color.RGBA) {
	b := img.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			if dx*dx+dy*dy <= rad*rad {
				img.SetRGBA(x, y, col(dx, dy))
			}
		}
	}
}

// blockB is block() for images larger than a tile.
func (p *painter) blockB(img *image.RGBA, x, y, w, d, h int, roof, wall color.RGBA) {
	rectB(img, x+1, y+1, x+w+1, y+d+h+1, p.shadow)
	rectB(img, x, y, x+w, y+d, roof)
	rectB(img, x, y, x+w, y+1, light(roof))
	rectB(img, x, y+d, x+w, y+d+h, wall)
	rectB(img, x, y+d, x+w, y+d+1, dark(wall))
}

// pad fills the footprint with paving and a 1 px kerb.
func (p *painter) pad(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	rectB(img, 0, 0, b.Dx(), b.Dy(), p.r.Sidewalk)
	rectB(img, 1, 1, b.Dx()-1, b.Dy()-1, c)
}

func (p *painter) plant() (*image.RGBA, []image.Point) {
	img := big(3)
	p.pad(img, p.pave)
	// Turbine hall with a bolt on the wall.
	p.blockB(img, 3, 22, 24, 10, 9, p.flatD, p.r.Walls[0])
	for x := 5; x < 25; x += 3 {
		img.SetRGBA(x, 34, p.glass)
	}
	bolt := []image.Point{{15, 33}, {14, 34}, {13, 35}, {14, 35}, {15, 35}, {14, 36}, {13, 37}}
	for _, pt := range bolt {
		img.SetRGBA(pt.X, pt.Y, p.r.Power)
	}
	// Cooling tower: a ring seen from above with a shaded throat.
	discB(img, 37.5, 17.5, 9.5, func(dx, dy float64) color.RGBA { return p.shadow })
	discB(img, 36, 16, 9.5, func(dx, dy float64) color.RGBA {
		if dx+dy < -6 {
			return light(p.flat)
		}
		return p.flat
	})
	discB(img, 36, 16, 6.5, func(dx, dy float64) color.RGBA {
		if dx+dy > 2 {
			return theme.Mix(p.flatD, p.r.Void, 0.4)
		}
		return p.flatD
	})
	// Stack with warning bands.
	rectB(img, 29, 5, 32, 24, p.shadow)
	rectB(img, 28, 4, 31, 23, p.flat)
	for y := 6; y < 22; y += 5 {
		rectB(img, 28, y, 31, y+2, p.r.UIErr)
	}
	rectB(img, 28, 4, 31, 5, p.r.Void)
	// Transformer yard.
	for x := 4; x < 22; x += 5 {
		rectB(img, x, 40, x+3, 44, p.flat)
		img.SetRGBA(x+1, 40, p.r.Power)
	}
	return img, []image.Point{{29, 3}, {36, 10}}
}

func (p *painter) pump() *image.RGBA {
	img := big(2)
	p.pad(img, p.pave)
	p.blockB(img, 3, 5, 14, 8, 7, theme.Mix(p.r.Pipe, p.flat, 0.4), p.r.Walls[2])
	rectB(img, 8, 15, 12, 20, p.door)
	// Droplet sign.
	for _, pt := range []image.Point{{14, 14}, {13, 15}, {14, 15}, {15, 15}, {13, 16}, {14, 16}, {15, 16}} {
		img.SetRGBA(pt.X, pt.Y, p.r.Pipe)
	}
	// Tank.
	discB(img, 24.5, 11.5, 5.5, func(dx, dy float64) color.RGBA { return p.shadow })
	discB(img, 24, 11, 5.5, func(dx, dy float64) color.RGBA {
		if dx+dy < -3 {
			return light(p.r.Pipe)
		}
		return theme.Mix(p.r.Pipe, p.flat, 0.5)
	})
	// Intake pipes.
	rectB(img, 5, 24, 27, 26, p.r.PipeDark)
	rectB(img, 5, 24, 27, 25, p.r.Pipe)
	return img
}

func (p *painter) police() *image.RGBA {
	img := big(2)
	p.pad(img, p.pave)
	roof := theme.Mix(p.r.ZoneC, p.flatD, 0.45)
	p.blockB(img, 3, 3, 26, 9, 8, roof, p.r.Walls[0])
	p.windows(img, 5, 28, 15, 2, 2, 2, p.glass)
	rectB(img, 14, 17, 18, 20, p.door)
	// Star on the roof.
	for _, pt := range []image.Point{{16, 5}, {15, 6}, {16, 6}, {17, 6}, {14, 7}, {15, 7}, {16, 7}, {17, 7}, {18, 7}, {15, 8}, {17, 8}} {
		img.SetRGBA(pt.X, pt.Y, p.r.Power)
	}
	// Two patrol cars.
	for _, x := range []int{6, 20} {
		rectB(img, x+1, 24, x+7, 28, p.shadow)
		rectB(img, x, 23, x+6, 27, p.r.Walls[0])
		rectB(img, x+2, 23, x+4, 27, p.r.ZoneC)
		img.SetRGBA(x+3, 23, p.r.UIErr)
	}
	return img
}

func (p *painter) fire() *image.RGBA {
	img := big(2)
	p.pad(img, p.pave)
	roof := theme.Mix(p.r.UIErr, p.flatD, 0.35)
	p.blockB(img, 3, 3, 26, 8, 10, roof, p.r.Walls[1])
	for _, x := range []int{6, 17} {
		rectB(img, x, 15, x+8, 21, p.door)
		for y := 16; y < 21; y += 2 {
			rectB(img, x, y, x+8, y+1, dark(p.door))
		}
	}
	rectB(img, 5, 6, 27, 8, light(roof))
	// Fire engine on the apron.
	rectB(img, 8, 25, 23, 29, p.shadow)
	rectB(img, 7, 24, 22, 28, p.r.UIErr)
	rectB(img, 18, 24, 22, 26, p.glass)
	rectB(img, 8, 25, 17, 26, light(p.r.Walls[0]))
	return img
}

func (p *painter) school() *image.RGBA {
	img := big(2)
	fill(img, p.r.Grass)
	rectB(img, 0, 0, 32, 32, p.r.Grass)
	p.blockB(img, 2, 2, 19, 8, 6, p.r.Roofs[2], p.r.Walls[1])
	p.windows(img, 4, 20, 12, 2, 2, 2, p.glass)
	rectB(img, 10, 14, 12, 16, p.door)
	// Wing.
	p.blockB(img, 2, 17, 8, 7, 5, dark(p.r.Roofs[2]), p.r.Walls[1])
	// Sports field with lines.
	rectB(img, 13, 19, 30, 30, theme.Mix(p.r.Grass, p.r.GrassLight, 0.6))
	line := theme.Mix(p.r.Walls[0], p.r.Grass, 0.2)
	rectB(img, 13, 19, 30, 20, line)
	rectB(img, 13, 29, 30, 30, line)
	rectB(img, 13, 19, 14, 30, line)
	rectB(img, 29, 19, 30, 30, line)
	rectB(img, 21, 19, 22, 30, line)
	// Flagpole.
	rectB(img, 25, 4, 26, 14, p.flatD)
	rectB(img, 26, 4, 29, 6, p.r.UIErr)
	return img
}

func (p *painter) wind() *image.RGBA {
	img := grass(p.r, 2)
	pole := theme.Mix(p.r.Walls[0], p.r.UIBorder, 0.2)
	rect(img, 8, 6, 9, 15, p.shadow)
	rect(img, 7, 6, 8, 15, pole)
	rect(img, 6, 14, 10, 15, p.flatD)
	// Three blades around the hub.
	blade := light(pole)
	for _, pt := range [][2]int{{7, 1}, {7, 2}, {7, 3}, {7, 4}, {8, 5}, {9, 6}, {10, 7}, {11, 8}, {12, 8}, {6, 6}, {5, 7}, {4, 8}, {3, 8}} {
		img.SetRGBA(pt[0], pt[1], blade)
	}
	rect(img, 6, 4, 9, 7, p.flatD)
	img.SetRGBA(7, 5, p.r.Power)
	return img
}

func (p *painter) tower1() *image.RGBA {
	img := grass(p.r, 1)
	legs := p.flatD
	rect(img, 5, 8, 6, 15, legs)
	rect(img, 10, 8, 11, 15, legs)
	rect(img, 7, 9, 9, 15, legs)
	rect(img, 5, 11, 11, 12, legs)
	disc(img, 9, 6.5, 5, func(dx, dy float64) color.RGBA { return p.shadow })
	tank := theme.Mix(p.r.Pipe, p.flat, 0.45)
	disc(img, 8, 5.5, 5, func(dx, dy float64) color.RGBA {
		if dx+dy < -2.5 {
			return light(tank)
		}
		if dx+dy > 3 {
			return dark(tank)
		}
		return tank
	})
	rect(img, 6, 5, 10, 6, p.r.Pipe)
	return img
}

func (p *painter) park() *image.RGBA {
	img := grass(p.r, 3)
	lawn := theme.Mix(p.r.Grass, p.r.GrassLight, 0.5)
	rect(img, 1, 1, 15, 15, lawn)
	// Gravel path in an L, a pond, two trees and a bench.
	path := theme.Mix(p.r.Sidewalk, p.r.Roofs[1], 0.25)
	rect(img, 7, 1, 9, 15, path)
	rect(img, 1, 9, 9, 11, path)
	disc(img, 4, 4.5, 2.6, func(dx, dy float64) color.RGBA {
		if dx+dy < -1.5 {
			return p.r.WaterLight
		}
		return p.r.Water
	})
	for _, c := range [][2]float64{{12.5, 4.5}, {12, 12.5}} {
		disc(img, c[0]+1, c[1]+1.5, 2.6, func(dx, dy float64) color.RGBA { return p.r.Shadow })
		disc(img, c[0], c[1], 2.6, func(dx, dy float64) color.RGBA {
			if dx+dy < -1 {
				return p.r.TreeLight
			}
			return p.r.Tree
		})
	}
	rect(img, 2, 12, 6, 13, p.door)
	return img
}

func (p *painter) cityHall() *image.RGBA {
	img := big(3)
	p.pad(img, p.pave)
	// Lawn in front with a path.
	rectB(img, 3, 33, 45, 46, p.r.Grass)
	rectB(img, 21, 33, 27, 46, p.r.Sidewalk)
	wall := light(p.r.Walls[0])
	p.blockB(img, 4, 8, 40, 12, 12, p.flat, wall)
	// Columns.
	for x := 7; x < 42; x += 4 {
		rectB(img, x, 22, x+2, 31, light(wall))
		rectB(img, x+2, 22, x+3, 31, dark(wall))
	}
	rectB(img, 4, 20, 44, 22, dark(wall))
	rectB(img, 21, 26, 27, 32, p.door)
	// Dome.
	discB(img, 24.5, 9.5, 7, func(dx, dy float64) color.RGBA { return p.shadow })
	discB(img, 24, 9, 7, func(dx, dy float64) color.RGBA {
		if dx+dy < -4 {
			return light(p.r.Roofs[2])
		}
		if dx+dy > 3 {
			return dark(p.r.Roofs[2])
		}
		return p.r.Roofs[2]
	})
	// Flag.
	rectB(img, 24, 0, 25, 4, p.flatD)
	rectB(img, 25, 0, 29, 2, p.r.UIErr)
	return img
}

func (p *painter) stadium() *image.RGBA {
	img := big(3)
	p.pad(img, p.pave)
	stand := theme.Mix(p.r.Walls[1], p.flat, 0.4)
	// Oval stands, then the pitch inside.
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			dx, dy := (float64(x)+0.5-24)/21, (float64(y)+0.5-24)/18
			d := dx*dx + dy*dy
			switch {
			case d <= 0.5:
				img.SetRGBA(x, y, theme.Mix(p.r.Grass, p.r.GrassLight, float64((x/3)%2)*0.5))
			case d <= 1:
				c := stand
				if (x+y)%3 == 0 {
					c = theme.Mix(stand, p.r.ZoneC, 0.3) // crowd
				}
				if dy > 0.3 {
					c = dark(c)
				}
				img.SetRGBA(x, y, c)
			}
		}
	}
	line := light(p.r.Walls[0])
	rectB(img, 23, 12, 25, 36, line)
	discB(img, 24, 24, 3.5, func(dx, dy float64) color.RGBA {
		if dx*dx+dy*dy > 6 {
			return line
		}
		return theme.Mix(p.r.Grass, p.r.GrassLight, 0.25)
	})
	// Floodlights.
	for _, c := range [][2]int{{4, 4}, {43, 4}, {4, 43}, {43, 43}} {
		rectB(img, c[0], c[1], c[0]+2, c[1]+2, p.r.Window)
	}
	return img
}

func (p *painter) busStop() *image.RGBA {
	img := newTile()
	fill(img, p.pave)
	rect(img, 0, 0, T, 1, p.r.Sidewalk)
	// Shelter roof, glass back wall and a sign post.
	rect(img, 3, 4, 13, 6, p.r.ZoneC)
	rect(img, 3, 6, 13, 9, theme.Mix(p.glass, p.pave, 0.3))
	rect(img, 3, 9, 4, 12, p.flatD)
	rect(img, 12, 9, 13, 12, p.flatD)
	rect(img, 6, 10, 10, 11, p.door) // bench
	rect(img, 14, 3, 15, 13, p.flatD)
	rect(img, 13, 2, 16, 5, p.r.Power)
	return img
}

func (p *painter) bus(img *image.RGBA, x, y int, horizontal bool) {
	body := p.r.ZoneC
	if horizontal {
		rectB(img, x+1, y+1, x+13, y+6, p.shadow)
		rectB(img, x, y, x+12, y+5, body)
		rectB(img, x+1, y+1, x+11, y+2, p.glass)
		return
	}
	rectB(img, x+1, y+1, x+6, y+13, p.shadow)
	rectB(img, x, y, x+5, y+12, body)
	rectB(img, x+1, y+1, x+4, y+3, p.glass)
}

func (p *painter) busDepot() *image.RGBA {
	img := big(2)
	p.pad(img, p.pave)
	p.blockB(img, 2, 2, 28, 7, 6, p.flat, p.r.Walls[2])
	for i := 0; i < 3; i++ {
		x := 4 + i*9
		rectB(img, x, 10, x+7, 15, p.door)
	}
	p.bus(img, 3, 19, true)
	p.bus(img, 17, 24, true)
	return img
}

func (p *painter) hospital() *image.RGBA {
	img := big(2)
	p.pad(img, p.pave)
	white := light(p.r.Walls[0])
	p.blockB(img, 2, 2, 22, 9, 10, theme.Mix(white, p.flat, 0.3), white)
	p.windows(img, 4, 23, 14, 3, 2, 2, p.glass)
	// Red cross on the roof.
	rectB(img, 11, 3, 15, 10, p.r.UIErr)
	rectB(img, 9, 5, 17, 8, p.r.UIErr)
	// Helipad.
	discB(img, 27.5, 26.5, 4, func(dx, dy float64) color.RGBA { return p.flatD })
	rectB(img, 26, 24, 27, 29, white)
	rectB(img, 29, 24, 30, 29, white)
	rectB(img, 26, 26, 30, 27, white)
	rectB(img, 12, 21, 16, 24, p.door)
	return img
}

func (p *painter) solar() *image.RGBA {
	img := big(2)
	fill(img, theme.Mix(p.r.Grass, p.r.GrassDark, 0.5))
	panel := theme.Mix(p.r.Water, p.glass, 0.4)
	for y := 2; y < 30; y += 7 {
		for x := 2; x < 30; x += 10 {
			rectB(img, x+1, y+1, x+9, y+6, p.shadow)
			rectB(img, x, y, x+8, y+5, panel)
			rectB(img, x, y, x+8, y+1, light(panel))
			rectB(img, x+4, y, x+5, y+5, dark(panel))
		}
	}
	return img
}

func (p *painter) university() *image.RGBA {
	img := big(3)
	fill(img, p.r.Grass)
	path := theme.Mix(p.r.Sidewalk, p.r.Roofs[1], 0.25)
	rectB(img, 22, 0, 26, 48, path)
	rectB(img, 0, 30, 48, 33, path)
	wall := p.r.Walls[1]
	p.blockB(img, 2, 3, 18, 8, 8, p.r.Roofs[1], wall)
	p.windows(img, 4, 19, 14, 2, 2, 2, p.glass)
	p.blockB(img, 28, 3, 18, 8, 8, p.r.Roofs[1], wall)
	p.windows(img, 30, 45, 14, 2, 2, 2, p.glass)
	p.blockB(img, 4, 36, 14, 6, 5, dark(p.r.Roofs[1]), wall)
	// Clock tower.
	rectB(img, 31, 34, 39, 47, p.shadow)
	rectB(img, 30, 33, 38, 46, light(wall))
	discB(img, 34, 37, 2.5, func(dx, dy float64) color.RGBA { return p.r.Walls[0] })
	img.SetRGBA(34, 36, p.flatD)
	img.SetRGBA(35, 37, p.flatD)
	for _, c := range [][2]float64{{8, 26}, {42, 26}, {16, 44}} {
		disc2 := func(dx, dy float64) color.RGBA { return p.r.Tree }
		discB(img, c[0], c[1], 3, disc2)
	}
	return img
}

func (p *painter) nuclear() *image.RGBA {
	img := big(3)
	p.pad(img, p.pave)
	for _, c := range [][2]float64{{12, 13}, {34, 13}} {
		discB(img, c[0]+1.5, c[1]+1.5, 9.5, func(dx, dy float64) color.RGBA { return p.shadow })
		discB(img, c[0], c[1], 9.5, func(dx, dy float64) color.RGBA {
			if dx+dy < -6 {
				return light(p.flat)
			}
			return p.flat
		})
		discB(img, c[0], c[1], 6, func(dx, dy float64) color.RGBA {
			return theme.Mix(p.r.Walls[0], p.flat, 0.3) // steam
		})
	}
	// Reactor dome and hall.
	p.blockB(img, 4, 29, 24, 8, 8, p.flatD, p.r.Walls[0])
	discB(img, 37.5, 35.5, 7, func(dx, dy float64) color.RGBA { return p.shadow })
	discB(img, 37, 35, 7, func(dx, dy float64) color.RGBA {
		if dx+dy < -4 {
			return light(p.r.Walls[0])
		}
		return p.r.Walls[0]
	})
	rectB(img, 8, 41, 12, 44, p.r.Power)
	return img
}

func (p *painter) monument() *image.RGBA {
	img := big(2)
	plaza := theme.Mix(p.r.Sidewalk, p.r.Walls[1], 0.3)
	fill(img, plaza)
	for i := 0; i < 32; i += 4 {
		rectB(img, i, 0, i+1, 32, dark(plaza))
		rectB(img, 0, i, 32, i+1, dark(plaza))
	}
	// Obelisk on a plinth, with its long shadow.
	rectB(img, 17, 10, 21, 30, p.shadow)
	rectB(img, 10, 22, 22, 28, p.flatD)
	rectB(img, 14, 3, 18, 24, light(p.r.Walls[0]))
	rectB(img, 17, 3, 18, 24, p.r.Walls[0])
	rectB(img, 15, 1, 17, 3, p.r.Power)
	for _, c := range [][2]float64{{5, 6}, {27, 6}, {5, 26}, {27, 26}} {
		discB(img, c[0], c[1], 2.5, func(dx, dy float64) color.RGBA { return p.r.Tree })
	}
	return img
}

// BuildIcons renders the no-power and no-water markers.
func BuildIcons(r theme.Roles) Icons {
	bolt := []string{
		"....##.",
		"...##..",
		"..##...",
		".#####.",
		"...##..",
		"..##...",
		".##....",
	}
	drop := []string{
		"...#...",
		"...#...",
		"..###..",
		".#####.",
		".#####.",
		".#####.",
		"..###..",
	}
	hatch := newTile()
	for y := 0; y < T; y++ {
		for x := 0; x < T; x++ {
			if (x+y)%4 < 2 {
				hatch.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}
	return Icons{NoPower: icon(bolt, r.Power), NoWater: icon(drop, r.Pipe), Hatch: hatch}
}

// icon draws a 7×7 glyph with a 1 px dark outline, 9×9 in total.
func icon(rows []string, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 9, 9))
	outline := color.RGBA{0x10, 0x10, 0x10, 0xff}
	on := func(x, y int) bool {
		return y >= 0 && y < 7 && x >= 0 && x < 7 && rows[y][x] == '#'
	}
	for y := -1; y < 8; y++ {
		for x := -1; x < 8; x++ {
			switch {
			case on(x, y):
				img.SetRGBA(x+1, y+1, c)
			case on(x-1, y) || on(x+1, y) || on(x, y-1) || on(x, y+1):
				img.SetRGBA(x+1, y+1, outline)
			}
		}
	}
	return img
}
