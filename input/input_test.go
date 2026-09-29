package input

import "testing"

// keyNames must stay value-parallel to the Key constants: the array
// length is compile-checked against NumKeys, but only this test pins
// the order. The spot checks land on group boundaries, so a key
// inserted mid-block with its name misplaced fails one of them.
func TestKeyNames(t *testing.T) {
	spot := map[Key]string{
		KeyA:            "a",
		KeyW:            "w",
		KeyZ:            "z",
		Key0:            "0",
		Key9:            "9",
		KeySlash:        "slash",
		KeyISOBackslash: "iso_backslash",
		KeySpace:        "space",
		KeyBackspace:    "backspace",
		KeyAltRight:     "alt_right",
		KeyF1:           "f1",
		KeyF12:          "f12",
		KeyPause:        "pause",
		KeyArrowRight:   "arrow_right",
		KeyNumpad0:      "numpad_0",
		KeyNumLock:      "num_lock",
	}
	for k, want := range spot {
		if keyNames[k] != want {
			t.Errorf("keyNames[%v] = %q, want %q", k, keyNames[k], want)
		}
	}

	// No empty names and no duplicates: these names become the
	// config-file vocabulary, where a collision is silently
	// ambiguous bindings.
	seen := make(map[string]bool, NumKeys)
	for i, name := range keyNames {
		if name == "" {
			t.Errorf("keyNames[%d] is empty", i)
		}
		if seen[name] {
			t.Errorf("keyNames: duplicate name %q", name)
		}
		seen[name] = true
	}
}

func TestKeyString(t *testing.T) {
	if got := KeyA.String(); got != "a" {
		t.Errorf("KeyA.String() = %q, want %q", got, "a")
	}
	if got := Key(-1).String(); got != "Key(-1)" {
		t.Errorf("Key(-1).String() = %q, want %q", got, "Key(-1)")
	}
	if got := Key(9999).String(); got != "Key(9999)" {
		t.Errorf("Key(9999).String() = %q, want %q", got, "Key(9999)")
	}
}

// The small enums are short enough to check against their full
// expected arrays, which pins both names and order.
func TestMouseButtonNames(t *testing.T) {
	want := [NumMouseButtons]string{"left", "right", "middle"}
	if mouseButtonNames != want {
		t.Errorf("mouseButtonNames = %v, want %v", mouseButtonNames, want)
	}
	if got := MouseButton(-1).String(); got != "MouseButton(-1)" {
		t.Errorf("MouseButton(-1).String() = %q, want %q", got, "MouseButton(-1)")
	}
	if got := MouseButton(9).String(); got != "MouseButton(9)" {
		t.Errorf("MouseButton(9).String() = %q, want %q", got, "MouseButton(9)")
	}
}

func TestPadButtonNames(t *testing.T) {
	want := [NumPadButtons]string{
		"south", "east", "west", "north",
		"left_shoulder", "right_shoulder",
		"left_stick", "right_stick",
		"back", "start", "guide",
		"dpad_up", "dpad_down", "dpad_left", "dpad_right",
	}
	if padButtonNames != want {
		t.Errorf("padButtonNames = %v, want %v", padButtonNames, want)
	}
	if got := PadButton(-1).String(); got != "PadButton(-1)" {
		t.Errorf("PadButton(-1).String() = %q, want %q", got, "PadButton(-1)")
	}
	if got := PadButton(99).String(); got != "PadButton(99)" {
		t.Errorf("PadButton(99).String() = %q, want %q", got, "PadButton(99)")
	}
}

func TestPadAxisNames(t *testing.T) {
	want := [NumPadAxes]string{
		"left_stick_x", "left_stick_y",
		"right_stick_x", "right_stick_y",
		"left_trigger", "right_trigger",
	}
	if padAxisNames != want {
		t.Errorf("padAxisNames = %v, want %v", padAxisNames, want)
	}
	if got := PadAxis(-1).String(); got != "PadAxis(-1)" {
		t.Errorf("PadAxis(-1).String() = %q, want %q", got, "PadAxis(-1)")
	}
	if got := PadAxis(9).String(); got != "PadAxis(9)" {
		t.Errorf("PadAxis(9).String() = %q, want %q", got, "PadAxis(9)")
	}
}
