package collision

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

func TestSystem_TouchingAcrossCellBoundary(t *testing.T) {
	world := newCollisionWorld(t)
	sys := NewSystem()
	collider, err := NewCollider(Circle{Radius: 1})
	if err != nil {
		t.Fatal(err)
	}
	a, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 49}}, collider)
	if err != nil {
		t.Fatal(err)
	}
	b, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 52}}, collider)
	if err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys)
	if err := b.SetComponent(world, core.Transform{Position: geom.Vector2{X: 51}}); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys)
	c := contactsOf(t, world, a)
	if len(c.Current) != 1 || c.Current[0].Penetration != 0 {
		t.Fatalf("touching pair was culled: %v", c)
	}
}

func TestSystem_AllocationsFollowMovement(t *testing.T) {
	for _, n := range []int{100, 1000} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			world := newCollisionWorld(t)
			sys := NewSystem()
			collider, err := NewCollider(Circle{Radius: 1})
			if err != nil {
				t.Fatal(err)
			}
			var moving *core.Entity
			for i := range n {
				entity, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: float64(i) * 100}}, collider)
				if err != nil {
					t.Fatal(err)
				}
				moving = entity
			}
			ctx := &core.Context{World: world}
			for range 4 {
				tick(t, world, sys)
			}
			if allocs := testing.AllocsPerRun(20, func() {
				if err := sys.Update(ctx); err != nil {
					t.Fatal(err)
				}
			}); allocs != 0 {
				t.Fatalf("settled world: %v allocations, want zero", allocs)
			}
			base := float64(n-1) * 100
			offset := 0.0
			if allocs := testing.AllocsPerRun(20, func() {
				offset = 0.5 - offset
				if err := moving.SetComponent(world, core.Transform{Position: geom.Vector2{X: base + offset}}); err != nil {
					t.Fatal(err)
				}
				if err := sys.Update(ctx); err != nil {
					t.Fatal(err)
				}
			}); allocs > 3 {
				t.Fatalf("one moving collider: %v allocations, want at most 3 independent of population", allocs)
			}
		})
	}
}

func TestSystem_SettledContactsAllocateNothing(t *testing.T) {
	world := newCollisionWorld(t)
	sys := NewSystem()
	player, _ := playerAndWall(t, world)
	ctx := &core.Context{World: world}
	for range 4 {
		tick(t, world, sys)
	}
	if allocs := testing.AllocsPerRun(20, func() {
		if err := sys.Update(ctx); err != nil {
			t.Fatal(err)
		}
	}); allocs != 0 {
		t.Fatalf("settled contacts allocated %v times, want zero", allocs)
	}
	c := contactsOf(t, world, player)
	if len(c.Current) != 1 || !slices.Equal(c.Current, c.Previous) {
		t.Fatalf("want stay, got %v", c)
	}
}

func TestSystem_ColliderOnlyAllocatesNothing(t *testing.T) {
	for _, n := range []int{100, 1000} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			world := newCollisionWorld(t)
			sys := NewSystem().(*system)
			collider, err := NewCollider(Circle{Radius: 1})
			if err != nil {
				t.Fatal(err)
			}
			for range n {
				if _, err := world.NewEntity(collider); err != nil {
					t.Fatal(err)
				}
			}
			ctx := &core.Context{World: world}
			for range 4 {
				tick(t, world, sys)
			}
			if allocs := testing.AllocsPerRun(20, func() {
				ctx.Tick++
				if err := sys.Update(ctx); err != nil {
					t.Fatal(err)
				}
			}); allocs != 0 {
				t.Errorf("collider-only world: %v allocations, want zero", allocs)
			}
			if len(sys.contacts) != 0 {
				t.Error("collider-only entities received private contact state")
			}
			for e := range sys.ensure.Execute() {
				t.Errorf("collider-only entity %d is eligible for transient Contacts", e.ID())
			}
			for e := range core.NewQuery(world).With(Contacts{}).Execute() {
				t.Errorf("collider-only entity %d received Contacts", e.ID())
			}
		})
	}
}

func TestSystem_MatchesFullPairDetection(t *testing.T) {
	type archetypeTag struct{}
	world := newCollisionWorld(t)
	sys := NewSystem()
	rng := rand.New(rand.NewPCG(72, 1))
	var entities []*core.Entity
	for i := range 24 {
		var shape ColliderShape = Circle{Radius: 8}
		if i%2 == 0 {
			shape = Box{Min: geom.Vector2{X: -5, Y: -7}, Max: geom.Vector2{X: 5, Y: 7}}
		}
		collider, err := NewCollider(shape)
		if err != nil {
			t.Fatal(err)
		}
		entity, err := world.NewEntity(core.Transform{}, collider)
		if err != nil {
			t.Fatal(err)
		}
		entities = append(entities, entity)
	}
	for step := range 20 {
		before := make(map[core.EntityID][]Contact)
		for _, entity := range entities {
			if c, ok := entity.Component[Contacts](world); ok {
				before[entity.ID()] = slices.Clone(c.Current)
			}
			if rng.IntN(6) == 0 && !entity.HasComponent[archetypeTag](world) {
				if err := entity.AddComponent(world, archetypeTag{}); err != nil {
					t.Fatal(err)
				}
			}
			if step > 0 && rng.IntN(4) != 0 {
				continue
			}
			if err := entity.SetComponent(world, core.Transform{
				Position: geom.Vector2{X: rng.Float64()*120 - 60, Y: rng.Float64()*120 - 60},
				Rotation: rng.Float64() * 2 * math.Pi,
			}); err != nil {
				t.Fatal(err)
			}
			if err := entity.Update(world, func(c *Collider) {
				c.Active = rng.IntN(5) != 0
				c.Mask = uint32(rng.IntN(2))
				c.Trigger = rng.IntN(2) == 0
				c.Offset.X = rng.Float64() * 3
			}); err != nil {
				t.Fatal(err)
			}
		}
		tick(t, world, sys)
		for _, a := range entities {
			ac, _ := a.Component[Collider](world)
			at, _ := a.Component[core.Transform](world)
			var want []Contact
			for _, b := range entities {
				bc, _ := b.Component[Collider](world)
				if a == b || !ac.Active || !bc.Active || !ac.CanCollideWith(bc) {
					continue
				}
				bt, _ := b.Component[core.Transform](world)
				ashape := transformShape(ac.Shape, ac.Offset, at).shape
				bshape := transformShape(bc.Shape, bc.Offset, bt).shape
				var hit contact
				if a.ID() < b.ID() {
					hit = contactBetween(ashape, bshape)
				} else {
					hit = contactBetween(bshape, ashape)
					hit.normal = hit.normal.Mul(-1)
				}
				if hit.collided {
					want = append(want, Contact{
						Other: b.ID(), Point: hit.point, Normal: hit.normal,
						Penetration: hit.penetration, Trigger: ac.Trigger || bc.Trigger,
					})
				}
			}
			got := contactsOf(t, world, a)
			if len(got.Current) != len(want) || !slices.Equal(got.Previous, before[a.ID()]) {
				t.Fatalf("step %d entity %d: got %v, want current %v previous %v", step, a.ID(), got, want, before[a.ID()])
			}
			for i := range want {
				// Canonical pair order can change the last rounding bit.
				actual, expected := got.Current[i], want[i]
				if actual.Other != expected.Other || actual.Trigger != expected.Trigger ||
					actual.Point.Sub(expected.Point).Length() > 1e-9 ||
					actual.Normal.Sub(expected.Normal).Length() > 1e-9 ||
					math.Abs(actual.Penetration-expected.Penetration) > 1e-9 {
					t.Fatalf("step %d entity %d: got %v, want %v", step, a.ID(), actual, expected)
				}
			}
		}
	}
}
