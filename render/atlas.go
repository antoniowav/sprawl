// Package render draws the world and the HUD with Ebitengine.
package render

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/antoniowav/sprawl/render/sprites"
	"github.com/antoniowav/sprawl/sim"
	"github.com/antoniowav/sprawl/theme"
)

// Atlas holds GPU images for every sprite. Ebitengine packs small images
// into shared textures itself, so draws of different tiles still batch.
type Atlas struct {
	Grass   [4]*ebiten.Image
	Lush    [2]*ebiten.Image
	Trees   [3]*ebiten.Image
	Water   [16]*ebiten.Image
	Shimmer [4]*ebiten.Image
	Road    [16]*ebiten.Image
	Line    [16]*ebiten.Image
	Pipe    [16]*ebiten.Image
	Lot     [3]*ebiten.Image
	Bridge  [16]*ebiten.Image
	Rock    [3]*ebiten.Image
	Bldg    [3][3][4]*ebiten.Image // [zone][level-1][variant]
	Lights  [3][3][4]*ebiten.Image
	CivicLt map[sim.Kind]*ebiten.Image
	Chimney [4][]image.Point
	Civic   map[sim.Kind]*ebiten.Image
	Stacks  []image.Point // power plant smoke sources
	NoPower *ebiten.Image
	NoWater *ebiten.Image
	Hatch   *ebiten.Image
	Tools   [sprites.IconCount]*ebiten.Image

	all []*ebiten.Image
}

var civicNames = map[sim.Kind]string{
	sim.PowerPlant: "plant", sim.WindTurbine: "wind", sim.WaterPump: "pump", sim.WaterTower: "tower",
	sim.Police: "police", sim.Fire: "fire", sim.School: "school", sim.Park: "park",
	sim.CityHall: "hall", sim.Stadium: "stadium",
}

// NewAtlas builds the sprites for roles and uploads them.
func NewAtlas(r theme.Roles) *Atlas {
	t := sprites.BuildTerrain(r)
	n := sprites.BuildNetwork(r)
	a := &Atlas{}
	a.up(a.Grass[:], t.Grass[:])
	a.up(a.Lush[:], t.Lush[:])
	a.up(a.Trees[:], t.Trees[:])
	a.up(a.Water[:], t.Water[:])
	a.up(a.Shimmer[:], t.Shimmer[:])
	a.up(a.Road[:], n.Road[:])
	a.up(a.Line[:], n.Line[:])
	a.up(a.Pipe[:], n.Pipe[:])
	a.up(a.Lot[:], n.Lot[:])
	a.up(a.Bridge[:], n.Bridge[:])
	a.up(a.Rock[:], t.Rock[:])
	b := sprites.BuildBuildings(r)
	for z := range b.Img {
		for l := range b.Img[z] {
			a.up(a.Bldg[z][l][:], b.Img[z][l][:])
			a.up(a.Lights[z][l][:], b.Lights[z][l][:])
		}
	}
	a.Chimney = b.Chimneys
	cv := sprites.BuildCivic(r)
	a.Civic, a.CivicLt = map[sim.Kind]*ebiten.Image{}, map[sim.Kind]*ebiten.Image{}
	for k, name := range civicNames {
		one := []*ebiten.Image{nil, nil}
		a.up(one, []*image.RGBA{cv.Img[name], cv.Lights[name]})
		a.Civic[k], a.CivicLt[k] = one[0], one[1]
	}
	a.Stacks = cv.PlantStacks
	ic := sprites.BuildIcons(r)
	icons := []*ebiten.Image{nil, nil, nil}
	a.up(icons, []*image.RGBA{ic.NoPower, ic.NoWater, ic.Hatch})
	a.NoPower, a.NoWater, a.Hatch = icons[0], icons[1], icons[2]
	ti := sprites.ToolIcons(r)
	a.up(a.Tools[:], ti[:])
	return a
}

// Dispose frees the GPU images.
func (a *Atlas) Dispose() {
	for _, img := range a.all {
		img.Deallocate()
	}
}

func (a *Atlas) up(dst []*ebiten.Image, src []*image.RGBA) {
	for i, s := range src {
		dst[i] = ebiten.NewImageFromImage(s)
		a.all = append(a.all, dst[i])
	}
}
