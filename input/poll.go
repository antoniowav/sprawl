package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Key repeat for named keys, in Update ticks (60/s).
const (
	repeatDelay = 15
	repeatEvery = 2
)

// Poller turns this tick's keyboard input into actions.
type Poller struct {
	km    *Keymap
	chars []rune
}

// NewPoller reads input through km.
func NewPoller(km *Keymap) *Poller { return &Poller{km: km} }

// Chars returns the characters typed this tick (for text prompts).
func (p *Poller) Chars() []rune {
	p.chars = ebiten.AppendInputChars(p.chars[:0])
	return p.chars
}

// Press is one key event: the bound action (empty if unbound) and, for
// typed characters, the rune. Prefix keys like z use the rune.
type Press struct {
	Action Action
	Rune   rune
}

// Poll returns this tick's key presses in a stable order.
func (p *Poller) Poll() []Press {
	var out []Press
	chars := p.Chars()
	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		chars = nil // some platforms still send the letter with Ctrl held
	}
	for _, r := range chars {
		a, _ := p.km.Lookup(Binding{Rune: r})
		out = append(out, Press{a, r})
	}
	shift := ebiten.IsKeyPressed(ebiten.KeyShift)
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl)
	for _, b := range p.km.named {
		if b.Shift == shift && b.Ctrl == ctrl && Repeated(b.Key) {
			a, _ := p.km.Lookup(b)
			out = append(out, Press{Action: a})
		}
	}
	return out
}

// Repeated is true on the first tick of a press and then at the repeat rate.
func Repeated(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || (d >= repeatDelay && (d-repeatDelay)%repeatEvery == 0)
}
