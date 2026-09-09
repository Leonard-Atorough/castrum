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
package timersystem

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/internal/ecs"
	"github.com/leonard-atorough/castrum/timers"
)

// TimerSystem is a System that updates all Timer components in the world.
// It accumulates delta time for each running timer, emits TimerCompletedEvent
// when duration is reached, and removes one-shot timers after firing.
//
// TimerSystem preallocates a cleanup bucket to avoid allocations for
// entities that have expired one-shot timers.
type TimerSystem struct {
	// Preallocated bucket for cleanup to avoid allocations per update
	timersToRemove []*ecs.Entity
	Capacity       int
	timerQuery     *ecs.Query
}

// NewTimerSystem creates a new TimerSystem with a given capacity for cleanup operations.
func NewTimerSystem(capacity int) *TimerSystem {
	return &TimerSystem{
		Capacity: capacity,
	}
}

func (ts *TimerSystem) Init(world *ecs.World) error {
	ts.timersToRemove = make([]*ecs.Entity, 0, ts.Capacity)
	ts.timerQuery = world.NewQuery().WithRequiredComponents(components.Timer{})
	return nil
}

func (ts *TimerSystem) Update(world *ecs.World, deltaTime float64) error {
	bus, ok := ecs.GetResource[*events.EventBus](world)
	if !ok {
		bus = nil // EventBus not registered
	}

	for result := range ts.timerQuery.Execute() {
		entityID := result.EntityID
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
			if bus != nil {
				bus.Emit(timers.TimerCompletedEvent{
					EntityID: entityID,
					TimerID:  timer.ID,
				}, "TimerSystem")
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
		world.RemoveComponent[components.Timer](entity.ID)
	}
	ts.timersToRemove = ts.timersToRemove[:0]
	return nil
}

func (ts *TimerSystem) Shutdown(world *ecs.World) error {

	for result := range ts.timerQuery.Execute() {
		entityID := result.EntityID
		timer, _ := world.GetComponent[components.Timer](entityID)
		timer.Stop()
		// Update the component back in the world since Timer is a value type
		world.SetComponent(entityID, timer)
	}
	// this helps in the future where timers need to be serialised and restored with their current state
	return nil
}
