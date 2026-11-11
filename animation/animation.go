// Package animation provides animation components, clip storage, and a system
// that advances sprite playback on fixed ticks and emits a [Completed] signal
// when an animation finishes its clip.
package animation

import (
	"fmt"
	"math"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/signal"
)

// ClipID identifies an animation clip in a [ClipStore].
type ClipID string

// Animation stores the playback state for a [Clip] displayed by an entity's
// [render.Sprite].
type Animation struct {
	// Clip identifies the animation definition to play.
	Clip ClipID
	// Current is the zero-based index of the frame to display.
	Current int
	// Elapsed is the time accumulated toward the next frame, in seconds.
	Elapsed float64
	// PlaybackMultiplier scales the clip's rate; zero means normal speed.
	PlaybackMultiplier float64
	// Paused prevents the current frame from advancing.
	Paused bool
	// CompletedOn is the fixed tick when the animation most recently completed.
	CompletedOn uint64
	// Loop selects what happens after the clip's final frame.
	Loop LoopMode
}

// Validate reports an error if Clip is empty, Current is negative, Elapsed
// or PlaybackMultiplier is non-finite or negative, or Loop is invalid. A zero
// PlaybackMultiplier means normal speed.
func (a Animation) Validate() error {
	if a.Clip == "" {
		return fmt.Errorf("animation: animation clip must not be empty")
	}
	if a.Current < 0 {
		return fmt.Errorf("animation: current must not be negative, got %d", a.Current)
	}
	if !isFinite(a.Elapsed) || a.Elapsed < 0 {
		return fmt.Errorf("animation: elapsed time must be finite and not negative, got %v", a.Elapsed)
	}
	if !isFinite(a.PlaybackMultiplier) || a.PlaybackMultiplier < 0 {
		return fmt.Errorf("animation: playback multiplier must be finite and not negative (0 is the normal rate), got %v", a.PlaybackMultiplier)
	}
	if a.Loop != LoopNone && a.Loop != LoopForever {
		return fmt.Errorf("animation: invalid loop mode %v", a.Loop)
	}
	return nil
}

// LoopMode controls what happens after an animation reaches its final frame.
type LoopMode int

const (
	// LoopNone plays the clip once, then pauses on its final frame. It is the
	// zero value, so animations play once by default.
	LoopNone LoopMode = iota
	// LoopForever wraps from the final frame back to the first.
	LoopForever
)

// Restart rewinds to the first frame and resumes playback.
func (a *Animation) Restart() {
	a.Current, a.Elapsed = 0, 0
	a.CompletedOn = 0
	a.Paused = false
}

// JustCompleted reports whether the animation completed on tick. Pass
// [core.Context.Tick] to check whether it completed during the current update.
func (a Animation) JustCompleted(tick uint64) bool {
	return a.CompletedOn == tick
}

// HasCompleted reports whether a non-looping animation has completed since
// its last [Animation.Restart].
//
// For [LoopForever], it always returns false. Use [Animation.JustCompleted]
// to observe each loop completion.
func (a Animation) HasCompleted() bool {
	return a.Loop == LoopNone && a.CompletedOn != 0
}

// Completed is the signal the animation system emits when an [Animation]
// reaches the end of its clip. For [LoopForever] animations, it is emitted
// each time the clip wraps to its first frame. Tick equals
// [Animation.CompletedOn].
type Completed struct {
	// Entity is the entity whose Animation finished the clip.
	Entity core.EntityID
	// Tick is the fixed tick of the completion.
	Tick uint64
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

// Validate reports an error if Source or Frames is empty, or FPS is not
// finite and positive.
func (c Clip) Validate() error {
	if c.Source == "" {
		return fmt.Errorf("animation: clip source must not be empty")
	}
	if len(c.Frames) == 0 {
		return fmt.Errorf("animation: clip must have at least one frame")
	}
	if !isFinite(c.FPS) || c.FPS <= 0 {
		return fmt.Errorf("animation: clip fps must be finite and positive, got %v", c.FPS)
	}
	return nil
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// ClipStore holds named [Clip] definitions for [NewAnimationSystem].
type ClipStore struct {
	clips map[ClipID]Clip
}

// NewClipStore returns an empty clip store.
func NewClipStore() *ClipStore {
	return &ClipStore{clips: make(map[ClipID]Clip)}
}

// Add registers clip under name. It copies clip.Frames so later changes to
// the input slice do not affect the registered clip. It returns an error if
// name is empty, already registered, or clip fails [Clip.Validate].
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

// Get returns the clip registered under name. Its [Clip.Frames] slice is
// shared with the store and must not be modified.
func (s *ClipStore) Get(name ClipID) (Clip, error) {
	clip, ok := s.clips[name]
	if !ok {
		return Clip{}, fmt.Errorf("animation: clip %q not found", name)
	}
	return clip, nil
}

// SystemName is the name used to register the animation system in the fixed
// schedule.
const SystemName = "engine.animation"

// NewAnimationSystem returns a [core.System] that advances [Animation]
// components on fixed updates using clips from store. Each animated entity
// must also have a [render.Sprite]. The system returns an error if an
// animation references a clip that is not registered, its current frame is
// outside the clip, or its frame advance cannot be represented as an int.
//
// The system resolves the world's [*signal.Hub] and emits a [Completed]
// signal when an animation finishes its clip.
//
// Enable the animation option when creating a game to register this system
// automatically. Otherwise, register it in the fixed schedule under
// [SystemName].
func NewAnimationSystem(store *ClipStore) core.System {
	var animations *core.Query
	var hub *signal.Hub
	return core.SystemFunc(func(ctx *core.Context) error {
		if animations == nil {
			animations = core.NewQuery(ctx.World).
				With(Animation{}, render.Sprite{})
		}
		if hub == nil {
			resolved, err := ctx.World.Resource[*signal.Hub]()
			if err != nil {
				return fmt.Errorf("animation system: resolve signal hub: %w; provide *signal.Hub to the world, typically signal.NewHub() through World.Provide, or construct the game with castrum.New", err)
			}
			hub = resolved
		}
		dt := ctx.DeltaTime.Seconds()
		for e := range animations.Execute() {
			anim, _ := e.Component[Animation]()
			clip, err := store.Get(anim.Clip)
			if err != nil {
				return fmt.Errorf("advancing entity %d: %w", e.ID(), err)
			}
			if anim.Current < 0 || anim.Current >= len(clip.Frames) {
				return fmt.Errorf("advancing entity %d: clip %q frame %d is outside [0, %d)",
					e.ID(), anim.Clip, anim.Current, len(clip.Frames))
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
				advance := anim.Elapsed / frameDuration
				maxInt := int(^uint(0) >> 1)
				if !isFinite(advance) || advance >= float64(maxInt) {
					return fmt.Errorf("advancing entity %d: clip %q frame advance %v is not representable",
						e.ID(), anim.Clip, advance)
				}
				crossed := int(advance)
				if crossed > maxInt-anim.Current {
					return fmt.Errorf("advancing entity %d: clip %q frame advance %v overflows the current frame",
						e.ID(), anim.Clip, advance)
				}
				anim.Elapsed -= float64(crossed) * frameDuration
				anim.Current += crossed
				if anim.Current > len(clip.Frames)-1 {
					anim.CompletedOn = ctx.Tick
					hub.Emit(Completed{Entity: e.ID(), Tick: ctx.Tick})

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
