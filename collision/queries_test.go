package collision

import (
	"math"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func layers2(t *testing.T) uint32 {
	t.Helper()
	layers, err := Layers(2)
	if err != nil {
		t.Fatal(err)
	}
	return layers
}

func TestQueriesRaycastUsesBroadphaseAndTargetLayers(t *testing.T) {
	world := newCollisionWorld(t)
	makeCollider := func(position geom.Vector2, layer int, mask uint32) core.Entity {
		t.Helper()
		collider, err := NewCollider(Circle{Radius: 1})
		if err != nil {
			t.Fatal(err)
		}
		layers, err := Layers(layer)
		if err != nil {
			t.Fatal(err)
		}
		collider.Layers = layers
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
	if err := system(&core.Context{World: world, Tick: 1}); err != nil {
		t.Fatal(err)
	}
	queries, err := world.Resource[*Queries]()
	if err != nil {
		t.Fatal(err)
	}

	segment := geom.Segment{Start: geom.Vector2{X: -3}, End: geom.Vector2{X: 10}}
	mask2, err := Mask(2)
	if err != nil {
		t.Fatal(err)
	}
	hit, ok := queries.Raycast(segment, RaycastFilter{Mask: mask2})
	if !ok || hit.Entity != far || hit.Distance != 6 {
		t.Fatalf("layer-filtered hit = %+v, %t; want entity %d at distance 6", hit, ok, far.ID())
	}
	if hit.Collider.Layers != layers2(t) || hit.Collider.Mask != math.MaxUint32 {
		t.Fatalf("hit collider = %+v; want layers %08b and mask %08b", hit.Collider, layers2(t), uint32(math.MaxUint32))
	}
	shape, ok := hit.Collider.Shape.(Circle)
	if !ok || shape.Radius != 1 {
		t.Fatalf("hit collider shape = %#v; want Circle with radius 1", hit.Collider.Shape)
	}
	mask1, err := Mask(1)
	if err != nil {
		t.Fatal(err)
	}
	hit, ok = queries.Raycast(segment, RaycastFilter{Mask: mask1})
	if !ok || hit.Entity != near {
		t.Fatalf("raycast should hit layer-matched collider with empty Collider.Mask: %+v, %t", hit, ok)
	}
	if _, ok := queries.Raycast(segment, RaycastFilter{}); ok {
		t.Fatal("raycast with empty mask should miss")
	}
	if _, ok := queries.Raycast(segment, RaycastFilter{Mask: mask2, Exclude: []core.Entity{far}}); ok {
		t.Fatal("raycast hit excluded collider")
	}
}

func TestQueriesRaycastLongSegment(t *testing.T) {
	for _, test := range []struct {
		center, radius float64
	}{{500000, 10}, {1e9, 1}} {
		world := newCollisionWorld(t)
		collider, err := NewCollider(Circle{Radius: test.radius})
		if err != nil {
			t.Fatal(err)
		}
		entity, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: test.center}}, collider)
		if err != nil {
			t.Fatal(err)
		}
		if err := NewSystem()(&core.Context{World: world, Tick: 1}); err != nil {
			t.Fatal(err)
		}
		queries, err := world.Resource[*Queries]()
		if err != nil {
			t.Fatal(err)
		}
		hit, ok := queries.Raycast(
			geom.Segment{End: geom.Vector2{X: 2 * test.center}},
			RaycastFilter{Mask: math.MaxUint32},
		)
		want := test.center - test.radius
		if !ok || hit.Entity != entity || math.Abs(hit.Distance-want) > 1e-6 ||
			math.Abs(hit.Point.X-want) > 1e-6 || hit.Normal != (geom.Vector2{X: -1}) {
			t.Fatalf("long-segment hit = %+v, %t; want entity %d at %v", hit, ok, entity.ID(), want)
		}
	}
}
