package ecs

import (
	"testing"
)

// movementSystem advances TestPosition by TestVelocity each tick using the
// generic typed helpers, mirroring how real game systems (e.g. RotatorSystem)
// access components.
type movementSystem struct {
	initCalled     bool
	shutdownCalled bool
	movementQuery  *Query
}

func (s *movementSystem) Init(world *World) error {
	s.initCalled = true
	s.movementQuery = world.NewQuery().WithRequiredComponents(TestPosition{}, TestVelocity{})
	return nil
}

func (s *movementSystem) Update(world *World, delta float64) error {
	for result := range s.movementQuery.Execute() {
		id := result.EntityID
		pos, err := world.GetComponent[TestPosition](id)
		if err != nil {
			return err
		}
		vel, err := world.GetComponent[TestVelocity](id)
		if err != nil {
			return err
		}
		pos.X += vel.X * delta
		pos.Y += vel.Y * delta
		if err := world.SetComponent(id, pos); err != nil {
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
		t.Errorf("Register failed: %v", err)
	}
	if !sys.initCalled {
		t.Error("expected Init to be called on registration")
	}

	unit, err := world.CreateWithComponents("Unit", TestPosition{X: 0, Y: 0}, TestVelocity{X: 1, Y: 2})
	if err != nil {
		t.Errorf("CreateWithComponents failed: %v", err)
	}

	turret, err := world.CreateWithComponents("Turret", TestPosition{X: 5, Y: 5})
	if err != nil {
		t.Errorf("CreateWithComponents failed: %v", err)
	}
	world.SetParent(turret.ID, unit.ID)

	const ticks = 3
	for i := 0; i < ticks; i++ {
		if err := manager.Update(world, 1.0); err != nil {
			t.Errorf("Update failed on tick %d: %v", i, err)
		}
	}

	gotPos, err := world.GetComponent[TestPosition](unit.ID)
	if err != nil {
		t.Errorf("GetComponent failed: %v", err)
	}
	if gotPos != (TestPosition{X: 3, Y: 6}) {
		t.Errorf("expected unit to have moved to (3,6) after %d ticks, got %#v", ticks, gotPos)
	}

	// The turret has no velocity, so the movement system must leave it untouched.
	turretPos, err := world.GetComponent[TestPosition](turret.ID)
	if err != nil {
		t.Errorf("GetComponent failed: %v", err)
	}
	if turretPos != (TestPosition{X: 5, Y: 5}) {
		t.Errorf("expected turret to stay put, got %#v", turretPos)
	}

	if err := world.DestroyEntity(unit.ID, true); err != nil {
		t.Errorf("DestroyEntity failed: %v", err)
	}
	world.Cleanup()

	if world.HasEntity(unit.ID) || world.HasEntity(turret.ID) {
		t.Error("cascade destroy should have removed both the unit and its child turret")
	}

	if err := manager.Shutdown(world); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
	if !sys.shutdownCalled {
		t.Error("expected Shutdown to be called")
	}
}
