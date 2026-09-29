package input

import (
	"fmt"
	"math"
)

// DefaultDeadzone is the default value of the Deadzone field in ActionMap.
// Below the value, input axes are considered inactive, i.e., treated as zero.
const DefaultDeadzone = 0.15

// Action is a human-readable alias for a game-level input role, such as "jump" or "move_x".
// It is used as the key in the Bindings map to associate actions with physical inputs.
type Action string

// Bindings maps [Action] to the physical inputs associated with that action.
// Each action can have multiple inputs bound to it, allowing flexible control schemes.
// Example:
//
//	Bindings{
//	    "jump":   []Input{KeyInput{KeySpace}, PadButtonInput{Button: PadSouth}},
//	    "move_x": []Input{KeyPairInput{Negative: KeyA, Positive: KeyD}, PadAxisInput{Axis: PadLeftStickX}},
//	    "p2_jump": []Input{PadButtonInput{Pad: 2, Button: PadSouth}},
//	}
type Bindings map[Action][]Input

// actionState is the runtime record of one action: its state for the
// current frame, the press and release events waiting for the next
// Tick, the events the current Tick delivered, the axis value, and
// the held duration.
type actionState struct {
	// thisFrame is the action's state for the current frame. Update
	// writes it every frame, and the frame queries (JustPressed,
	// JustReleased, Held) read it.
	thisFrame State
	// sinceLastTick records whether a press or a release happened at
	// least once since the last Tick. Duplicate presses count as
	// one, and a press and a release can both be recorded. Tick
	// clears it.
	sinceLastTick State
	// thisTick holds the events Tick delivered for the current tick.
	// It stays constant for the whole tick, so every system reading
	// it in that tick sees the same thing.
	thisTick State
	// axis is the action's current axis value: the strongest value
	// among its bound axes. Pad values below the deadzone read as
	// zero.
	axis float64
	// previousAxis is the action's axis value from the previous
	// frame. The finish phase of Update uses it to detect the axis
	// crossing the deadzone.
	previousAxis float64
	// duration is how long the action has been held without
	// interruption, in seconds. Update adds the frame's dt while the
	// action is held, and resets it to zero as soon as it is not.
	duration float64
}

// ActionMap resolves bound physical inputs into per-action
// state. Update resolves one frame, Tick delivers one tick, and the
// query methods report the results. Deadzone is the pad axis
// deadzone, defaulting to DefaultDeadzone.
//
// A nil ActionMap reads zero on every query, so the context can
// publish nil when no bindings were configured. The mutators
// (Update, Tick, SetBindings) require a real map.
type ActionMap struct {
	deadzone float64
	bindings Bindings
	states   map[Action]*actionState
}

// New returns an ActionManager over a copy of bindings. A deadzone
// of zero selects DefaultDeadzone. Nil bindings, or a deadzone
// outside [0, 1], are errors.
func New(bindings Bindings, deadzone float64) (*ActionMap, error) {
	if bindings == nil {
		return nil, fmt.Errorf("bindings cannot be nil")
	}
	if deadzone < 0 || deadzone > 1 {
		return nil, fmt.Errorf("deadzone must be between 0 and 1")
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
// The new bindings take effect at the next Update.
func (am *ActionMap) SetBindings(bindings Bindings) error {
	if bindings == nil {
		return fmt.Errorf("cannot update bindings: bindings cannot be nil")
	}
	am.bindings = cloneBindings(bindings)
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
		for _, bound := range inputs {
			switch src := bound.(type) {
			case KeyInput:
				if !modifiersHeld(s, src.Modifiers) {
					break
				}
				if s.KeyPressed(src.Key) {
					frame.Pressed = true
				}
				if s.KeyHeld(src.Key) {
					frame.Held = true
				}
				if s.KeyReleased(src.Key) {
					frame.Released = true
				}
			case MouseButtonInput:
				if s.MousePressed(src.Button) {
					frame.Pressed = true
				}
				if s.MouseHeld(src.Button) {
					frame.Held = true
				}
				if s.MouseReleased(src.Button) {
					frame.Released = true
				}
			case PadButtonInput:
				ps := padState(s, src.Pad, src.Button)
				frame.Pressed = frame.Pressed || ps.Pressed
				frame.Held = frame.Held || ps.Held
				frame.Released = frame.Released || ps.Released
			case PadAxisInput:
				val := am.padAxisValue(s, src.Pad, src.Axis, src.Direction)
				if math.Abs(val) > math.Abs(axis) {
					axis = val
				}
			case KeyPairInput:
				var pair float64
				if s.KeyHeld(src.Negative) {
					pair -= 1
				}
				if s.KeyHeld(src.Positive) {
					pair += 1
				}
				if math.Abs(pair) > math.Abs(axis) {
					axis = pair
				}
			}
		}

		active := math.Abs(axis) >= am.deadzone
		wasActive := math.Abs(st.previousAxis) >= am.deadzone
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

// Axis reports the current axis value for the action.
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

// modifiersHeld reports whether every non-empty modifier slot is held.
// KeyNone slots read as empty, so a zero-value array gates nothing.
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

// cloneBindings returns a copy the ActionMap owns: the map and every
// input slice are fresh, so the caller can mutate their Bindings
// after. Every Input member is a value type, so this is a complete
// deep copy.
func cloneBindings(bindings Bindings) Bindings {
	owned := make(Bindings, len(bindings))
	for action, inputs := range bindings {
		owned[action] = append([]Input(nil), inputs...)
	}
	return owned
}

// padState folds a pad button binding's tri-state across the pads its
// Pad address selects. Pad 0 - the zero value - is any connected pad;
// a player number addresses that one pad.
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

// padAxisValue returns the strongest axis value among the pads a
// binding's Pad address selects - Pad 0 is any connected pad, a player
// number addresses one pad. Values below the deadzone read as zero,
// and Direction gates the halves: a gated-away deflection reads as
// zero per pad, before the strongest wins, so a wrong-direction pad
// cannot beat a right-direction one.
func (am *ActionMap) padAxisValue(s *Snapshot, pad int, axis PadAxis, direction int) float64 {
	best := 0.0
	for padIndex := range MaxPads {
		if pad != 0 && padIndex != pad-1 {
			continue
		}
		if !s.PadConnected(padIndex) {
			continue
		}
		val := s.PadAxis(padIndex, axis)
		if math.Abs(val) < am.deadzone {
			val = 0
		}
		if (direction < 0 && val > 0) || (direction > 0 && val < 0) {
			val = 0
		}
		if math.Abs(val) > math.Abs(best) {
			best = val
		}
	}
	return best
}
