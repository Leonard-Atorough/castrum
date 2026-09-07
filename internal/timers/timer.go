// Package timers provides a Timer component and TimerSystem for managing
// one-shot and repeating timers within the ECS. Timers accumulate delta time
// and fire callbacks when their duration is reached.
//
// A Timer is a component that can be attached to any entity. The TimerSystem
// queries for timers each frame and updates them based on delta time.
// One-shot timers are automatically removed after firing; repeating timers
// reset and continue.
//
// Example usage:
//
//	entity, _ := world.CreateWithComponents(
//		"my-entity",
//		Timer{
//			ID:       "cooldown",
//			Duration: 2.0,
//			Running:  true,
//			Once:     true,
//			OnTimerTick: func(e *core.Entity) {
//				fmt.Println("Timer fired!")
//			},
//		},
//	)
//	// In game loop, register TimerSystem:
//	systems.Register(timers.NewTimerSystem(64))
package timers

import (
	"github.com/leonard-atorough/castrum/internal/core"
)

// TimerID uniquely identifies a timer within a Manager.
type TimerID string

// Timer is a component that tracks elapsed time and fires a callback when
// its duration is reached. Timers can be one-shot (fires once then is removed)
// or repeating (resets and continues firing).
//
// Timer is a value type and should be attached to entities via AddComponent.
// The TimerSystem handles update logic and callback firing.
type Timer struct {
	// Unique identifier for this timer (useful for multiple timers per entity)
	ID TimerID
	// Duration for which the timer runs
	Duration float64
	// Elapsed time since the timer started (accumulated by TimerSystem)
	ElapsedTime float64
	// Whether the timer is currently running
	Running bool
	// Whether this is a one-shot timer (fires once then stops)
	Once bool
	// optional callback function to be called when the timer completes
	OnTimerTick func(e *core.Entity)
}

// Start begins the timer, resetting elapsed time to zero.
func (t *Timer) Start() {
	t.Running = true
	t.ElapsedTime = 0
}

// Stop pauses the timer without resetting elapsed time.
// Resume can be called to continue from the same elapsed time.
func (t *Timer) Stop() {
	t.Running = false
}

// Resume continues a stopped timer from its current elapsed time.
func (t *Timer) Resume() {
	t.Running = true
}

// TimerSystem is a System that updates all Timer components in the world.
// It accumulates delta time for each running timer, fires callbacks when
// duration is reached, and removes one-shot timers after firing.
//
// TimerSystem preallocates a cleanup bucket to avoid allocations for
// entities that have expired one-shot timers.
type TimerSystem struct {
	// Preallocated bucket for cleanup to avoid allocations per update
	timersToRemove []*core.Entity
	Capacity       int
}

func (ts *TimerSystem) Init(world *core.World) error {
	ts.timersToRemove = make([]*core.Entity, 0, ts.Capacity)
	return nil
}

func (ts *TimerSystem) Update(world *core.World, deltaTime float64) error {
	timers := core.QueryFor[Timer](world)
	for _, entityID := range timers {
		timer, err := world.GetComponent[Timer](entityID)
		if err != nil {
			continue
		}

		entity, exists := world.GetEntity(entityID)
		if !exists {
			continue
		}

		if timer.Running == false {
			continue
		}
		timer.ElapsedTime += deltaTime
		if timer.ElapsedTime >= timer.Duration {
			if timer.OnTimerTick != nil {
				timer.OnTimerTick(entity)
			}
			if timer.Once {
				ts.timersToRemove = append(ts.timersToRemove, entity)
			} else {
				timer.ElapsedTime = 0
			}
		}
		// Update the component back in the world since Timer is a value type
		world.SetComponent(entityID, timer)
	}

	for _, entity := range ts.timersToRemove {
		world.RemoveComponent[Timer](entity.ID)
	}
	ts.timersToRemove = ts.timersToRemove[:0]
	return nil
}

func (ts *TimerSystem) Shutdown(world *core.World) error {
	timers := core.QueryFor[Timer](world)
	for _, entityID := range timers {
		timer, _ := world.GetComponent[Timer](entityID)
		timer.Stop()
		// Update the component back in the world since Timer is a value type
		world.SetComponent(entityID, timer)
	}
	// this helps in the future where timers need to be serialised and restored with their current state
	return nil
}
