package animation

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/signal"
)

// primaryTrack is the index of the track that drives state-level events and
// transitions; the other tracks only resolve their own frame at a boundary.
const primaryTrack = 0

const SystemName = "animation"

// Animate is the system responsible for updating and applying animation
// states to entities. Games created by castrum.New register it in
// [core.PhaseFixed] automatically; when registering it manually, provide an
// [Sets] resource (World.SetResource) and a [*signal.Hub] resource.
//
// The resources are resolved only when an [Animator] exists, so worlds
// without animators never require them.
func Animate(ctx *core.Context) error {
	animators := core.NewQuery(ctx.World).With(Animator{})
	if _, ok := animators.First(); !ok {
		return nil
	}
	animations, err := ctx.World.Resource[Sets]()
	if err != nil {
		return fmt.Errorf("animation: %w; register animation sets with World.SetResource", err)
	}
	hub, err := ctx.World.Resource[*signal.Hub]()
	if err != nil {
		return fmt.Errorf("animation: %w; provide *signal.Hub to the world, typically signal.NewHub() through World.Provide, or construct the game with castrum.New", err)
	}

	dt := ctx.DeltaTime.Seconds()
	for e := range animators.Execute() {
		animator, _ := e.Component[Animator]()
		if err := animator.Validate(); err != nil {
			return fmt.Errorf("animation: entity %d: %w", e.ID(), err)
		}
		set, ok := animations.Get(animator.Set)
		if !ok {
			return fmt.Errorf("animation: entity %d: unknown set %q", e.ID(), animator.Set)
		}

		started := animator.current == "" || animator.State != animator.current
		if started {
			if err := animator.start(set, animator.State); err != nil {
				return fmt.Errorf("animation: entity %d: %w", e.ID(), err)
			}
			hub.Emit(Started{Entity: e.Entity(), Tick: ctx.Tick, State: animator.current})
		}
		if animator.Paused || animator.finished {
			if started {
				e.SetComponent(animator) // persist a same-tick start
			}
			continue
		}
		state, ok := set.States[animator.current]
		if !ok {
			return fmt.Errorf("animation: entity %d: set %q has no state %q", e.ID(), animator.Set, animator.current)
		}
		speed := animator.Speed
		if speed == 0 {
			speed = 1
		}

		if animator.blend.duration > 0 {
			animator.blend.elapsed += speed * dt
			if animator.blend.elapsed >= animator.blend.duration {
				animator.blend = blendState{}
			}
		}

		for i := range animator.tracks {
			track := &animator.tracks[i]
			def := state.Tracks[i]
			track.elapsed += speed * dt
			frameDuration := 1.0 / def.FPS
			if track.elapsed < frameDuration {
				continue // no frame boundary crossed this tick
			}
			advance := track.elapsed / frameDuration
			if !isFinite(advance) || advance >= float64(math.MaxInt) {
				return fmt.Errorf("animation: entity %d: frame advance %v is not representable", e.ID(), advance)
			}
			crossed := int(advance)
			if crossed > math.MaxInt-track.frame {
				return fmt.Errorf("animation: entity %d: frame advance %v overflows the current frame", e.ID(), advance)
			}
			track.elapsed -= float64(crossed) * frameDuration

			step := crossed
			if animator.reversed {
				step = -crossed
			}
			total := track.frame + step
			length := len(def.Frames)
			if total >= 0 && total < length {
				track.frame = total
				continue
			}

			// Boundary crossed: the primary track drives state-level events.
			passes := uint64(1)
			if state.Loop == Loop && total >= length {
				passes = uint64(total / length)
			}
			if i == primaryTrack {
				hub.Emit(Completed{Entity: e.Entity(), Tick: ctx.Tick, State: animator.current, Loops: passes})
				if state.Next != "" {
					if err := animator.start(set, state.Next); err != nil {
						return fmt.Errorf("animation: entity %d: %w", e.ID(), err)
					}
					hub.Emit(Started{Entity: e.Entity(), Tick: ctx.Tick, State: animator.current})
					break
				}
			}

			switch state.Loop {
			case Loop:
				track.frame = ((total % length) + length) % length
			case LoopPingPong:
				if i == primaryTrack {
					animator.reversed = !animator.reversed
				}
				if length == 1 {
					track.frame = 0
				} else {
					// Reflect the virtual position into [0, length-1].
					per := 2 * (length - 1)
					m := total % per
					if m < 0 {
						m += per
					}
					track.frame = m
					if m > length-1 {
						track.frame = per - m
					}
				}
			default: // LoopOnce holds the final frame.
				track.frame = length - 1
				if i == primaryTrack {
					animator.finished = true
				}
			}
		}
		e.SetComponent(animator)
	}
	return nil
}
