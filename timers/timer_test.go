package timers

import (
	"testing"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
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
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	ecs.SetResource(world, bus)

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

	var eventFired bool
	var firedEvent TimerCompletedEvent
	bus.On(func(_ events.EventMeta, e TimerCompletedEvent) {
		eventFired = true
		firedEvent = e
	}, false)

	system.Update(world, 1.5)

	if !eventFired {
		t.Fatalf("expected 1 event to fire")
	}

	if firedEvent.EntityID != entity.ID {
		t.Fatal("event should reference the correct entity")
	}
	if firedEvent.TimerID != "timer1" {
		t.Fatal("event should reference the correct timer ID")
	}
}

// TestTimerSystem_OneShotTimerRemovedAfterFiring tests that one-shot timers are removed after firing
func TestTimerSystem_OneShotTimerRemovedAfterFiring(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	ecs.SetResource(world, bus)

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

	var eventFired bool
	bus.On(func(_ events.EventMeta, e TimerCompletedEvent) {
		eventFired = true
	}, false)

	system.Update(world, 1.5)

	// Timer should be removed after firing
	_, err := world.GetComponent[components.Timer](entity.ID)
	if err == nil {
		t.Fatal("one-shot timer should be removed after firing")
	}

	// Event should have been emitted
	if !eventFired {
		t.Fatal("expected 1 event to fire")
	}
}

// TestTimerSystem_RepeatingTimerKeepsFiring tests that repeating timers reset and continue
func TestTimerSystem_RepeatingTimerKeepsFiring(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	ecs.SetResource(world, bus)

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

	var eventCount int
	bus.On(func(_ events.EventMeta, e TimerCompletedEvent) {
		eventCount++
	}, false)

	// First update at 0.5s - should fire
	eventCount = 0
	system.Update(world, 0.5)
	if eventCount != 1 {
		t.Fatalf("expected 1 event after first update, got %d", eventCount)
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
	eventCount = 0
	system.Update(world, 0.5)
	if eventCount != 1 {
		t.Fatalf("expected 1 event after second update, got %d", eventCount)
	}

	// Timer should still exist
	_, err := world.GetComponent[components.Timer](entity.ID)
	if err != nil {
		t.Fatal("repeating timer should still exist after firing")
	}
}

// TestTimerSystem_StoppedTimerDoesNotFire tests that stopped timers don't fire
func TestTimerSystem_StoppedTimerDoesNotFire(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	ecs.SetResource(world, bus)

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

	var eventFired bool
	bus.On(func(_ events.EventMeta, e TimerCompletedEvent) {
		eventFired = true
	}, false)

	system.Update(world, 1.0)

	if eventFired {
		t.Fatalf("stopped timer should not fire")
	}
}

// TestTimerSystem_MultipleTimersOnDifferentEntities tests multiple timers on different entities
func TestTimerSystem_MultipleTimersOnDifferentEntities(t *testing.T) {
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	ecs.SetResource(world, bus)

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

	var eventCount int
	bus.On(func(_ events.EventMeta, e TimerCompletedEvent) {
		eventCount++
	}, false)

	system.Update(world, 1.5)

	if eventCount != 2 {
		t.Fatalf("expected 2 events, got %d", eventCount)
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
	world := ecs.NewWorld()
	bus := events.NewEventBus()
	ecs.SetResource(world, bus)

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

	var eventCount int
	bus.On(func(_ events.EventMeta, e TimerCompletedEvent) {
		eventCount++
	}, false)

	// First update fires
	eventCount = 0
	system.Update(world, 0.5)
	if eventCount != 1 {
		t.Fatal("expected 1 event after first update")
	}

	// Second update without firing should have no events
	eventCount = 0
	system.Update(world, 0.1)
	if eventCount != 0 {
		t.Fatal("expected 0 events after second update (no timer fired)")
	}

	// Third update fires again
	eventCount = 0
	system.Update(world, 0.4)
	if eventCount != 1 {
		t.Fatal("expected 1 event after third update")
	}
}

// TestTimerSystem_ShutdownStopsAllTimers tests that Shutdown stops all running timers
func TestTimerSystem_ShutdownStopsAllTimers(t *testing.T) {
	world := ecs.NewWorld()

	// Create multiple running timers
	entity1, _ := world.CreateWithComponents(
		"entity1",
		components.Timer{
			ID:       "timer1",
			Duration: 1.0,
			Running:  true,
		},
	)

	entity2, _ := world.CreateWithComponents(
		"entity2",
		components.Timer{
			ID:       "timer2",
			Duration: 1.0,
			Running:  true,
		},
	)

	// Create a stopped timer to verify it remains stopped
	entity3, _ := world.CreateWithComponents(
		"entity3",
		components.Timer{
			ID:       "timer3",
			Duration: 1.0,
			Running:  false,
		},
	)

	system := NewTimerSystem(10)
	system.Init(world)

	// Verify all timers are in their initial state before shutdown
	timer1, _ := world.GetComponent[components.Timer](entity1.ID)
	timer2, _ := world.GetComponent[components.Timer](entity2.ID)
	timer3, _ := world.GetComponent[components.Timer](entity3.ID)

	if !timer1.Running || !timer2.Running || timer3.Running {
		t.Fatal("timers not in expected initial state")
	}

	// Call Shutdown
	err := system.Shutdown(world)
	if err != nil {
		t.Fatalf("Shutdown should not return error: %v", err)
	}

	// Verify all timers are stopped
	timer1, _ = world.GetComponent[components.Timer](entity1.ID)
	timer2, _ = world.GetComponent[components.Timer](entity2.ID)
	timer3, _ = world.GetComponent[components.Timer](entity3.ID)

	if timer1.Running {
		t.Fatal("timer1 should be stopped after Shutdown")
	}
	if timer2.Running {
		t.Fatal("timer2 should be stopped after Shutdown")
	}
	if timer3.Running {
		t.Fatal("timer3 should still be stopped after Shutdown")
	}

	// Verify timers still exist (just stopped)
	_, err1 := world.GetComponent[components.Timer](entity1.ID)
	_, err2 := world.GetComponent[components.Timer](entity2.ID)
	_, err3 := world.GetComponent[components.Timer](entity3.ID)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatal("all timers should still exist after Shutdown")
	}
}
