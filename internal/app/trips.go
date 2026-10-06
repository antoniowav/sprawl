package app

import (
	"math"
	"math/rand/v2"

	"github.com/antoniowav/sprawl/render"
	"github.com/antoniowav/sprawl/sim"
)

// Cars are sampled real trips. One car stands for about carSample (5)
// commuters: it leaves a home in the morning rush, follows the routed
// road path through the junctions to work, parks, and drives back home in
// the evening. A few shopping trips go out at midday. Cars are only for
// show (the simulation already counts every commuter in aggregate), so
// they live here, not in the saved city.

const (
	carSample  = 5
	maxCars    = 500
	carSpeed   = 2.5 // tiles per second of sim time on a free road
	morningOn  = 0.25
	morningOff = 0.38
	middayOn   = 0.46
	middayOff  = 0.58
	eveningOn  = 0.68
	eveningOff = 0.80
)

type car struct {
	path  []int
	i     int // segment: from path[i] to path[i+1]
	frac  float64
	color uint8
	work  bool // driving to work (parks there) rather than home
	shop  bool // a midday errand: comes straight back
}

type parked struct {
	path   []int // home → work, driven backwards to go home
	color  uint8
	leave  float64 // day phase when it heads home
	errand bool
}

type traffic struct {
	cars     []car
	parked   []parked
	clock    float64 // seconds; a day is dayLength
	spawnAcc float64
	homes    []int // developed homes, refreshed now and then
	refresh  float64
	rng      *rand.Rand
	city     *sim.City
}

func newTraffic(c *sim.City) *traffic {
	return &traffic{city: c, rng: rand.New(rand.NewPCG(uint64(c.Seed), 99)), clock: dayLength * 0.3}
}

// phase is the time of day, 0 midnight .. 0.5 noon.
func (t *traffic) phase() float64 { return math.Mod(t.clock/dayLength, 1) }

func in(p, a, b float64) bool { return p >= a && p < b }

// step advances every car and starts new trips; dt is sim seconds.
func (t *traffic) step(dt float64) {
	c := t.city
	t.clock += dt
	p := t.phase()

	t.refresh -= dt
	if t.refresh <= 0 {
		t.refresh = 5
		t.homes = t.homes[:0]
		for i := range c.Tiles {
			if tl := &c.Tiles[i]; tl.Kind == sim.ZoneR && tl.Level > 0 && tl.Commute >= 0 {
				t.homes = append(t.homes, i)
			}
		}
	}

	// How many trips start now: a day's commuters spread over the rush.
	commuters := c.Stats.Workforce()
	perDay := math.Min(maxCars, commuters/carSample)
	rate := 0.0
	switch {
	case in(p, morningOn, morningOff):
		rate = perDay / ((morningOff - morningOn) * dayLength)
	case in(p, middayOn, middayOff):
		rate = 0.15 * perDay / ((middayOff - middayOn) * dayLength)
	}
	t.spawnAcc += rate * dt
	for t.spawnAcc >= 1 && len(t.homes) > 0 && len(t.cars) < maxCars {
		t.spawnAcc--
		home := t.homes[t.rng.IntN(len(t.homes))]
		route := c.TripRoute(home)
		if len(route) < 2 {
			continue
		}
		shop := in(p, middayOn, middayOff)
		t.cars = append(t.cars, car{path: route, color: uint8(t.rng.IntN(5)), work: true, shop: shop})
	}
	t.spawnAcc = math.Min(t.spawnAcc, 5)

	// Parked cars head home when their time comes.
	kept := t.parked[:0]
	for _, pk := range t.parked {
		if p >= pk.leave && (pk.errand || p < morningOn || p >= eveningOn) && len(t.cars) < maxCars {
			back := make([]int, len(pk.path))
			for i, v := range pk.path {
				back[len(back)-1-i] = v
			}
			t.cars = append(t.cars, car{path: back, color: pk.color})
			continue
		}
		kept = append(kept, pk)
	}
	t.parked = kept

	// Drive.
	live := t.cars[:0]
	for _, cr := range t.cars {
		at := cr.path[cr.i]
		if c.Tiles[at].Kind != sim.Road {
			continue // the road was bulldozed under it
		}
		cong := c.Tiles[at].Congestion()
		cr.frac += dt * carSpeed / (1 + 2*math.Max(0, cong-0.5))
		for cr.frac >= 1 && cr.i < len(cr.path)-1 {
			cr.frac--
			cr.i++
		}
		if cr.i >= len(cr.path)-1 {
			if cr.work { // arrived: park until the evening (or a short errand)
				leave := eveningOn + t.rng.Float64()*(eveningOff-eveningOn)
				if cr.shop {
					leave = math.Min(0.99, p+0.01+t.rng.Float64()*0.03)
				}
				t.parked = append(t.parked, parked{path: cr.path, color: cr.color, leave: leave, errand: cr.shop})
			}
			continue
		}
		live = append(live, cr)
	}
	t.cars = live
}

// view returns cars inside the tile rectangle, in world pixels.
func (t *traffic) view(x0, y0, x1, y1 int) []render.Car {
	c := t.city
	var out []render.Car
	ts := float64(render.TileSize)
	for _, cr := range t.cars {
		a, b := cr.path[cr.i], cr.path[cr.i+1]
		ax, ay, bx, by := a%c.W, a/c.W, b%c.W, b/c.W
		if max(ax, bx) < x0-1 || min(ax, bx) > x1+1 || max(ay, by) < y0-1 || min(ay, by) > y1+1 {
			continue
		}
		dx, dy := float64(bx-ax), float64(by-ay)
		// Tile centre to tile centre, keeping to the right-hand lane.
		x := (float64(ax)+0.5+dx*cr.frac)*ts - dy*2.5
		y := (float64(ay)+0.5+dy*cr.frac)*ts + dx*2.5
		out = append(out, render.Car{X: x, Y: y, Horizontal: dx != 0, Color: int(cr.color)})
	}
	return out
}

// stepTrips advances the car trips for the current city.
func (a *App) stepTrips(dt float64) {
	if !a.cfg.Animations {
		a.trips = nil
		return
	}
	if a.trips == nil || a.trips.city != a.city {
		a.trips = newTraffic(a.city)
	}
	a.trips.step(dt)
}

func (a *App) tripView() []render.Car {
	if a.trips == nil || a.trips.city != a.city {
		return nil
	}
	v := a.viewTiles()
	return a.trips.view(v.Min.X, v.Min.Y, v.Max.X, v.Max.Y)
}
