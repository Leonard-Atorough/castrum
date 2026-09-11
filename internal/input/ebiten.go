package input

import (
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	publicinput "github.com/leonard-atorough/castrum/input"
)

// Handler polls Ebiten and resolves its physical input into game actions.
// It is internal so gameplay code does not depend on Ebiten's key types.
type Handler struct {
	actions         *publicinput.ActionMap
	lastFrameTime   time.Time
	currentKeys     map[string]publicinput.KeyState
	previousKeys    map[string]publicinput.KeyState
	justPressedKeys []ebiten.Key
	pressedKeys     []ebiten.Key
	releasedKeys    []ebiten.Key
}

// New creates an Ebiten-backed input handler.
func New(bindings publicinput.Bindings) *Handler {
	return &Handler{
		actions:      publicinput.NewActionMap(bindings),
		currentKeys:  make(map[string]publicinput.KeyState),
		previousKeys: make(map[string]publicinput.KeyState),
	}
}

// Snapshot polls Ebiten once and updates the resolved action states.
func (h *Handler) Snapshot() {
	now := time.Now()
	delta := 0.0
	if !h.lastFrameTime.IsZero() {
		delta = now.Sub(h.lastFrameTime).Seconds()
	}
	h.lastFrameTime = now

	clear(h.currentKeys)
	h.justPressedKeys = inpututil.AppendJustPressedKeys(h.justPressedKeys[:0])
	h.pressedKeys = inpututil.AppendPressedKeys(h.pressedKeys[:0])
	h.releasedKeys = inpututil.AppendJustReleasedKeys(h.releasedKeys[:0])

	for _, key := range h.pressedKeys {
		name := keyName(key)
		previous := h.previousKeys[name]
		h.currentKeys[name] = publicinput.KeyState{
			Pressed:  !previous.Held,
			Held:     true,
			Duration: previous.Duration + delta,
		}
	}
	for _, key := range h.justPressedKeys {
		name := keyName(key)
		state := h.currentKeys[name]
		state.Pressed = true
		state.Held = true
		h.currentKeys[name] = state
	}
	for _, key := range h.releasedKeys {
		name := keyName(key)
		h.currentKeys[name] = publicinput.KeyState{Released: true}
	}

	modifiers := publicinput.Modifiers{
		Shift: ebiten.IsKeyPressed(ebiten.KeyShift),
		Ctrl:  ebiten.IsKeyPressed(ebiten.KeyControl),
		Alt:   ebiten.IsKeyPressed(ebiten.KeyAlt),
	}
	h.actions.Update(h.currentKeys, modifiers)
	h.previousKeys = cloneStates(h.currentKeys)
}

// ActionPressed reports whether an action was pressed this frame.
func (h *Handler) ActionPressed(action publicinput.Action) bool {
	return h.actions.ActionPressed(action)
}

// ActionHeld reports whether an action is currently held.
func (h *Handler) ActionHeld(action publicinput.Action) bool {
	return h.actions.ActionHeld(action)
}

// ActionReleased reports whether an action was released this frame.
func (h *Handler) ActionReleased(action publicinput.Action) bool {
	return h.actions.ActionReleased(action)
}

func cloneStates(states map[string]publicinput.KeyState) map[string]publicinput.KeyState {
	cloned := make(map[string]publicinput.KeyState, len(states))
	for key, state := range states {
		cloned[key] = state
	}
	return cloned
}

func keyName(key ebiten.Key) string {
	return strings.ToLower(key.String())
}
