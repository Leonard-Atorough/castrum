package main

import (
	"math"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
)

func TestTimeControlsPauseScaleAndResumeOrbit(t *testing.T) {
	g, err := castrum.New(castrum.WithFixedTPS(100), castrum.WithBindings(timeBindings()))
	if err != nil {
		t.Fatal(err)
	}
	inner, err := g.World().NewEntity(core.Transform{})
	if err != nil {
		t.Fatal(err)
	}
	outer, err := g.World().NewEntity(core.Transform{})
	if err != nil {
		t.Fatal(err)
	}
	state, err := g.World().NewEntity(render.Sprite{})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystem(core.PhaseFixed, "orbit", orbitSystem(inner, outer)); err != nil {
		t.Fatal(err)
	}
	if err := g.AddSystem(core.PhaseFrame, "time control", timeControlSystem(g, state)); err != nil {
		t.Fatal(err)
	}
	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	if err := g.Advance(10 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	checkOrbits := func(angle float64) {
		t.Helper()
		center := geom.Vector2{X: screenW / 2, Y: screenH / 2}
		for _, orbit := range []struct {
			entity *core.Entity
			want   geom.Vector2
		}{
			{inner, center.Add(geom.Vector2{X: 140 * math.Cos(angle), Y: 140 * math.Sin(angle)})},
			{outer, center.Add(geom.Vector2{X: 240 * math.Cos(angle/2), Y: 240 * math.Sin(angle/2)})},
		} {
			transform, _ := orbit.entity.Component[core.Transform](g.World())
			if !transform.Position.AlmostEqual(orbit.want, 1e-9) {
				t.Fatalf("orbit position=%v, want %v", transform.Position, orbit.want)
			}
		}
	}
	checkOrbits(0.015)
	for _, step := range []struct {
		key     input.Key
		pressed bool
		delta   time.Duration
		scale   float64
		text    string
		angle   float64
	}{
		{input.KeySpace, true, 10 * time.Millisecond, 1, "1.00x, paused", 0.015},
		{input.KeyArrowUp, false, 200 * time.Millisecond, 1.3, "1.30x, paused", 0.015},
		{input.KeyW, false, 200 * time.Millisecond, 1.6, "1.60x, paused", 0.015},
		{input.KeyP, true, 25 * time.Millisecond, 1.6, "1.60x, running", 0.075},
	} {
		snapshot := &input.Snapshot{}
		snapshot.Keys[step.key] = input.State{Held: true, Pressed: step.pressed}
		g.Context().Input = snapshot
		if err := g.Advance(step.delta); err != nil {
			t.Fatal(err)
		}
		if math.Abs(g.TimeScale()-step.scale) > 1e-9 {
			t.Fatalf("scale=%v, want %v", g.TimeScale(), step.scale)
		}
		sprite, _ := state.Component[render.Sprite](g.World())
		source, ok := sprite.Drawable.(render.TextSource)
		if !ok || source.Text != step.text {
			t.Fatalf("status=%v, want %q", sprite.Drawable, step.text)
		}
		checkOrbits(step.angle)
	}
}

func TestTimeScaleControlsClampWhilePaused(t *testing.T) {
	for _, step := range []struct {
		key   input.Key
		scale float64
		want  float64
	}{
		{input.KeyArrowUp, 3.9, 4},
		{input.KeyArrowDown, 0.3, 0.25},
		{input.KeyS, 1, 0.7},
	} {
		g, err := castrum.New(castrum.WithBindings(timeBindings()))
		if err != nil {
			t.Fatal(err)
		}
		state, err := g.World().NewEntity(render.Sprite{})
		if err != nil {
			t.Fatal(err)
		}
		if err := g.AddSystem(core.PhaseFrame, "time control", timeControlSystem(g, state)); err != nil {
			t.Fatal(err)
		}
		if err := g.Startup(); err != nil {
			t.Fatal(err)
		}
		g.Pause()
		if err := g.SetTimeScale(step.scale); err != nil {
			t.Fatal(err)
		}
		snapshot := &input.Snapshot{}
		snapshot.Keys[step.key].Held = true
		g.Context().Input = snapshot
		if err := g.Advance(200 * time.Millisecond); err != nil {
			t.Fatal(err)
		}
		if !g.Paused() || math.Abs(g.TimeScale()-step.want) > 1e-9 {
			t.Fatalf("scale=%v paused=%t, want %v while paused", g.TimeScale(), g.Paused(), step.want)
		}
	}
}
