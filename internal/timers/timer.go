// Package timers provides a Timer component and TimerSystem for managing
// one-shot and repeating timers within the ECS. Timers accumulate delta time
// and emit TimerCompletedEvent when their duration is reached.
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
//		components.Timer{
//			ID:       "cooldown",
//			Duration: 2.0,
//			Running:  true,
//			Once:     true,
//		},
//	)
//	// In game loop, register TimerSystem:
//	system := timers.NewTimerSystem(64)
//	systems.Register("timers", -1, system, world)
//	// Listen for events:
//	for _, event := range system.Events() {
//		fmt.Printf("Timer %s completed on entity %v\n", event.TimerID, event.EntityID)
//	}
package timers

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/core"
)

type TimerCompletedEvent struct {
	EntityID core.EntityID
	TimerID  components.TimerID
}

// TimerSystem is a System that updates all Timer components in the world.
// It accumulates delta time for each running timer, emits TimerCompletedEvent
// when duration is reached, and removes one-shot timers after firing.
//
// TimerSystem preallocates a cleanup bucket to avoid allocations for
// entities that have expired one-shot timers.
type TimerSystem struct {
	// Preallocated bucket for cleanup to avoid allocations per update
	timersToRemove []*core.Entity
	// Events from the last update
	events   []TimerCompletedEvent
	Capacity int
}

// NewTimerSystem creates a new TimerSystem with a given capacity for cleanup operations.
func NewTimerSystem(capacity int) *TimerSystem {
	return &TimerSystem{
		Capacity: capacity,
	}
}

// Events returns timer events emitted during the last Update.
func (ts *TimerSystem) Events() []TimerCompletedEvent {
	if ts.events == nil {
		return []TimerCompletedEvent{}
	}
	return ts.events
}

func (ts *TimerSystem) Init(world *core.World) error {
	ts.timersToRemove = make([]*core.Entity, 0, ts.Capacity)
	ts.events = make([]TimerCompletedEvent, 0, ts.Capacity)
	return nil
}

func (ts *TimerSystem) Update(world *core.World, deltaTime float64) error {
	// Clear previous frame's events
	ts.events = ts.events[:0]

	timers := core.QueryFor[components.Timer](world)
	for _, entityID := range timers {
		timer, err := world.GetComponent[components.Timer](entityID)
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
			// Emit a TimerCompletedEvent for this timer
			ts.events = append(ts.events, TimerCompletedEvent{
				EntityID: entityID,
				TimerID:  timer.ID,
			})

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
		world.RemoveComponent[components.Timer](entity.ID)
	}
	ts.timersToRemove = ts.timersToRemove[:0]
	return nil
}

func (ts *TimerSystem) Shutdown(world *core.World) error {
	timers := core.QueryFor[components.Timer](world)
	for _, entityID := range timers {
		timer, _ := world.GetComponent[components.Timer](entityID)
		timer.Stop()
		// Update the component back in the world since Timer is a value type
		world.SetComponent(entityID, timer)
	}
	// this helps in the future where timers need to be serialised and restored with their current state
	return nil
}
