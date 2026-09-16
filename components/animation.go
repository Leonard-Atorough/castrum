package components

import "fmt"

// Animation holds the per-entity playback state for a sprite animation.
//
// An Animation component references a clip by ID in the animation system's
// [ClipStore]. The clip definition (frames, FPS, loop mode) lives on the
// clip, not on this component — Animation only carries the runtime state
// that changes frame to frame.
//
// When an entity has both an Animation and a [Sprite], the animation system
// advances FrameIndex and FrameTime each tick and the renderer uses the
// current frame's atlas region instead of Sprite.RegionName.
//
// Field roles:
//   - ClipID and PlaybackSpeed are configuration set at creation time.
//   - FrameIndex, FrameTime, and Playing are runtime state mutated by the
//     animation system via the get-mutate-set pattern (GetComponent returns
//     a copy; SetComponent writes it back).
type Animation struct {
	ClipID        string  // ID to look up the AnimationClip in the ClipStore
	FrameIndex    int     // current frame index
	FrameTime     float64 // accumulated time for the current frame (seconds)
	Playing       bool    // whether the animation is currently running
	PlaybackSpeed float64 // playback multiplier (1.0 = normal speed)
}

// NewAnimation creates an Animation for the given clip ID. When autoplay is
// true, Playing is set to true so the animation system begins advancing
// immediately. PlaybackSpeed defaults to 1.0.
func NewAnimation(clipID string, autoplay bool) Animation {
	return Animation{
		ClipID:        clipID,
		PlaybackSpeed: 1.0,
		Playing:       autoplay,
	}
}

// Validate checks that the Animation has a non-empty ClipID, non-negative
// FrameIndex and FrameTime, and a positive PlaybackSpeed.
func (a Animation) Validate() error {
	if a.ClipID == "" {
		return fmt.Errorf("clipID must be specified")
	}
	if a.FrameIndex < 0 {
		return fmt.Errorf("frameIndex cannot be negative")
	}
	if a.FrameTime < 0 {
		return fmt.Errorf("frameTime cannot be negative")
	}
	if a.PlaybackSpeed <= 0 {
		return fmt.Errorf("playbackSpeed must be positive")
	}
	return nil
}

// Serialize converts the Animation to a map suitable for blueprint
// serialization or save-game storage. All fields are included.
func (a Animation) Serialize() (map[string]any, error) {
	return map[string]any{
		"clipID":        a.ClipID,
		"frameIndex":    a.FrameIndex,
		"frameTime":     a.FrameTime,
		"playing":       a.Playing,
		"playbackSpeed": a.PlaybackSpeed,
	}, nil
}

// Deserialize populates an Animation from a serialized map, returning the
// reconstructed component. The result is validated before returning.
func (a Animation) Deserialize(data map[string]any) (Animation, error) {
	if v, ok := data["clipID"].(string); ok {
		a.ClipID = v
	}
	if v, ok := data["frameIndex"].(float64); ok {
		a.FrameIndex = int(v)
	}
	if v, ok := data["frameTime"].(float64); ok {
		a.FrameTime = v
	}
	if v, ok := data["playing"].(bool); ok {
		a.Playing = v
	}
	if v, ok := data["playbackSpeed"].(float64); ok {
		a.PlaybackSpeed = v
	}
	return a, a.Validate()
}