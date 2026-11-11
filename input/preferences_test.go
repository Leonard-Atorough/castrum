package input

import (
	"reflect"
	"testing"
)

func TestInputPreferencesValidationAndCopies(t *testing.T) {
	if _, err := NewPreferences(nil); err == nil {
		t.Fatal("nil bindings accepted")
	}

	bindings := Bindings{
		"jump": {Chord(KeyInput{Key: KeyA}, KeyInput{Key: KeyB})},
	}
	prefs, err := NewPreferences(bindings)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := prefs.Value()
	bindings["jump"][0].(ChordInput).Inputs[0] = KeyInput{Key: KeyC}
	copy, _ := prefs.Value()
	copy.Bindings["jump"][0].(ChordInput).Inputs[1] = KeyInput{Key: KeyD}
	delete(copy.Bindings, "jump")
	if got, _ := prefs.Value(); !reflect.DeepEqual(got, original) {
		t.Fatal("mutating input or snapshot changed stored bindings")
	}
	if err := prefs.Set(Preferences{Bindings: Bindings{"jump": {Chord(KeyInput{Key: KeyA})}}}); err == nil {
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
	bindings["jump"][0].(ChordInput).Inputs[0] = KeyInput{Key: KeyD}
	if got, _ := prefs.Value(); reflect.DeepEqual(got.Bindings, bindings) {
		t.Fatal("Set retained caller's bindings")
	}
}

func TestPreferencesValidateAndClone(t *testing.T) {
	original := Preferences{Bindings: Bindings{
		"jump": {Chord(KeyInput{Key: KeyA}, KeyInput{Key: KeyB})},
	}}
	if err := original.Validate(); err != nil {
		t.Fatal(err)
	}
	copy := original.Clone()
	copy.Bindings["jump"][0].(ChordInput).Inputs[0] = KeyInput{Key: KeyC}
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
