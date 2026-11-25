package animation

import (
	"encoding/json"
	"fmt"

	"github.com/Leonard-Atorough/castrum/geom"
)

// Sets is a named collection of animation sets, intended as a world resource.
type Sets struct {
	sets map[string]Set
}

// Validate returns an error if the collection is empty or contains an invalid
// set or a set whose name does not match its key.
func (ss Sets) Validate() error {
	if len(ss.sets) == 0 {
		return fmt.Errorf("animation: sets must have at least one set")
	}
	for name, set := range ss.sets {
		if set.Name != name {
			return fmt.Errorf("animation: set key %q has mismatched name %q", name, set.Name)
		}
		if err := set.Validate(); err != nil {
			return fmt.Errorf("animation: set %q: %w", name, err)
		}
	}
	return nil
}

// NewSets returns a validated [Sets] containing deep copies of the input sets.
// It returns an error if the collection is empty or any set or key is invalid.
func NewSets(sets map[string]Set) (Sets, error) {
	out := Sets{sets: make(map[string]Set, len(sets))}
	for name, set := range sets {
		if set.Name != name {
			return Sets{}, fmt.Errorf("animation: set key %q has mismatched name %q", name, set.Name)
		}
		if err := set.Validate(); err != nil {
			return Sets{}, fmt.Errorf("animation: set %q: %w", name, err)
		}
		out.sets[name] = set.clone()
	}
	if err := out.Validate(); err != nil {
		return Sets{}, err
	}
	return out, nil
}

// Get returns the named set and whether it exists.
// The returned set's maps and slices share storage with the collection.
func (ss Sets) Get(name string) (Set, bool) {
	set, ok := ss.sets[name]
	return set, ok
}

// Names returns the set names in unspecified order.
func (ss Sets) Names() []string {
	names := make([]string, 0, len(ss.sets))
	for name := range ss.sets {
		names = append(names, name)
	}
	return names
}

// MarshalJSON encodes the collection as a JSON object keyed by set name.
func (ss Sets) MarshalJSON() ([]byte, error) {
	return json.Marshal(ss.sets)
}

// UnmarshalJSON decodes a JSON object into the collection without validating it.
func (ss *Sets) UnmarshalJSON(data []byte) error {
	if ss.sets == nil {
		ss.sets = make(map[string]Set)
	}
	return json.Unmarshal(data, &ss.sets)
}

// Set groups named animation states.
type Set struct {
	// Name is the set's name.
	Name string `json:"name"`
	// States maps state names to states.
	States map[string]State `json:"states"`
}

// NewSet returns a set with an initialized, empty state map.
// The set is invalid until it has a name and at least one state.
func NewSet() Set {
	return Set{
		States: make(map[string]State),
	}
}

// Validate returns an error if the set, its states, or their transitions are invalid.
func (s Set) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("set name cannot be empty")
	}
	if len(s.States) == 0 {
		return fmt.Errorf("set %q must have at least one state", s.Name)
	}
	for name, entry := range s.States {
		if entry.Name != name {
			return fmt.Errorf("state key %q has mismatched name %q", name, entry.Name)
		}
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("state %q: %w", name, err)
		}
		if entry.Next != "" {
			if _, ok := s.States[entry.Next]; !ok {
				return fmt.Errorf("state %q transitions to unknown state %q", name, entry.Next)
			}
		}
	}
	return nil
}

func (s Set) clone() Set {
	out := Set{Name: s.Name, States: make(map[string]State, len(s.States))}
	for name, state := range s.States {
		out.States[name] = state.clone()
	}
	return out
}

// State defines an animation state and its tracks.
type State struct {
	// Name is the state's name.
	Name string `json:"name"`
	// Tracks are the layers played in the state.
	Tracks []Track `json:"tracks"`
	// Loop controls whether the state repeats.
	Loop LoopMode `json:"loop,omitempty"`
	// EnterBlend is the duration in seconds of the transition into the state.
	EnterBlend float64 `json:"enterBlend,omitempty"`
	// BlendEase names the easing function for the enter transition. Empty
	// defaults to linear; unknown names are rejected by [State.Validate].
	BlendEase string `json:"blendEase,omitempty"`
	// Next is the state to enter when the primary track reaches a boundary.
	// An empty name disables the automatic transition.
	Next string `json:"next,omitempty"`
}

// Validate returns an error if the state or its tracks are invalid.
func (e State) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("entry name cannot be empty")
	}
	if len(e.Tracks) == 0 {
		return fmt.Errorf("entry must have at least one track")
	}
	seen := make(map[string]bool, len(e.Tracks))
	for _, track := range e.Tracks {
		if seen[track.Name] {
			return fmt.Errorf("duplicate track name %q", track.Name)
		}
		seen[track.Name] = true
		if err := track.Validate(); err != nil {
			return fmt.Errorf("track %q: %w", track.Name, err)
		}
	}
	if e.Loop != LoopOnce && e.Loop != Loop && e.Loop != LoopPingPong {
		return fmt.Errorf("invalid loop mode")
	}
	if !isFinite(e.EnterBlend) || e.EnterBlend < 0 {
		return fmt.Errorf("invalid enter blend value")
	}
	if e.BlendEase != "" {
		if _, ok := blendEase(e.BlendEase); !ok {
			return fmt.Errorf("unknown blend ease %q", e.BlendEase)
		}
	}
	return nil
}

func (s State) clone() State {
	out := s
	out.Tracks = make([]Track, len(s.Tracks))
	for i, track := range s.Tracks {
		out.Tracks[i] = track.clone()
	}
	return out
}

// LoopMode controls what happens after an animation reaches its final frame.
type LoopMode int

const (
	// LoopOnce plays the state once and holds its final frame if there is no
	// automatic transition. It is the zero value.
	LoopOnce LoopMode = iota
	// Loop wraps from the final frame back to the first.
	Loop
	// LoopPingPong plays the state forward, then backward, repeatedly.
	LoopPingPong
)

func blendEase(ease string) (geom.Ease, bool) {
	switch ease {
	case "linear":
		return geom.EaseLinear, true
	case "easeInQuad":
		return geom.EaseInQuad, true
	case "easeOutQuad":
		return geom.EaseOutQuad, true
	case "easeInOutQuad":
		return geom.EaseInOutQuad, true
	case "easeInCubic":
		return geom.EaseInCubic, true
	case "easeOutCubic":
		return geom.EaseOutCubic, true
	case "easeInOutCubic":
		return geom.EaseInOutCubic, true
	default:
		return nil, false
	}
}
