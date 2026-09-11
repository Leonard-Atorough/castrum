package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	publicinput "github.com/leonard-atorough/castrum/input"
)

type fakeBackend struct {
	justPressed  []ebiten.Key
	pressed      []ebiten.Key
	justReleased []ebiten.Key
	keys         map[ebiten.Key]bool
}

func (f *fakeBackend) JustPressedKeys() []ebiten.Key  { return f.justPressed }
func (f *fakeBackend) PressedKeys() []ebiten.Key      { return f.pressed }
func (f *fakeBackend) JustReleasedKeys() []ebiten.Key { return f.justReleased }
func (f *fakeBackend) IsKeyPressed(key ebiten.Key) bool {
	return f.keys[key]
}

func TestHandler_SnapshotResolvesSpaceBinding(t *testing.T) {
	backend := &fakeBackend{
		justPressed: []ebiten.Key{ebiten.KeySpace},
		pressed:     []ebiten.Key{ebiten.KeySpace},
		keys:        make(map[ebiten.Key]bool),
	}
	handler := New(publicinput.Bindings{
		publicinput.Action("Jump_Up"): {{Key: "space"}},
	})
	handler.backend = backend

	handler.Snapshot()

	if !handler.ActionPressed(publicinput.Action("Jump_Up")) {
		t.Fatal("expected Jump_Up to be pressed")
	}
	if !handler.ActionHeld(publicinput.Action("Jump_Up")) {
		t.Fatal("expected Jump_Up to be held")
	}
}

func TestKeyName_NormalizesEbitenKey(t *testing.T) {
	if got := keyName(ebiten.KeySpace); got != "space" {
		t.Fatalf("keyName(KeySpace) = %q, want space", got)
	}
	if got := keyName(ebiten.KeyArrowUp); got != "arrowup" {
		t.Fatalf("keyName(KeyArrowUp) = %q, want arrowup", got)
	}
}
