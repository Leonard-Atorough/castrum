package input

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func TestInputPreferencesValidationAndCopies(t *testing.T) {
	if _, err := NewPreferences(nil); err == nil {
		t.Fatal("nil bindings accepted")
	}

	bindings := Bindings{
		"jump": {Chord(KeyInput{Key: KeyA}.Binding(), KeyInput{Key: KeyB}.Binding())},
	}
	prefs, err := NewPreferences(bindings)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := prefs.Value()
	bindings["jump"][0].Chord.Inputs[0] = KeyInput{Key: KeyC}.Binding()
	copy, _ := prefs.Value()
	copy.Bindings["jump"][0].Chord.Inputs[0].Key.Modifiers[0] = KeyShiftLeft
	copy.Bindings["jump"][0].Chord.Inputs[1] = KeyInput{Key: KeyD}.Binding()
	delete(copy.Bindings, "jump")
	if got, _ := prefs.Value(); !reflect.DeepEqual(got, original) {
		t.Fatal("mutating input or snapshot changed stored bindings")
	}
	if err := prefs.Set(Preferences{Bindings: Bindings{"jump": {Chord(KeyInput{Key: KeyA}.Binding())}}}); err == nil {
		t.Fatal("invalid chord accepted")
	}
	if err := prefs.Set(original); err != nil {
		t.Fatal(err)
	}
	if got, _ := prefs.Value(); !reflect.DeepEqual(got, original) {
		t.Fatal("invalid or unchanged update changed preferences")
	}
	if err := prefs.Set(Preferences{Bindings: bindings}); err != nil {
		t.Fatal(err)
	}
	bindings["jump"][0].Chord.Inputs[0] = KeyInput{Key: KeyD}.Binding()
	if got, _ := prefs.Value(); reflect.DeepEqual(got.Bindings, bindings) {
		t.Fatal("Set retained caller's bindings")
	}
}

func TestPreferencesValidateAndClone(t *testing.T) {
	original := Preferences{Bindings: Bindings{
		"jump": {Chord(KeyInput{Key: KeyA}.Binding(), KeyInput{Key: KeyB}.Binding())},
	}}
	if err := original.Validate(); err != nil {
		t.Fatal(err)
	}
	copy := original.Clone()
	copy.Bindings["jump"][0].Chord.Inputs[0].Key.Key = KeyC
	if reflect.DeepEqual(original, copy) {
		t.Fatal("clone shares chord inputs")
	}
	if err := (Preferences{}).Validate(); err == nil {
		t.Fatal("nil bindings accepted")
	}
	if clone := (Preferences{}).Clone(); clone.Bindings != nil {
		t.Fatal("cloning nil bindings changed them")
	}
}

func TestPreferencesRejectInvalidJSONBindings(t *testing.T) {
	prefs, err := NewPreferences(Bindings{"jump": {KeyInput{Key: KeySpace}.Binding()}})
	if err != nil {
		t.Fatal(err)
	}
	original, err := prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	for _, document := range []string{
		`{}`,
		`{"bindings":{"jump":[{}]}}`,
		`{"bindings":{"jump":[{"key":{},"mouse_button":{}}]}}`,
		`{"bindings":{"jump":[{"pad_axis":{"deadzone":1.1}}]}}`,
		`{"bindings":{},"deadzone":1.1}`,
		`{"bindings":{"jump":[{"chord":{"inputs":[{"key":{}}]}}]}}`,
		`{"bindings":{"jump":[{"chord":{"inputs":[{"key":{}},{"chord":{"inputs":[{"key":{}},{"key":{}}]}}]}}]}}`,
	} {
		t.Run(document, func(t *testing.T) {
			var value Preferences
			if err := json.Unmarshal([]byte(document), &value); err != nil {
				t.Fatal(err)
			}
			if err := prefs.Set(value); err == nil {
				t.Fatal("invalid restored bindings accepted")
			}
			got, err := prefs.Value()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, original) {
				t.Fatal("invalid restored bindings replaced the group")
			}
		})
	}
}

func TestGlobalDeadzoneValidationAndApplication(t *testing.T) {
	bindings := Bindings{
		"move":     {PadAxisInput{Axis: PadLeftStickX}.Binding()},
		"override": {PadAxisInput{Axis: PadLeftStickX, Deadzone: 0.05}.Binding()},
		"chord":    {Chord(KeyInput{Key: KeyA}.Binding(), PadAxisInput{Axis: PadLeftStickX}.Binding())},
	}
	prefs, err := NewPreferences(bindings)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := New(bindings, 0)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &Snapshot{}
	snapshot.Keys[KeyA].Held = true
	snapshot.Pads[0].Connected = true
	snapshot.Pads[0].Axes[PadLeftStickX] = 0.2
	actions.Update(snapshot, 0)
	actions.Tick()
	actions.Tick()

	if err := prefs.Update(func(p Preferences) Preferences {
		p.Deadzone = 0.3
		return p
	}); err != nil {
		t.Fatal(err)
	}
	actions.Update(snapshot, 0)
	if actions.Axis("move") != 0.2 || !actions.Held("chord") {
		t.Fatal("desired deadzone changed the map before application")
	}
	value, err := prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	if err := actions.SetPreferences(value); err != nil {
		t.Fatal(err)
	}
	actions.Update(snapshot, 0)
	if actions.Axis("move") != 0 || !actions.JustReleased("move") || !actions.JustReleased("chord") || actions.Axis("override") != 0.2 {
		t.Fatal("global deadzone did not affect axes and chords while preserving overrides")
	}
	actions.Tick()
	if !actions.Released("move") || !actions.Released("chord") {
		t.Fatal("deadzone changes lost fixed-tick releases")
	}
	before := actions.Preferences()
	for _, invalid := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		candidate := Preferences{Bindings: Bindings{}, Deadzone: invalid}
		if err := prefs.Set(candidate); err == nil {
			t.Fatal("group accepted invalid deadzone")
		}
		if err := actions.SetPreferences(candidate); err == nil {
			t.Fatal("action map accepted invalid deadzone")
		}
		if got, err := prefs.Value(); err != nil || !reflect.DeepEqual(got, value) {
			t.Fatal("invalid update changed desired preferences")
		}
		if !reflect.DeepEqual(actions.Preferences(), before) {
			t.Fatal("invalid update partially changed the action map")
		}
	}
	for _, deadzone := range []float64{0, 1} {
		if err := actions.SetPreferences(Preferences{Bindings: bindings, Deadzone: deadzone}); err != nil {
			t.Fatal(err)
		}
		actions.Update(snapshot, 0)
		want := 0.0
		if deadzone == 0 {
			want = 0.2
		}
		if actions.Axis("move") != want || actions.Axis("override") != 0.2 {
			t.Fatalf("deadzone %v: move=%v, override=%v", deadzone, actions.Axis("move"), actions.Axis("override"))
		}
	}
}
