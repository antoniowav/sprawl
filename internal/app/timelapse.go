package app

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/antoniowav/sprawl/render"
	"github.com/antoniowav/sprawl/sim"
)

// The title screen's backdrop: a town that builds itself. A plan of build
// steps is laid out around the map's start point and played back a step
// at a time while the simulation runs fast, so roads get drawn, lots get
// zoned and buildings rise behind the menu. After a few minutes it starts
// over on a new map.

type buildStep struct {
	tool sim.Tool
	a, b sim.Pt
}

type timelapse struct {
	steps   []buildStep
	next    int
	stepAcc float64
	tickAcc float64
	age     float64
	orbit   float64
}

const (
	timelapseStep    = 0.35  // seconds between build steps
	timelapseTicks   = 24.0  // sim ticks per second (fast forward)
	timelapseRestart = 240.0 // seconds before a new town
	timelapseDay     = 4.0   // day/night runs this much faster than in game
)

// newTimelapse creates the backdrop city and its build plan.
func (a *App) newTimelapse() {
	c := sim.NewMap(96, 96, rand.Int64N(1_000_000), sim.MapTypes[rand.IntN(3)])
	c.Funds = 1e9
	c.PeakPop = 1 << 30 // everything unlocked
	a.city = c
	a.tl = &timelapse{steps: planTown(c), orbit: rand.Float64() * 2 * math.Pi}
	a.grows = map[sim.Pt]time.Time{}
	a.chunks.Reset()
	a.miniDirty = true
	a.cam.Zoom = 2
	a.dayClock = dayLength * 0.3
}

// planTown lays out a small grid town around the start point: streets,
// a plant, zones (shops in the middle, industry on one side), water and a
// few services. Steps that don't fit the terrain simply fail when played.
func planTown(c *sim.City) []buildStep {
	s := c.Start
	p := func(dx, dy int) sim.Pt { return sim.Pt{X: s.X + dx, Y: s.Y + dy} }
	var st []buildStep
	road := func(a, b sim.Pt) { st = append(st, buildStep{sim.ToolRoad, a, b}) }
	add := func(t sim.Tool, a, b sim.Pt) { st = append(st, buildStep{t, a, b}) }

	// Main street in short segments, so it visibly grows.
	for x := -15; x < 15; x += 5 {
		road(p(x, 0), p(x+5, 0))
	}
	add(sim.ToolPlant, p(-18, -2), p(-18, -2))
	for _, x := range []int{-12, -6, 0, 6, 12} {
		road(p(x, -8), p(x, 0))
		road(p(x, 0), p(x, 8))
	}
	road(p(-12, -8), p(12, -8))
	road(p(-12, 8), p(12, 8))

	// Zones along the streets, block by block.
	for _, bx := range []int{-11, -5, 1, 7} {
		kind := sim.ToolZoneR
		switch {
		case bx == -5 || bx == 1:
			kind = sim.ToolZoneC
		case bx == 7:
			kind = sim.ToolZoneI
		}
		add(kind, p(bx, 1), p(bx+4, 1))
		add(sim.ToolZoneR, p(bx, -1), p(bx+4, -1))
		add(sim.ToolZoneR, p(bx, -7), p(bx+4, -7))
		add(sim.ToolZoneR, p(bx, 7), p(bx+4, 7))
	}
	// Water: towers and pipes under the streets.
	add(sim.ToolTower, p(-2, -3), p(-2, -3))
	add(sim.ToolTower, p(4, 3), p(4, 3))
	add(sim.ToolPipe, p(-12, 0), p(12, 0))
	for _, x := range []int{-6, 0, 6} {
		add(sim.ToolPipe, p(x, -8), p(x, 8))
	}
	add(sim.ToolPipe, p(-12, -8), p(12, -8))
	add(sim.ToolPipe, p(-12, 8), p(12, 8))
	// Services and green.
	add(sim.ToolSchool, p(-10, -5), p(-10, -5))
	add(sim.ToolPolice, p(2, -5), p(2, -5))
	add(sim.ToolPark, p(-3, -4), p(-3, -4))
	add(sim.ToolPark, p(9, 4), p(9, 4))
	add(sim.ToolHospital, p(-10, 3), p(-10, 3))
	add(sim.ToolBusDepot, p(8, -5), p(8, -5))
	add(sim.ToolBusStop, p(-3, 1), p(-3, 1))
	add(sim.ToolBusStop, p(3, -1), p(3, -1))
	return st
}

func (a *App) timelapseApply(t sim.Tool, pts []sim.Pt) bool {
	p, _ := a.city.ApplyEdit(t, pts, false)
	if p.Err != "" {
		return false
	}
	for _, pt := range p.Tiles {
		a.chunks.Touch(pt.X, pt.Y)
	}
	a.miniDirty = true
	return true
}

// spiral lists points around c, nearest first, out to radius r.
func spiral(c sim.Pt, r int) []sim.Pt {
	pts := []sim.Pt{c}
	for d := 1; d <= r; d++ {
		for dy := -d; dy <= d; dy++ {
			for dx := -d; dx <= d; dx++ {
				if max(dx, -dx, dy, -dy) == d {
					pts = append(pts, sim.Pt{X: c.X + dx, Y: c.Y + dy})
				}
			}
		}
	}
	return pts
}

// updateTimelapse advances the backdrop: build steps, fast simulation,
// quick day and night, and a slow orbit of the camera over the town.
func (a *App) updateTimelapse(dt float64) {
	tl := a.tl
	if tl == nil || tl.age > timelapseRestart {
		a.newTimelapse()
		tl = a.tl
	}
	tl.age += dt
	c := a.city
	now := time.Now()

	tl.stepAcc += dt
	for tl.stepAcc >= timelapseStep && tl.next < len(tl.steps) {
		tl.stepAcc -= timelapseStep
		s := tl.steps[tl.next]
		tl.next++
		if _, ok := s.tool.Building(); ok {
			// Buildings try spots spiralling out from the plan until one fits.
			for _, at := range spiral(s.a, 5) {
				if a.timelapseApply(s.tool, []sim.Pt{at}) {
					break
				}
			}
			continue
		}
		a.timelapseApply(s.tool, sim.Selection(s.tool, s.a, s.b, false))
	}

	tl.tickAcc += dt * timelapseTicks
	for n := 0; tl.tickAcc >= 1 && n < 8; n++ {
		tl.tickAcc--
		for _, p := range c.Tick() {
			a.chunks.Touch(p.X, p.Y)
			if a.cfg.Animations {
				a.grows[p] = now
			}
		}
	}
	if a.cfg.TimeOfDay == "cycle" {
		a.dayClock += dt * timelapseDay
	}
	a.stepSmoke(dt)
	a.carClock += dt
	a.stepTrips(dt * timelapseDay)

	// Orbit the town slowly.
	tl.orbit += dt * 0.04
	r := 4.0 * render.TileSize
	cx := float64(c.Start.X*render.TileSize) + math.Cos(tl.orbit)*r
	cy := float64(c.Start.Y*render.TileSize) + math.Sin(tl.orbit)*r*0.6 - 2*render.TileSize
	a.cam.Jump(cx, cy)
	a.dirty = true
}
