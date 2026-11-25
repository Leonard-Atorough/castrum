package animation

import (
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/sprite"
)

// Animator is a per-entity animation component with serializable desired state.
type Animator struct {
	// Set is the key of the animation set.
	Set string `json:"set"`
	// State is the desired state name.
	State string `json:"state"`
	// Speed is the playback multiplier; zero means normal speed.
	Speed float64 `json:"speed,omitempty"`
	// Paused controls whether playback is paused.
	Paused bool `json:"paused,omitempty"`
	// Style controls the appearance of the animation.
	Style sprite.Style `json:"style,omitempty"`

	current  string       // state actually playing
	tracks   []trackState // runtime, one per track of the current entry
	blend    blendState   // active enter-transition, if any
	reversed bool
	finished bool
}

type trackState struct {
	frame   int
	elapsed float64
}

type blendState struct {
	from, to          string
	fromTracks        []trackState // frozen frames of the outgoing state
	elapsed, duration float64
}

// WeightedFrame is one drawable layer: a track's current region and its blend weight.
type WeightedFrame struct {
	// Track is the name of the track.
	Track string
	// Source is the sheet containing the region.
	Source asset.SheetRef
	// Region is the name of the sheet region.
	Region string
	// Weight is the layer's blend weight.
	Weight float64
}

// NewAnimator returns an [Animator] with the given set and desired state.
// It returns an error if either name is empty.
func NewAnimator(set, state string) (Animator, error) {
	a := Animator{Set: set, State: state}
	if err := a.Validate(); err != nil {
		return Animator{}, err
	}
	return a, nil
}

// Validate returns an error if the desired state or runtime fields are invalid.
// It does not check whether the set or state exists in [Sets].
func (a Animator) Validate() error {
	if a.Set == "" {
		return fmt.Errorf("animation: animator set cannot be empty")
	}
	if a.State == "" {
		return fmt.Errorf("animation: animator state cannot be empty")
	}
	if !isFinite(a.Speed) || a.Speed < 0 {
		return fmt.Errorf("animation: animator speed must be finite and non-negative, got %v", a.Speed)
	}
	if err := a.Style.Validate(); err != nil {
		return fmt.Errorf("animation: style: %w", err)
	}
	if a.current == "" {
		if a.tracks != nil || !a.blend.zero() || a.reversed || a.finished {
			return fmt.Errorf("animation: animator without a current state has runtime state")
		}
		return nil
	}
	if a.tracks == nil {
		return fmt.Errorf("animation: animator playing %q has no track state", a.current)
	}
	if a.blend.duration > 0 {
		if a.blend.from == "" || a.blend.to == "" {
			return fmt.Errorf("animation: blend needs both states")
		}
		if !isFinite(a.blend.elapsed) || a.blend.elapsed < 0 || a.blend.elapsed > a.blend.duration {
			return fmt.Errorf("animation: blend elapsed %v outside [0, %v]", a.blend.elapsed, a.blend.duration)
		}
	} else if !a.blend.zero() {
		return fmt.Errorf("animation: blend state without a duration")
	}
	return nil
}

// Current returns the state actually playing, which may differ from State.
// It returns an empty string before playback starts.
func (a Animator) Current() string {
	return a.current
}

// Finished reports whether a [LoopOnce] state has completed and is holding
// its final frame. It remains true until [Animate] starts another state.
func (a Animator) Finished() bool {
	return a.finished
}

// Frames returns the current state's weighted layers for drawing.
// It returns nil without error before playback starts.
//
// During a blend, outgoing layers precede incoming layers.
// It returns an error if the [Animator] is invalid or a required set or state
// is missing, or if the blend easing name is unknown.
func (a Animator) Frames(sets Sets) ([]WeightedFrame, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	if a.current == "" {
		return nil, nil
	}
	set, ok := sets.Get(a.Set)
	if !ok {
		return nil, fmt.Errorf("animation: unknown set %q", a.Set)
	}
	cur, ok := set.States[a.current]
	if !ok {
		return nil, fmt.Errorf("animation: set %q has no state %q", a.Set, a.current)
	}

	out := make([]WeightedFrame, 0, 2*len(cur.Tracks))
	weight := 1.0
	if a.blend.duration > 0 {
		t := a.blend.elapsed / a.blend.duration
		if t > 1 {
			t = 1
		}
		to, ok := set.States[a.blend.to]
		if !ok {
			return nil, fmt.Errorf("animation: set %q has no blend target state %q", a.Set, a.blend.to)
		}
		ease := geom.EaseLinear
		if to.BlendEase != "" {
			var known bool
			ease, known = blendEase(to.BlendEase)
			if !known {
				return nil, fmt.Errorf("animation: state %q has unknown blend ease %q", a.blend.to, to.BlendEase)
			}
		}
		progress := ease(t)
		from, ok := set.States[a.blend.from]
		if !ok {
			return nil, fmt.Errorf("animation: set %q has no blend source state %q", a.Set, a.blend.from)
		}
		out = appendWeightedFrames(out, from.Tracks, a.blend.fromTracks, 1-progress)
		weight = progress
	}
	return appendWeightedFrames(out, cur.Tracks, a.tracks, weight), nil
}

func (a *Animator) start(set Set, name string) error {
	state, ok := set.States[name]
	if !ok {
		return fmt.Errorf("animation: set %q has no state %q", set.Name, name)
	}
	prev := a.current
	var prevTracks []trackState
	if prev != "" {
		prevTracks = slices.Clone(a.tracks) // snapshot the outgoing frames for the blend
	}
	a.current = name
	a.State = name
	a.finished = false
	a.reversed = false
	a.tracks = make([]trackState, len(state.Tracks))
	a.blend = blendState{}
	if prev != "" && state.EnterBlend > 0 {
		a.blend = blendState{from: prev, to: name, fromTracks: prevTracks, duration: state.EnterBlend}
	}
	return nil
}

func (b blendState) zero() bool {
	return b.from == "" && b.to == "" && b.elapsed == 0 && b.duration == 0 && b.fromTracks == nil
}

func appendWeightedFrames(out []WeightedFrame, tracks []Track, states []trackState, progress float64) []WeightedFrame {
	for i, track := range tracks {
		if i >= len(states) {
			break
		}
		frame := states[i].frame
		if frame >= len(track.Frames) {
			frame = len(track.Frames) - 1
		}
		out = append(out, WeightedFrame{
			Track:  track.Name,
			Source: track.SheetRef(),
			Region: track.Frames[frame],
			Weight: progress,
		})
	}
	return out
}
