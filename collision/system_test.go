package collision

import (
	"math"
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
	playerCollider, err := NewCollider(CircleShape{Radius: 2})
	if err != nil {
		t.Fatal(err)
	}
	wallCollider, err := NewCollider(RectShape{Min: geom.Vector2{X: 1, Y: -2}, Max: geom.Vector2{X: 3, Y: 2}})
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

	playerCollider, err := NewCollider(CircleShape{Radius: 2})
	if err != nil {
		t.Fatal(err)
	}
	wallCollider, err := NewCollider(RectShape{Min: geom.Vector2{X: 1, Y: -2}, Max: geom.Vector2{X: 3, Y: 2}})
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

	playerCollider, err := NewCollider(CircleShape{Radius: 10})
	if err != nil {
		t.Fatal(err)
	}
	player, err := world.NewEntity(core.Transform{}, playerCollider)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		wallCollider, err := NewCollider(CircleShape{Radius: 2})
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
