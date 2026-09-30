package animation

import (
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
)

// ClipID identifies an animation clip in a ClipStore.
type ClipID string

// Animation is a component that plays an AnimationClip on the
// entity's Sprite.
type Animation struct {
	// Clip identifies the clip being played.
	Clip ClipID
	// Current is the index of the frame being displayed.
	Current int
	// Elapsed is the time accumulated within the current frame, in
	// seconds.
	Elapsed float64
	// PlaybackMultiplier scales the clip's rate. 0 - the zero
	// value - is the normal rate; 1 is identity; 2 is double speed.
	PlaybackMultiplier float64
	// Paused stops playback.
	Paused bool
}

// Validate implements the Validatable contract.
func (a Animation) Validate() error {
	if a.Clip == "" {
		return fmt.Errorf("animation: animation clip must not be empty")
	}
	if a.Current < 0 {
		return fmt.Errorf("animation: current must not be negative, got %d", a.Current)
	}
	if a.Elapsed < 0 {
		return fmt.Errorf("animation: elapsed must not be negative, got %v", a.Elapsed)
	}
	if a.PlaybackMultiplier < 0 {
		return fmt.Errorf("animation: playback multiplier must not be negative (0 is the normal rate), got %v", a.PlaybackMultiplier)
	}
	return nil
}

// LoopMode controls how an animation repeats after its last frame.
type LoopMode int

const (
	// LoopNone plays the clip once, then holds the last frame paused.
	// It is the zero value: the field named Loop reads truthfully.
	LoopNone LoopMode = iota
	// LoopForever wraps from the last frame back to the first.
	LoopForever
)

// Pause stops the animation from advancing.
func (a *Animation) Pause() {
	a.Paused = true
}

// Resume unpauses the animation.
func (a *Animation) Resume() {
	a.Paused = false
}

// Restart rewinds to the first frame and resumes playback.
func (a *Animation) Restart() {
	a.Current, a.Elapsed = 0, 0
	a.Resume()
}

// Stop rewinds to the first frame and pauses: the animation resumes
// from the beginning, not where it stopped.
func (a *Animation) Stop() {
	a.Current, a.Elapsed = 0, 0
	a.Paused = true
}

// AnimationClip is an animation definition: named atlas regions
// played in order at a fixed rate, with a looping behavior.
type AnimationClip struct {
	// Source is the atlas holding the frames.
	Source asset.AtlasID
	// Frames holds the frame region names, in playback order.
	Frames []string
	// FPS is the playback rate in frames per second.
	FPS float64
	// Loop defines the looping behavior.
	Loop LoopMode
}

// Validate implements the Validatable contract.
func (c AnimationClip) Validate() error {
	if c.Source == "" {
		return fmt.Errorf("animation: clip source must not be empty")
	}
	if len(c.Frames) == 0 {
		return fmt.Errorf("animation: clip must have at least one frame")
	}
	if c.FPS <= 0 {
		return fmt.Errorf("animation: clip fps must be positive, got %v", c.FPS)
	}
	return nil
}

// ClipStore is the game's library of named animation clips, provided
// as a world resource by castrum.New.
type ClipStore struct {
	clips map[ClipID]AnimationClip
}

// NewClipStore returns an empty clip store.
func NewClipStore() *ClipStore {
	return &ClipStore{clips: make(map[ClipID]AnimationClip)}
}

// Add registers clip under name. Frames is copied, so the store owns
// its slice. Errors on an empty or duplicate name, or on a clip
// failing Validate.
func (s *ClipStore) Add(name ClipID, clip AnimationClip) error {
	if name == "" {
		return fmt.Errorf("animation: clip name must not be empty")
	}
	if _, ok := s.clips[name]; ok {
		return fmt.Errorf("animation: clip %q already registered", name)
	}
	if err := clip.Validate(); err != nil {
		return fmt.Errorf("animation: clip %q: %w", name, err)
	}
	clip.Frames = slices.Clone(clip.Frames)
	s.clips[name] = clip
	return nil
}

// Get returns the clip registered under name. The returned value
// shares the store's Frames slice: do not mutate it in place.
func (s *ClipStore) Get(name ClipID) (AnimationClip, error) {
	clip, ok := s.clips[name]
	if !ok {
		return AnimationClip{}, fmt.Errorf("animation: clip %q not found", name)
	}
	return clip, nil
}

// NewAnimationSystem returns the engine's animation advancer,
// registered in the fixed phase by castrum.New.
func NewAnimationSystem(store *ClipStore) core.System {
	var animations *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if animations == nil {
			animations = core.NewQuery(ctx.World).
				With(Animation{}, core.Sprite{})
		}
		dt := ctx.DeltaTime.Seconds()
		for e := range animations.Execute() {
			anim, _ := e.Component[Animation]()
			clip, err := store.Get(anim.Clip)
			if err != nil {
				return fmt.Errorf("advancing entity %d: %w", e.ID(), err)
			}
			if anim.Paused {
				continue
			}
			speed := anim.PlaybackMultiplier
			if speed == 0 {
				speed = 1
			}
			anim.Elapsed += dt * speed
			frameDuration := 1.0 / clip.FPS
			if anim.Elapsed < frameDuration {
				e.SetComponent(anim)
			} else {
				crossed := int(anim.Elapsed / frameDuration)
				anim.Elapsed -= float64(crossed) * frameDuration
				anim.Current += crossed
				switch {
				case clip.Loop == LoopForever:
					anim.Current %= len(clip.Frames)
				case anim.Current >= len(clip.Frames):
					anim.Current = len(clip.Frames) - 1
					anim.Paused = true
				}
				e.SetComponent(anim)
			}

			sprite, _ := e.Component[core.Sprite]()
			target := core.AtlasSource{Atlas: clip.Source, Region: clip.Frames[anim.Current]}
			if sprite.Drawable != target {
				sprite.Drawable = target
				e.SetComponent(sprite)
			}
		}
		return nil
	})
}
