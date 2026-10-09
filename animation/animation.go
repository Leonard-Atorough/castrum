// Package animation provides animation components, clip storage, and a system
// that advances sprite playback on fixed ticks.
package animation

import (
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/render"
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
	// CompletedOn is the tick stamp when the animation last completed.
	CompletedOn uint64
	// Loop defines the looping behavior.
	Loop LoopMode
}

// Validate reports whether a has valid playback state: a non-empty clip ID,
// non-negative frame index and elapsed time, a non-negative playback
// multiplier, and a valid loop mode. A zero playback multiplier means normal
// speed.
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
	if a.Loop != LoopNone && a.Loop != LoopForever {
		return fmt.Errorf("animation: invalid loop mode %v", a.Loop)
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

// Restart rewinds to the first frame and resumes playback.
func (a *Animation) Restart() {
	a.Current, a.Elapsed = 0, 0
	a.CompletedOn = 0
	a.Paused = false
}

// JustCompleted reports whether the animation completed on the given
// fixed tick. Pass the ctx's current tick to check if the animation just
// completed.
func (a Animation) JustCompleted(tick uint64) bool {
	return a.CompletedOn == tick
}

// HasCompleted reports whether a non-looping animation has finished playing.
//
// For [LoopForever], it always returns false. Use [Animation.JustCompleted]
// to observe each loop completion.
func (a Animation) HasCompleted() bool {
	return a.Loop == LoopNone && a.CompletedOn != 0
}

// Clip is a reusable definition of an animation sequence. It specifies the
// source atlas, frame names in playback order, and playback rate.
type Clip struct {
	// Source is the atlas holding the frames.
	Source asset.AtlasID
	// Frames holds the frame region names, in playback order.
	Frames []string
	// FPS is the playback rate in frames per second.
	FPS float64
}

// Validate reports whether c has a source atlas, at least one frame, and a
// positive frame rate.
func (c Clip) Validate() error {
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
	clips map[ClipID]Clip
}

// NewClipStore returns an empty clip store.
func NewClipStore() *ClipStore {
	return &ClipStore{clips: make(map[ClipID]Clip)}
}

// Add registers clip under name. It copies clip.Frames so later changes to
// the caller's slice do not affect the stored clip. It returns an error if
// name is empty or already registered, or if clip fails validation.
func (s *ClipStore) Add(name ClipID, clip Clip) error {
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

// Get returns the clip registered under name. The returned Clip shares its
// Frames slice with the store; do not mutate that slice.
func (s *ClipStore) Get(name ClipID) (Clip, error) {
	clip, ok := s.clips[name]
	if !ok {
		return Clip{}, fmt.Errorf("animation: clip %q not found", name)
	}
	return clip, nil
}

// SystemName is the name castrum.New registers the animation system
// under in the fixed schedule.
const SystemName = "engine.animation"

// NewAnimationSystem returns a system that advances animations during fixed
// updates. castrum.New registers this system, under [SystemName], when the
// game is configured with castrum.WithAnimation; games that register it
// themselves do not need the option. An animation that references an
// unregistered clip causes the system to return an error.
func NewAnimationSystem(store *ClipStore) core.System {
	var animations *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if animations == nil {
			animations = core.NewQuery(ctx.World).
				With(Animation{}, render.Sprite{})
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
				if anim.Current > len(clip.Frames)-1 {
					anim.CompletedOn = ctx.Tick

					if anim.Loop == LoopForever {
						anim.Current %= len(clip.Frames)
					} else {
						anim.Current = len(clip.Frames) - 1
						anim.Paused = true
					}
				}
				e.SetComponent(anim)
			}

			sprite, _ := e.Component[render.Sprite]()
			target := render.AtlasSource{Atlas: clip.Source, Region: clip.Frames[anim.Current]}
			if sprite.Drawable != target {
				sprite.Drawable = target
				e.SetComponent(sprite)
			}
		}
		return nil
	})
}
