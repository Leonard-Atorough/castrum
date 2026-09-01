package core

import "testing"

// movementSystem advances TestPosition by TestVelocity each tick using the
// generic typed helpers, mirroring how real game systems (e.g. RotatorSystem)
// access components.
type movementSystem struct {
	initCalled     bool
	shutdownCalled bool
}

func (s *movementSystem) Init(world *World) error {
	s.initCalled = true
	return nil
}

func (s *movementSystem) Update(world *World, delta float64) error {
	for _, id := range QueryFor[TestPosition](world) {
		if !HasComponent[TestVelocity](world, id) {
			continue
		}
		pos, err := GetComponent[TestPosition](world, id)
		if err != nil {
			return err
		}
		vel, err := GetComponent[TestVelocity](world, id)
		if err != nil {
			return err
		}
		pos.X += vel.X * delta
		pos.Y += vel.Y * delta
		if err := SetComponent(world, id, pos); err != nil {
			return err
		}
	}
	return nil
}

func (s *movementSystem) Shutdown(world *World) error {
	s.shutdownCalled = true
	return nil
}

// TestWorld_EndToEndGameLoop exercises World, Manager and the typed component
// helpers together the way real game code composes them: spawn entities,
// run several update ticks through a registered system, tear down part of
// the hierarchy, and verify the final world state.
func TestWorld_EndToEndGameLoop(t *testing.T) {
	world := NewWorld()
	manager := NewManager()

	sys := &movementSystem{}
	if err := manager.Register("movement", 0, sys, world); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if !sys.initCalled {
		t.Fatal("expected Init to be called on registration")
	}

	unit, err := world.CreateWithComponents("Unit", TestPosition{X: 0, Y: 0}, TestVelocity{X: 1, Y: 2})
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}

	turret, err := world.CreateWithComponents("Turret", TestPosition{X: 5, Y: 5})
	if err != nil {
		t.Fatalf("CreateWithComponents failed: %v", err)
	}
	world.SetParent(turret.ID, unit.ID)

	const ticks = 3
	for i := 0; i < ticks; i++ {
		if err := manager.Update(world, 1.0); err != nil {
			t.Fatalf("Update failed on tick %d: %v", i, err)
		}
	}

	gotPos, err := GetComponent[TestPosition](world, unit.ID)
	if err != nil {
		t.Fatalf("GetComponent failed: %v", err)
	}
	if gotPos != (TestPosition{X: 3, Y: 6}) {
		t.Fatalf("expected unit to have moved to (3,6) after %d ticks, got %#v", ticks, gotPos)
	}

	// The turret has no velocity, so the movement system must leave it untouched.
	turretPos, err := GetComponent[TestPosition](world, turret.ID)
	if err != nil {
		t.Fatalf("GetComponent failed: %v", err)
	}
	if turretPos != (TestPosition{X: 5, Y: 5}) {
		t.Fatalf("expected turret to stay put, got %#v", turretPos)
	}

	if err := world.DestroyEntity(unit.ID, true); err != nil {
		t.Fatalf("DestroyEntity failed: %v", err)
	}
	world.Cleanup()

	if world.HasEntity(unit.ID) || world.HasEntity(turret.ID) {
		t.Fatal("cascade destroy should have removed both the unit and its child turret")
	}

	if err := manager.Shutdown(world); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	if !sys.shutdownCalled {
		t.Fatal("expected Shutdown to be called")
	}
}
