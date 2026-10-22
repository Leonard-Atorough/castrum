package collision

import (
	"math"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func TestSystem_RetryAfterInvalidTransformRemovesNewProxies(t *testing.T) {
	world := newCollisionWorld(t)
	sys := NewSystem().(*system)
	collider, err := NewCollider(Circle{Radius: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 1000}}, collider); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys)
	transient, err := world.NewEntity(core.Transform{}, collider)
	if err != nil {
		t.Fatal(err)
	}
	invalid, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: math.NaN()}}, collider)
	if err != nil {
		t.Fatal(err)
	}
	if err := sys.Update(&core.Context{World: world}); err == nil {
		t.Fatal("non-finite transform should fail the tick")
	}
	if _, ok := sys.proxies[transient.ID()]; !ok {
		t.Fatal("setup: transient proxy was not indexed before the failure")
	}
	if err := world.DestroyEntity(transient); err != nil {
		t.Fatal(err)
	}
	if err := invalid.SetComponent(world, core.Transform{Position: geom.Vector2{X: 2000}}); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys)
	if len(sys.proxies) != 2 {
		t.Fatalf("retry left stale proxies: %v", sys.proxies)
	}
	bounds := geom.Rect{Min: geom.Vector2{X: -2, Y: -2}, Max: geom.Vector2{X: 2, Y: 2}}
	if ids := sys.grid.QueryOverlapsInto(bounds, nil); len(ids) != 0 {
		t.Fatalf("retry left stale grid entries: %v", ids)
	}
}

func TestSystem_RetryAfterInvalidTransformReplaysDirtyProxies(t *testing.T) {
	for _, change := range []string{"spawn", "position", "shape", "filter"} {
		t.Run(change, func(t *testing.T) {
			world := newCollisionWorld(t)
			sys := NewSystem().(*system)
			collider, err := NewCollider(Circle{Radius: 1})
			if err != nil {
				t.Fatal(err)
			}
			wall, err := world.NewEntity(core.Transform{}, collider)
			if err != nil {
				t.Fatal(err)
			}
			var target *core.Entity
			if change != "spawn" {
				transform := core.Transform{Position: geom.Vector2{X: 10}}
				initial := collider
				if change == "filter" {
					transform.Position.X = 0
					initial.Mask = 0
				}
				target, err = world.NewEntity(transform, initial)
				if err != nil {
					t.Fatal(err)
				}
			}
			tick(t, world, sys)
			if c := contactsOf(t, world, wall); len(c.Current) != 0 {
				t.Fatalf("setup: wall should have no contacts, got %v", c)
			}
			switch change {
			case "spawn":
				target, err = world.NewEntity(core.Transform{}, collider)
			case "position":
				err = target.SetComponent(world, core.Transform{})
			case "shape":
				err = target.Update(world, func(c *Collider) { c.Shape = Circle{Radius: 10} })
			case "filter":
				err = target.Update(world, func(c *Collider) { c.Mask = 1 })
			}
			if err != nil {
				t.Fatal(err)
			}
			invalid, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: math.NaN()}}, collider)
			if err != nil {
				t.Fatal(err)
			}
			ctx := &core.Context{World: world}
			for range 2 {
				if err := sys.Update(ctx); err == nil {
					t.Fatal("non-finite transform should fail the tick")
				}
				p := sys.proxies[target.ID()]
				transform, _ := target.Component[core.Transform](world)
				current, _ := target.Component[Collider](world)
				if p == nil || p.position != transform.Position || p.collider != current {
					t.Fatal("setup: target inputs were not cached before the failure")
				}
			}
			if err := invalid.SetComponent(world, core.Transform{Position: geom.Vector2{X: 1000}}); err != nil {
				t.Fatal(err)
			}
			tick(t, world, sys)
			for _, entity := range []*core.Entity{target, wall} {
				c := contactsOf(t, world, entity)
				if len(c.Current) != 1 || len(c.Previous) != 0 {
					t.Fatalf("retry should discover an enter, got %v", c)
				}
			}
			tick(t, world, sys)
			if len(sys.dirty) != 0 {
				t.Fatalf("successful discovery left pending dirty work: %v", sys.dirty)
			}
			for _, entity := range []*core.Entity{target, wall} {
				c := contactsOf(t, world, entity)
				if len(c.Current) != 1 || len(c.Previous) != 1 {
					t.Fatalf("following tick should report stay, got %v", c)
				}
			}
		})
	}
}
