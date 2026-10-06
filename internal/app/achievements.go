package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/antoniowav/sprawl/internal/meta"
	"github.com/antoniowav/sprawl/sim"
)

// achievement is a goal that, once reached in any city, stays unlocked.
type achievement struct {
	id, name, desc string
	met            func(a *App) bool
}

var achievements = []achievement{
	{"first100", "First steps", "Reach 100 people", func(a *App) bool { return a.city.Stats.Residents >= 100 }},
	{"town", "Proper town", "Reach 1,000 people", func(a *App) bool { return a.city.Stats.Residents >= 1000 }},
	{"city", "Big city", "Reach 10,000 people", func(a *App) bool { return a.city.Stats.Residents >= 10000 }},
	{"clean", "Clean energy", "Run on at least 200 power with no smoky plants", func(a *App) bool {
		return a.city.Power.Capacity >= 200 && a.count(func(t *sim.Tile) bool { return t.Kind == sim.PowerPlant }) == 0
	}},
	{"black", "In the black", "Earn money 24 months in a row", func(a *App) bool { return profitStreak(a.city) >= 24 }},
	{"commute", "Smooth commute", "2,000 people and no jammed road", func(a *App) bool {
		if a.city.Stats.Residents < 2000 {
			return false
		}
		return a.count(func(t *sim.Tile) bool { return t.Kind == sim.Road && t.Congestion() > 1 }) == 0
	}},
	{"bridges", "Island hopper", "Build 20 bridge tiles", func(a *App) bool {
		return a.count(func(t *sim.Tile) bool { return t.Kind == sim.Road && t.Terrain == sim.Water }) >= 20
	}},
	{"scholar", "Scholar city", "Build a university", func(a *App) bool {
		return a.count(func(t *sim.Tile) bool { return t.Kind == sim.University }) > 0
	}},
	{"debtfree", "Debt-free decade", "Ten years without ever going into debt", func(a *App) bool {
		return a.city.Day >= 10*sim.DaysPerMonth*sim.MonthsPerYear && !a.city.EverInDebt
	}},
	{"boomtown", "Boomtown", "Complete the Boomtown scenario", func(a *App) bool {
		return a.city.ScenarioID == "boomtown" && a.city.ScenarioResult == sim.ScenarioWon
	}},
	{"island", "Islander", "Complete the Island scenario", func(a *App) bool {
		return a.city.ScenarioID == "island" && a.city.ScenarioResult == sim.ScenarioWon
	}},
}

// profitStreak counts the latest months in a row where funds went up.
func profitStreak(c *sim.City) int {
	h := c.History
	n := 0
	for i := len(h) - 1; i > 0 && h[i].Funds > h[i-1].Funds; i-- {
		n++
	}
	return n
}

func achievementsPath() string { return filepath.Join(meta.DataDir(), "achievements.json") }

// loadAchievements reads unlocked achievements (id -> date).
func loadAchievements() map[string]string {
	got := map[string]string{}
	if b, err := os.ReadFile(achievementsPath()); err == nil {
		json.Unmarshal(b, &got)
	}
	return got
}

// checkAchievements unlocks anything newly reached and announces it.
func (a *App) checkAchievements() {
	if a.tl != nil || a.shotPath != "" {
		return // the title backdrop and scripted screenshots don't count
	}
	changed := false
	for _, ac := range achievements {
		if _, ok := a.unlocked[ac.id]; ok || !ac.met(a) {
			continue
		}
		a.unlocked[ac.id] = time.Now().Format("2006-01-02")
		changed = true
		a.city.Logf(sim.Info, "achievement: %s", ac.name)
		a.toast("Achievement: "+ac.name+" · "+ac.desc, true)
		a.snd.Play(soundMilestone)
	}
	if changed {
		os.MkdirAll(meta.DataDir(), 0o755)
		if b, err := json.MarshalIndent(a.unlocked, "", "  "); err == nil {
			os.WriteFile(achievementsPath(), b, 0o644)
		}
	}
}
