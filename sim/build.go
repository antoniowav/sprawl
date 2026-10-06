package sim

import "fmt"

// Tool is something the player can place or remove.
type Tool uint8

const (
	ToolRoad Tool = iota
	ToolLine
	ToolPipe
	ToolBulldoze
	ToolZoneR
	ToolZoneC
	ToolZoneI
	ToolPlant
	ToolPump
	ToolPolice
	ToolFire
	ToolSchool
	ToolPark
	ToolWind
	ToolTower
	ToolCityHall
	ToolStadium
	ToolBusStop
	ToolBusDepot
	ToolHospital
	ToolSolar
	ToolUniversity
	ToolNuclear
	ToolMonument
	ToolRaise
	ToolLower
	ToolLevel
)

var toolNames = [...]string{"road", "power line", "water pipe", "bulldoze", "residential", "commercial",
	"industrial", "power plant", "water pump", "police station", "fire station", "school",
	"park", "wind turbine", "water tower", "city hall", "stadium", "bus stop", "bus depot", "hospital",
	"solar farm", "university", "nuclear plant", "monument", "raise ground", "lower ground", "level ground"}

// BuildingSpec describes a placeable building and everything it does.
type BuildingSpec struct {
	Tool          Tool
	Kind          Kind
	Size          int // footprint is Size×Size
	Cost          float64
	Upkeep        float64 // per month
	PowerCap      int     // power it supplies
	WaterCap      int     // water it supplies (pumps also need water terrain)
	PowerUse      int
	WaterUse      int
	Service       int     // index into Tile.Cover, or -1
	ServiceRadius int     // overrides the service's usual radius when > 0
	LVBonus       float64 // land value added at its centre, fading over LVRadius
	LVRadius      int
	Smog          float64 // pollution at its centre, fading over SmogRadius
	SmogRad       int
	Unlock        int    // population needed before it can be built
	Note          string // one line for the build menu
}

// Buildings lists placeable buildings in menu order.
var Buildings = []BuildingSpec{
	{Tool: ToolPlant, Kind: PowerPlant, Size: 3, Cost: 3000, Upkeep: 100, PowerCap: PlantCapacity, Service: -1,
		Smog: 0.5, SmogRad: 8, Note: "200 power · smoky"},
	{Tool: ToolSolar, Kind: SolarFarm, Size: 2, Cost: 2000, Upkeep: 20, PowerCap: 60, Service: -1,
		Unlock: 500, Note: "60 power · clean"},
	{Tool: ToolNuclear, Kind: NuclearPlant, Size: 3, Cost: 15000, Upkeep: 400, PowerCap: 1000, WaterUse: 10, Service: -1,
		Unlock: 5000, Note: "1,000 power · no smog"},
	{Tool: ToolWind, Kind: WindTurbine, Size: 1, Cost: 500, Upkeep: 10, PowerCap: 25, Service: -1,
		Note: "25 power · clean"},
	{Tool: ToolPump, Kind: WaterPump, Size: 2, Cost: 1500, Upkeep: 50, WaterCap: PumpCapacity, PowerUse: 3, Service: -1,
		Note: "150 water · next to water"},
	{Tool: ToolTower, Kind: WaterTower, Size: 1, Cost: 700, Upkeep: 20, WaterCap: 40, PowerUse: 1, Service: -1,
		Note: "40 water · anywhere"},
	{Tool: ToolPolice, Kind: Police, Size: 2, Cost: 500, Upkeep: 60, PowerUse: 4, WaterUse: 1, Service: 0,
		Note: "safety · radius 10"},
	{Tool: ToolFire, Kind: Fire, Size: 2, Cost: 500, Upkeep: 60, PowerUse: 4, WaterUse: 1, Service: 1,
		Note: "fire cover · radius 10"},
	{Tool: ToolSchool, Kind: School, Size: 2, Cost: 800, Upkeep: 80, PowerUse: 4, WaterUse: 1, Service: 2,
		Note: "education · radius 12"},
	{Tool: ToolHospital, Kind: Hospital, Size: 2, Cost: 1500, Upkeep: 120, PowerUse: 6, WaterUse: 2, Service: 3,
		Unlock: 250, Note: "health · radius 14"},
	{Tool: ToolUniversity, Kind: University, Size: 3, Cost: 8000, Upkeep: 200, PowerUse: 10, WaterUse: 3, Service: 2,
		ServiceRadius: 22, LVBonus: 0.08, LVRadius: 16, Unlock: 2500, Note: "education · radius 22"},
	{Tool: ToolBusStop, Kind: BusStop, Size: 1, Cost: 200, Upkeep: 15, Service: -1,
		Note: "30% of commuters ride · radius 6"},
	{Tool: ToolBusDepot, Kind: BusDepot, Size: 2, Cost: 1200, Upkeep: 80, PowerUse: 4, WaterUse: 1, Service: -1,
		Note: "runs the bus stops"},
	{Tool: ToolPark, Kind: Park, Size: 1, Cost: 150, Upkeep: 5, Service: -1, LVBonus: 0.15, LVRadius: 4,
		Note: "land value · radius 4"},
	{Tool: ToolCityHall, Kind: CityHall, Size: 3, Cost: 4000, Upkeep: 100, PowerUse: 6, WaterUse: 2, Service: -1,
		LVBonus: 0.12, LVRadius: 16, Unlock: 1000, Note: "+5% taxes · land value"},
	{Tool: ToolStadium, Kind: Stadium, Size: 3, Cost: 6000, Upkeep: 150, PowerUse: 8, WaterUse: 3, Service: -1,
		LVBonus: 0.08, LVRadius: 12, Unlock: 2500, Note: "people want to live here"},
	{Tool: ToolMonument, Kind: Monument, Size: 2, Cost: 10000, Upkeep: 50, Service: -1,
		LVBonus: 0.25, LVRadius: 10, Unlock: 10000, Note: "the city's pride"},
}

// Building returns the spec for a building tool.
func (t Tool) Building() (BuildingSpec, bool) {
	for _, b := range Buildings {
		if b.Tool == t {
			return b, true
		}
	}
	return BuildingSpec{}, false
}

// SpecOf returns the spec for a building kind.
func SpecOf(k Kind) (BuildingSpec, bool) {
	for _, b := range Buildings {
		if b.Kind == k {
			return b, true
		}
	}
	return BuildingSpec{}, false
}

// spec is SpecOf for kinds known to be buildings.
func spec(k Kind) *BuildingSpec {
	for i := range Buildings {
		if Buildings[i].Kind == k {
			return &Buildings[i]
		}
	}
	return nil
}

func (t Tool) String() string { return toolNames[t] }

// Line tools draw paths; area tools fill rectangles.
func (t Tool) IsLine() bool { return t == ToolRoad || t == ToolLine || t == ToolPipe }

// Costs per tile.
const (
	CostRoad          = 10
	CostLine          = 5
	CostPipe          = 5
	CostZone          = 5
	CostBulldoze      = 1
	CostBulldozeBuilt = 20 // extra for a developed tile or a building
	CostBridge        = 60 // road tile over water
)

// Pt is a tile coordinate.
type Pt struct{ X, Y int }

// Plan is the outcome of applying a tool to a set of tiles.
type Plan struct {
	Tool    Tool
	Tiles   []Pt // tiles that will change
	Skipped int  // tiles where the tool doesn't apply
	Cost    float64
	Err     string // set when the whole action is refused
}

// PlanTool works out what Apply would do, without changing anything.
// pipesOnly makes bulldoze remove pipes instead of surface things
// (used in underground view).
// Terraform maps a terrain tool to its operation.
func (t Tool) Terraform() (int, bool) {
	switch t {
	case ToolRaise:
		return TerraRaise, true
	case ToolLower:
		return TerraLower, true
	case ToolLevel:
		return TerraLevel, true
	}
	return 0, false
}

func (c *City) PlanTool(t Tool, pts []Pt, pipesOnly bool) Plan {
	p := Plan{Tool: t}
	if op, ok := t.Terraform(); ok {
		tp := c.PlanTerraform(op, pts)
		return Plan{Tool: t, Tiles: tp.Tiles, Cost: tp.Cost, Err: tp.Err}
	}
	if spec, ok := t.Building(); ok {
		return c.planBuilding(spec, pts)
	}
	seen := map[int32]bool{} // buildings already counted by bulldoze
	for _, pt := range pts {
		if !c.In(pt.X, pt.Y) {
			continue
		}
		tl := c.At(pt.X, pt.Y)
		if t == ToolBulldoze && !pipesOnly && tl.Anchor >= 0 {
			// A building goes as a whole, once, whichever tile was hit.
			if !seen[tl.Anchor] {
				seen[tl.Anchor] = true
				for _, bt := range c.footprint(tl.Anchor) {
					p.Tiles = append(p.Tiles, bt)
				}
				p.Cost += CostBulldoze + CostBulldozeBuilt
			}
			continue
		}
		cost, ok := c.tileCost(t, tl, pipesOnly, c.Slope(pt.X, pt.Y))
		if !ok {
			p.Skipped++
			continue
		}
		p.Tiles = append(p.Tiles, pt)
		p.Cost += cost
	}
	switch {
	case len(p.Tiles) == 0 && t == ToolBulldoze:
		p.Err = "nothing to bulldoze"
	case len(p.Tiles) == 0:
		p.Err = fmt.Sprintf("can't build %s here", t)
	case t != ToolBulldoze && c.Funds < 0:
		p.Err = "in debt: only bulldozing allowed"
	case t != ToolBulldoze && p.Cost > c.Funds: // bulldozing is always allowed, even into debt
		p.Err = fmt.Sprintf("not enough funds ($%.0f needed)", p.Cost)
	}
	return p
}

// Apply executes a plan made by PlanTool on the current state. It returns
// the plan, with Err set if nothing was done.
func (c *City) Apply(t Tool, pts []Pt, pipesOnly bool) Plan {
	if op, ok := t.Terraform(); ok {
		tp := c.Terraform(op, pts)
		return Plan{Tool: t, Tiles: tp.Tiles, Cost: tp.Cost, Err: tp.Err}
	}
	p := c.PlanTool(t, pts, pipesOnly)
	if p.Err != "" {
		return p
	}
	if spec, ok := t.Building(); ok {
		anchor := int32(p.Tiles[0].Y*c.W + p.Tiles[0].X)
		for _, pt := range p.Tiles {
			tl := c.At(pt.X, pt.Y)
			if tl.Terrain == Trees {
				tl.Terrain = Land
			}
			tl.Kind, tl.Level, tl.Line, tl.Anchor = spec.Kind, 0, false, anchor
		}
	} else {
		for _, pt := range p.Tiles {
			c.applyTile(t, c.At(pt.X, pt.Y), pipesOnly)
		}
	}
	c.Funds -= p.Cost
	c.lvDirty = true
	return p
}

func zoneKind(t Tool) Kind { return ZoneR + Kind(t-ToolZoneR) }

// tileCost says whether t changes this tile and what it costs.
func (c *City) tileCost(t Tool, tl *Tile, pipesOnly bool, slope int) (float64, bool) {
	buildable := tl.Terrain != Water && tl.Terrain != Rock
	switch t {
	case ToolRoad:
		if tl.Terrain == Water && tl.Kind == Empty {
			return CostBridge, true // bridge
		}
		if !buildable || slope > roadClimb || tl.Kind == Road || (tl.Kind != Empty && !isEmptyLot(tl)) {
			return 0, false
		}
		if slope == roadClimb {
			return 3 * CostRoad, true // a steep stretch: cuttings and switchbacks
		}
		return CostRoad, true
	case ToolLine:
		if tl.Line || (tl.Kind != Road && (!buildable || tl.Kind != Empty)) {
			return 0, false
		}
		return CostLine, true
	case ToolPipe:
		if !buildable || tl.Pipe {
			return 0, false
		}
		return CostPipe, true
	case ToolZoneR, ToolZoneC, ToolZoneI:
		k := zoneKind(t)
		if !buildable || slope > gentle || tl.Kind == k || (tl.Kind != Empty && !isEmptyLot(tl)) {
			return 0, false
		}
		return CostZone, true
	case ToolBulldoze:
		if pipesOnly {
			return CostBulldoze, tl.Pipe
		}
		switch {
		case tl.Kind != Empty:
			if tl.Level > 0 {
				return CostBulldoze + CostBulldozeBuilt, true
			}
			return CostBulldoze, true
		case tl.Line || tl.Terrain == Trees:
			return CostBulldoze, true
		}
	}
	return 0, false
}

func isEmptyLot(tl *Tile) bool { return tl.IsZone() && tl.Level == 0 }

// IsZone reports whether the tile is zoned R, C or I.
func (t *Tile) IsZone() bool { return t.Kind >= ZoneR && t.Kind <= ZoneI }

func (c *City) applyTile(t Tool, tl *Tile, pipesOnly bool) {
	clearTrees := func() {
		if tl.Terrain == Trees {
			tl.Terrain = Land
		}
	}
	switch t {
	case ToolRoad:
		clearTrees()
		tl.Kind, tl.Level = Road, 0
	case ToolLine:
		clearTrees()
		tl.Line = true
	case ToolPipe:
		tl.Pipe = true
	case ToolZoneR, ToolZoneC, ToolZoneI:
		clearTrees()
		tl.Kind, tl.Level = zoneKind(t), 0
		tl.Line = false // zones conduct power themselves
		tl.Variant = uint8(c.rng.IntN(4))
	case ToolBulldoze:
		if pipesOnly {
			tl.Pipe = false
			return
		}
		clearTrees()
		tl.Kind, tl.Level, tl.Line, tl.Anchor = Empty, 0, false, -1
	}
}

// footprint lists the tiles of the building anchored at index a.
func (c *City) footprint(a int32) []Pt {
	ax, ay := int(a)%c.W, int(a)/c.W
	size := 2 // big zone buildings
	if spec, ok := SpecOf(c.Tiles[a].Kind); ok {
		size = spec.Size
	}
	var pts []Pt
	for y := ay; y < ay+size; y++ {
		for x := ax; x < ax+size; x++ {
			pts = append(pts, Pt{x, y})
		}
	}
	return pts
}

// planBuilding places a building with its top-left at the first point.
func (c *City) planBuilding(spec BuildingSpec, pts []Pt) Plan {
	p := Plan{Tool: spec.Tool}
	if len(pts) == 0 {
		p.Err = "nowhere to build"
		return p
	}
	o := pts[0]
	touchesWater := false
	lo, hi := uint8(255), uint8(0)
	for y := o.Y; y < o.Y+spec.Size; y++ {
		for x := o.X; x < o.X+spec.Size; x++ {
			if !c.In(x, y) {
				p.Err = "doesn't fit on the map"
				continue
			}
			t := c.At(x, y)
			p.Tiles = append(p.Tiles, Pt{x, y}) // kept on error, for the preview
			lo, hi = min(lo, t.Height), max(hi, t.Height)
			if t.Terrain == Water || t.Terrain == Rock || (t.Kind != Empty && !isEmptyLot(t)) {
				p.Err = fmt.Sprintf("%s needs %d×%d clear land", spec.Tool, spec.Size, spec.Size)
			}
			for _, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
				if c.In(x+d[0], y+d[1]) && c.At(x+d[0], y+d[1]).Terrain == Water {
					touchesWater = true
				}
			}
		}
	}
	p.Cost = spec.Cost
	switch {
	case p.Err != "":
	case hi-lo > gentle:
		p.Err = fmt.Sprintf("%s needs flatter ground (level it with t f)", spec.Tool)
	case !c.Unlocked(spec):
		p.Err = fmt.Sprintf("%s unlocks at %d people", spec.Tool, spec.Unlock)
	case spec.Kind == WaterPump && !touchesWater:
		p.Err = "a water pump must touch water"
	case c.Funds < 0:
		p.Err = "in debt: only bulldozing allowed"
	case p.Cost > c.Funds:
		p.Err = fmt.Sprintf("not enough funds ($%.0f needed)", p.Cost)
	}
	return p
}

// Unlocked reports whether the city has been big enough to build b.
func (c *City) Unlocked(b BuildingSpec) bool { return c.PeakPop >= b.Unlock }

// IsBuilding reports whether the tile is part of a service or utility building.
func (t *Tile) IsBuilding() bool { return t.Kind >= PowerPlant }

// LPath returns tiles from a to b: along x first then y, or y first when
// vertFirst is set. Both ends are included.
func LPath(a, b Pt, vertFirst bool) []Pt {
	var pts []Pt
	step := func(v, to int) int {
		switch {
		case v < to:
			return 1
		case v > to:
			return -1
		}
		return 0
	}
	cur := a
	pts = append(pts, cur)
	walk := func(horizontal bool) {
		for {
			if horizontal && cur.X != b.X {
				cur.X += step(cur.X, b.X)
			} else if !horizontal && cur.Y != b.Y {
				cur.Y += step(cur.Y, b.Y)
			} else {
				return
			}
			pts = append(pts, cur)
		}
	}
	walk(!vertFirst)
	walk(vertFirst)
	return pts
}

// RectPts returns every tile in the rectangle spanned by a and b.
func RectPts(a, b Pt) []Pt {
	x0, x1 := min(a.X, b.X), max(a.X, b.X)
	y0, y1 := min(a.Y, b.Y), max(a.Y, b.Y)
	pts := make([]Pt, 0, (x1-x0+1)*(y1-y0+1))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			pts = append(pts, Pt{x, y})
		}
	}
	return pts
}

// Selection returns the tiles a tool affects between anchor a and cursor b.
// For levelling, the anchor comes first: it sets the height.
func Selection(t Tool, a, b Pt, vertFirst bool) []Pt {
	if t.IsLine() {
		return LPath(a, b, vertFirst)
	}
	pts := RectPts(a, b)
	if t == ToolLevel {
		for i, p := range pts {
			if p == a {
				pts[0], pts[i] = pts[i], pts[0]
			}
		}
	}
	return pts
}
