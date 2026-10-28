package input

import (
	"fmt"
	"math"
)

// DefaultDeadzone is the default gamepad axis deadzone used by [New].
// Axis values with magnitude below it are treated as zero.
const DefaultDeadzone = 0.15

// FrameSystemName is the name castrum.New registers the frame input
// system under: the system that resolves one display frame of input
// into the frame view by calling [ActionMap.Update].
const FrameSystemName = "engine.input-update"

// TickSystemName is the name castrum.New registers the fixed input
// system under: the system that delivers the press and release edges
// accumulated since the previous tick by calling [ActionMap.Tick].
const TickSystemName = "engine.input-tick"

// Action is a game-level input role, such as "jump" or "move_x".
type Action string

// Bindings maps each [Action] to the physical inputs that activate it.
// Multiple inputs can be bound to one action.
//
//	Bindings{
//	    "jump":   []Input{KeyInput{KeySpace}, PadButtonInput{Button: PadSouth}},
//	    "move_x": []Input{KeyPairInput{Negative: KeyA, Positive: KeyD}, PadAxisInput{Axis: PadLeftStickX}},
//	    "p2_jump": []Input{PadButtonInput{Pad: 2, Button: PadSouth}},
//	}
type Bindings map[Action][]Input

type actionState struct {
	thisFrame     State
	sinceLastTick State
	thisTick      State
	axis          float64
	previousAxis  float64
	// Chord transitions are tracked per binding because action state is OR-folded.
	chordHeld []bool
	duration  float64
}

// ActionMap resolves [Bindings] against physical input snapshots and exposes
// per-action state.
//
// A nil ActionMap returns zero from queries; mutating methods require a
// non-nil map.
type ActionMap struct {
	deadzone float64
	bindings Bindings
	states   map[Action]*actionState
}

// New creates an [ActionMap] with a copy of bindings. A deadzone of zero
// selects [DefaultDeadzone]. It returns an error if bindings is nil or
// deadzone is not finite or outside [0, 1], or a binding is invalid.
// Bindings must use value-form input types; chords require at least two
// members and cannot contain other chords.
func New(bindings Bindings, deadzone float64) (*ActionMap, error) {
	if err := validateBindings(bindings); err != nil {
		return nil, err
	}
	if math.IsNaN(deadzone) || deadzone < 0 || deadzone > 1 {
		return nil, fmt.Errorf("deadzone must be finite and between 0 and 1")
	}

	if deadzone == 0 {
		deadzone = DefaultDeadzone
	}
	am := &ActionMap{
		deadzone: deadzone,
		bindings: cloneBindings(bindings),
		states:   make(map[Action]*actionState),
	}
	return am, nil
}

// SetBindings replaces the bindings with a copy of the argument.
// The new bindings take effect at the next Update; chords start inactive
// and may produce a new press. Invalid bindings leave the map unchanged.
func (am *ActionMap) SetBindings(bindings Bindings) error {
	if err := validateBindings(bindings); err != nil {
		return fmt.Errorf("cannot update bindings: %w", err)
	}
	am.bindings = cloneBindings(bindings)
	for _, st := range am.states {
		st.chordHeld = nil
	}
	return nil
}

// Bindings returns a copy of the current bindings.
func (am *ActionMap) Bindings() Bindings {
	return cloneBindings(am.bindings)
}

// Update resolves one frame of input into per-action state. Press
// and release events accumulate in sinceLastTick until the next
// Tick delivers them. Call it once per frame, before any Tick. A
// nil snapshot reads as every input off.
func (am *ActionMap) Update(s *Snapshot, dt float64) {
	for action, inputs := range am.bindings {
		st := am.states[action]
		if st == nil {
			st = &actionState{}
			am.states[action] = st
		}

		var frame State
		var axis float64
		for i, bound := range inputs {
			var state State
			var value float64
			if chord, ok := bound.(ChordInput); ok {
				if st.chordHeld == nil {
					st.chordHeld = make([]bool, len(inputs))
				}
				held := am.chordActive(s, chord)
				state = State{
					Held:     held,
					Pressed:  held && !st.chordHeld[i],
					Released: !held && st.chordHeld[i],
				}
				st.chordHeld[i] = held
			} else {
				state, value = am.resolveInput(s, bound, 0)
			}
			frame.Pressed = frame.Pressed || state.Pressed
			frame.Held = frame.Held || state.Held
			frame.Released = frame.Released || state.Released
			if math.Abs(value) > math.Abs(axis) {
				axis = value
			}
		}

		active := axis != 0
		wasActive := st.previousAxis != 0
		frame.Held = frame.Held || active
		frame.Pressed = frame.Pressed || active && !wasActive
		frame.Released = frame.Released || !active && wasActive

		st.thisFrame = frame
		st.axis = axis
		if frame.Pressed {
			st.sinceLastTick.Pressed = true
		}
		if frame.Released {
			st.sinceLastTick.Released = true
		}
		if frame.Held {
			st.duration += dt
		} else {
			st.duration = 0
		}
		st.previousAxis = axis

	}
}

// Tick delivers the press and release events accumulated since the
// last tick. Call it once per fixed tick, before fixed systems read
// Pressed or Released.
func (am *ActionMap) Tick() {
	for action := range am.bindings {
		st := am.states[action]
		if st == nil {
			continue
		}

		st.thisTick = st.sinceLastTick
		st.sinceLastTick = State{}
	}
}

// JustPressed reports whether the action was pressed in the current
// frame.
func (am *ActionMap) JustPressed(action Action) bool {
	if am == nil {
		return false
	}
	st := am.states[action]
	if st == nil {
		return false
	}
	return st.thisFrame.Pressed
}

// JustReleased reports whether the action was released in the current frame.
func (am *ActionMap) JustReleased(action Action) bool {
	if am == nil {
		return false
	}
	st := am.states[action]
	if st == nil {
		return false
	}
	return st.thisFrame.Released
}

// Held reports whether the action is held in the current frame.
func (am *ActionMap) Held(action Action) bool {
	if am == nil {
		return false
	}
	st := am.states[action]
	if st == nil {
		return false
	}
	return st.thisFrame.Held
}

// Pressed reports whether a press was latched for the current tick.
// It reports true for every read in the tick that consumed the
// press. Frame-rate code wants JustPressed instead.
func (am *ActionMap) Pressed(action Action) bool {
	if am == nil {
		return false
	}
	st := am.states[action]
	if st == nil {
		return false
	}
	return st.thisTick.Pressed
}

// Released reports whether a release was latched for the current
// tick. It reports true for every read in the tick that consumed
// the release. Frame-rate code wants JustReleased instead.
func (am *ActionMap) Released(action Action) bool {
	if am == nil {
		return false
	}
	st := am.states[action]
	if st == nil {
		return false
	}
	return st.thisTick.Released
}

// Duration reports how long the action has been held, in seconds.
func (am *ActionMap) Duration(action Action) float64 {
	if am == nil {
		return 0
	}
	st := am.states[action]
	if st == nil {
		return 0
	}
	return st.duration
}

// Axis reports the action's latest frame-sampled axis value. Fixed catch-up
// ticks share that value; Tick does not accumulate, consume, or clear it.
func (am *ActionMap) Axis(action Action) float64 {
	if am == nil {
		return 0
	}
	st := am.states[action]
	if st == nil {
		return 0
	}
	return st.axis
}

// FlushEdges clears pending JustPressed/JustReleased edges (frame and tick
// latches) without touching Held or durations. Called on scene transitions
// so a press that triggered a switch cannot fire in the next scene.
func (am *ActionMap) FlushEdges() {
	if am == nil {
		return
	}
	for _, st := range am.states {
		st.thisFrame = State{}
		st.thisTick = State{}
	}
}

func modifiersHeld(s *Snapshot, modifiers [3]Key) bool {
	for _, mod := range modifiers {
		if mod == KeyNone {
			continue
		}
		if !s.KeyHeld(mod) {
			return false
		}
	}
	return true
}

func validateBindings(bindings Bindings) error {
	if bindings == nil {
		return fmt.Errorf("bindings cannot be nil")
	}
	for action, inputs := range bindings {
		for i, bound := range inputs {
			if err := validateInput(bound); err != nil {
				return fmt.Errorf("action %q binding %d: %w", action, i, err)
			}
		}
	}
	return nil
}

func validateInput(bound Input) error {
	switch src := bound.(type) {
	case KeyInput, MouseButtonInput, PadButtonInput, KeyPairInput:
		return nil
	case PadAxisInput:
		if math.IsNaN(src.Deadzone) || src.Deadzone < 0 || src.Deadzone > 1 {
			return fmt.Errorf("deadzone must be finite and between 0 and 1")
		}
	case ChordInput:
		if len(src.Inputs) < 2 {
			return fmt.Errorf("chord requires at least two inputs")
		}
		for i, member := range src.Inputs {
			if _, nested := member.(ChordInput); nested {
				return fmt.Errorf("chord member %d: nested chords are not supported", i)
			}
			if err := validateInput(member); err != nil {
				return fmt.Errorf("chord member %d: %w", i, err)
			}
		}
	default:
		return fmt.Errorf("unsupported input type %T; use value-form bindings", bound)
	}
	return nil
}

// Copy chord members too, so callers cannot mutate stored bindings through a
// chord.
func cloneBindings(bindings Bindings) Bindings {
	owned := make(Bindings, len(bindings))
	for action, inputs := range bindings {
		owned[action] = append([]Input(nil), inputs...)
		for i, bound := range owned[action] {
			if chord, ok := bound.(ChordInput); ok {
				chord.Inputs = append([]Input(nil), chord.Inputs...)
				owned[action][i] = chord
			}
		}
	}
	return owned
}

func (am *ActionMap) resolveInput(s *Snapshot, bound Input, anyPad int) (State, float64) {
	switch src := bound.(type) {
	case KeyInput:
		if !modifiersHeld(s, src.Modifiers) {
			return State{}, 0
		}
		return State{Pressed: s.KeyPressed(src.Key), Held: s.KeyHeld(src.Key), Released: s.KeyReleased(src.Key)}, 0
	case MouseButtonInput:
		return State{Pressed: s.MousePressed(src.Button), Held: s.MouseHeld(src.Button), Released: s.MouseReleased(src.Button)}, 0
	case PadButtonInput:
		if src.Pad == 0 {
			src.Pad = anyPad
		}
		return padState(s, src.Pad, src.Button), 0
	case PadAxisInput:
		if src.Pad == 0 {
			src.Pad = anyPad
		}
		return State{}, am.padAxisValue(s, src)
	case KeyPairInput:
		var value float64
		if s.KeyHeld(src.Negative) {
			value--
		}
		if s.KeyHeld(src.Positive) {
			value++
		}
		return State{}, value
	default:
		panic(fmt.Sprintf("input: unsupported binding %T reached resolver", bound))
	}
}

func (am *ActionMap) chordActive(s *Snapshot, chord ChordInput) bool {
	for pad := 1; pad <= MaxPads; pad++ {
		active := true
		for _, member := range chord.Inputs {
			state, value := am.resolveInput(s, member, pad)
			if !state.Held && value == 0 {
				active = false
				break
			}
		}
		if active {
			return true
		}
	}
	return false
}

func padState(s *Snapshot, pad int, button PadButton) State {
	var state State
	for padIndex := range MaxPads {
		if pad != 0 && padIndex != pad-1 {
			continue
		}
		if !s.PadConnected(padIndex) {
			continue
		}
		if s.PadPressed(padIndex, button) {
			state.Pressed = true
		}
		if s.PadHeld(padIndex, button) {
			state.Held = true
		}
		if s.PadReleased(padIndex, button) {
			state.Released = true
		}
	}
	return state
}

func (am *ActionMap) padAxisValue(s *Snapshot, src PadAxisInput) float64 {
	deadzone := src.Deadzone
	if deadzone == 0 {
		deadzone = am.deadzone
	}
	best := 0.0
	for padIndex := range MaxPads {
		if src.Pad != 0 && padIndex != src.Pad-1 {
			continue
		}
		if !s.PadConnected(padIndex) {
			continue
		}
		val := s.PadAxis(padIndex, src.Axis)
		if math.Abs(val) < deadzone {
			val = 0
		}
		if (src.Direction < 0 && val > 0) || (src.Direction > 0 && val < 0) {
			val = 0
		}
		if math.Abs(val) > math.Abs(best) {
			best = val
		}
	}
	return best
}
