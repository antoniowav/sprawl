// Package sim is the pure city simulation. It must not import rendering or
// input packages; everything here is unit tested.
package sim

import "math/rand/v2"

// Terrain is the natural ground of a tile.
type Terrain uint8

const (
	Land Terrain = iota
	Water
	Trees
)

// Kind is what has been built on a tile.
type Kind uint8

const (
	Empty Kind = iota
	Road
	ZoneR
	ZoneC
	ZoneI
	PowerPlant
	WaterPump
	Police
	Fire
	School
	Park
	WindTurbine
	WaterTower
	CityHall
	Stadium
)

// Tile is one map cell. Derived fields are recomputed by the sim and are
// not saved.
type Tile struct {
	Terrain Terrain
	Kind    Kind
	Level   uint8 // zones: 0 empty lot, 1 low, 2 medium, 3 high
	Variant uint8 // sprite variant, rerolled when Level changes
	Line    bool  // overhead power line
	Pipe    bool  // underground pipe
	Anchor  int32 // multi-tile buildings: index of top-left tile, else -1

	// Derived each day; not saved.
	Powered, Watered bool
	LandValue        float32    // 0..1
	Pollution        float32    // 0..1
	Cover            [3]float32 // police, fire, school
	UnpoweredDays    uint16
}

// City is the whole simulation state.
type City struct {
	Name   string
	W, H   int
	Tiles  []Tile
	Seed   int64
	Ticks  int64
	Day    int
	Funds  float64
	Tax    [3]int
	Demand [3]float64
	Stats  Stats
	Power  Utility
	Water  Utility
	Log    []Event

	PeakPop int // highest population reached; drives unlocks and milestones
	History []Sample

	Loan       *Loan
	DebtMonths int
	Bankrupt   bool
	LastMonth  Ledger

	rng      *rand.Rand
	pcg      *rand.PCG // rng's source, kept for saving
	highDays [3]int
	lvDirty  bool

	brownout, waterShort bool
	lvBonus              []float32 // scratch for updateLandValue
	yearNet              float64
}

// StartingFunds is the money a new city begins with.
const StartingFunds = 20000

// New generates a w×h city from seed.
func New(w, h int, seed int64) *City {
	c := &City{
		W:     w,
		H:     h,
		Tiles: make([]Tile, w*h),
		Seed:  seed,
		Funds: StartingFunds,
		Tax:   [3]int{9, 9, 9},
	}
	c.pcg = rand.NewPCG(uint64(seed), 0x5eed)
	c.rng = rand.New(c.pcg)
	for i := range c.Tiles {
		c.Tiles[i].Anchor = -1
	}
	generateTerrain(c)
	c.Name = cityName(c.rng)
	return c
}

// In reports whether (x, y) is on the map.
func (c *City) In(x, y int) bool { return x >= 0 && y >= 0 && x < c.W && y < c.H }

// At returns the tile at (x, y). The caller must check In first.
func (c *City) At(x, y int) *Tile { return &c.Tiles[y*c.W+x] }

// Hash is a stable per-tile hash for cosmetic choices (sprite variants).
func (c *City) Hash(x, y int) uint32 {
	h := uint64(x)*0x9E3779B97F4A7C15 ^ uint64(y)*0xC2B2AE3D27D4EB4F ^ uint64(c.Seed)
	h ^= h >> 31
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 29
	return uint32(h)
}

var (
	namePre = []string{"Ash", "Bright", "Cedar", "Elm", "Fair", "Glen", "Haven", "Iron",
		"Lake", "Maple", "North", "Oak", "Pine", "River", "Stone", "West"}
	namePost = []string{"ford", "field", "haven", "wood", "ton", "bridge", "dale", "port", "view", "brook"}
)

func cityName(r *rand.Rand) string {
	return namePre[r.IntN(len(namePre))] + namePost[r.IntN(len(namePost))]
}
