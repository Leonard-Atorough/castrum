// Package timers defines timer-related events emitted by the engine's timer
// system. The system itself lives in internal/timers, coupled to the ECS World.
package timers

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/ecs"
)

// TimerCompletedEvent is emitted when a Timer component's duration elapses.
type TimerCompletedEvent struct {
	EntityID ecs.EntityID
	TimerID  components.TimerID
}
