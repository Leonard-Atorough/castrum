package animation

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/assets"
	"github.com/leonard-atorough/castrum/internal/core"
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
// and emits events when clips finish or loop.
type System struct {
	events      []AnimationEvent
	query       *core.Query
	assetLoader *assets.Assets
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
		components.Animation{},
		components.Renderable{},
	)
	as.events = make([]AnimationEvent, 0, 64)

	// For now, create a default assets loader. In production, inject this from outside.
	as.assetLoader = assets.NewAssets(nil)

	return nil
}

// Update processes all Animation components, advancing frame time and emitting events.
func (as *System) Update(world *core.World, delta float64) error {
	as.events = as.events[:0] // clear previous frame events

	for entry := range as.query.Execute() {
		anim, _ := entry.Get[components.Animation]()
		entityID := entry.EntityID

		if !anim.Playing {
			continue
		}

		// Load the animation clip from assets
		res, err := as.assetLoader.Load(anim.ClipPath)
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
					as.events = append(as.events, AnimationEvent{
						EntityID: entityID,
						ClipPath: anim.ClipPath,
						Type:     EventClipLooped,
					})
				} else {
					anim.FrameIndex = len(clip.Frames) - 1
					anim.Playing = false
					as.events = append(as.events, AnimationEvent{
						EntityID: entityID,
						ClipPath: anim.ClipPath,
						Type:     EventClipFinished,
					})
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
