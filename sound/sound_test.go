package sound

import (
	"math"
	"testing"
)

func TestEffectsAreShortAndClean(t *testing.T) {
	for e := Effect(0); e < effectCount; e++ {
		s := synth(e)
		if len(s) == 0 || len(s) > 2*rate {
			t.Errorf("effect %d: %d samples", e, len(s))
		}
		for _, v := range s {
			if math.IsNaN(v) || math.Abs(v) > 1 {
				t.Fatalf("effect %d clips or is NaN: %v", e, v)
			}
		}
	}
}

func TestMutedNeverOpensDevice(t *testing.T) {
	p := New(0)
	p.Play(Click)
	if p.ctx != nil {
		t.Fatal("muted player opened the audio device")
	}
}
