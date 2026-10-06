package app

import (
	"os"
	"path/filepath"

	"github.com/antoniowav/sprawl/internal/meta"
	"github.com/antoniowav/sprawl/render"
	"github.com/antoniowav/sprawl/sim"
)

// guideStep is one getting-started step; done is checked against the city.
type guideStep struct {
	text string
	done func(a *App) bool
}

var guideSteps = []guideStep{
	{"Build a road. Pick the road tool (r, or the second toolbar button) and drag on the map.",
		func(a *App) bool { return a.count(func(t *sim.Tile) bool { return t.Kind == sim.Road }) > 0 }},
	{"Zone homes along the road: residential tool (z then r), drag next to the road. Lots only grow where they touch a road.",
		func(a *App) bool { return a.count(func(t *sim.Tile) bool { return t.Kind == sim.ZoneR }) > 0 }},
	{"Power it: open Buildings (b) and place a power plant touching your road. Power runs along roads and power lines.",
		func(a *App) bool {
			return a.count(func(t *sim.Tile) bool { return t.Kind == sim.ZoneR && t.Powered }) > 0
		}},
	{"People need jobs. Zone some industry (z then i) along a road, a little away from the homes: it's smoky.",
		func(a *App) bool { return a.count(func(t *sim.Tile) bool { return t.Kind == sim.ZoneI }) > 0 }},
	{"Let it grow. Watch the R C I demand bars on the right; reach 50 people. Space pauses, 1 2 3 set the speed.",
		func(a *App) bool { return a.city.Stats.Residents >= 50 }},
	{"Shops follow people: zone commercial (z then c) once the C bar is up.",
		func(a *App) bool {
			return a.count(func(t *sim.Tile) bool { return t.Kind == sim.ZoneC && t.Level > 0 }) > 0
		}},
	{"Bigger buildings need water. Place a water pump next to water (b, 2) and lay pipes (w) to your blocks. Homes next to a pipe get water.",
		func(a *App) bool {
			return a.count(func(t *sim.Tile) bool { return t.IsZone() && t.Level > 0 && t.Watered }) > 0
		}},
	{"Raise land value with a park, school, police or fire station (b). Check it with the overlay button (o).",
		func(a *App) bool {
			return a.count(func(t *sim.Tile) bool {
				return t.Kind == sim.Police || t.Kind == sim.Fire || t.Kind == sim.School || t.Kind == sim.Park
			}) > 0
		}},
	{"Keep an eye on money: open the budget (Ctrl+B). Taxes: type :tax 10. Save any time with Ctrl+S.",
		func(a *App) bool { return a.sawBudget }},
}

func (a *App) count(f func(*sim.Tile) bool) int {
	n := 0
	for i := range a.city.Tiles {
		if f(&a.city.Tiles[i]) {
			n++
		}
	}
	return n
}

func guideMarker() string { return filepath.Join(meta.DataDir(), "guide-done") }

// guideFirstRun is true until the player has finished the guide once.
func guideFirstRun() bool {
	_, err := os.Stat(guideMarker())
	return err != nil
}

// guideView returns the panel for the current step, or nil when hidden.
// Finishing the last step remembers it so later cities start without it.
func (a *App) guideView() *render.Guide {
	if !a.showGuide {
		return nil
	}
	for i, s := range guideSteps {
		if !s.done(a) {
			return &render.Guide{Step: i + 1, Total: len(guideSteps), Text: s.text}
		}
	}
	if guideFirstRun() {
		os.MkdirAll(meta.DataDir(), 0o755)
		os.WriteFile(guideMarker(), nil, 0o644)
		a.city.Logf(sim.Info, "getting started: done. F1 brings the guide back")
	}
	return &render.Guide{Step: len(guideSteps), Total: len(guideSteps), Done: true,
		Text: "That's the whole loop. Grow, balance R/C/I demand, keep the budget positive. ? lists every key. F1 hides this."}
}
