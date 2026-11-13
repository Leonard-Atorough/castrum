package input

import (
	"fmt"
	"math"
	"reflect"
)

// DefaultDeadzone is the default gamepad axis deadzone used by [New].
// Axis values with magnitude below it are treated as zero.
const DefaultDeadzone = 0.15

// FrameSystemName is the name castrum.New registers the pre-frame input
// system under: the system that resolves one display frame of input into
// the frame view by calling [ActionMap.Update].
const FrameSystemName = "engine.input-update"

// TickSystemName is the name castrum.New registers the pre-fixed input
// system under: the system that delivers the press and release edges
// accumulated since the previous tick by calling [ActionMap.Tick].
const TickSystemName = "engine.input-tick"

// Action is a game-level input role, such as "jump" or "move_x".
type Action string

// Bindings maps each [Action] to the physical inputs that activate it.
// Multiple inputs can be bound to one action.
//
//	Bindings{
//	    "jump":   {KeyInput{Key: KeySpace}.Binding(), PadButtonInput{Button: PadSouth}.Binding()},
//	    "move_x": {KeyPairInput{Negative: KeyA, Positive: KeyD}.Binding(), PadAxisInput{Axis: PadLeftStickX}.Binding()},
//	    "p2_jump": {PadButtonInput{Pad: 2, Button: PadSouth}.Binding()},
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
// Each input must select exactly one binding type; chords require at least
// two members and cannot contain other chords.
func New(bindings Bindings, deadzone float64) (*ActionMap, error) {
	if err := validateBindings(bindings); err != nil {
		return nil, err
	}
	if err := validateDeadzone(deadzone); err != nil {
		return nil, err
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
// and may produce a new press. Removing a held action releases it on the next
// Update. Pending fixed-tick edges are preserved, including that release.
// Invalid bindings leave the map unchanged.
func (am *ActionMap) SetBindings(bindings Bindings) error {
	if err := validateBindings(bindings); err != nil {
		return fmt.Errorf("cannot update bindings: %w", err)
	}
	am.setBindings(bindings)
	return nil
}

func (am *ActionMap) setBindings(bindings Bindings) {
	am.bindings = cloneBindings(bindings)
	for _, st := range am.states {
		st.chordHeld = nil
	}
}

// Bindings returns a copy of the current bindings.
func (am *ActionMap) Bindings() Bindings {
	return cloneBindings(am.bindings)
}

// Preferences returns an owned snapshot with the effective map-wide deadzone.
func (am *ActionMap) Preferences() Preferences {
	return Preferences{Bindings: am.Bindings(), Deadzone: am.deadzone}
}

// SetPreferences validates and applies bindings and deadzone together for the
// next Update. Invalid values leave the map unchanged. Changed bindings restart
// chords as in [ActionMap.SetBindings]; unchanged bindings preserve chord state.
// Pending fixed-tick edges are preserved.
func (am *ActionMap) SetPreferences(p Preferences) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("cannot update preferences: %w", err)
	}
	if !reflect.DeepEqual(am.bindings, p.Bindings) {
		am.setBindings(p.Bindings)
	}
	am.deadzone = p.Deadzone
	if am.deadzone == 0 {
		am.deadzone = DefaultDeadzone
	}
	return nil
}

func validateDeadzone(deadzone float64) error {
	if math.IsNaN(deadzone) || deadzone < 0 || deadzone > 1 {
		return fmt.Errorf("deadzone must be finite and between 0 and 1")
	}
	return nil
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
			if bound.Chord != nil {
				if st.chordHeld == nil {
					st.chordHeld = make([]bool, len(inputs))
				}
				held := am.chordActive(s, *bound.Chord)
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
	for action, st := range am.states {
		if _, bound := am.bindings[action]; bound {
			continue
		}
		held := st.thisFrame.Held
		st.thisFrame = State{Released: held}
		st.axis = 0
		st.previousAxis = 0
		st.duration = 0
		st.chordHeld = nil
		if held {
			st.sinceLastTick.Released = true
		} else if !st.sinceLastTick.Pressed && !st.sinceLastTick.Released &&
			!st.thisTick.Pressed && !st.thisTick.Released {
			delete(am.states, action)
		}
	}
}

// Tick delivers the press and release events accumulated since the
// last tick. Call it once per fixed tick, before fixed systems read
// Pressed or Released.
func (am *ActionMap) Tick() {
	for action, st := range am.states {
		st.thisTick = st.sinceLastTick
		st.sinceLastTick = State{}
		if _, bound := am.bindings[action]; !bound &&
			!st.thisFrame.Held && !st.thisFrame.Pressed && !st.thisFrame.Released &&
			!st.thisTick.Pressed && !st.thisTick.Released {
			delete(am.states, action)
		}
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

// FlushEdges clears pending press and release edges without touching Held or
// durations. Called on scene transitions so a press that triggered a switch
// cannot fire in the next scene.
func (am *ActionMap) FlushEdges() {
	if am == nil {
		return
	}

	for _, st := range am.states {
		st.thisFrame.Pressed = false
		st.thisFrame.Released = false
		st.sinceLastTick = State{}
		st.thisTick.Pressed = false
		st.thisTick.Released = false
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
	count := 0
	for _, present := range [...]bool{
		bound.Key != nil, bound.MouseButton != nil, bound.PadButton != nil,
		bound.PadAxis != nil, bound.KeyPair != nil, bound.Chord != nil,
	} {
		if present {
			count++
		}
	}
	if count != 1 {
		return fmt.Errorf("input must select exactly one binding type")
	}
	if src := bound.PadAxis; src != nil {
		if err := validateDeadzone(src.Deadzone); err != nil {
			return err
		}
	}
	if src := bound.Chord; src != nil {
		if len(src.Inputs) < 2 {
			return fmt.Errorf("chord requires at least two inputs")
		}
		for i, member := range src.Inputs {
			if member.Chord != nil {
				return fmt.Errorf("chord member %d: nested chords are not supported", i)
			}
			if err := validateInput(member); err != nil {
				return fmt.Errorf("chord member %d: %w", i, err)
			}
		}
	}
	return nil
}

func cloneBindings(bindings Bindings) Bindings {
	owned := make(Bindings, len(bindings))
	for action, inputs := range bindings {
		owned[action] = append([]Input(nil), inputs...)
		for i, bound := range owned[action] {
			bound = cloneInputFields(bound)
			if chord := bound.Chord; chord != nil {
				chord.Inputs = append([]Input(nil), chord.Inputs...)
				// Nested chords are invalid, so only one member level needs copying.
				for j, member := range chord.Inputs {
					chord.Inputs[j] = cloneInputFields(member)
				}
			}
			owned[action][i] = bound
		}
	}
	return owned
}

func cloneInputFields(bound Input) Input {
	bound.Key = clonePointer(bound.Key)
	bound.MouseButton = clonePointer(bound.MouseButton)
	bound.PadButton = clonePointer(bound.PadButton)
	bound.PadAxis = clonePointer(bound.PadAxis)
	bound.KeyPair = clonePointer(bound.KeyPair)
	bound.Chord = clonePointer(bound.Chord)
	return bound
}

func clonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func (am *ActionMap) resolveInput(s *Snapshot, bound Input, anyPad int) (State, float64) {
	switch {
	case bound.Key != nil:
		src := bound.Key
		if !modifiersHeld(s, src.Modifiers) {
			return State{}, 0
		}
		return State{Pressed: s.KeyPressed(src.Key), Held: s.KeyHeld(src.Key), Released: s.KeyReleased(src.Key)}, 0
	case bound.MouseButton != nil:
		src := bound.MouseButton
		return State{Pressed: s.MousePressed(src.Button), Held: s.MouseHeld(src.Button), Released: s.MouseReleased(src.Button)}, 0
	case bound.PadButton != nil:
		src := *bound.PadButton
		if src.Pad == 0 {
			src.Pad = anyPad
		}
		return padState(s, src.Pad, src.Button), 0
	case bound.PadAxis != nil:
		src := *bound.PadAxis
		if src.Pad == 0 {
			src.Pad = anyPad
		}
		return State{}, am.padAxisValue(s, src)
	case bound.KeyPair != nil:
		src := bound.KeyPair
		var value float64
		if s.KeyHeld(src.Negative) {
			value--
		}
		if s.KeyHeld(src.Positive) {
			value++
		}
		return State{}, value
	default:
		panic(fmt.Sprintf("input: invalid binding %+v reached resolver", bound))
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
