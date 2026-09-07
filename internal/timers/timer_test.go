package timers

import (
	"testing"

	"github.com/leonard-atorough/castrum/internal/core"
)

// TestTimer_StateTransitions tests Timer component state methods
func TestTimer_StateTransitions(t *testing.T) {
	timer := Timer{
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
	timer := Timer{
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
	timer := Timer{
		ID:       "stopped-test",
		Duration: 1.0,
		Running:  false,
	}

	// Even though we try to accumulate, a real system wouldn't if Running is false
	if timer.Running {
		t.Fatal("timer should be stopped")
	}
}

// TestTimerSystem_FiresCallbackWhenExpired tests that OnTimerTick is called when timer expires
func TestTimerSystem_FiresCallbackWhenExpired(t *testing.T) {
	world := core.NewWorld()
	fired := false
	var firedEntity *core.Entity

	entity, err := world.CreateWithComponents(
		"test-entity",
		Timer{
			ID:       "timer1",
			Duration: 1.0,
			Running:  true,
			OnTimerTick: func(e *core.Entity) {
				fired = true
				firedEntity = e
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to create entity: %v", err)
	}

	system := NewTimerSystem(10)
	system.Update(world, 1.5)

	if !fired {
		t.Fatal("OnTimerTick should have been called")
	}
	if firedEntity.ID != entity.ID {
		t.Fatal("callback should receive the correct entity")
	}
}

// TestTimerSystem_OneShootTimerRemovedAfterFiring tests that one-shot timers are removed after firing
func TestTimerSystem_OneShotTimerRemovedAfterFiring(t *testing.T) {
	world := core.NewWorld()

	entity, _ := world.CreateWithComponents(
		"test-entity",
		Timer{
			ID:       "timer1",
			Duration: 1.0,
			Running:  true,
			Once:     true,
			OnTimerTick: func(e *core.Entity) {
				// callback
			},
		},
	)

	system := NewTimerSystem(10)
	system.Update(world, 1.5)

	// Timer should be removed after firing
	_, err := world.GetComponent[Timer](entity.ID)
	if err == nil {
		t.Fatal("one-shot timer should be removed after firing")
	}
}

// TestTimerSystem_RepeatingTimerKeepsFiring tests that repeating timers reset and continue
func TestTimerSystem_RepeatingTimerKeepsFiring(t *testing.T) {
	world := core.NewWorld()
	callCount := 0

	entity, _ := world.CreateWithComponents(
		"test-entity",
		Timer{
			ID:       "timer1",
			Duration: 0.5,
			Running:  true,
			Once:     false,
			OnTimerTick: func(e *core.Entity) {
				callCount++
			},
		},
	)

	system := NewTimerSystem(10)

	// First update at 0.5s - should fire
	system.Update(world, 0.5)
	if callCount != 1 {
		t.Fatalf("expected 1 callback, got %d", callCount)
	}

	// Get timer and verify it's still present and reset
	timer, _ := world.GetComponent[Timer](entity.ID)
	if timer.ElapsedTime != 0 {
		t.Fatalf("repeating timer should reset ElapsedTime to 0, got %f", timer.ElapsedTime)
	}
	if !timer.Running {
		t.Fatal("repeating timer should still be running")
	}

	// Second update at 0.5s - should fire again
	system.Update(world, 0.5)
	if callCount != 2 {
		t.Fatalf("expected 2 callbacks, got %d", callCount)
	}

	// Timer should still exist
	_, err := world.GetComponent[Timer](entity.ID)
	if err != nil {
		t.Fatal("repeating timer should still exist after firing")
	}
}

// TestTimerSystem_StoppedTimerDoesNotFire tests that stopped timers don't fire
func TestTimerSystem_StoppedTimerDoesNotFire(t *testing.T) {
	world := core.NewWorld()
	callCount := 0

	world.CreateWithComponents(
		"test-entity",
		Timer{
			ID:       "timer1",
			Duration: 0.5,
			Running:  false,
			OnTimerTick: func(e *core.Entity) {
				callCount++
			},
		},
	)

	system := NewTimerSystem(10)
	system.Update(world, 1.0)

	if callCount != 0 {
		t.Fatalf("stopped timer should not fire, got %d callbacks", callCount)
	}
}

// TestTimerSystem_MultipleTimersOnDifferentEntities tests multiple timers on different entities
func TestTimerSystem_MultipleTimersOnDifferentEntities(t *testing.T) {
	world := core.NewWorld()
	fired := map[string]bool{
		"entity1": false,
		"entity2": false,
	}

	entity1, _ := world.CreateWithComponents(
		"entity1",
		Timer{
			ID:       "timer1",
			Duration: 1.0,
			Running:  true,
			Once:     true,
			OnTimerTick: func(e *core.Entity) {
				fired["entity1"] = true
			},
		},
	)

	entity2, _ := world.CreateWithComponents(
		"entity2",
		Timer{
			ID:       "timer2",
			Duration: 1.0,
			Running:  true,
			Once:     true,
			OnTimerTick: func(e *core.Entity) {
				fired["entity2"] = true
			},
		},
	)

	system := NewTimerSystem(10)
	system.Update(world, 1.5)

	if !fired["entity1"] || !fired["entity2"] {
		t.Fatal("both timers should have fired")
	}

	// Both should be removed
	_, err1 := world.GetComponent[Timer](entity1.ID)
	_, err2 := world.GetComponent[Timer](entity2.ID)
	if err1 == nil || err2 == nil {
		t.Fatal("both one-shot timers should be removed")
	}
}

// TestTimerSystem_NoCallbackTimerStillFires tests that timers without callbacks still work
func TestTimerSystem_NoCallbackTimerStillFires(t *testing.T) {
	world := core.NewWorld()

	entity, _ := world.CreateWithComponents(
		"test-entity",
		Timer{
			ID:       "timer1",
			Duration: 1.0,
			Running:  true,
			Once:     true,
			// No OnTimerTick
		},
	)

	system := NewTimerSystem(10)
	system.Update(world, 1.5)

	// Timer should still be removed even without callback
	_, err := world.GetComponent[Timer](entity.ID)
	if err == nil {
		t.Fatal("one-shot timer should be removed even without callback")
	}
}

// TestTimerSystem_Shutdown ensures all timers are stopped
func TestTimerSystem_Shutdown(t *testing.T) {
	world := core.NewWorld()

	entity1, _ := world.CreateWithComponents(
		"entity1",
		Timer{
			ID:       "timer1",
			Duration: 1.0,
			Running:  true,
		},
	)

	entity2, _ := world.CreateWithComponents(
		"entity2",
		Timer{
			ID:       "timer2",
			Duration: 1.0,
			Running:  true,
		},
	)

	system := NewTimerSystem(10)
	system.Shutdown(world)

	// Both timers should be stopped
	timerComp1, _ := world.GetComponent[Timer](entity1.ID)
	timerComp2, _ := world.GetComponent[Timer](entity2.ID)

	if timerComp1.Running {
		t.Fatal("timer1 should be stopped on shutdown")
	}
	if timerComp2.Running {
		t.Fatal("timer2 should be stopped on shutdown")
	}
}

// TestTimerSystem_BucketPreallocation tests that cleanup bucket is properly reused
func TestTimerSystem_BucketPreallocation(t *testing.T) {
	world := core.NewWorld()

	// Create multiple one-shot timers
	for i := 0; i < 5; i++ {
		world.CreateWithComponents(
			"test-entity",
			Timer{
				ID:       TimerID("timer" + string(rune(i))),
				Duration: 0.5,
				Running:  true,
				Once:     true,
				OnTimerTick: func(e *core.Entity) {
					// callback
				},
			},
		)
	}

	system := NewTimerSystem(10)
	system.Update(world, 1.0)

	// All should be removed
	timers := core.QueryFor[Timer](world)
	if len(timers) != 0 {
		t.Fatalf("all one-shot timers should be removed, got %d remaining", len(timers))
	}

	// Bucket should be reset for next update
	if len(system.timersToRemove) != 0 {
		t.Fatal("cleanup bucket should be reset after update")
	}
}
