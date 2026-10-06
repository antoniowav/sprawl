package app

import (
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Frame pacing. In low-power mode Ebitengine only runs a frame on new input
// or when ScheduleFrame is called, so a resting game costs almost nothing.
// The pacer asks for frames at the rate the game currently needs.
//
// FPSModeVsyncOffMinimum is marked deprecated in Ebitengine 2.5+, but it is
// still implemented on desktop in 2.10 and is the only way to stop the
// 60 Hz loop entirely. config: low_power = false falls back to plain vsync.
const (
	hzActive = 60 // keys held, camera easing, mouse dragging
	hzGrow   = 10 // buildings rising or cars driving: smooth enough for pixel art
	hzAnim   = 8  // animations visible (they step at 4 fps)
	hzIdle   = 2  // theme polling and message timeouts
)

type pacer struct{ hz atomic.Int32 }

func startPacer() *pacer {
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMinimum) //nolint:staticcheck // see comment above
	p := &pacer{}
	p.hz.Store(hzActive)
	go func() {
		for {
			time.Sleep(time.Second / time.Duration(p.hz.Load()))
			ebiten.ScheduleFrame() //nolint:staticcheck
		}
	}()
	return p
}

// set picks the rate for the coming frames. simRate is the sim's ticks per
// second (0 when paused); frames run at twice that so ticks land evenly.
func (p *pacer) set(active, growing, animating bool, simRate float64) {
	if p == nil {
		return
	}
	hz := int32(hzIdle)
	switch {
	case active || anyInputHeld():
		hz = hzActive
	case growing:
		hz = hzGrow
	case animating:
		hz = hzAnim
	}
	hz = max(hz, min(hzActive, int32(simRate*2)))
	p.hz.Store(hz)
}

func anyInputHeld() bool {
	if len(inpututil.AppendPressedKeys(nil)) > 0 {
		return true
	}
	for _, b := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight, ebiten.MouseButtonMiddle} {
		if ebiten.IsMouseButtonPressed(b) {
			return true
		}
	}
	return false
}
