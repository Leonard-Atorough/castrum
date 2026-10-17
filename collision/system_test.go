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

// tick runs one fixed collision tick against world.
func tick(t *testing.T, world *core.World, sys core.System) {
	t.Helper()
	if err := sys.Update(&core.Context{World: world}); err != nil {
		t.Fatalf("collision tick: %v", err)
	}
}

func contactsOf(t *testing.T, world *core.World, entity *core.Entity) Contacts {
	t.Helper()
	contacts, ok := entity.Component[Contacts](world)
	if !ok {
		t.Fatalf("entity %d has no Contacts", entity.ID())
	}
	return contacts
}

// playerAndWall spawns a circle collider at the origin and a rect
// collider covering world x [1, 3]: the pair overlaps with the circle
// penetrating the rect's near edge by exactly one unit.
func playerAndWall(t *testing.T, world *core.World) (player, wall *core.Entity) {
	t.Helper()
	playerCollider, err := NewCollider(Circle{Radius: 2})
	if err != nil {
		t.Fatal(err)
	}
	wallCollider, err := NewCollider(Box{Min: geom.Vector2{X: 1, Y: -2}, Max: geom.Vector2{X: 3, Y: 2}})
	if err != nil {
		t.Fatal(err)
	}
	player, err = world.NewEntity(core.Transform{}, playerCollider)
	if err != nil {
		t.Fatal(err)
	}
	wall, err = world.NewEntity(core.Transform{}, wallCollider)
	if err != nil {
		t.Fatal(err)
	}
	return player, wall
}

// The lifecycle is component state: enter is Current without
// Previous, stay is both, exit is Previous without Current.
func TestSystem_ContactLifecycle(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	player, wall := playerAndWall(t, world)

	tick(t, world, sys)
	playerContacts := contactsOf(t, world, player)
	wallContacts := contactsOf(t, world, wall)
	if len(playerContacts.Current) != 1 || len(playerContacts.Previous) != 0 {
		t.Fatalf("tick 1 player contacts = %v / %v, want enter", playerContacts.Current, playerContacts.Previous)
	}
	if len(wallContacts.Current) != 1 || len(wallContacts.Previous) != 0 {
		t.Fatalf("tick 1 wall contacts = %v / %v, want enter", wallContacts.Current, wallContacts.Previous)
	}

	// The records mirror each other: same point and penetration,
	// normals pointing toward the other side.
	playerHit := playerContacts.Current[0]
	wallHit := wallContacts.Current[0]
	if playerHit.Other != wall.ID() || wallHit.Other != player.ID() {
		t.Fatalf("contact others = %d / %d", playerHit.Other, wallHit.Other)
	}
	if math.Abs(playerHit.Penetration-1) > 1e-9 {
		t.Errorf("penetration = %v, want 1", playerHit.Penetration)
	}
	if playerHit.Normal != (geom.Vector2{X: 1, Y: 0}) || wallHit.Normal != (geom.Vector2{X: -1, Y: 0}) {
		t.Errorf("normals = %v / %v, want (1, 0) and (-1, 0)", playerHit.Normal, wallHit.Normal)
	}
	if playerHit.Point != (geom.Vector2{X: 1, Y: 0}) {
		t.Errorf("point = %v, want (1, 0)", playerHit.Point)
	}

	// An unchanged tick stays: the enter becomes both Current and
	// Previous.
	tick(t, world, sys)
	playerContacts = contactsOf(t, world, player)
	if len(playerContacts.Current) != 1 || len(playerContacts.Previous) != 1 {
		t.Fatalf("tick 2 player contacts = %v / %v, want stay", playerContacts.Current, playerContacts.Previous)
	}
	if !slices.Equal(playerContacts.Current, playerContacts.Previous) {
		t.Errorf("stay contact changed across an unchanged tick: %v vs %v", playerContacts.Current, playerContacts.Previous)
	}

	// Separation exits: Current empties, Previous keeps the contact.
	if err := player.SetComponent(world, core.Transform{Position: geom.Vector2{X: 10, Y: 0}}); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys)
	playerContacts = contactsOf(t, world, player)
	wallContacts = contactsOf(t, world, wall)
	if len(playerContacts.Current) != 0 || len(playerContacts.Previous) != 1 {
		t.Fatalf("tick 3 player contacts = %v / %v, want exit", playerContacts.Current, playerContacts.Previous)
	}
	if len(wallContacts.Current) != 0 || len(wallContacts.Previous) != 1 {
		t.Fatalf("tick 3 wall contacts = %v / %v, want exit", wallContacts.Current, wallContacts.Previous)
	}

	// After the exit tick the pair state is gone entirely.
	tick(t, world, sys)
	playerContacts = contactsOf(t, world, player)
	if len(playerContacts.Current) != 0 || len(playerContacts.Previous) != 0 {
		t.Fatalf("tick 4 player contacts = %v / %v, want empty", playerContacts.Current, playerContacts.Previous)
	}
}

// The system attaches Contacts itself; a game spawns only the
// Collider.
func TestSystem_AttachesContactsToColliders(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	player, _ := playerAndWall(t, world)

	if _, ok := player.Component[Contacts](world); ok {
		t.Fatal("test setup: entity should not have Contacts before the first tick")
	}
	tick(t, world, sys)
	if _, ok := player.Component[Contacts](world); !ok {
		t.Fatal("collision system should have attached Contacts")
	}
}

func TestSystem_LayerMaskFiltering(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	player, wall := playerAndWall(t, world)

	// The wall sits on layer 1 and listens only to layer 1; the
	// player lives on layer 0. One-sided masks do not collide.
	collider, _ := wall.Component[Collider](world)
	collider.Layers = 1 << 1
	collider.Mask = 1 << 1
	if err := wall.SetComponent(world, collider); err != nil {
		t.Fatal(err)
	}

	tick(t, world, sys)
	playerContacts := contactsOf(t, world, player)
	wallContacts := contactsOf(t, world, wall)
	if len(playerContacts.Current) != 0 {
		t.Errorf("layer 0 player should not contact a wall deaf to layer 0, got %v", playerContacts.Current)
	}
	if len(wallContacts.Current) != 0 {
		t.Errorf("wall deaf to layer 0 should have no contacts, got %v", wallContacts.Current)
	}
}

func TestSystem_TriggerFlagInContacts(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	player, wall := playerAndWall(t, world)

	collider, _ := player.Component[Collider](world)
	collider.Trigger = true
	if err := player.SetComponent(world, collider); err != nil {
		t.Fatal(err)
	}

	tick(t, world, sys)
	playerContacts := contactsOf(t, world, player)
	wallContacts := contactsOf(t, world, wall)
	if len(playerContacts.Current) != 1 || !playerContacts.Current[0].Trigger {
		t.Fatalf("trigger pair should record a trigger contact, got %v", playerContacts.Current)
	}
	if len(wallContacts.Current) != 1 || !wallContacts.Current[0].Trigger {
		t.Fatalf("the trigger flag mirrors to the partner, got %v", wallContacts.Current)
	}
}

// A collider switched off stops contacting this tick and shows the
// exit through Previous, exactly like separation.
func TestSystem_InactiveColliderStopsColliding(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	player, wall := playerAndWall(t, world)

	tick(t, world, sys)
	if len(contactsOf(t, world, player).Current) != 1 {
		t.Fatal("test setup: pair should be colliding")
	}

	collider, _ := player.Component[Collider](world)
	collider.Active = false
	if err := player.SetComponent(world, collider); err != nil {
		t.Fatal(err)
	}

	tick(t, world, sys)
	playerContacts := contactsOf(t, world, player)
	wallContacts := contactsOf(t, world, wall)
	if len(playerContacts.Current) != 0 || len(playerContacts.Previous) != 1 {
		t.Fatalf("inactive player contacts = %v / %v, want exit", playerContacts.Current, playerContacts.Previous)
	}
	if len(wallContacts.Current) != 0 || len(wallContacts.Previous) != 1 {
		t.Fatalf("wall contacts = %v / %v, want exit", wallContacts.Current, wallContacts.Previous)
	}
}

// A pair that was never colliding is picked up when one side moves
// into the other - the dirty proxy path through the spatial index.
func TestSystem_MovingColliderEntersNewPair(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()

	playerCollider, err := NewCollider(Circle{Radius: 2})
	if err != nil {
		t.Fatal(err)
	}
	wallCollider, err := NewCollider(Box{Min: geom.Vector2{X: 1, Y: -2}, Max: geom.Vector2{X: 3, Y: 2}})
	if err != nil {
		t.Fatal(err)
	}
	player, err := world.NewEntity(core.Transform{Position: geom.Vector2{X: 50, Y: 0}}, playerCollider)
	if err != nil {
		t.Fatal(err)
	}
	wall, err := world.NewEntity(core.Transform{}, wallCollider)
	if err != nil {
		t.Fatal(err)
	}

	tick(t, world, sys)
	if got := contactsOf(t, world, player).Current; len(got) != 0 {
		t.Fatalf("separated pair should not contact, got %v", got)
	}

	if err := player.SetComponent(world, core.Transform{Position: geom.Vector2{X: 0, Y: 0}}); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys)
	playerContacts := contactsOf(t, world, player)
	if len(playerContacts.Current) != 1 || len(playerContacts.Previous) != 0 {
		t.Fatalf("moved-in player contacts = %v / %v, want enter", playerContacts.Current, playerContacts.Previous)
	}
	if playerContacts.Current[0].Other != wall.ID() {
		t.Errorf("contact other = %d, want wall %d", playerContacts.Current[0].Other, wall.ID())
	}
}

// Current is sorted by Other so successive ticks compare stably.
func TestSystem_ContactsSortedByOther(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()

	playerCollider, err := NewCollider(Circle{Radius: 10})
	if err != nil {
		t.Fatal(err)
	}
	player, err := world.NewEntity(core.Transform{}, playerCollider)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		wallCollider, err := NewCollider(Circle{Radius: 2})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := world.NewEntity(
			core.Transform{Position: geom.Vector2{X: float64(i + 1), Y: 0}},
			wallCollider,
		); err != nil {
			t.Fatal(err)
		}
	}

	tick(t, world, sys)
	playerContacts := contactsOf(t, world, player)
	if len(playerContacts.Current) != 3 {
		t.Fatalf("player contacts = %v, want 3", playerContacts.Current)
	}
	for i := 1; i < len(playerContacts.Current); i++ {
		if playerContacts.Current[i-1].Other >= playerContacts.Current[i].Other {
			t.Errorf("contacts not sorted by Other: %v", playerContacts.Current)
			break
		}
	}
}

// A collider removed mid-game does not leave its last contacts
// behind: the entity loses its Contacts, and the partner observes the
// exit through its own Current emptying.
func TestSystem_RemovedColliderClearsContacts(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem()
	player, wall := playerAndWall(t, world)

	tick(t, world, sys)
	if len(contactsOf(t, world, player).Current) != 1 {
		t.Fatal("test setup: pair should be colliding")
	}

	if err := wall.RemoveComponent[Collider](world); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys)
	if _, ok := wall.Component[Contacts](world); ok {
		t.Error("removed collider should have no Contacts left")
	}
	playerContacts := contactsOf(t, world, player)
	if len(playerContacts.Current) != 0 || len(playerContacts.Previous) != 1 {
		t.Errorf("player contacts = %v / %v, want the exit edge", playerContacts.Current, playerContacts.Previous)
	}
}

func TestSystem_GeometryAndFilterChanges(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*Collider, *core.Transform)
		hit    bool
	}{
		{name: "position", change: func(_ *Collider, tr *core.Transform) { tr.Position.X = 10 }},
		{name: "rotation", change: func(_ *Collider, tr *core.Transform) { tr.Rotation = math.Pi / 2 }},
		{name: "shape", change: func(c *Collider, _ *core.Transform) { c.Shape = Circle{Radius: 0.5} }},
		{name: "shape type", change: func(c *Collider, _ *core.Transform) {
			c.Shape = Box{Min: geom.Vector2{X: -0.5, Y: -0.5}, Max: geom.Vector2{X: 0.5, Y: 0.5}}
		}},
		{name: "offset", change: func(c *Collider, _ *core.Transform) { c.Offset.Y = 10 }},
		{name: "mask", change: func(c *Collider, _ *core.Transform) { c.Mask = 0 }},
		{name: "layers", change: func(c *Collider, _ *core.Transform) { c.Layers = 0 }},
		{name: "trigger", hit: true, change: func(c *Collider, _ *core.Transform) { c.Trigger = true }},
		{name: "ignored scale and transform offset", hit: true, change: func(_ *Collider, tr *core.Transform) {
			tr.Scale = geom.Vector2{X: 10, Y: 0.1}
			tr.Offset.X = 100
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			world := core.NewWorld()
			sys := NewSystem().(*system)
			player, wall := playerAndWall(t, world)
			if test.name == "rotation" {
				if err := player.Update(world, func(c *Collider) { c.Offset.Y = 2 }); err != nil {
					t.Fatal(err)
				}
			}
			tick(t, world, sys)
			original := contactsOf(t, world, player).Current[0]
			collider, _ := player.Component[Collider](world)
			transform, _ := player.Component[core.Transform](world)
			test.change(&collider, &transform)
			if err := player.SetComponent(world, collider); err != nil {
				t.Fatal(err)
			}
			if err := player.SetComponent(world, transform); err != nil {
				t.Fatal(err)
			}
			tick(t, world, sys)
			for _, entity := range []*core.Entity{player, wall} {
				c := contactsOf(t, world, entity)
				if (len(c.Current) == 1) != test.hit || len(c.Previous) != 1 {
					t.Fatalf("contacts = %v / %v, want hit %v and previous contact", c.Current, c.Previous, test.hit)
				}
			}
			if got := contactsOf(t, world, player).Previous[0]; got != original {
				t.Fatalf("previous contact overwritten: got %v, want %v", got, original)
			}
			tick(t, world, sys)
			if !test.hit {
				c := contactsOf(t, world, player)
				if len(c.Current) != 0 || len(c.Previous) != 0 {
					t.Fatalf("exit did not clear: %v", c)
				}
			}
		})
	}
}

func TestSystem_ReactivationAndComponentLifecycle(t *testing.T) {
	for _, action := range []string{"inactive", "collider removed", "transform removed", "destroyed"} {
		t.Run(action, func(t *testing.T) {
			world := core.NewWorld()
			sys := NewSystem()
			player, wall := playerAndWall(t, world)
			tick(t, world, sys)
			collider, _ := wall.Component[Collider](world)
			transform, _ := wall.Component[core.Transform](world)
			var err error
			switch action {
			case "inactive":
				off := collider
				off.Active = false
				err = wall.SetComponent(world, off)
			case "collider removed":
				err = wall.RemoveComponent[Collider](world)
			case "transform removed":
				err = wall.RemoveComponent[core.Transform](world)
			case "destroyed":
				err = world.DestroyEntity(wall)
			}
			if err != nil {
				t.Fatal(err)
			}
			tick(t, world, sys)
			c := contactsOf(t, world, player)
			if len(c.Current) != 0 || len(c.Previous) != 1 {
				t.Fatalf("want exit, got %v", c)
			}
			tick(t, world, sys)
			c = contactsOf(t, world, player)
			if len(c.Current) != 0 || len(c.Previous) != 0 {
				t.Fatalf("want cleared exit, got %v", c)
			}
			switch action {
			case "inactive":
				err = wall.SetComponent(world, collider)
			case "collider removed":
				err = wall.AddComponent(world, collider)
			case "transform removed":
				err = wall.AddComponent(world, transform)
			case "destroyed":
				wall, err = world.NewEntity(transform, collider)
			}
			if err != nil {
				t.Fatal(err)
			}
			tick(t, world, sys)
			for _, entity := range []*core.Entity{player, wall} {
				c = contactsOf(t, world, entity)
				if len(c.Current) != 1 || len(c.Previous) != 0 {
					t.Fatalf("want fresh enter, got %v", c)
				}
			}
		})
	}
}

func TestSystem_StaticFiltersCanCreateAndChangeContacts(t *testing.T) {
	world := core.NewWorld()
	sys := NewSystem().(*system)
	player, wall := playerAndWall(t, world)
	collider, _ := player.Component[Collider](world)
	collider.Mask = 0
	if err := player.SetComponent(world, collider); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys)
	shape := sys.proxies[player.ID()].shape
	collider.Mask = 1
	if err := player.SetComponent(world, collider); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys)
	if c := contactsOf(t, world, player); len(c.Current) != 1 || len(c.Previous) != 0 {
		t.Fatalf("mask change should create an enter: %v", c)
	}
	collider.Trigger = true
	if err := player.SetComponent(world, collider); err != nil {
		t.Fatal(err)
	}
	tick(t, world, sys)
	for _, entity := range []*core.Entity{player, wall} {
		c := contactsOf(t, world, entity)
		if !c.Current[0].Trigger || c.Previous[0].Trigger {
			t.Fatalf("trigger change overwrote history: %v", c)
		}
	}
	if sys.proxies[player.ID()].shape != shape {
		t.Fatal("filter-only changes rebuilt geometry")
	}
}

func TestSystem_TouchingAcrossCellBoundary(t *testing.T) {
	world := core.NewWorld()
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
			world := core.NewWorld()
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
	world := core.NewWorld()
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

func TestSystem_RetryAfterInvalidTransformRemovesNewProxies(t *testing.T) {
	world := core.NewWorld()
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
			world := core.NewWorld()
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

func TestSystem_MatchesFullPairDetection(t *testing.T) {
	type archetypeTag struct{}
	world := core.NewWorld()
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
