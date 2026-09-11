package physics

import (
	"testing"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/geom"
)

func TestSystem_RectCollision(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)

	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: true})

	if err := collisionSys.Init(world); err != nil {
		t.Errorf("Init failed: %v", err)
	}

	// Create player at origin with box collider
	player, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-10, -10), geom.NewVector2(10, 10)), true, false, 0, 1),
	)

	// Create obstacle at (5, 5) - should collide
	obstacle, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 5, Y: 5}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-10, -10), geom.NewVector2(10, 10)), true, false, 1, 0),
	)

	// Update spatial index
	if err := collisionSys.Update(world, 0); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	// Test collision detection
	result, err := collisionSys.TestCollision(world, player.ID, obstacle.ID)
	if err != nil {
		t.Errorf("TestCollision failed: %v", err)
	}
	if !result.Collided {
		t.Error("Expected collision between overlapping rectangles")
	}
}

func TestSystem_NoCollisionWhenFar(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)

	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: true})

	collisionSys.Init(world)

	// Create player at origin
	player, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-10, -10), geom.NewVector2(10, 10)), true, false, 0, 1),
	)

	// Create obstacle far away at (200, 200) - should NOT collide
	obstacle, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 200, Y: 200}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-10, -10), geom.NewVector2(10, 10)), true, false, 1, 0),
	)

	collisionSys.Update(world, 0)

	result, err := collisionSys.TestCollision(world, player.ID, obstacle.ID)
	if err != nil {
		t.Errorf("TestCollision failed: %v", err)
	}
	if result.Collided {
		t.Error("Expected no collision between distant rectangles")
	}
}

func TestSystem_CircleCollision(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: true})

	collisionSys.Init(world)

	// Create player circle at origin
	player, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 15}, true, false, 0, 1),
	)

	// Create obstacle circle at (20, 0) - should collide (distance 20, radii sum to 30)
	obstacle, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 20, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 15}, true, false, 1, 0),
	)

	if err := collisionSys.Update(world, 0); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	result, err := collisionSys.TestCollision(world, player.ID, obstacle.ID)
	if err != nil {
		t.Errorf("TestCollision failed: %v", err)
	}
	if !result.Collided {
		t.Error("Expected collision between overlapping circles")
	}
}

func TestSystem_CircleRectCollision(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: true})

	collisionSys.Init(world)

	// Create rect at origin
	rect, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-10, -10), geom.NewVector2(10, 10)), true, false, 0, 1),
	)

	// Create circle at (15, 0) - should collide
	circle, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 15, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 8}, true, false, 1, 0),
	)

	if err := collisionSys.Update(world, 0); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	result, err := collisionSys.TestCollision(world, rect.ID, circle.ID)
	if err != nil {
		t.Errorf("TestCollision failed: %v", err)
	}
	if !result.Collided {
		t.Error("Expected collision between rect and circle")
	}
}

func TestSystem_EventLifecycle(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: true})
	collisionSys.Init(world)

	// Create two separated entities
	_, _ = world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 10}, true, false, 0, 1),
	)
	enemy, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 100, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 10}, true, false, 1, 0),
	)

	collisionSys.Update(world, 0)

	// Track event state across updates
	var lastEvent CollisionEvent
	var eventFired bool

	// Subscribe to collision events once for the entire test
	subID := bus.On(func(_ events.EventMeta, e CollisionEvent) {
		lastEvent = e
		eventFired = true
	}, false)

	// First update: no collision
	eventFired = false
	collisionSys.Update(world, 0)
	if eventFired {
		t.Error("Expected no events when separated")
	}

	// Move enemy into collision range
	world.SetComponent(enemy.ID, components.Transform{Position: geom.Vector2{X: 15, Y: 0}})

	// Update after move: should emit Enter
	eventFired = false
	if err := collisionSys.Update(world, 0); err != nil {
		t.Errorf("Update failed: %v", err)
	}
	if !eventFired || lastEvent.CollisionEventType != CollisionEnter {
		t.Error("Expected CollisionEnter event after moving into collision range")
	}

	// Next update with no movement: should emit Stay
	eventFired = false
	collisionSys.Update(world, 0)
	if !eventFired || lastEvent.CollisionEventType != CollisionStay {
		t.Error("Expected CollisionStay event when still colliding")
	}

	// Move enemy away
	world.SetComponent(enemy.ID, components.Transform{Position: geom.Vector2{X: 100, Y: 0}})

	// Update after moving away: should emit Exit
	eventFired = false
	if err := collisionSys.Update(world, 0); err != nil {
		t.Errorf("Update failed: %v", err)
	}
	if !eventFired || lastEvent.CollisionEventType != CollisionExit {
		t.Error("Expected CollisionExit event after moving out of collision range")
	}

	bus.Unsubscribe(subID)
}

func TestSystem_LayerMaskFiltering(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: true})
	collisionSys.Init(world)

	// Entity on layer 0, collides with [1]
	player, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 15}, true, false, 0, 1),
	)

	// Entity on layer 2 (not in mask [1])
	enemy, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 20, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 15}, true, false, 2, 0),
	)

	if err := collisionSys.Update(world, 0); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	// Should not collide due to layer mismatch
	result, err := collisionSys.TestCollision(world, player.ID, enemy.ID)
	if err != nil {
		t.Errorf("TestCollision failed: %v", err)
	}
	if result.Collided {
		t.Error("Expected no collision between incompatible layers")
	}
}

func TestSystem_InactiveColliderSkipped(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: true})
	collisionSys.Init(world)

	// Active collider
	player, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 15}, true, false, 0, 1),
	)

	// Inactive collider (overlapping)
	enemy, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 20, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 15}, false, false, 1, 0),
	)

	if err := collisionSys.Update(world, 0); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	// Should not collide because enemy is inactive
	result, err := collisionSys.TestCollision(world, player.ID, enemy.ID)
	if err == nil || result.Collided {
		t.Error("Expected no collision with inactive collider")
	}
}

func TestSystem_CircleCircleContact(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: true})
	collisionSys.Init(world)

	// Two circles
	circle1, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 10}, true, false, 0, 1),
	)
	circle2, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 15, Y: 0}},
		components.NewCollider(geom.Circle{Center: geom.Vector2{}, Radius: 10}, true, false, 1, 0),
	)

	collisionSys.Update(world, 0)

	result, err := collisionSys.TestCollision(world, circle1.ID, circle2.ID)
	if err != nil {
		t.Errorf("TestCollision failed: %v", err)
	}
	if !result.Collided {
		t.Error("Expected collision between circles")
	}

	// Verify contact geometry is computed
	if result.Point == (geom.Vector2{}) {
		t.Error("Expected non-zero contact point")
	}
	if result.Normal == (geom.Vector2{}) {
		t.Error("Expected non-zero normal")
	}
	if result.Penetration == 0 {
		t.Error("Expected non-zero penetration depth")
	}

	// Verify normal is unit-ish (circles 15 apart, radii 10+10=20, penetration should be 5)
	if result.Penetration < 4.9 || result.Penetration > 5.1 {
		t.Errorf("Expected penetration ~5, got %f", result.Penetration)
	}
}

func TestSystem_QueryCollisions(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: true})

	collisionSys.Init(world)

	// Create player
	player, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-10, -10), geom.NewVector2(10, 10)), true, false, 0, 1),
	)

	// Create two colliding obstacles
	obs1, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 5, Y: 0}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-5, -5), geom.NewVector2(5, 5)), true, false, 1, 0),
	)

	obs2, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 8}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-5, -5), geom.NewVector2(5, 5)), true, false, 1, 0),
	)

	// Create one non-colliding obstacle
	obs3, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 100, Y: 100}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-5, -5), geom.NewVector2(5, 5)), true, false, 1, 0),
	)

	if err := collisionSys.Update(world, 0); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	// Query collisions for player
	collisions, err := collisionSys.CollidingWith(world, player.ID)
	if err != nil {
		t.Errorf("QueryCollisions failed: %v", err)
	}

	if len(collisions) != 2 {
		t.Errorf("Expected 2 collisions, got %d", len(collisions))
	}

	// Verify the colliding entities are in the result
	found := make(map[ecs.EntityID]bool)
	for _, id := range collisions {
		found[id] = true
	}

	if !found[obs1.ID] {
		t.Error("Expected obs1 in collisions")
	}
	if !found[obs2.ID] {
		t.Error("Expected obs2 in collisions")
	}
	if found[obs3.ID] {
		t.Error("Did not expect obs3 in collisions")
	}
}

func TestSystem_DisabledCollision(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: false})

	collisionSys.Init(world)

	// Create colliding entities
	_, _ = world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-10, -10), geom.NewVector2(10, 10)), true, false, 0, 1),
	)

	obstacle, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 5, Y: 5}},
		components.NewCollider(geom.NewRect(geom.NewVector2(-10, -10), geom.NewVector2(10, 10)), true, false, 1, 0),
	)

	collisionSys.Update(world, 0)

	// Run with collision disabled
	if err := collisionSys.Update(world, 0); err != nil {
		t.Errorf("Update failed: %v", err)
	}

	// Obstacle should still exist (not destroyed)
	_, getErr := world.GetComponent[components.Collider](obstacle.ID)
	if getErr != nil {
		t.Error("Expected obstacle to still exist when collision is disabled")
	}
}

func TestSystem_TestCollisionMissingComponent(t *testing.T) {
	world := ecs.NewWorld()
	collisionSys := NewSystem(PhysicsConfig{QueryRadius: 300, Enabled: true})

	collisionSys.Init(world)

	// Create entity without collider
	entity, _ := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 0, Y: 0}},
	)

	// Should return error when entity has no collider
	_, testErr := collisionSys.TestCollision(world, entity.ID, entity.ID)
	if testErr == nil {
		t.Error("Expected error when testing collision on entity without collider")
	}
}
