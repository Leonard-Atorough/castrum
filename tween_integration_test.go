package castrum

import (
	"math"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/tween"
)

func TestTweenSchedulerClocks(t *testing.T) {
	t.Parallel()

	type step struct {
		delta      time.Duration
		pause      bool
		resume     bool
		scale      float64
		fixedTime  float64
		frameTime  float64
		fixedCalls int
	}
	tests := []struct {
		name     string
		maxTicks int
		steps    []step
	}{
		{
			name: "normal accumulation",
			steps: []step{
				{delta: 125 * time.Millisecond, frameTime: 0.125},
				{delta: 125 * time.Millisecond, fixedTime: 0.25, frameTime: 0.25, fixedCalls: 1},
				{delta: 250 * time.Millisecond, fixedTime: 0.5, frameTime: 0.5, fixedCalls: 2},
			},
		},
		{
			name: "pause and resume preserve remainder",
			steps: []step{
				{delta: 125 * time.Millisecond, frameTime: 0.125},
				{delta: 250 * time.Millisecond, pause: true, frameTime: 0.375},
				{delta: 250 * time.Millisecond, frameTime: 0.625},
				{delta: 125 * time.Millisecond, resume: true, fixedTime: 0.25, frameTime: 0.75, fixedCalls: 1},
				{delta: 250 * time.Millisecond, fixedTime: 0.5, frameTime: 1, fixedCalls: 2},
			},
		},
		{
			name: "half speed",
			steps: []step{
				{delta: 250 * time.Millisecond, scale: 0.5, frameTime: 0.25},
				{delta: 250 * time.Millisecond, fixedTime: 0.25, frameTime: 0.5, fixedCalls: 1},
				{delta: 250 * time.Millisecond, fixedTime: 0.25, frameTime: 0.75, fixedCalls: 1},
				{delta: 250 * time.Millisecond, fixedTime: 0.5, frameTime: 1, fixedCalls: 2},
			},
		},
		{
			name: "double speed",
			steps: []step{
				{delta: 125 * time.Millisecond, scale: 2, fixedTime: 0.25, frameTime: 0.125, fixedCalls: 1},
				{delta: 125 * time.Millisecond, fixedTime: 0.5, frameTime: 0.25, fixedCalls: 2},
				{delta: 250 * time.Millisecond, fixedTime: 1, frameTime: 0.5, fixedCalls: 4},
				{delta: 250 * time.Millisecond, fixedTime: 1, frameTime: 0.75, fixedCalls: 6},
				{delta: 250 * time.Millisecond, fixedTime: 1, frameTime: 1, fixedCalls: 8},
				{delta: 250 * time.Millisecond, fixedTime: 1, frameTime: 1, fixedCalls: 10},
			},
		},
		{
			name: "scale change preserves remainder",
			steps: []step{
				{delta: 250 * time.Millisecond, scale: 0.5, frameTime: 0.25},
				{delta: 125 * time.Millisecond, scale: 2, fixedTime: 0.25, frameTime: 0.375, fixedCalls: 1},
				{delta: 125 * time.Millisecond, fixedTime: 0.5, frameTime: 0.5, fixedCalls: 2},
			},
		},
		{
			name: "frame time clamp",
			steps: []step{
				{delta: 5 * time.Second, fixedTime: 0.25, frameTime: 0.25, fixedCalls: 1},
				{delta: 250 * time.Millisecond, fixedTime: 0.5, frameTime: 0.5, fixedCalls: 2},
			},
		},
		{
			name:     "tick limit discards backlog",
			maxTicks: 1,
			steps: []step{
				{delta: 250 * time.Millisecond, scale: 4, fixedTime: 0.25, frameTime: 0.25, fixedCalls: 1},
				{delta: 0, fixedTime: 0.25, frameTime: 0.25, fixedCalls: 1},
				{delta: 250 * time.Millisecond, scale: 1, fixedTime: 0.5, frameTime: 0.5, fixedCalls: 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := []option{WithFixedTPS(4)}
			if tt.maxTicks > 0 {
				opts = append(opts, WithMaxTicksPerFrame(tt.maxTicks))
			}
			g, err := New(opts...)
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}
			fixed, err := tween.NewTween(time.Second, geom.EaseInQuad)
			if err != nil {
				t.Fatalf("fixed NewTween() error: %v", err)
			}
			frame, err := tween.NewTween(time.Second, geom.EaseInQuad)
			if err != nil {
				t.Fatalf("frame NewTween() error: %v", err)
			}
			var fixedCalls, frameCalls int
			if err := g.AddSystem(core.PhaseFixed, "tween.fixed", core.SystemFunc(func(ctx *core.Context) error {
				fixedCalls++
				if ctx.DeltaTime != 250*time.Millisecond {
					t.Errorf("fixed DeltaTime = %v, want 250ms", ctx.DeltaTime)
				}
				_, err := fixed.Update(ctx.DeltaTime)
				return err
			})); err != nil {
				t.Fatalf("AddSystem(fixed) error: %v", err)
			}
			if err := g.AddSystem(core.PhaseFrame, "tween.frame", core.SystemFunc(func(ctx *core.Context) error {
				frameCalls++
				_, err := frame.Update(ctx.DeltaTime)
				return err
			})); err != nil {
				t.Fatalf("AddSystem(frame) error: %v", err)
			}
			if err := g.Startup(); err != nil {
				t.Fatalf("Startup() error: %v", err)
			}

			for i, step := range tt.steps {
				if step.pause {
					g.Pause()
				}
				if step.resume {
					g.Resume()
				}
				if step.scale > 0 {
					if err := g.SetTimeScale(step.scale); err != nil {
						t.Fatalf("step %d: SetTimeScale() error: %v", i, err)
					}
				}
				if err := g.Advance(step.delta); err != nil {
					t.Fatalf("step %d: Advance() error: %v", i, err)
				}
				if fixedCalls != step.fixedCalls || frameCalls != i+1 {
					t.Errorf("step %d: fixed/frame calls = %d/%d, want %d/%d",
						i, fixedCalls, frameCalls, step.fixedCalls, i+1)
				}
				for _, clock := range []struct {
					name  string
					tween *tween.Tween
					time  float64
				}{
					{"fixed", &fixed, step.fixedTime},
					{"frame", &frame, step.frameTime},
				} {
					want := clock.time * clock.time
					if got := clock.tween.Progress(); math.Abs(got-want) > 1e-12 {
						t.Errorf("step %d: %s Progress() = %v, want %v", i, clock.name, got, want)
					}
					if got := clock.tween.IsComplete(); got != (clock.time == 1) {
						t.Errorf("step %d: %s IsComplete() = %v, want %v", i, clock.name, got, clock.time == 1)
					}
					if got := clock.tween.Progress(); math.Abs(got-want) > 1e-12 {
						t.Errorf("step %d: observing %s changed progress to %v", i, clock.name, got)
					}
				}
			}
		})
	}
}
