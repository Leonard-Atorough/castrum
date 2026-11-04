package collision

import (
	"math"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func TestBroadphaseSyncIsTransactional(t *testing.T) {
	world := newCollisionWorld(t)
	collider, err := NewCollider(Circle{Radius: 1})
	if err != nil {
		t.Fatal(err)
	}
	target, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 1000}}, collider)
	if err != nil {
		t.Fatal(err)
	}
	invalid, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: math.MaxFloat64}}, collider)
	if err != nil {
		t.Fatal(err)
	}
	system := NewSystem()
	queries, err := world.Resource[*Queries]()
	if err != nil {
		t.Fatal(err)
	}

	if hit, ok := queries.Raycast(pointSegment(geom.Vector2{X: 1000}), RaycastFilter{Mask: math.MaxUint32}); ok {
		t.Fatalf("raycast before first sync hit entity %d", hit.Entity)
	}
	if err := system.Update(&core.Context{World: world, Tick: 1}); err == nil {
		t.Fatal("sync with an unindexable transform should fail")
	}
	if queries.Tick() != 0 {
		t.Fatalf("failed initial sync published tick %d", queries.Tick())
	}

	if err := invalid.SetComponent(world, core.Transform{Position: geom.Vector2{X: 2000}}); err != nil {
		t.Fatal(err)
	}
	if err := system.Update(&core.Context{World: world, Tick: 1}); err != nil {
		t.Fatalf("initial valid sync: %v", err)
	}
	if queries.Tick() != 1 {
		t.Fatalf("published tick = %d, want 1", queries.Tick())
	}
	if hit, ok := queries.Raycast(pointSegment(geom.Vector2{X: 1000}), RaycastFilter{Mask: math.MaxUint32}); !ok || hit.Entity != target.ID() {
		t.Fatalf("initial raycast = %+v, %t; want target %d", hit, ok, target.ID())
	}

	if err := target.SetComponent(world, core.Transform{}); err != nil {
		t.Fatal(err)
	}
	if err := invalid.SetComponent(world, core.Transform{Position: geom.Vector2{X: math.MaxFloat64}}); err != nil {
		t.Fatal(err)
	}
	if err := system.Update(&core.Context{World: world, Tick: 2}); err == nil {
		t.Fatal("sync with a non-finite transform should fail")
	}
	if queries.Tick() != 1 {
		t.Fatalf("failed sync published tick %d, want previous tick 1", queries.Tick())
	}
	if hit, ok := queries.Raycast(pointSegment(geom.Vector2{X: 1000}), RaycastFilter{Mask: math.MaxUint32}); !ok || hit.Entity != target.ID() {
		t.Fatalf("raycast after failed sync = %+v, %t; want previous target %d", hit, ok, target.ID())
	}
	if _, ok := queries.Raycast(pointSegment(geom.Vector2{}), RaycastFilter{Mask: math.MaxUint32}); ok {
		t.Fatal("failed sync exposed staged collider state")
	}

	if err := invalid.SetComponent(world, core.Transform{Position: geom.Vector2{X: 2000}}); err != nil {
		t.Fatal(err)
	}
	if err := system.Update(&core.Context{World: world, Tick: 2}); err != nil {
		t.Fatalf("retry sync: %v", err)
	}
	if queries.Tick() != 2 {
		t.Fatalf("retried sync published tick %d, want 2", queries.Tick())
	}
	if hit, ok := queries.Raycast(pointSegment(geom.Vector2{}), RaycastFilter{Mask: math.MaxUint32}); !ok || hit.Entity != target.ID() {
		t.Fatalf("raycast after successful retry = %+v, %t; want target %d", hit, ok, target.ID())
	}
	if _, ok := queries.Raycast(pointSegment(geom.Vector2{X: 1000}), RaycastFilter{Mask: math.MaxUint32}); ok {
		t.Fatal("successful retry retained target's old bounds")
	}
}

func pointSegment(point geom.Vector2) geom.Segment {
	return geom.Segment{Start: point, End: point}
}
