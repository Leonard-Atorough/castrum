package main

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func TestWanderMovesAndSelectsTargetsOnArrival(t *testing.T) {
	world := core.NewWorld()
	sprite, err := world.NewEntity(core.Transform{})
	if err != nil {
		t.Fatal(err)
	}
	targets := []geom.Vector2{{X: 3, Y: 4}, {X: 9, Y: 12}, {X: 9, Y: 12}}
	calls := 0
	system := wanderSystem(sprite, func() geom.Vector2 {
		if calls >= len(targets) {
			t.Fatal("selected more targets than arrivals")
		}
		target := targets[calls]
		calls++
		return target
	})
	ctx := &core.Context{World: world}
	for _, step := range []struct {
		delta time.Duration
		want  geom.Vector2
		calls int
	}{
		{10 * time.Millisecond, geom.Vector2{X: 0.72, Y: 0.96}, 1},
		{100 * time.Millisecond, targets[0], 2},
		{0, targets[0], 2},
		{100 * time.Millisecond, targets[1], 3},
	} {
		ctx.DeltaTime = step.delta
		if err := system.Update(ctx); err != nil {
			t.Fatal(err)
		}
		got, ok := sprite.Component[core.Transform](world)
		if !ok || !got.Position.AlmostEqual(step.want, 1e-9) || calls != step.calls {
			t.Fatalf("delta %v: position=%v targets=%d, want %v and %d", step.delta, got.Position, calls, step.want, step.calls)
		}
	}
}

func TestWanderAlreadyAtTarget(t *testing.T) {
	world := core.NewWorld()
	sprite, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 5, Y: 8}})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	system := wanderSystem(sprite, func() geom.Vector2 {
		calls++
		return geom.Vector2{X: 5, Y: 8}
	})
	if err := system.Update(&core.Context{World: world}); err != nil {
		t.Fatal(err)
	}
	got, _ := sprite.Component[core.Transform](world)
	if got.Position != (geom.Vector2{X: 5, Y: 8}) || calls != 2 {
		t.Fatalf("stationary arrival: position=%v targets=%d", got.Position, calls)
	}
}
