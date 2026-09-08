package animation

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/assets"
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
	ClipPath string
	Type     AnimationEventType
}

// System is a lifecycle handler for Animation components.
// It processes all Animation components, advancing frame time against loaded AnimationClips,
// and emits events to the event bus when clips finish or loop.
type System struct {
	query       *core.Query
	assetLoader *assets.Assets
}

func (as *System) Init(world *core.World) error {
	as.query = world.NewQuery().WithRequiredComponents(
		components.Animation{},
		components.Renderable{},
	)

	// For now, create a default assets loader. In production, inject this from outside.
	as.assetLoader = assets.NewAssets(nil)

	return nil
}

// Update processes all Animation components, advancing frame time and emitting events to the bus.
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

		// Load the animation clip from assets
		res, err := as.assetLoader.LoadSync(anim.ClipPath)
		if err != nil || res == nil {
			// Skip if clip not found; log in production
			continue
		}

		clip := res.(*assets.AnimationClip)

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
						ClipPath: anim.ClipPath,
						Type:     EventClipLooped,
					}, "AnimationSystem")
				} else {
					anim.FrameIndex = len(clip.Frames) - 1
					anim.Playing = false
					bus.Emit(AnimationEvent{
						EntityID: entityID,
						ClipPath: anim.ClipPath,
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

