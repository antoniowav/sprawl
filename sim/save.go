package sim

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
)

// SaveVersion is the current save format. Older versions are migrated on
// load; newer ones are refused.
const SaveVersion = 1

// saveFile is the on-disk form: gzip'd JSON with per-field tile arrays
// ([]byte fields become base64), so a 128×128 city stays small. Derived
// per-tile values are saved too, so a loaded game continues exactly as the
// original would have.
type saveFile struct {
	Version    int
	Name       string
	W, H       int
	Seed       int64
	Ticks      int64
	Day        int
	Funds      float64
	Tax        [3]int
	Demand     [3]float64
	Stats      Stats
	Power      Utility
	Water      Utility
	Log        []Event
	Loan       *Loan
	DebtMonths int
	Bankrupt   bool
	LastMonth  Ledger
	HighDays   [3]int
	LvDirty    bool
	Brownout   bool
	WaterShort bool
	YearNet    float64
	RNG        []byte
	PeakPop    int
	History    []Sample
	Map        MapType
	Start      Pt
	ScenarioID string

	Terrain, Kind, Level, Variant, Flags []byte
	Anchor                               []int32
	LandValue, Pollution                 []float32
	Cover                                [3][]float32
	UnpoweredDays                        []uint16
}

const (
	flagLine = 1 << iota
	flagPipe
	flagPowered
	flagWatered
)

// Save writes the city to w.
func (c *City) Save(w io.Writer) error {
	rng, err := c.pcg.MarshalBinary()
	if err != nil {
		return err
	}
	n := len(c.Tiles)
	f := saveFile{
		Version: SaveVersion, Name: c.Name, W: c.W, H: c.H, Seed: c.Seed, Ticks: c.Ticks, Day: c.Day,
		Funds: c.Funds, Tax: c.Tax, Demand: c.Demand, Stats: c.Stats, Power: c.Power, Water: c.Water,
		Log: c.Log, Loan: c.Loan, DebtMonths: c.DebtMonths, Bankrupt: c.Bankrupt, LastMonth: c.LastMonth,
		HighDays: c.highDays, LvDirty: c.lvDirty, Brownout: c.brownout, WaterShort: c.waterShort,
		YearNet: c.yearNet, RNG: rng, PeakPop: c.PeakPop, History: c.History, Map: c.Map, Start: c.Start, ScenarioID: c.ScenarioID,
		Terrain: make([]byte, n), Kind: make([]byte, n), Level: make([]byte, n), Variant: make([]byte, n),
		Flags: make([]byte, n), Anchor: make([]int32, n), LandValue: make([]float32, n),
		Pollution: make([]float32, n), UnpoweredDays: make([]uint16, n),
	}
	for k := range f.Cover {
		f.Cover[k] = make([]float32, n)
	}
	for i, t := range c.Tiles {
		f.Terrain[i], f.Kind[i], f.Level[i], f.Variant[i] = byte(t.Terrain), byte(t.Kind), t.Level, t.Variant
		f.Flags[i] = flag(t.Line, flagLine) | flag(t.Pipe, flagPipe) | flag(t.Powered, flagPowered) | flag(t.Watered, flagWatered)
		f.Anchor[i], f.LandValue[i], f.Pollution[i], f.UnpoweredDays[i] = t.Anchor, t.LandValue, t.Pollution, t.UnpoweredDays
		for k := range f.Cover {
			f.Cover[k][i] = t.Cover[k]
		}
	}
	zw := gzip.NewWriter(w)
	if err := json.NewEncoder(zw).Encode(&f); err != nil {
		return err
	}
	return zw.Close()
}

func flag(b bool, f byte) byte {
	if b {
		return f
	}
	return 0
}

// Load reads a city written by Save.
func Load(r io.Reader) (*City, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("not a save file: %w", err)
	}
	var f saveFile
	if err := json.NewDecoder(zr).Decode(&f); err != nil {
		return nil, fmt.Errorf("corrupt save: %w", err)
	}
	if f.Version > SaveVersion {
		return nil, fmt.Errorf("save is version %d; this build reads up to %d", f.Version, SaveVersion)
	}
	migrate(&f)
	n := f.W * f.H
	if f.W <= 0 || f.H <= 0 || len(f.Terrain) != n || len(f.Kind) != n || len(f.Level) != n ||
		len(f.Variant) != n || len(f.Flags) != n || len(f.Anchor) != n || len(f.LandValue) != n ||
		len(f.Pollution) != n || len(f.UnpoweredDays) != n {
		return nil, fmt.Errorf("corrupt save: tile arrays don't match %dx%d", f.W, f.H)
	}
	for k := range f.Cover {
		if len(f.Cover[k]) != n {
			return nil, fmt.Errorf("corrupt save: coverage arrays")
		}
	}
	c := &City{
		Name: f.Name, W: f.W, H: f.H, Seed: f.Seed, Ticks: f.Ticks, Day: f.Day, Funds: f.Funds,
		Tax: f.Tax, Demand: f.Demand, Stats: f.Stats, Power: f.Power, Water: f.Water, Log: f.Log,
		Loan: f.Loan, DebtMonths: f.DebtMonths, Bankrupt: f.Bankrupt, LastMonth: f.LastMonth,
		highDays: f.HighDays, lvDirty: f.LvDirty, brownout: f.Brownout, waterShort: f.WaterShort,
		yearNet: f.YearNet, Tiles: make([]Tile, n), PeakPop: f.PeakPop, History: f.History, Map: f.Map, Start: f.Start, ScenarioID: f.ScenarioID,
	}
	if c.LastMonth.Buildings == nil {
		c.LastMonth.Buildings = map[Kind]float64{}
	}
	c.pcg = &rand.PCG{}
	if err := c.pcg.UnmarshalBinary(f.RNG); err != nil {
		return nil, fmt.Errorf("corrupt save: rng: %w", err)
	}
	c.rng = rand.New(c.pcg)
	for i := range c.Tiles {
		t := &c.Tiles[i]
		t.Terrain, t.Kind, t.Level, t.Variant = Terrain(f.Terrain[i]), Kind(f.Kind[i]), f.Level[i], f.Variant[i]
		fl := f.Flags[i]
		t.Line, t.Pipe, t.Powered, t.Watered = fl&flagLine != 0, fl&flagPipe != 0, fl&flagPowered != 0, fl&flagWatered != 0
		t.Anchor, t.LandValue, t.Pollution, t.UnpoweredDays = f.Anchor[i], f.LandValue[i], f.Pollution[i], f.UnpoweredDays[i]
		for k := range t.Cover {
			t.Cover[k] = f.Cover[k][i]
		}
		if t.Anchor < -1 || int(t.Anchor) >= n {
			return nil, fmt.Errorf("corrupt save: bad anchor at tile %d", i)
		}
	}
	return c, nil
}

// migrate upgrades older save versions in place. Version 1 is the first.
func migrate(f *saveFile) {}
