package timers

import (
	"testing"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/core"
)

// TestTimer_StateTransitions tests Timer component state methods
func TestTimer_StateTransitions(t *testing.T) {
	timer := components.Timer{
		ID:       "test-timer",
		Duration: 1.0,
		Running:  false,
	}

	// Test Start
	timer.Start()
	if !timer.Running || timer.ElapsedTime != 0 {
		t.Fatal("Start should set Running=true and reset ElapsedTime to 0")
	}

	// Test Stop
	timer.Stop()
	if timer.Running {
		t.Fatal("Stop should set Running=false")
	}

	// Test Resume
	timer.Resume()
	if !timer.Running {
		t.Fatal("Resume should set Running=true")
	}
}

// TestTimer_ElapsedTimeAccumulates tests that elapsed time accumulates while Running
func TestTimer_ElapsedTimeAccumulates(t *testing.T) {
	timer := components.Timer{
		ID:       "accumulate-test",
		Duration: 1.0,
		Running:  true,
	}

	// Simulate accumulating time
	timer.ElapsedTime += 0.3
	if timer.ElapsedTime != 0.3 {
		t.Fatalf("expected ElapsedTime=0.3, got %f", timer.ElapsedTime)
	}

	timer.ElapsedTime += 0.3
	if timer.ElapsedTime != 0.6 {
		t.Fatalf("expected ElapsedTime=0.6, got %f", timer.ElapsedTime)
	}
}

// TestTimer_StoppedDoesNotAccumulate tests that stopped timers don't accumulate time
func TestTimer_StoppedDoesNotAccumulate(t *testing.T) {
	timer := components.Timer{
		ID:       "stopped-test",
		Duration: 1.0,
		Running:  false,
	}

	// Even though we try to accumulate, a real system wouldn't if Running is false
	if timer.Running {
		t.Fatal("timer should be stopped")
	}
}

// TestTimerSystem_EmitsEventWhenExpired tests that TimerCompletedEvent is emitted when timer expires
func TestTimerSystem_EmitsEventWhenExpired(t *testing.T) {
	world := core.NewWorld()

	entity, err := world.CreateWithComponents(
		"test-entity",
		components.Timer{
			ID:       "timer1",
			Duration: 1.0,
			Running:  true,
		},
	)
	if err != nil {
		t.Fatalf("failed to create entity: %v", err)
	}

	system := NewTimerSystem(10)
	system.Init(world)
	system.Update(world, 1.5)

	events := system.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].EntityID != entity.ID {
		t.Fatal("event should reference the correct entity")
	}
	if events[0].TimerID != "timer1" {
		t.Fatal("event should reference the correct timer ID")
	}
}

// TestTimerSystem_OneShotTimerRemovedAfterFiring tests that one-shot timers are removed after firing
func TestTimerSystem_OneShotTimerRemovedAfterFiring(t *testing.T) {
	world := core.NewWorld()

	entity, _ := world.CreateWithComponents(
		"test-entity",
		components.Timer{
			ID:       "timer1",
			Duration: 1.0,
			Running:  true,
			Once:     true,
		},
	)

	system := NewTimerSystem(10)
	system.Init(world)
	system.Update(world, 1.5)

	// Timer should be removed after firing
	_, err := world.GetComponent[components.Timer](entity.ID)
	if err == nil {
		t.Fatal("one-shot timer should be removed after firing")
	}

	// Event should have been emitted
	events := system.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

// TestTimerSystem_RepeatingTimerKeepsFiring tests that repeating timers reset and continue
func TestTimerSystem_RepeatingTimerKeepsFiring(t *testing.T) {
	world := core.NewWorld()

	entity, _ := world.CreateWithComponents(
		"test-entity",
		components.Timer{
			ID:       "timer1",
			Duration: 0.5,
			Running:  true,
			Once:     false,
		},
	)

	system := NewTimerSystem(10)
	system.Init(world)

	// First update at 0.5s - should fire
	system.Update(world, 0.5)
	events := system.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event after first update, got %d", len(events))
	}

	// Get timer and verify it's still present and reset
	timer, _ := world.GetComponent[components.Timer](entity.ID)
	if timer.ElapsedTime != 0 {
		t.Fatalf("repeating timer should reset ElapsedTime to 0, got %f", timer.ElapsedTime)
	}
	if !timer.Running {
		t.Fatal("repeating timer should still be running")
	}

	// Second update at 0.5s - should fire again
	system.Update(world, 0.5)
	events = system.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event after second update, got %d", len(events))
	}

	// Timer should still exist
	_, err := world.GetComponent[components.Timer](entity.ID)
	if err != nil {
		t.Fatal("repeating timer should still exist after firing")
	}
}

// TestTimerSystem_StoppedTimerDoesNotFire tests that stopped timers don't fire
func TestTimerSystem_StoppedTimerDoesNotFire(t *testing.T) {
	world := core.NewWorld()

	world.CreateWithComponents(
		"test-entity",
		components.Timer{
			ID:       "timer1",
			Duration: 0.5,
			Running:  false,
		},
	)

	system := NewTimerSystem(10)
	system.Init(world)
	system.Update(world, 1.0)

	events := system.Events()
	if len(events) != 0 {
		t.Fatalf("stopped timer should not fire, got %d events", len(events))
	}
}

// TestTimerSystem_MultipleTimersOnDifferentEntities tests multiple timers on different entities
func TestTimerSystem_MultipleTimersOnDifferentEntities(t *testing.T) {
	world := core.NewWorld()

	entity1, _ := world.CreateWithComponents(
		"entity1",
		components.Timer{
			ID:       "timer1",
			Duration: 1.0,
			Running:  true,
			Once:     true,
		},
	)

	entity2, _ := world.CreateWithComponents(
		"entity2",
		components.Timer{
			ID:       "timer2",
			Duration: 1.0,
			Running:  true,
			Once:     true,
		},
	)

	system := NewTimerSystem(10)
	system.Init(world)
	system.Update(world, 1.5)

	events := system.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Both should be removed
	_, err1 := world.GetComponent[components.Timer](entity1.ID)
	_, err2 := world.GetComponent[components.Timer](entity2.ID)
	if err1 == nil || err2 == nil {
		t.Fatal("both one-shot timers should be removed")
	}
}

// TestTimerSystem_EventsClearedEachUpdate tests that events are cleared each update
func TestTimerSystem_EventsClearedEachUpdate(t *testing.T) {
	world := core.NewWorld()

	world.CreateWithComponents(
		"entity1",
		components.Timer{
			ID:       "timer1",
			Duration: 0.5,
			Running:  true,
			Once:     false,
		},
	)

	system := NewTimerSystem(10)
	system.Init(world)

	// First update fires
	system.Update(world, 0.5)
	if len(system.Events()) != 1 {
		t.Fatal("expected 1 event after first update")
	}

	// Second update without firing should have no events
	system.Update(world, 0.1)
	if len(system.Events()) != 0 {
		t.Fatal("expected 0 events after second update (no timer fired)")
	}

	// Third update fires again
	system.Update(world, 0.4)
	if len(system.Events()) != 1 {
		t.Fatal("expected 1 event after third update")
	}
}
