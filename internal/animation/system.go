package animation

import (
	"fmt"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
)

// System processes animation playback for entities with
// [components.Animation] and [components.Sprite] components. It reads
// clips from the [*ClipStore] world resource and emits [AnimationEvent]
// values via the [*events.EventBus] world resource.
type System struct {
	query    *ecs.Query
	store    *ClipStore
	eventBus *events.EventBus
}

// NewSystem creates an animation system. The clip store and event bus
// are resolved from world resources during [System.Init].
func NewSystem() *System {
	return &System{}
}

func (s *System) Init(world *ecs.World) error {
	s.query = world.NewQuery().WithRequiredComponents(
		components.Animation{},
		components.Sprite{},
	)

	if store, ok := world.GetResource[*ClipStore](); ok {
		s.store = store
	} else {
		return fmt.Errorf("failed to get ClipStore resource")
	}
	if eventBus, ok := world.GetResource[*events.EventBus](); ok {
		s.eventBus = eventBus
	} else {
		return fmt.Errorf("failed to get EventBus resource")
	}
	return nil
}

func (s *System) Update(world *ecs.World, delta float64) error {
	if s.query == nil || s.store == nil || s.eventBus == nil {
		return fmt.Errorf("system not properly initialized")
	}
	for entry := range s.query.Execute() {
		anim, _ := entry.Get[components.Animation]()
		if !anim.Playing {
			continue
		}

		clip, ok := s.store.Get(anim.ClipID)
		if !ok {
			continue
		}
		anim.FrameTime += delta * anim.PlaybackSpeed
		frameDuration := 1 / clip.FPS

		for anim.FrameTime >= frameDuration {
			anim.FrameTime -= frameDuration
			anim.FrameIndex++

			// Emit frame events for the index we just advanced to,
			// but only if it's in range.
			if anim.FrameIndex < len(clip.Frames) && clip.FrameEvents != nil {
				if eventNames, hasEvent := clip.FrameEvents[anim.FrameIndex]; hasEvent {
					for _, eventName := range eventNames {
						s.eventBus.Emit(AnimationEvent{
							EntityID:   entry.EntityID,
							ClipID:     anim.ClipID,
							Type:       AnimationEventFrame,
							FrameIndex: anim.FrameIndex,
							EventName:  eventName,
						}, "animation.System")
					}
				}
			}

			if anim.FrameIndex >= len(clip.Frames) {
				switch clip.Loop {
				case LoopForever:
					anim.FrameIndex = 0
					s.eventBus.Emit(AnimationEvent{
						EntityID: entry.EntityID,
						ClipID:   anim.ClipID,
						Type:     AnimationEventLooped,
					}, "animation.System")
				case LoopNone:
					anim.FrameIndex = len(clip.Frames) - 1
					anim.Playing = false
					s.eventBus.Emit(AnimationEvent{
						EntityID: entry.EntityID,
						ClipID:   anim.ClipID,
						Type:     AnimationEventFinished,
					}, "animation.System")
				}
				break
			}
		}
		_ = world.SetComponent(entry.EntityID, anim)
	}
	return nil
}

func (s *System) Shutdown(world *ecs.World) error {
	// Perform any necessary cleanup here
	return nil
}
