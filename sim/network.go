package sim

import "sort"

// Utility capacities and per-tile consumption (SPEC §6.4, §6.5).
const (
	PlantCapacity      = 200
	PumpCapacity       = 150
	unpoweredDecayDays = 30
)

// powerUse is what a tile draws. Buildings draw once, on their anchor tile.
func powerUse(t *Tile, i int) int {
	switch {
	case t.Kind == ZoneR:
		return int(t.Level)
	case t.Kind == ZoneC:
		return 2 * int(t.Level)
	case t.Kind == ZoneI:
		return 3 * int(t.Level)
	case t.IsBuilding() && int(t.Anchor) == i:
		return spec(t.Kind).PowerUse
	}
	return 0
}

func waterUse(t *Tile, i int) int {
	switch {
	case t.Kind == ZoneR:
		return 2 * int(t.Level)
	case t.Kind == ZoneC:
		return int(t.Level)
	case t.Kind == ZoneI:
		return 2 * int(t.Level)
	case t.IsBuilding() && int(t.Anchor) == i:
		return spec(t.Kind).WaterUse
	}
	return 0
}

// Utility is one network's totals for the HUD.
type Utility struct {
	Capacity int // sum of connected source capacity
	Demand   int // sum of what connected consumers want
	Served   int // what they got
}

var dirs4 = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

// updateUtilities recomputes Powered and Watered for every tile. Power goes
// first because pumps need it.
func (c *City) updateUtilities() {
	c.updatePower()
	c.updateWater()
}

// conductsPower: roads carry power as well as lines, so a block is powered
// as soon as it touches a road connected to a plant. Lines reach the rest.
func (c *City) conductsPower(t *Tile) bool {
	return t.Line || t.Kind == Road || t.IsZone() || t.IsBuilding()
}

// updatePower floods out from the plants of each connected network and
// serves consumers nearest-first until capacity runs out.
func (c *City) updatePower() {
	n := len(c.Tiles)
	for i := range c.Tiles {
		c.Tiles[i].Powered = false
	}
	comp := make([]int32, n) // 0 = unvisited
	var total Utility
	short := false
	var id int32
	for i := range c.Tiles {
		t := &c.Tiles[i]
		if comp[i] != 0 || !t.IsBuilding() || int(t.Anchor) != i || spec(t.Kind).PowerCap == 0 {
			continue
		}
		// Collect this network's plants: flood the component first.
		id++
		members := c.flood(i, comp, id, c.conductsPower)
		var sources []int
		supply := 0
		for _, m := range members {
			if mt := &c.Tiles[m]; mt.IsBuilding() && spec(mt.Kind).PowerCap > 0 {
				sources = append(sources, m)
				if int(mt.Anchor) == m {
					supply += spec(mt.Kind).PowerCap
				}
			}
		}
		total.Capacity += supply
		// Nearest-first: BFS from all plant tiles over this component.
		for _, m := range c.bfsOrder(sources, func(j int) bool { return comp[j] == id }) {
			use := powerUse(&c.Tiles[m], m)
			total.Demand += use
			if use <= supply {
				supply -= use
				total.Served += use
				c.Tiles[m].Powered = true
			} else {
				short = true
			}
		}
	}
	// Buildings and zones with no draw of their own share their anchor's state;
	// empty lots on a live network count as powered.
	for i := range c.Tiles {
		t := &c.Tiles[i]
		if t.IsBuilding() && int(t.Anchor) != i {
			t.Powered = c.Tiles[t.Anchor].Powered
		}
		if t.IsBuilding() && (spec(t.Kind).PowerCap > 0 || spec(t.Kind).PowerUse == 0) {
			t.Powered = true // sources, and buildings that need no power
		}
		if (t.IsZone() && t.Level == 0 || t.Kind == Road || t.Line) && comp[i] != 0 {
			t.Powered = true
		}
		if t.IsZone() && t.Level > 0 && !t.Powered {
			t.UnpoweredDays++
		} else {
			t.UnpoweredDays = 0
		}
	}
	c.Power = total
	c.edgeEvent(&c.brownout, short, "brownout: power demand exceeds plant capacity", "power restored")
}

// updateWater floods pipes from powered pumps. A consumer is served by a
// pipe on its own tile or a 4-neighbour.
func (c *City) updateWater() {
	n := len(c.Tiles)
	for i := range c.Tiles {
		c.Tiles[i].Watered = false
	}
	isSource := func(t *Tile) bool { return t.IsBuilding() && spec(t.Kind).WaterCap > 0 }
	conducts := func(t *Tile) bool { return t.Pipe || isSource(t) }
	comp := make([]int32, n)
	dist := make([]int32, n)
	var total Utility
	short := false
	var id int32
	type consumer struct{ i, d int }
	for i := range c.Tiles {
		t := &c.Tiles[i]
		if comp[i] != 0 || !isSource(t) || int(t.Anchor) != i {
			continue
		}
		id++
		members := c.flood(i, comp, id, conducts)
		supply := 0
		var sources []int
		for _, m := range members {
			if mt := &c.Tiles[m]; isSource(mt) {
				sources = append(sources, m)
				if int(mt.Anchor) == m && mt.Powered {
					supply += spec(mt.Kind).WaterCap
				}
			}
		}
		total.Capacity += supply
		order := c.bfsOrder(sources, func(j int) bool { return comp[j] == id })
		for d, m := range order {
			dist[m] = int32(d)
		}
		// Consumers: tiles on or next to this network, nearest pipe first.
		best := map[int]int{}
		for _, m := range order {
			mx, my := m%c.W, m/c.W
			for _, dd := range [5][2]int{{0, 0}, {0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
				x, y := mx+dd[0], my+dd[1]
				if !c.In(x, y) {
					continue
				}
				j := y*c.W + x
				if waterUse(&c.Tiles[j], j) == 0 && !c.Tiles[j].IsZone() && !c.Tiles[j].IsBuilding() {
					continue
				}
				if d, ok := best[j]; !ok || int(dist[m]) < d {
					best[j] = int(dist[m])
				}
			}
		}
		cons := make([]consumer, 0, len(best))
		for j, d := range best {
			cons = append(cons, consumer{j, d})
		}
		sort.Slice(cons, func(a, b int) bool {
			if cons[a].d != cons[b].d {
				return cons[a].d < cons[b].d
			}
			return cons[a].i < cons[b].i
		})
		for _, cn := range cons {
			t := &c.Tiles[cn.i]
			if t.Watered {
				continue // already served by another network
			}
			use := waterUse(t, cn.i)
			total.Demand += use
			if use <= supply {
				supply -= use
				total.Served += use
				t.Watered = true
			} else {
				short = true
			}
		}
	}
	for i := range c.Tiles {
		t := &c.Tiles[i]
		if t.IsBuilding() && int(t.Anchor) != i {
			t.Watered = c.Tiles[t.Anchor].Watered
		}
	}
	c.Water = total
	c.edgeEvent(&c.waterShort, short, "water shortage: demand exceeds pump capacity", "water supply restored")
}

// flood marks the 4-connected component of start (tiles where ok holds)
// with id and returns its members.
func (c *City) flood(start int, comp []int32, id int32, ok func(*Tile) bool) []int {
	members := []int{start}
	comp[start] = id
	for k := 0; k < len(members); k++ {
		x, y := members[k]%c.W, members[k]/c.W
		for _, d := range dirs4 {
			nx, ny := x+d[0], y+d[1]
			if !c.In(nx, ny) {
				continue
			}
			j := ny*c.W + nx
			if comp[j] == 0 && ok(&c.Tiles[j]) {
				comp[j] = id
				members = append(members, j)
			}
		}
	}
	return members
}

// bfsOrder returns tiles where in(j) holds, in breadth-first order from
// the sources.
func (c *City) bfsOrder(sources []int, in func(int) bool) []int {
	seen := make(map[int]bool, len(sources))
	order := append([]int(nil), sources...)
	for _, s := range sources {
		seen[s] = true
	}
	for k := 0; k < len(order); k++ {
		x, y := order[k]%c.W, order[k]/c.W
		for _, d := range dirs4 {
			nx, ny := x+d[0], y+d[1]
			if !c.In(nx, ny) {
				continue
			}
			j := ny*c.W + nx
			if !seen[j] && in(j) {
				seen[j] = true
				order = append(order, j)
			}
		}
	}
	return order
}

// edgeEvent logs once when a condition starts and once when it ends.
func (c *City) edgeEvent(state *bool, now bool, start, end string) {
	switch {
	case now && !*state:
		c.Logf(Warn, "%s", start)
	case !now && *state:
		c.Logf(Info, "%s", end)
	}
	*state = now
}
