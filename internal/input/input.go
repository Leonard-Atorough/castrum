package input

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type KeyState struct {
	Pressed  bool
	Held     bool
	Released bool
	Duration float64
}

type InputState struct {
	Keyboard map[ebiten.Key]KeyState
	Shift    bool
	Ctrl     bool
	Alt      bool
	Mouse    struct {
		X       int
		Y       int
		Buttons map[ebiten.MouseButton]KeyState
	}
}

func NewInputState() InputState {
	return InputState{
		Keyboard: make(map[ebiten.Key]KeyState),
		Mouse: struct {
			X       int
			Y       int
			Buttons map[ebiten.MouseButton]KeyState
		}{
			Buttons: make(map[ebiten.MouseButton]KeyState),
		},
	}
}

func (i *InputState) Reset() {
	for key := range i.Keyboard {
		i.Keyboard[key] = KeyState{}
	}
	i.Shift = false
	i.Ctrl = false
	i.Alt = false
	for button := range i.Mouse.Buttons {
		i.Mouse.Buttons[button] = KeyState{}
	}
}
