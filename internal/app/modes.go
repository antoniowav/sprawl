package app

import "github.com/antoniowav/sprawl/sim"

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
