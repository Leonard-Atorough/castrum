package input

import (
	"fmt"

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

type InputBuffer struct {
	buffer []InputState
	size   int
	count  int
	head   int
	tail   int
}

func NewInputBuffer(size int) *InputBuffer {
	return &InputBuffer{
		buffer: make([]InputState, size),
		size:   size,
		head:   -1,
		tail:   0,
	}
}

func (b *InputBuffer) Push(input InputState) {

	b.head = (b.head + 1) % b.size
	if b.count != b.size {
		b.count++
	}

	b.buffer[b.head] = input

	if b.tail == b.head {
		b.tail = (b.tail + 1) % b.size
	}
}

func (b *InputBuffer) Pop() (InputState, error) {
	if b.count == 0 || b.head == -1 {
		return InputState{}, fmt.Errorf("Input buffer is empty")
	}
	// pop needs to read the value at tail, decrement the count, move the tail forward
	value := b.buffer[b.tail]
	b.tail = (b.tail + 1) % b.size
	b.count--
	return value, nil
}

func (b *InputBuffer) Peek() (InputState, error) {
	if b.count == 0 {
		return InputState{}, fmt.Errorf("Input buffer is empty")
	}
	return b.buffer[b.tail], nil
}

func (b *InputBuffer) IsEmpty() bool {
	return b.count == 0
}

func (b *InputBuffer) IsFull() bool {
	return b.count == b.size
}

func (b *InputBuffer) Clear() {
	b.buffer = make([]InputState, b.size)
	b.count = 0
	b.head = -1
	b.tail = 0
}

func (b *InputBuffer) Size() int {
	return b.size
}
func (b *InputBuffer) Count() int {
	return b.count
}
