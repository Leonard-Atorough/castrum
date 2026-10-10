package castrum

import (
	"math"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
)

func TestPauseStopsFixedOnlyFrameContinues(t *testing.T) {
	// FixedTPS 4 => fixedDT = 250ms.
	g, _ := New(WithFixedTPS(4))
	var ticks, frames int
	var frameDT time.Duration
	g.AddSystem(core.PhaseFrame, "frame counter", core.SystemFunc(func(ctx *core.Context) error {
		frames++
		frameDT = ctx.DeltaTime
		return nil
	}))
	g.AddSystem(core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error {
		ticks++
		return nil
	}))

	step := 100 * time.Millisecond
	g.Advance(step)
	if ticks != 0 || frames != 1 {
		t.Fatalf("before pause: ticks=%d frames=%d, want 0/1", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.4) > 1e-9 {
		t.Fatalf("alpha before pause = %v, want 0.4", alpha)
	}

	g.Pause()
	if !g.Paused() {
		t.Fatal("Paused = false after Pause")
	}
	g.Advance(step)
	g.Advance(step)
	if ticks != 0 || frames != 3 {
		t.Fatalf("while paused: ticks=%d frames=%d, want 0/3 - frame phase must continue", ticks, frames)
	}
	if frameDT != step {
		t.Errorf("frame DeltaTime while paused = %v, want %v (unscaled wall-clock time)", frameDT, step)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.4) > 1e-9 {
		t.Errorf("alpha while paused = %v, want 0.4 (frozen)", alpha)
	}

	g.Resume()
	if g.Paused() {
		t.Fatal("Paused = true after Resume")
	}
	// The accumulator was frozen at 100ms; resuming continues from there,
	// so the tick that was half-due before the pause still needs the rest
	// of its interval.
	g.Advance(step)
	if ticks != 0 || frames != 4 {
		t.Fatalf("after resume: ticks=%d frames=%d, want 0/4 (paused time is not recovered)", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.8) > 1e-9 {
		t.Errorf("alpha after resume = %v, want 0.8 (frozen 0.4 plus one frame)", alpha)
	}
	g.Advance(step)
	if ticks != 1 || frames != 5 {
		t.Fatalf("tick after resume: ticks=%d frames=%d, want 1/5", ticks, frames)
	}
}

func TestPauseSkipsSpiralGuardAndClamp(t *testing.T) {
	// MaxFrameTime 50ms and a 10s stall while paused: no clamping side
	// effects, no ticks, alpha stays zero.
	g, _ := New(WithFixedTPS(1), WithMaxFrameTime(50*time.Millisecond))
	var ticks int
	g.AddSystem(core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error { ticks++; return nil }))
	g.Advance(10 * time.Second)
	if ticks != 0 {
		t.Fatalf("ticks before pause = %d, want 0", ticks)
	}

	g.Pause()
	g.Advance(10 * time.Second)
	if ticks != 0 {
		t.Errorf("ticks while paused = %d, want 0", ticks)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.05) > 1e-9 {
		t.Errorf("alpha while paused = %v, want 0.05 (frozen at its pre-pause value)", alpha)
	}
}

func TestTimeScaleScalesAccumulationOnly(t *testing.T) {
	// FixedTPS 10 => fixedDT = 100ms.
	g, _ := New(WithFixedTPS(10))
	var ticks, frames int
	var frameDT time.Duration
	g.AddSystem(core.PhaseFrame, "frame counter", core.SystemFunc(func(ctx *core.Context) error {
		frames++
		frameDT = ctx.DeltaTime
		return nil
	}))
	g.AddSystem(core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error { ticks++; return nil }))

	if err := g.SetTimeScale(0.5); err != nil {
		t.Fatalf("SetTimeScale(0.5): %v", err)
	}
	step := 100 * time.Millisecond
	g.Advance(step)
	if ticks != 0 || frames != 1 {
		t.Fatalf("half speed, first advance: ticks=%d frames=%d, want 0/1", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha-0.5) > 1e-9 {
		t.Errorf("alpha at half speed = %v, want 0.5", alpha)
	}
	if frameDT != step {
		t.Errorf("frame DeltaTime at half speed = %v, want %v (frame time is never scaled)", frameDT, step)
	}

	g.Advance(step)
	if ticks != 1 || frames != 2 {
		t.Fatalf("half speed, second advance: ticks=%d frames=%d, want 1/2", ticks, frames)
	}
	if alpha := g.Alpha(); math.Abs(alpha) > 1e-9 {
		t.Errorf("alpha after tick = %v, want 0", alpha)
	}
}

func TestTimeScaleFastForwardRespectsSpiralGuard(t *testing.T) {
	// FixedTPS 100 => fixedDT = 10ms; scale 10 turns a clamped 250ms frame
	// into 2.5s due - capped at MaxTicksPerFrame ticks, backlog dropped.
	g, _ := New(WithFixedTPS(100), WithMaxFrameTime(time.Second))
	var ticks int
	g.AddSystem(core.PhaseFixed, "tick counter", core.SystemFunc(func(ctx *core.Context) error { ticks++; return nil }))

	if err := g.SetTimeScale(10); err != nil {
		t.Fatalf("SetTimeScale(10): %v", err)
	}
	g.Advance(time.Second)
	if ticks != 5 {
		t.Errorf("ticks = %d, want 5 (fast forward still capped at MaxTicksPerFrame)", ticks)
	}
	if alpha := g.Alpha(); alpha != 0 {
		t.Errorf("alpha = %v, want 0 (backlog dropped)", alpha)
	}
}

func TestSetTimeScaleRejectsNonPositive(t *testing.T) {
	g, _ := New()
	if g.TimeScale() != 1 {
		t.Fatalf("default TimeScale = %v, want 1", g.TimeScale())
	}
	for _, bad := range []float64{0, -0.5, -2} {
		if err := g.SetTimeScale(bad); err == nil {
			t.Errorf("SetTimeScale(%v) should return an error", bad)
		}
	}
	if g.TimeScale() != 1 {
		t.Errorf("TimeScale after rejected values = %v, want 1 (unchanged)", g.TimeScale())
	}

	g.SetTimeScale(2)
	if g.TimeScale() != 2 {
		t.Errorf("TimeScale = %v, want 2", g.TimeScale())
	}

	// Scale is remembered while paused and applies after Resume.
	g.Pause()
	g.SetTimeScale(0.25)
	g.Resume()
	if g.TimeScale() != 0.25 {
		t.Errorf("TimeScale after pause/resume = %v, want 0.25", g.TimeScale())
	}
}
