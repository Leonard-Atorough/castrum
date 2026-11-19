// Package animation provides reusable sprite clips and per-entity playback.
// Its fixed-tick system advances animated sprites and emits completion signals.
package animation

import (
	"fmt"
	"math"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/signal"
	"github.com/Leonard-Atorough/castrum/sprite"
)

// LoopMode controls what happens after an animation reaches its final frame.
type LoopMode int

const (
	// LoopNone plays the clip once, then pauses on its final frame. It is the
	// zero value, so animations play once by default.
	LoopNone LoopMode = iota
	// LoopAll wraps from the final frame back to the first.
	LoopAll
)

// Completed is emitted when an [AnimatedSprite] reaches the end of its clip.
// The system emits one signal per sprite per update, even if a looping clip
// wraps multiple times.
type Completed struct {
	// Entity is the entity whose clip completed.
	Entity core.Entity
	// Tick is the fixed tick of the completion.
	Tick uint64
	// Loops is the number of wraps in this update; 1 for a LoopNone completion.
	Loops uint64
}

// SpriteClip is a reusable sequence of named sheet regions and a playback rate.
// Its frames are copied on construction and can be shared by animated sprites.
// Its zero value is invalid. An in-memory sheet must remain read-only.
type SpriteClip struct {
	source asset.SheetRef
	frames []string
	fps    float64
}

// Validate reports an error if the clip's source is invalid, it has no frames,
// or its playback rate is not finite and positive.
func (c SpriteClip) Validate() error {
	if err := c.source.Validate(); err != nil {
		return fmt.Errorf("animation: clip source: %w", err)
	}
	if len(c.frames) == 0 {
		return fmt.Errorf("animation: clip must have at least one frame")
	}
	if !isFinite(c.fps) || c.fps <= 0 {
		return fmt.Errorf("animation: clip fps must be finite and positive, got %v", c.fps)
	}
	return nil
}

// NewSpriteClip returns a clip that plays frames in order at fps. It copies
// frames but does not load the sheet or check that the named regions exist.
// It returns an error if source is invalid, frames is empty or contains an
// empty name, or fps is not finite and positive.
func NewSpriteClip(source asset.SheetRef, frames []string, fps float64) (SpriteClip, error) {
	clip := SpriteClip{source: source, frames: frames, fps: fps}
	if err := clip.Validate(); err != nil {
		return SpriteClip{}, err
	}
	for i, frame := range frames {
		if frame == "" {
			return SpriteClip{}, fmt.Errorf("animation: clip frame %d must have a name", i)
		}
	}
	clip.frames = slices.Clone(frames)
	return clip, nil
}

// AnimatedSprite is a sprite with its own animation state.
// Its zero value has no clip and draws nothing.
//
// It requires a [core.Transform] to render.
type AnimatedSprite struct {
	// Loop selects what happens after the clip's final frame.
	Loop LoopMode
	// PlaybackMultiplier scales the clip's rate; zero means normal speed.
	// It must be finite and non-negative.
	PlaybackMultiplier float64
	// Paused prevents the current frame from advancing.
	Paused bool
	// Style holds the sprite's visual style.
	Style sprite.Style

	clip        SpriteClip
	hasClip     bool
	frame       int
	elapsed     float64
	completedOn uint64
	completed   bool
}

// NewAnimatedSprite returns a sprite with the clip's first frame selected.
// It returns an error if the clip is invalid.
func NewAnimatedSprite(clip SpriteClip) (AnimatedSprite, error) {
	if err := clip.Validate(); err != nil {
		return AnimatedSprite{}, fmt.Errorf("animation: animated sprite clip: %w", err)
	}
	return AnimatedSprite{clip: clip, hasClip: true}, nil
}

// Validate reports whether the sprite's style and playback state are valid.
func (a AnimatedSprite) Validate() error {
	if err := a.Style.Validate(); err != nil {
		return fmt.Errorf("animation: style: %w", err)
	}
	if a.Loop != LoopNone && a.Loop != LoopAll {
		return fmt.Errorf("animation: invalid loop mode %v", a.Loop)
	}
	if !isFinite(a.PlaybackMultiplier) || a.PlaybackMultiplier < 0 {
		return fmt.Errorf("animation: playback multiplier must be finite and non-negative")
	}
	if !isFinite(a.elapsed) || a.elapsed < 0 {
		return fmt.Errorf("animation: elapsed time must be finite and non-negative")
	}
	if !a.hasClip {
		if a.frame != 0 || a.elapsed != 0 || a.completedOn != 0 || a.completed {
			return fmt.Errorf("animation: clipless sprite has playback state")
		}
		return nil
	}
	if err := a.clip.Validate(); err != nil {
		return fmt.Errorf("animation: clip: %w", err)
	}
	if a.frame < 0 || a.frame >= len(a.clip.frames) {
		return fmt.Errorf("animation: frame %d is out of range", a.frame)
	}
	return nil
}

// CurrentFrame returns the sheet reference and region name of the frame
// to draw. It reports false for a clipless sprite and an error if the
// sprite is invalid.
func (a AnimatedSprite) CurrentFrame() (asset.SheetRef, string, bool, error) {
	if err := a.Validate(); err != nil {
		return asset.SheetRef{}, "", false, err
	}
	if !a.hasClip {
		return asset.SheetRef{}, "", false, nil
	}
	return a.clip.source, a.clip.frames[a.frame], true, nil
}

// JustCompleted reports whether the clip completed on tick. Pass
// [core.Context.Tick] to check whether it completed during the current update.
func (a AnimatedSprite) JustCompleted(tick uint64) bool {
	return a.hasClip && a.completedOn != 0 && a.completedOn == tick
}

// HasCompleted reports whether a non-looping clip has completed since its
// last [AnimatedSprite.Restart] or [AnimatedSprite.SetClip].
func (a AnimatedSprite) HasCompleted() bool {
	return a.hasClip && a.Loop == LoopNone && a.completed
}

// SetClip replaces the clip, selects its first frame, and resumes playback.
// It resets completion even if clip is unchanged. If clip is invalid, it
// returns an error without changing the sprite.
func (a *AnimatedSprite) SetClip(clip SpriteClip) error {
	if err := clip.Validate(); err != nil {
		return fmt.Errorf("animation: animated sprite clip: %w", err)
	}
	a.clip = clip
	a.hasClip, a.Paused = true, false
	a.frame, a.elapsed, a.completedOn = 0, 0, 0
	a.completed = false
	return nil
}

// Restart rewinds and resumes playback. It does nothing without a clip.
func (a *AnimatedSprite) Restart() {
	if !a.hasClip {
		return
	}
	a.frame, a.elapsed, a.completedOn = 0, 0, 0
	a.completed = false
	a.Paused = false
}

// SpriteAnimationSystem is the animation system's fixed-phase registration name.
const SpriteAnimationSystem = "engine.animation"

type animationSystem struct {
	animations *core.Query
	hub        *signal.Hub
}

// NewSpriteAnimationSystem returns a system that advances [AnimatedSprite]
// components and emits [Completed] signals.
//
// Games created by castrum.New register it in [core.PhaseFixed] automatically.
// When registering it manually, provide a [*signal.Hub] resource and run it
// in the fixed phase. Update returns an error if the hub is missing, a sprite
// is invalid, or a frame advance cannot be represented.
func NewSpriteAnimationSystem() core.System {
	return &animationSystem{}
}

func (s *animationSystem) Update(ctx *core.Context) error {
	if s.animations == nil {
		s.animations = core.NewQuery(ctx.World).With(AnimatedSprite{})
	}
	if s.hub == nil {
		resolved, err := ctx.World.Resource[*signal.Hub]()
		if err != nil {
			return fmt.Errorf("animated sprite system: %w; provide *signal.Hub to the world, typically signal.NewHub() through World.Provide, or construct the game with castrum.New", err)
		}
		s.hub = resolved
	}

	dt := ctx.DeltaTime.Seconds()
	for e := range s.animations.Execute() {
		anim, _ := e.Component[AnimatedSprite]()
		if err := anim.Validate(); err != nil {
			return fmt.Errorf("advancing entity %d: %w", e.ID(), err)
		}
		if anim.Paused || !anim.hasClip {
			continue
		}
		speed := anim.PlaybackMultiplier
		if speed == 0 {
			speed = 1
		}
		anim.elapsed += speed * dt
		if frameDuration := 1.0 / anim.clip.fps; anim.elapsed >= frameDuration {
			advance := anim.elapsed / frameDuration
			maxInt := int(^uint(0) >> 1)
			if !isFinite(advance) || advance >= float64(maxInt) {
				return fmt.Errorf("advancing entity %d: frame advance %v is not representable",
					e.ID(), advance)
			}
			crossed := int(advance)
			if crossed > maxInt-anim.frame {
				return fmt.Errorf("advancing entity %d: frame advance %v overflows the current frame",
					e.ID(), advance)
			}
			anim.elapsed -= float64(crossed) * frameDuration
			total := anim.frame + crossed
			frameCount := len(anim.clip.frames)
			switch {
			case total < frameCount:
				anim.frame = total
			case anim.Loop == LoopAll:
				anim.completedOn = ctx.Tick
				anim.frame = total % frameCount
				s.hub.Emit(Completed{Entity: e.Entity(), Tick: ctx.Tick, Loops: uint64(total / frameCount)})
			default:
				anim.completedOn = ctx.Tick
				anim.completed = true
				anim.Paused = true
				anim.frame = frameCount - 1
				s.hub.Emit(Completed{Entity: e.Entity(), Tick: ctx.Tick, Loops: 1})
			}
		}
		e.SetComponent(anim)
	}
	return nil
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
