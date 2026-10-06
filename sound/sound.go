// Package sound synthesises Sprawl's sound effects in code (no audio files)
// and plays them with oto, Ebitengine's audio library. The device is opened
// on the first sound and the stream is suspended a few seconds after the
// last one: an open stream keeps feeding silence and costs about 1.5 % of a
// core, which Ebitengine's own audio package has no way to pause.
package sound

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Effect is a sound effect.
type Effect int

const (
	Click Effect = iota
	Place
	Zone
	Building
	Bulldoze
	Error
	Milestone
	effectCount
)

const rate = 44100

// Player plays effects at a volume of 0..1.
type Player struct {
	mu        sync.Mutex
	ctx       *oto.Context
	pcm       [effectCount][]byte
	last      [effectCount]time.Time
	volume    float64
	failed    bool
	suspended bool
	idle      *time.Timer
	playing   []*oto.Player
}

// idleAfter is how long the stream stays open after the last sound.
const idleAfter = 3 * time.Second

// New prepares the effects; nothing is opened until the first Play.
func New(volume float64) *Player {
	p := &Player{volume: volume}
	for e := Effect(0); e < effectCount; e++ {
		p.pcm[e] = encode(synth(e))
	}
	return p
}

// SetVolume changes the volume (0 mutes).
func (p *Player) SetVolume(v float64) {
	p.mu.Lock()
	p.volume = max(0, min(1, v))
	p.mu.Unlock()
}

// Play starts an effect. Repeats of the same effect within 60 ms are
// dropped, so painting a road doesn't turn into a buzz.
func (p *Player) Play(e Effect) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.volume <= 0 || p.failed || time.Since(p.last[e]) < 60*time.Millisecond {
		return
	}
	p.last[e] = time.Now()
	if p.ctx == nil && !p.open() {
		return
	}
	if p.suspended {
		if err := p.ctx.Resume(); err != nil {
			return
		}
		p.suspended = false
	}
	// Drop finished players so they can be collected.
	live := p.playing[:0]
	for _, pl := range p.playing {
		if pl.IsPlaying() {
			live = append(live, pl)
		}
	}
	pl := p.ctx.NewPlayer(bytes.NewReader(p.pcm[e]))
	pl.SetVolume(p.volume)
	pl.Play()
	p.playing = append(live, pl)
	if p.idle == nil {
		p.idle = time.AfterFunc(idleAfter, p.suspend)
	} else {
		p.idle.Reset(idleAfter)
	}
}

func (p *Player) suspend() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, pl := range p.playing {
		if pl.IsPlaying() {
			p.idle.Reset(idleAfter)
			return
		}
	}
	if p.ctx != nil && !p.suspended && p.ctx.Suspend() == nil {
		p.suspended = true
	}
}

// open creates the audio context; a machine without a sound device just
// stays silent.
func (p *Player) open() bool {
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate: rate, ChannelCount: 2, Format: oto.FormatFloat32LE,
	})
	if err != nil {
		p.failed = true
		return false
	}
	<-ready
	p.ctx = ctx
	return true
}

// encode turns mono samples into interleaved stereo float32 little-endian.
func encode(s []float64) []byte {
	b := make([]byte, len(s)*8)
	for i, v := range s {
		bits := math.Float32bits(float32(v))
		binary.LittleEndian.PutUint32(b[i*8:], bits)
		binary.LittleEndian.PutUint32(b[i*8+4:], bits)
	}
	return b
}

// tone is a note with an envelope: attack in seconds, then exponential decay.
func tone(dur, freq, freqEnd, attack, decay float64, wave func(phase float64) float64) []float64 {
	n := int(dur * rate)
	out := make([]float64, n)
	phase := 0.0
	for i := range out {
		t := float64(i) / rate
		f := freq + (freqEnd-freq)*t/dur
		phase += f / rate
		env := math.Exp(-t / decay)
		if t < attack {
			env *= t / attack
		}
		out[i] = wave(phase) * env
	}
	return out
}

func sine(p float64) float64 { return math.Sin(2 * math.Pi * p) }
func tri(p float64) float64  { return 4*math.Abs(p-math.Floor(p+0.5)) - 1 }
func square(p float64) float64 {
	if p-math.Floor(p) < 0.5 {
		return 0.6
	}
	return -0.6
}

func mix(parts ...[]float64) []float64 {
	n := 0
	for _, p := range parts {
		n = max(n, len(p))
	}
	out := make([]float64, n)
	for _, p := range parts {
		for i, v := range p {
			out[i] += v
		}
	}
	return out
}

func seq(parts ...[]float64) []float64 {
	var out []float64
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func gain(s []float64, g float64) []float64 {
	for i := range s {
		s[i] *= g
	}
	return s
}

// noise is low-passed white noise with a decay.
func noise(dur, cutoff, decay float64, seed uint64) []float64 {
	r := rand.New(rand.NewPCG(seed, 7))
	n := int(dur * rate)
	out := make([]float64, n)
	a := math.Exp(-2 * math.Pi * cutoff / rate)
	y := 0.0
	for i := range out {
		y = a*y + (1-a)*(r.Float64()*2-1)
		out[i] = y * 3 * math.Exp(-float64(i)/rate/decay)
	}
	return out
}

func synth(e Effect) []float64 {
	switch e {
	case Click:
		return gain(tone(0.03, 1400, 1100, 0.001, 0.008, sine), 0.35)
	case Place:
		return gain(tone(0.06, 520, 760, 0.002, 0.02, tri), 0.35)
	case Zone:
		return gain(seq(tone(0.05, 660, 660, 0.002, 0.02, tri), tone(0.07, 880, 880, 0.002, 0.03, tri)), 0.3)
	case Building:
		return gain(mix(tone(0.22, 160, 70, 0.003, 0.07, sine), noise(0.12, 900, 0.03, 1)), 0.6)
	case Bulldoze:
		return gain(mix(noise(0.25, 600, 0.08, 2), tone(0.18, 90, 60, 0.005, 0.06, square)), 0.45)
	case Error:
		return gain(seq(tone(0.07, 180, 170, 0.003, 0.05, square), make([]float64, rate/30), tone(0.07, 150, 140, 0.003, 0.05, square)), 0.25)
	case Milestone:
		var notes [][]float64
		for _, f := range []float64{523.25, 659.25, 783.99, 1046.5} {
			notes = append(notes, tone(0.12, f, f, 0.004, 0.09, tri))
		}
		tail := tone(0.5, 1046.5, 1046.5, 0.004, 0.2, tri)
		return gain(seq(append(notes, tail)...), 0.35)
	}
	return nil
}
