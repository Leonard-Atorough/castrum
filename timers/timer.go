// Package timers defines timer-related events emitted by the engine's timer
// system. The system itself lives in internal/timers, coupled to the ECS World.
package timers

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
)

// TimerCompletedEvent is emitted when a Timer component's duration elapses.
type TimerCompletedEvent struct {
	EntityID ecs.EntityID
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
	bus, ok := world.GetResource[*events.EventBus]()
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
				bus.Emit(TimerCompletedEvent{
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
