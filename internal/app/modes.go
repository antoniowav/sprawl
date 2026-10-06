package app

import (
	"fmt"

	"github.com/antoniowav/sprawl/sim"
)

// gameMode is a choice in the new-city form: sandbox or a scenario.
type gameMode struct {
	name     string
	scenario *sim.Scenario
}

var gameModes = func() []gameMode {
	ms := []gameMode{{name: "sandbox"}}
	for _, s := range sim.Scenarios {
		ms = append(ms, gameMode{name: s.Name, scenario: s})
	}
	return ms
}()

// scenarioOver shows the win or lose screen.
func (a *App) scenarioOver(result int) {
	s := sim.ScenarioByID(a.city.ScenarioID)
	if s == nil {
		return
	}
	years := a.city.Day / (sim.DaysPerMonth * sim.MonthsPerYear)
	keep := button{'c', "Keep playing", a.closeDialog}
	title := button{'t', "Title screen", func() { a.closeDialog(); a.guard("Leave this city?", a.showTitle) }}
	if result == sim.ScenarioWon {
		a.snd.Play(soundMilestone)
		a.openDialog(&dialog{kind: dlgConfirm, title: "scenario complete",
			text: []string{fmt.Sprintf("%s: done in %d years, %d months.", s.Name, years, a.city.Day/sim.DaysPerMonth%12),
				"The city is yours to keep building."},
			buttons: []button{keep, title}})
		return
	}
	retry := button{'r', "Try again", func() {
		a.closeDialog()
		c := s.NewCity()
		a.installCity(c, s.Seed)
		c.Logf(sim.Info, "%s founded · %s", c.Name, s.Name)
		s.Begin(c)
	}}
	a.openDialog(&dialog{kind: dlgConfirm, title: "time's up",
		text:    []string{s.Name + " wasn't completed in time.", "Try again, or keep this city as a sandbox."},
		buttons: []button{retry, keep, title}})
}
