package animation

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/events"
)

type AnimationEventType int

const (
	EventClipFinished AnimationEventType = iota
	EventClipLooped
)

type AnimationEvent struct {
	EntityID core.EntityID
	ClipID   string
	Type     AnimationEventType
}

// System processes animation playback for entities with Animation components.
// It delegates to the Manager for clip resolution and orchestrates frame advancement
// and event emission. Does not handle clip creation or configuration.
type System struct {
	query   *core.Query
	manager *Manager
}

// NewSystem creates a new animation system with the given manager.
func NewSystem(manager *Manager) *System {
	return &System{
		manager: manager,
	}
}

func (as *System) Init(world *core.World) error {
	as.query = world.NewQuery().WithRequiredComponents(
		components.Animation{},
		components.Renderable{},
	)

	return nil
}

// Update processes all Animation components, advancing frame time and emitting events.
func (as *System) Update(world *core.World, delta float64) error {
	bus, ok := core.GetResource[*events.EventBus](world)
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
					bus.Emit(AnimationEvent{
						EntityID: entityID,
						ClipID:   anim.ClipPath,
						Type:     EventClipLooped,
					}, "AnimationSystem")
				} else {
					anim.FrameIndex = len(clip.Frames) - 1
					anim.Playing = false
					bus.Emit(AnimationEvent{
						EntityID: entityID,
						ClipID:   anim.ClipPath,
						Type:     EventClipFinished,
					}, "AnimationSystem")
				}
			}
		}

		_ = world.SetComponent(entityID, anim)
	}
	return nil
}

func (as *System) Shutdown(world *core.World) error {
	return nil
}
