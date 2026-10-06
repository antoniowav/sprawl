package sim

// Balance constants (SPEC §6). Tuning passes should only need this file.

// Time.
const TicksPerDay = 4

// Capacity per zone level (index = level).
var (
	resCap = [4]int{0, 8, 30, 90}
	comCap = [4]int{0, 4, 15, 45}
	indCap = [4]int{0, 6, 20, 50}
)

// Demand.
const (
	workforceShare = 0.5  // L = share · P
	resBootstrap   = 20.0 // jobs-equivalent so an empty city wants houses
	comPerResident = 0.2  // commercial jobs wanted per resident
	comMinBase     = 10.0
	indPerWorker   = 0.8 // > 0.6 so jobs per worker exceed 1 and growth never stalls
	indBootstrap   = 15.0
	taxNeutral     = 9
	taxSlope       = 0.04
	demandSmooth   = 0.15
	demandHigh     = 0.7 // event threshold
	demandHighDays = 10
)

// Growth.
const (
	growMax       = 0.10 // max chance per day to gain a level
	growScale     = 0.15
	decayMax      = 0.06
	decayScale    = 0.05
	decayFloor    = -0.2 // score below this starts decline
	noAccessDecay = 0.05
	lvWeight      = 0.5
	coverWeight   = 0.2
)

// Land value (cover terms arrive with services in M6).
const (
	lvBase           = 0.30
	lvWater          = 0.15
	lvWaterRadius    = 3
	lvNeighbourhood  = 0.10
	lvNeighbourRad   = 2
	lvPollution      = 0.35
	lvLowMax         = 0.35 // LV below this caps a tile at level 1
	lvMidMax         = 0.60 // below this caps at level 2
	pollIndustry     = 0.08 // per level
	pollIndustryRad  = 6
	landValueEveryNd = 5 // recompute every N days
)

// Service radii, indexed like Tile.Cover: police, fire, school.
var serviceRadius = [3]int{10, 10, 12}

// coverPlateau scales coverage so it stays full near the building.
const coverPlateau = 1.25
