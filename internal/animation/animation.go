package animation

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/core"
)

type AnimationEventType int

const (
	FrameEventType AnimationEventType = iota
	CompleteEventType
)

type AnimationEvent struct {
	EntityID   core.EntityID
	Type       AnimationEventType
	FrameIndex int
}

// System is a lifecycle handler for animation components.
// It processes all Animatable components and emits animation events.
// To control animations, modify Animatable components directly via world.SetComponent.
type System struct {
	events []AnimationEvent
	query  *core.Query
}

// Events returns animation events emitted during the last Update.
func (as *System) Events() []AnimationEvent {
	if as.events == nil {
		return []AnimationEvent{}
	}
	return as.events
}

func (as *System) Init(world *core.World) error {
	as.query = world.NewQuery().WithRequiredComponents(
		components.Animatable{},
		components.Renderable{},
	)
	as.events = make([]AnimationEvent, 0, 64)

	return nil
}

// Update processes all Animatable components, advancing frame time and emitting events.
func (as *System) Update(world *core.World, delta float64) error {
	as.events = as.events[:0] // clear previous frame events

	// Use the new query builder to iterate over Animatable entities
	for entry := range as.query.Execute() {
		animComp, _ := entry.Get[components.Animatable]()
		entityID := entry.EntityID

		if _, exists := animComp.Animations[animComp.CurrentAnimation]; !exists {
			continue
		}

		if !animComp.Animations[animComp.CurrentAnimation].Playing {
			continue
		}

		current := animComp.Animations[animComp.CurrentAnimation]
		current.FrameTime += delta

		if current.FrameTime >= current.FrameSpeed {
			current.FrameTime = 0
			current.FrameIndex++

			// Emit frame event
			if current.FrameIndex < len(current.Frames) {
				as.events = append(as.events, AnimationEvent{
					EntityID:   entityID,
					Type:       FrameEventType,
					FrameIndex: current.FrameIndex,
				})

				// Trigger frame callback if defined
				if current.FrameEvents != nil {
					if callback, exists := current.FrameEvents[current.FrameIndex]; exists {
						callback()
					}
				}
			}

			if current.FrameIndex >= len(current.Frames) {
				if current.Loop {
					current.FrameIndex = 0
				} else {
					current.FrameIndex = len(current.Frames) - 1
					current.Playing = false

					// Emit completion event
					as.events = append(as.events, AnimationEvent{
						EntityID: entityID,
						Type:     CompleteEventType,
					})

					// Trigger completion callback if defined
					if current.Callback != nil {
						current.Callback()
					}
				}
			}
		}

		animComp.Animations[animComp.CurrentAnimation] = current
		_ = world.SetComponent(entityID, animComp)
	}
	return nil
}

func (as *System) Shutdown(world *core.World) error {
	return nil
}
