package app

import (
	"sort"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/antoniowav/sprawl/input"
	"github.com/antoniowav/sprawl/sim"
)

// commandNames are completed with Tab.
var commandNames = []string{"budget", "e", "help", "load", "loan", "name", "new", "q", "quit", "repay", "save", "tax", "theme", "w", "wq"}

func (a *App) updatePrompt() {
	for _, r := range a.poll.Chars() {
		a.prompt += string(r)
		a.dirty = true
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		a.mode, a.dirty = modeNormal, true
	case inpututil.IsKeyJustPressed(ebiten.KeyEnter):
		cmd := strings.TrimSpace(a.prompt)
		a.mode, a.dirty = modeNormal, true
		if cmd != "" && (len(a.history) == 0 || a.history[len(a.history)-1] != cmd) {
			a.history = append(a.history, cmd)
		}
		a.histIdx = len(a.history)
		a.run(cmd)
	case inpututil.IsKeyJustPressed(ebiten.KeyTab):
		a.complete()
	case input.Repeated(ebiten.KeyArrowUp):
		if a.histIdx > 0 {
			a.histIdx--
			a.prompt = a.history[a.histIdx]
		}
		a.dirty = true
	case input.Repeated(ebiten.KeyArrowDown):
		if a.histIdx < len(a.history) {
			a.histIdx++
		}
		a.prompt = ""
		if a.histIdx < len(a.history) {
			a.prompt = a.history[a.histIdx]
		}
		a.dirty = true
	case input.Repeated(ebiten.KeyBackspace):
		if a.prompt == "" {
			a.mode = modeNormal
		} else {
			r := []rune(a.prompt)
			a.prompt = string(r[:len(r)-1])
		}
		a.dirty = true
	}
}

// complete fills in the command name when the prefix is unique, otherwise
// lists the candidates.
func (a *App) complete() {
	a.dirty = true
	if strings.Contains(a.prompt, " ") {
		return
	}
	var m []string
	for _, c := range commandNames {
		if strings.HasPrefix(c, a.prompt) {
			m = append(m, c)
		}
	}
	switch len(m) {
	case 0:
		a.flash(sim.Warn, "no command starts with %q", a.prompt)
	case 1:
		a.prompt = m[0] + " "
	default:
		sort.Strings(m)
		a.flash(sim.Info, "%s", strings.Join(m, " "))
	}
}

func (a *App) run(line string) {
	f := strings.Fields(line)
	if len(f) == 0 {
		return
	}
	rest := strings.Join(f[1:], " ")
	arg := func(i int) string {
		if i < len(f) {
			return f[i]
		}
		return ""
	}
	switch f[0] {
	case "q!":
		a.quit = true
	case "q", "quit":
		a.requestQuit()
	case "wq":
		if a.save(rest) {
			a.quit = true
		}
	case "w", "save":
		a.save(rest)
	case "e", "load":
		if rest == "" {
			a.openSaves()
			return
		}
		a.guard("Open "+rest+"?", func() { a.load(rest) })
	case "help", "h":
		a.mode = modeHelp
	case "budget":
		a.mode = modeBudget
	case "tax":
		a.cmdTax(f[1:])
	case "loan":
		a.report(a.city.TakeLoan())
	case "repay":
		a.report(a.city.Repay())
	case "name":
		if n := rest; n != "" {
			a.city.Name = n
			a.flash(sim.Info, "city renamed to %s", n)
		}
	case "new":
		seed := a.seed + 1
		if s, err := strconv.ParseInt(arg(1), 10, 64); err == nil {
			seed = s
		}
		a.guard("Start a new city?", func() { a.newCity(seed) })
	case "theme":
		if arg(1) == "reload" {
			a.loadTheme(true)
			return
		}
		a.flash(sim.Info, "theme: %s · :theme reload", a.pal.Name)
	default:
		a.flash(sim.Err, "unknown command: %s (Tab completes)", f[0])
	}
}

func (a *App) report(err error) {
	if err != nil {
		a.flash(sim.Warn, "%v", err)
	}
	a.dirty = true
}

// cmdTax handles ":tax 12" and ":tax r 12".
func (a *App) cmdTax(args []string) {
	usage := func() { a.flash(sim.Warn, "usage: :tax <0-20> or :tax r|c|i <0-20>") }
	z := -1
	if len(args) == 2 {
		i := strings.Index("rci", strings.ToLower(args[0]))
		if len(args[0]) != 1 || i < 0 {
			usage()
			return
		}
		z, args = i, args[1:]
	}
	if len(args) != 1 {
		usage()
		return
	}
	n, err := strconv.Atoi(args[0])
	if err != nil {
		usage()
		return
	}
	if err := a.city.SetTax(z, n); err != nil {
		a.report(err)
		return
	}
	t := a.city.Tax
	a.flash(sim.Info, "taxes r/c/i: %d/%d/%d", t[0], t[1], t[2])
}
