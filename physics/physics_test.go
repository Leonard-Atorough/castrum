package physics

import (
	"math"
	"testing"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/geom"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.CellSize != 50 {
		t.Errorf("DefaultConfig().CellSize = %v, want 50", config.CellSize)
	}
	if !config.Enabled {
		t.Error("DefaultConfig().Enabled = false, want true")
	}
}

func TestPhysicsSystem_InitAndShutdown(t *testing.T) {
	world := ecs.NewWorld()
	system := NewSystem(PhysicsConfig{Enabled: true})

	if err := system.Init(world); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if system.query == nil {
		t.Fatal("Init() did not create the collider query")
	}
	if err := system.Shutdown(world); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestPhysicsSystem_UpdateRequiresEventBusWhenEnabled(t *testing.T) {
	world := ecs.NewWorld()
	system := NewSystem(PhysicsConfig{Enabled: true})
	if err := system.Init(world); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := system.Update(world, 0); err == nil {
		t.Fatal("Update() error = nil, want missing event bus error")
	}
}

func TestPhysicsSystem_DisabledUpdateDoesNotRequireEventBus(t *testing.T) {
	world := ecs.NewWorld()
	system := NewSystem(PhysicsConfig{Enabled: false})
	if err := system.Init(world); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := system.Update(world, 0); err != nil {
		t.Fatalf("disabled Update() error = %v, want nil", err)
	}
	if len(system.lastProxies) != 0 {
		t.Fatalf("disabled Update() indexed %d proxies, want 0", len(system.lastProxies))
	}
}

func TestPhysicsSystem_InactiveColliderRemovesPreviousPair(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	system := NewSystem(PhysicsConfig{CellSize: 10, Enabled: true})
	if err := system.Init(world); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	first, err := world.CreateWithComponents("",
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
		components.NewCollider(geom.Circle{Radius: 10}, true, false, 0, 1),
	)
	if err != nil {
		t.Fatalf("Create first entity: %v", err)
	}
	second, err := world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 5}, Scale: geom.Vector2{X: 1, Y: 1}},
		components.NewCollider(geom.Circle{Radius: 10}, true, false, 1, 0),
	)
	if err != nil {
		t.Fatalf("Create second entity: %v", err)
	}

	var eventsSeen []CollisionEventType
	bus.On(func(_ events.EventMeta, event CollisionEvent) {
		eventsSeen = append(eventsSeen, event.CollisionEventType)
	}, false)

	if err := system.Update(world, 0); err != nil {
		t.Fatalf("initial Update() error = %v", err)
	}
	if len(system.previousPairs) != 1 {
		t.Fatalf("previousPairs after collision = %d, want 1", len(system.previousPairs))
	}

	eventsSeen = nil
	if err := world.SetComponent(second.ID, components.Collider{
		Shape:  geom.Circle{Radius: 10},
		Active: false,
		Layer:  1,
		Mask:   0,
	}); err != nil {
		t.Fatalf("deactivate collider: %v", err)
	}
	if err := system.Update(world, 0); err != nil {
		t.Fatalf("Update() after deactivation error = %v", err)
	}

	if len(system.lastProxies) != 1 {
		t.Fatalf("lastProxies after deactivation = %d, want 1", len(system.lastProxies))
	}
	if _, ok := system.lastProxies[first.ID]; !ok {
		t.Fatal("active entity proxy was removed")
	}
	if len(system.previousPairs) != 0 {
		t.Fatalf("previousPairs after deactivation = %d, want 0", len(system.previousPairs))
	}
	if len(eventsSeen) != 0 {
		t.Fatalf("events after deactivation = %v, want none", eventsSeen)
	}
}

func TestCollisionProxyChanged_WhenFilterChanges(t *testing.T) {
	previous := collisionProxy{layer: 1, mask: 1 << 2}

	current := previous
	current.layer = 2
	if !collisionProxyChanged(previous, current) {
		t.Fatal("layer change did not mark collision proxy dirty")
	}

	current = previous
	current.mask = 1 << 3
	if !collisionProxyChanged(previous, current) {
		t.Fatal("mask change did not mark collision proxy dirty")
	}
}

func TestPhysicsSystem_FilterChangesReevaluateStaticPair(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	system := NewSystem(PhysicsConfig{Enabled: true})
	if err := system.Init(world); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	first, err := world.CreateWithComponents("",
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
		components.NewCollider(geom.Circle{Radius: 10}, true, false, 0, 1),
	)
	if err != nil {
		t.Fatalf("Create first entity: %v", err)
	}
	_, err = world.CreateWithComponents("",
		components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
		components.NewCollider(geom.Circle{Radius: 10}, true, false, 1, 0),
	)
	if err != nil {
		t.Fatalf("Create second entity: %v", err)
	}

	var eventTypes []CollisionEventType
	bus.On(func(_ events.EventMeta, event CollisionEvent) {
		eventTypes = append(eventTypes, event.CollisionEventType)
	}, false)

	if err := system.Update(world, 0); err != nil {
		t.Fatalf("initial Update() error = %v", err)
	}
	if len(eventTypes) != 1 || eventTypes[0] != CollisionEnter {
		t.Fatalf("initial events = %v, want [CollisionEnter]", eventTypes)
	}

	eventTypes = nil
	if err := world.SetComponent(first.ID, components.NewCollider(geom.Circle{Radius: 10}, true, false, 0)); err != nil {
		t.Fatalf("remove collision mask: %v", err)
	}
	if err := system.Update(world, 0); err != nil {
		t.Fatalf("Update() after filter removal error = %v", err)
	}
	if len(eventTypes) != 1 || eventTypes[0] != CollisionExit {
		t.Fatalf("events after filter removal = %v, want [CollisionExit]", eventTypes)
	}

	eventTypes = nil
	if err := world.SetComponent(first.ID, components.NewCollider(geom.Circle{Radius: 10}, true, false, 0, 1)); err != nil {
		t.Fatalf("restore collision mask: %v", err)
	}
	if err := system.Update(world, 0); err != nil {
		t.Fatalf("Update() after filter restoration error = %v", err)
	}
	if len(eventTypes) != 1 || eventTypes[0] != CollisionEnter {
		t.Fatalf("events after filter restoration = %v, want [CollisionEnter]", eventTypes)
	}

}

func TestPhysicsSystem_EventIncludesContactDepthAndTriggerState(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	system := NewSystem(PhysicsConfig{Enabled: true})
	if err := system.Init(world); err != nil {
		t.Fatal(err)
	}

	_, err := world.CreateWithComponents("",
		components.Transform{},
		components.NewCollider(geom.Circle{Radius: 10}, true, true, 0, 1),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = world.CreateWithComponents("",
		components.Transform{Position: geom.Vector2{X: 15}},
		components.NewCollider(geom.Circle{Radius: 10}, true, false, 1, 0),
	)
	if err != nil {
		t.Fatal(err)
	}

	var event CollisionEvent
	bus.On(func(_ events.EventMeta, received CollisionEvent) {
		event = received
	}, false)
	if err := system.Update(world, 0); err != nil {
		t.Fatal(err)
	}
	if event.CollisionEventType != CollisionEnter {
		t.Fatalf("event type = %v, want CollisionEnter", event.CollisionEventType)
	}
	if !event.Trigger {
		t.Error("trigger collision event was not marked as a trigger")
	}
	if event.Penetration < 4.9 || event.Penetration > 5.1 {
		t.Errorf("event penetration = %v, want approximately 5", event.Penetration)
	}
}

func TestPhysicsSystem_OffscreenCollidersStillCollide(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	world.SetResource(bus)
	system := NewSystem(PhysicsConfig{Enabled: true})
	if err := system.Init(world); err != nil {
		t.Fatal(err)
	}

	for index, position := range []geom.Vector2{{X: 100000, Y: 100000}, {X: 100005, Y: 100000}} {
		if _, err := world.CreateWithComponents("",
			components.Transform{Position: position},
			components.NewCollider(geom.Circle{Radius: 10}, true, false, uint8(index), 1-uint(index)),
		); err != nil {
			t.Fatal(err)
		}
	}

	entered := false
	bus.On(func(_ events.EventMeta, event CollisionEvent) {
		entered = entered || event.CollisionEventType == CollisionEnter
	}, false)
	if err := system.Update(world, 0); err != nil {
		t.Fatal(err)
	}
	if !entered {
		t.Error("off-screen colliders did not emit CollisionEnter")
	}
}

func TestPhysicsSystem_TransformOnlyChangesReevaluatePair(t *testing.T) {
	t.Run("rotation", func(t *testing.T) {
		world := ecs.NewWorld()
		bus := events.NewEventBus()
		world.SetResource(bus)
		system := NewSystem(PhysicsConfig{Enabled: true})
		if err := system.Init(world); err != nil {
			t.Fatal(err)
		}
		first, _ := world.CreateWithComponents("",
			components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
			components.NewCollider(geom.Rect{Min: geom.Vector2{X: -5, Y: -1}, Max: geom.Vector2{X: 5, Y: 1}}, true, false, 0, 1),
		)
		_, _ = world.CreateWithComponents("",
			components.Transform{Position: geom.Vector2{X: 0, Y: 6}, Scale: geom.Vector2{X: 1, Y: 1}},
			components.NewCollider(geom.Rect{Min: geom.Vector2{X: -5, Y: -1}, Max: geom.Vector2{X: 5, Y: 1}}, true, false, 1, 0),
		)
		var enters int
		bus.On(func(_ events.EventMeta, event CollisionEvent) {
			if event.CollisionEventType == CollisionEnter {
				enters++
			}
		}, false)
		if err := system.Update(world, 0); err != nil {
			t.Fatal(err)
		}
		if err := world.SetComponent(first.ID, components.Transform{Rotation: math.Pi / 2, Scale: geom.Vector2{X: 1, Y: 1}}); err != nil {
			t.Fatal(err)
		}
		if err := system.Update(world, 0); err != nil {
			t.Fatal(err)
		}
		if enters != 1 {
			t.Fatalf("rotation-only update enter count = %d, want 1", enters)
		}
	})

	t.Run("scale", func(t *testing.T) {
		world := ecs.NewWorld()
		bus := events.NewEventBus()
		world.SetResource(bus)
		system := NewSystem(PhysicsConfig{Enabled: true})
		if err := system.Init(world); err != nil {
			t.Fatal(err)
		}
		first, _ := world.CreateWithComponents("",
			components.Transform{Scale: geom.Vector2{X: 1, Y: 1}},
			components.NewCollider(geom.Circle{Radius: 1}, true, false, 0, 1),
		)
		_, _ = world.CreateWithComponents("",
			components.Transform{Position: geom.Vector2{X: 5}, Scale: geom.Vector2{X: 1, Y: 1}},
			components.NewCollider(geom.Circle{Radius: 1}, true, false, 1, 0),
		)
		var enters int
		bus.On(func(_ events.EventMeta, event CollisionEvent) {
			if event.CollisionEventType == CollisionEnter {
				enters++
			}
		}, false)
		if err := system.Update(world, 0); err != nil {
			t.Fatal(err)
		}
		if err := world.SetComponent(first.ID, components.Transform{Scale: geom.Vector2{X: 5, Y: 5}}); err != nil {
			t.Fatal(err)
		}
		if err := system.Update(world, 0); err != nil {
			t.Fatal(err)
		}
		if enters != 1 {
			t.Fatalf("scale-only update enter count = %d, want 1", enters)
		}
	})
}
