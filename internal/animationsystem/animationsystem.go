package animationsystem

import (
	"github.com/leonard-atorough/castrum/animation"
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/events"
	"github.com/leonard-atorough/castrum/internal/ecs"
)

// System processes animation playback for entities with Animation components.
// It delegates to the Manager for clip resolution and orchestrates frame advancement
// and event emission. Does not handle clip creation or configuration.
type System struct {
	query   *ecs.Query
	manager *animation.AnimationClipStore
}

// NewSystem creates a new animation system with the given manager.
func NewSystem(manager *animation.AnimationClipStore) *System {
	return &System{
		manager: manager,
	}
}

func (as *System) Init(world *ecs.World) error {
	as.query = world.NewQuery().WithRequiredComponents(
		components.Animation{},
		components.Sprite{},
	)

	return nil
}

// Update processes all Animation components, advancing frame time and emitting events.
func (as *System) Update(world *ecs.World, delta float64) error {
	bus, ok := ecs.GetResource[*events.EventBus](world)
	if !ok {
		return nil // EventBus not registered, skip event emission
	}

	for entry := range as.query.Execute() {
		anim, _ := entry.Get[components.Animation]()
		entityID := entry.EntityID

		if !anim.Playing {
			continue
		}

		// Look up the clip from the manager
		clip := as.manager.Get(anim.ClipPath)
		if clip == nil {
			// Skip if clip not found; log in production
			continue
		}

		// Advance frame time by delta * playback speed
		anim.FrameTime += delta * anim.PlaybackSpeed

		// Advance frames
		if anim.FrameTime >= clip.FrameSpeed {
			anim.FrameTime -= clip.FrameSpeed
			anim.FrameIndex++

			// Handle loop or stop
			if anim.FrameIndex >= len(clip.Frames) {
				if clip.Loop {
					anim.FrameIndex = 0
					bus.Emit(animation.AnimationEvent{
						EntityID: entityID,
						ClipID:   anim.ClipPath,
						Type:     animation.EventClipLooped,
					}, "AnimationSystem")
				} else {
					anim.FrameIndex = len(clip.Frames) - 1
					anim.Playing = false
					bus.Emit(animation.AnimationEvent{
						EntityID: entityID,
						ClipID:   anim.ClipPath,
						Type:     animation.EventClipFinished,
					}, "AnimationSystem")
				}
			}
		}

		_ = world.SetComponent(entityID, anim)
	}
	return nil
}

func (as *System) Shutdown(world *ecs.World) error {
	return nil
}
