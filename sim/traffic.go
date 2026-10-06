package sim

// Traffic (SPEC §6.9). Every few days each developed home sends its
// commuters along the roads to the nearest jobs: a breadth-first search
// from all roads that touch a shop or factory gives every road tile its
// distance to work, and each home walks that distance field downhill,
// adding its commuters to every road tile on the way. No agents, just
// aggregate flow per tile.

const (
	roadCapacity    = 150 // commuters a road tile carries before it's congested
	transitShare    = 0.3 // share of commuters a nearby bus stop takes
	busStopRadius   = 6
	noiseMax        = 0.15 // land value lost next to a gridlocked road
	commuteFar      = 40   // road tiles; longer commutes slow growth
	noJobsPenalty   = 0.3  // growth score for homes with no road to any job
	farCommutePenal = 0.15
)

// Congestion is a road tile's load over its capacity (1 = full).
func (t *Tile) Congestion() float64 { return float64(t.Traffic) / roadCapacity }

func (c *City) updateTraffic() {
	n := len(c.Tiles)
	dist := make([]int32, n)
	for i := range dist {
		dist[i] = -1
	}
	isRoad := func(i int) bool { return c.Tiles[i].Kind == Road }
	jobs := func(t *Tile) bool { return (t.Kind == ZoneC || t.Kind == ZoneI) && t.Level > 0 }

	// Sources: roads next to developed shops and factories.
	var queue []int
	for i := range c.Tiles {
		if !isRoad(i) {
			continue
		}
		x, y := i%c.W, i/c.W
		for _, d := range dirs4 {
			if c.In(x+d[0], y+d[1]) && jobs(c.At(x+d[0], y+d[1])) {
				dist[i] = 0
				queue = append(queue, i)
				break
			}
		}
	}
	for k := 0; k < len(queue); k++ {
		i := queue[k]
		x, y := i%c.W, i/c.W
		for _, d := range dirs4 {
			nx, ny := x+d[0], y+d[1]
			if !c.In(nx, ny) {
				continue
			}
			j := ny*c.W + nx
			if isRoad(j) && dist[j] < 0 {
				dist[j] = dist[i] + 1
				queue = append(queue, j)
			}
		}
	}

	c.jobDist = dist

	// Bus stops only run if a powered depot exists somewhere.
	transit := make([]bool, n)
	depot := false
	for i := range c.Tiles {
		if t := &c.Tiles[i]; t.Kind == BusDepot && t.Powered {
			depot = true
			break
		}
	}
	if depot {
		for i := range c.Tiles {
			if c.Tiles[i].Kind == BusStop {
				c.spreadIndex(float64(i%c.W)+0.5, float64(i/c.W)+0.5, busStopRadius, 1, func(j int, v float64) {
					transit[j] = true
				})
			}
		}
	}

	load := make([]float64, n)
	for i := range c.Tiles {
		t := &c.Tiles[i]
		if t.Kind != ZoneR || t.Level == 0 {
			t.Commute = 0
			continue
		}
		// Start from the adjacent road closest to work.
		x, y := i%c.W, i/c.W
		start := -1
		for _, d := range dirs4 {
			nx, ny := x+d[0], y+d[1]
			if !c.In(nx, ny) {
				continue
			}
			j := ny*c.W + nx
			if isRoad(j) && dist[j] >= 0 && (start < 0 || dist[j] < dist[start]) {
				start = j
			}
		}
		if start < 0 {
			t.Commute = -1
			continue
		}
		t.Commute = int16(min(int(dist[start]), 32767))
		commuters := workforceShare * float64(resCap[t.Level])
		if transit[i] {
			commuters *= 1 - transitShare
		}
		// Walk downhill to the jobs.
		for cur := start; ; {
			load[cur] += commuters
			if dist[cur] == 0 {
				break
			}
			cx, cy := cur%c.W, cur/c.W
			next := -1
			for _, d := range dirs4 {
				nx, ny := cx+d[0], cy+d[1]
				if c.In(nx, ny) && dist[ny*c.W+nx] == dist[cur]-1 {
					next = ny*c.W + nx
					break
				}
			}
			if next < 0 {
				break
			}
			cur = next
		}
	}

	// Store load and spread the noise of congested roads to their neighbours.
	if len(c.noise) != n {
		c.noise = make([]float32, n)
	}
	clear(c.noise)
	for i := range c.Tiles {
		t := &c.Tiles[i]
		if !isRoad(i) {
			t.Traffic = 0
			continue
		}
		t.Traffic = uint16(min(load[i], 65535))
		over := t.Congestion() - 0.5
		if over <= 0 {
			continue
		}
		v := float32(min(noiseMax, noiseMax*over))
		x, y := i%c.W, i/c.W
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if c.In(x+dx, y+dy) {
					j := (y+dy)*c.W + x + dx
					c.noise[j] = max(c.noise[j], v)
				}
			}
		}
	}
}

// commutePenalty lowers the growth score of homes that can't reach jobs.
func (c *City) commutePenalty(t *Tile) float64 {
	if t.Kind != ZoneR || c.Stats.CommJobs+c.Stats.IndJobs == 0 {
		return 0
	}
	switch {
	case t.Commute < 0:
		return noJobsPenalty
	case t.Commute > commuteFar:
		return farCommutePenal
	}
	return 0
}

// TripRoute is the road path a commuter from home tile i drives to work:
// road tile indexes from the road beside the home to the road beside the
// nearest jobs. Nil if the home has no road route to any job.
func (c *City) TripRoute(i int) []int {
	if len(c.jobDist) != len(c.Tiles) {
		c.updateTraffic()
	}
	dist := c.jobDist
	x, y := i%c.W, i/c.W
	start := -1
	for _, d := range dirs4 {
		nx, ny := x+d[0], y+d[1]
		if !c.In(nx, ny) {
			continue
		}
		j := ny*c.W + nx
		if c.Tiles[j].Kind == Road && dist[j] >= 0 && (start < 0 || dist[j] < dist[start]) {
			start = j
		}
	}
	if start < 0 {
		return nil
	}
	path := []int{start}
	for cur := start; dist[cur] > 0; {
		cx, cy := cur%c.W, cur/c.W
		next := -1
		for _, d := range dirs4 {
			nx, ny := cx+d[0], cy+d[1]
			if c.In(nx, ny) && dist[ny*c.W+nx] == dist[cur]-1 {
				next = ny*c.W + nx
				break
			}
		}
		if next < 0 {
			break
		}
		path = append(path, next)
		cur = next
	}
	return path
}
