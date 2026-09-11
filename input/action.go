package input

import "strings"

// Action identifies a game-level input action such as "move_up" or "jump".
// Actions are intentionally independent of a physical keyboard or controller.
type Action string

// Binding connects an action to a physical key name and optional modifiers.
// Key names are backend-independent names such as "w", "arrow_up", or
// "space".
type Binding struct {
	Key       string    `yaml:"key"`
	Modifiers Modifiers `yaml:"modifiers"`
}

// Bindings maps action names to one or more physical bindings.
type Bindings map[Action][]Binding

// Reader exposes resolved game actions to gameplay systems.
type Reader interface {
	ActionPressed(action Action) bool
	ActionHeld(action Action) bool
	ActionReleased(action Action) bool
}

// ActionMap resolves physical input states into action states.
type ActionMap struct {
	bindings Bindings
	states   map[Action]KeyState
}

// NewActionMap creates an action map from configured bindings.
func NewActionMap(bindings Bindings) *ActionMap {
	owned := make(Bindings, len(bindings))
	for action, actionBindings := range bindings {
		ownedBindings := append([]Binding(nil), actionBindings...)
		for i := range ownedBindings {
			ownedBindings[i].Key = normalizeKeyName(ownedBindings[i].Key)
		}
		owned[action] = ownedBindings
	}
	return &ActionMap{
		bindings: owned,
		states:   make(map[Action]KeyState, len(owned)),
	}
}

// Update resolves the current physical key states into action states.
func (m *ActionMap) Update(keys map[string]KeyState, modifiers Modifiers) {
	for action, actionBindings := range m.bindings {
		state := KeyState{}
		for _, binding := range actionBindings {
			if !matchesModifiers(modifiers, binding.Modifiers) {
				continue
			}
			candidate := keys[normalizeKeyName(binding.Key)]
			state.Pressed = state.Pressed || candidate.Pressed
			state.Held = state.Held || candidate.Held
			state.Released = state.Released || candidate.Released
			if candidate.Duration > state.Duration {
				state.Duration = candidate.Duration
			}
		}
		m.states[action] = state
	}
}

// ActionPressed reports whether an action was pressed this frame.
func (m *ActionMap) ActionPressed(action Action) bool {
	return m.states[action].Pressed
}

// ActionHeld reports whether an action is currently held.
func (m *ActionMap) ActionHeld(action Action) bool {
	return m.states[action].Held
}

// ActionReleased reports whether an action was released this frame.
func (m *ActionMap) ActionReleased(action Action) bool {
	return m.states[action].Released
}

func matchesModifiers(actual, required Modifiers) bool {
	return (!required.Ctrl || actual.Ctrl) &&
		(!required.Shift || actual.Shift) &&
		(!required.Alt || actual.Alt)
}

func normalizeKeyName(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	return key
}
