package collision

import (
	"math"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func TestQueriesRaycastUsesBroadphaseAndTargetLayers(t *testing.T) {
	world := newCollisionWorld(t)
	makeCollider := func(position geom.Vector2, layer int, mask uint32) *core.Entity {
		t.Helper()
		collider, err := NewCollider(Circle{Radius: 1})
		if err != nil {
			t.Fatal(err)
		}
		collider.Layers = Layers(layer)
		collider.Mask = mask
		entity, err := world.NewEntity(core.Transform{Position: position}, collider)
		if err != nil {
			t.Fatal(err)
		}
		return entity
	}

	near := makeCollider(geom.Vector2{}, 1, 0)
	far := makeCollider(geom.Vector2{X: 4}, 2, math.MaxUint32)
	system := NewSystem()
	if err := system.Update(&core.Context{World: world, Tick: 1}); err != nil {
		t.Fatal(err)
	}
	queries, err := world.Resource[*Queries]()
	if err != nil {
		t.Fatal(err)
	}

	segment := geom.Segment{Start: geom.Vector2{X: -3}, End: geom.Vector2{X: 10}}
	hit, ok := queries.Raycast(segment, RaycastFilter{Mask: Mask(2)})
	if !ok || hit.Entity != far.ID() || hit.Distance != 6 {
		t.Fatalf("layer-filtered hit = %+v, %t; want entity %d at distance 6", hit, ok, far.ID())
	}
	if hit.Collider.Layers != Layers(2) || hit.Collider.Mask != math.MaxUint32 {
		t.Fatalf("hit collider = %+v; want layers %08b and mask %08b", hit.Collider, Layers(2), uint32(math.MaxUint32))
	}
	shape, ok := hit.Collider.Shape.(Circle)
	if !ok || shape.Radius != 1 {
		t.Fatalf("hit collider shape = %#v; want Circle with radius 1", hit.Collider.Shape)
	}
	hit, ok = queries.Raycast(segment, RaycastFilter{Mask: Mask(1)})
	if !ok || hit.Entity != near.ID() {
		t.Fatalf("raycast should hit layer-matched collider with empty Collider.Mask: %+v, %t", hit, ok)
	}
	if _, ok := queries.Raycast(segment, RaycastFilter{}); ok {
		t.Fatal("raycast with empty mask should miss")
	}
	if _, ok := queries.Raycast(segment, RaycastFilter{Mask: Mask(2), Exclude: []core.EntityID{far.ID()}}); ok {
		t.Fatal("raycast hit excluded collider")
	}
}

func TestQueriesRaycastLongSegment(t *testing.T) {
	world := newCollisionWorld(t)
	collider, err := NewCollider(Circle{Radius: 10})
	if err != nil {
		t.Fatal(err)
	}
	entity, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 500000}}, collider)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewSystem().Update(&core.Context{World: world, Tick: 1}); err != nil {
		t.Fatal(err)
	}
	queries, err := world.Resource[*Queries]()
	if err != nil {
		t.Fatal(err)
	}

	hit, ok := queries.Raycast(
		geom.Segment{Start: geom.Vector2{}, End: geom.Vector2{X: 1000000}},
		RaycastFilter{Mask: math.MaxUint32},
	)
	if !ok || hit.Entity != entity.ID() {
		t.Fatalf("long-segment hit = %+v, %t; want entity %d", hit, ok, entity.ID())
	}
}
