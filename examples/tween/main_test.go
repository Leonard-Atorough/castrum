package main

import (
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
)

func TestTweenExamplePauseAndRestart(t *testing.T) {
	g, d, err := newGame()
	if err != nil {
		t.Fatalf("newGame() error: %v", err)
	}
	if err := g.Startup(); err != nil {
		t.Fatalf("Startup() error: %v", err)
	}
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("Advance() error: %v", err)
	}
	motion := d.motion.Progress()
	before, ok := d.body.Component[core.Transform](g.World())
	if !ok {
		t.Fatal("body has no Transform")
	}
	if before.Position.X <= 240 {
		t.Error("body did not move on the fixed tick")
	}
	initial, ok := d.swatch.Component[render.Sprite](g.World())
	if !ok {
		t.Fatal("swatch has no Sprite")
	}

	snapshot := &input.Snapshot{}
	snapshot.Keys[input.KeySpace].Pressed = true
	g.Context().Input = snapshot
	if err := g.Advance(250 * time.Millisecond); err != nil {
		t.Fatalf("pause Advance() error: %v", err)
	}
	g.Context().Input = nil
	if !g.Paused() {
		t.Fatal("pause action did not pause the game")
	}
	for i := 0; i < 8; i++ {
		if err := g.Advance(250 * time.Millisecond); err != nil {
			t.Fatalf("paused Advance() error: %v", err)
		}
	}
	if got := d.motion.Progress(); got != motion {
		t.Errorf("paused motion progress = %v, want %v", got, motion)
	}
	after, ok := d.body.Component[core.Transform](g.World())
	if !ok {
		t.Fatal("body lost its Transform")
	}
	if after.Position != before.Position {
		t.Errorf("paused position = %v, want %v", after.Position, before.Position)
	}
	tinted, ok := d.swatch.Component[render.Sprite](g.World())
	if !ok {
		t.Fatal("swatch lost its Sprite")
	}
	if tinted.Color == initial.Color {
		t.Error("frame tint did not change while paused")
	}

	snapshot = &input.Snapshot{}
	snapshot.Keys[input.KeyR].Pressed = true
	g.Context().Input = snapshot
	if err := g.Advance(0); err != nil {
		t.Fatalf("restart Advance() error: %v", err)
	}
	g.Context().Input = nil
	if d.motion.Progress() != 0 || d.tint.Progress() != 0 {
		t.Error("restart did not reset both playheads")
	}
	tinted, _ = d.swatch.Component[render.Sprite](g.World())
	if got := tinted.Color; got != (color.NRGBA{R: 255, G: 180, B: 60, A: 255}) {
		t.Errorf("restarted tint = %v, want initial orange", got)
	}
	after, _ = d.body.Component[core.Transform](g.World())
	if after.Position != before.Position {
		t.Error("restart mutated the simulation transform while paused")
	}

	snapshot = &input.Snapshot{}
	snapshot.Keys[input.KeySpace].Pressed = true
	g.Context().Input = snapshot
	if err := g.Advance(g.Options().FixedDT()); err != nil {
		t.Fatalf("resume Advance() error: %v", err)
	}
	g.Context().Input = nil
	if g.Paused() || d.motion.Progress() <= 0 {
		t.Error("resume did not advance the restarted movement")
	}
}

func TestTweenExampleEndpoints(t *testing.T) {
	g, d, err := newGame()
	if err != nil {
		t.Fatalf("newGame() error: %v", err)
	}
	if err := g.Startup(); err != nil {
		t.Fatalf("Startup() error: %v", err)
	}
	delta := g.Options().FixedDT()
	steps := int(tweenDuration/delta) + 2
	for i := 0; i < steps; i++ {
		if err := g.Advance(delta); err != nil {
			t.Fatalf("step %d: Advance() error: %v", i, err)
		}
	}
	if !d.motion.IsComplete() || !d.tint.IsComplete() {
		t.Fatal("tweens did not complete")
	}
	body, ok := d.body.Component[core.Transform](g.World())
	if !ok {
		t.Fatal("body has no Transform")
	}
	if body.Position.X != 1040 || body.Position.Y != 260 {
		t.Errorf("completed position = %v, want (1040, 260)", body.Position)
	}
	swatch, ok := d.swatch.Component[render.Sprite](g.World())
	if !ok {
		t.Fatal("swatch has no Sprite")
	}
	if got := swatch.Color; got != (color.NRGBA{R: 80, G: 160, B: 255, A: 255}) {
		t.Errorf("completed tint = %v, want final blue", got)
	}
}

func TestTweenExampleSpeedControls(t *testing.T) {
	for _, tt := range []struct {
		name  string
		key   input.Key
		scale float64
		ticks int
	}{
		{"half", input.Key1, 0.5, 0},
		{"normal", input.Key2, 1, 1},
		{"double", input.Key3, 2, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g, d, err := newGame()
			if err != nil {
				t.Fatalf("newGame() error: %v", err)
			}
			if err := g.Startup(); err != nil {
				t.Fatalf("Startup() error: %v", err)
			}
			snapshot := &input.Snapshot{}
			snapshot.Keys[tt.key].Pressed = true
			g.Context().Input = snapshot
			delta := g.Options().FixedDT()
			if err := g.Advance(delta); err != nil {
				t.Fatalf("Advance() error: %v", err)
			}
			if got := g.TimeScale(); got != tt.scale {
				t.Errorf("TimeScale() = %v, want %v", got, tt.scale)
			}
			wantFrame := float64(delta) / float64(tweenDuration)
			wantFrame = 2 * wantFrame * wantFrame
			if got := d.tint.Progress(); math.Abs(got-wantFrame) > 1e-12 {
				t.Errorf("frame progress = %v, want unscaled %v", got, wantFrame)
			}
			wantFixed := float64(time.Duration(tt.ticks)*delta) / float64(tweenDuration)
			wantFixed = 4 * wantFixed * wantFixed * wantFixed
			if got := d.motion.Progress(); math.Abs(got-wantFixed) > 1e-12 {
				t.Errorf("fixed progress = %v, want %v", got, wantFixed)
			}
		})
	}
}
