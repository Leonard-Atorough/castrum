package main

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
)

func TestWheelZoomRunsOncePerFrame(t *testing.T) {
	for _, tt := range []struct {
		name    string
		elapsed time.Duration
		paused  bool
		ticks   int
	}{
		{"no fixed ticks", 0, false, 0},
		{"one fixed tick", 10 * time.Millisecond, false, 1},
		{"catch-up ticks", 30 * time.Millisecond, false, 3},
		{"paused simulation", 30 * time.Millisecond, true, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g, err := castrum.New(castrum.WithFixedTPS(100))
			if err != nil {
				t.Fatal(err)
			}
			camera := g.MainCamera()
			if err := camera.Update(g.World(), func(c *core.Transform) {
				c.Scale = geom.Vector2{X: 1, Y: 1}
			}); err != nil {
				t.Fatal(err)
			}
			if err := g.AddSystem(core.PhaseFrame, "zoom", zoomSystem(camera)); err != nil {
				t.Fatal(err)
			}
			ticks := 0
			if err := g.AddSystem(core.PhaseFixed, "counter", core.SystemFunc(func(ctx *core.Context) error {
				ticks++
				return nil
			})); err != nil {
				t.Fatal(err)
			}
			if err := g.Startup(); err != nil {
				t.Fatal(err)
			}
			if tt.paused {
				g.Pause()
			}
			snapshot := &input.Snapshot{}
			snapshot.Mouse.Wheel = geom.Vector2{X: 0.25, Y: 2.5}
			g.Context().Input = snapshot
			if err := g.Advance(tt.elapsed); err != nil {
				t.Fatal(err)
			}
			if ticks != tt.ticks {
				t.Fatalf("fixed ticks = %d, want %d", ticks, tt.ticks)
			}
			want := geom.Vector2{X: 1.25, Y: 1.25}
			if got, ok := camera.Component[core.Transform](g.World()); !ok || got.Scale != want {
				t.Fatalf("zoom scale = %v, want %v after one frame", got.Scale, want)
			}
			g.Resume()
			snapshot.Mouse.Wheel = geom.Vector2{}
			if err := g.Advance(30 * time.Millisecond); err != nil {
				t.Fatal(err)
			}
			if got, ok := camera.Component[core.Transform](g.World()); !ok || got.Scale != want {
				t.Fatalf("zero-scroll update reapplied movement: scale=%v", got.Scale)
			}
		})
	}
}
