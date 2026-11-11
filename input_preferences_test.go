package castrum

import (
	"reflect"
	"testing"

	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/settings"
)

func applyInputPreferences(t *testing.T, prefs *settings.Group[input.Preferences], actions *input.ActionMap) {
	t.Helper()
	value, err := prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actions.Bindings(), value.Bindings) {
		if err := actions.SetBindings(value.Bindings); err != nil {
			t.Fatal(err)
		}
	}
}

func samplePreferenceBindings() input.Bindings {
	return input.Bindings{
		"jump": {input.Chord(input.KeyInput{Key: input.KeyA}, input.KeyInput{Key: input.KeyB})},
	}
}

func TestExplicitControlsRemovalReleasesHeldAction(t *testing.T) {
	bindings := input.Bindings{"move": {input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD}}}
	prefs, err := input.NewPreferences(bindings)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := input.New(bindings, 0)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &input.Snapshot{}
	snapshot.Keys[input.KeyD] = input.State{Held: true}
	actions.Update(snapshot, 0)
	if err := prefs.Set(input.Preferences{Bindings: input.Bindings{}}); err != nil {
		t.Fatal(err)
	}
	applyInputPreferences(t, prefs, actions)
	actions.Update(nil, 0)
	if actions.Held("move") || actions.Axis("move") != 0 || !actions.JustReleased("move") {
		t.Fatal("removing a binding did not release its action")
	}
	actions.Tick()
	if !actions.Pressed("move") || !actions.Released("move") {
		t.Fatal("fixed tick did not receive the accumulated edges")
	}
}

func TestExplicitControlsApplication(t *testing.T) {
	initial := samplePreferenceBindings()
	prefs, err := input.NewPreferences(initial)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := input.New(input.Bindings{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	applyInputPreferences(t, prefs, actions)
	if !reflect.DeepEqual(actions.Bindings(), initial) {
		t.Fatal("initial bindings not applied")
	}
	replacement := input.Bindings{"move": {input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD}}}
	if err := prefs.Set(input.Preferences{Bindings: replacement}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actions.Bindings(), initial) {
		t.Fatal("changing preferences changed the action map without application")
	}
	applyInputPreferences(t, prefs, actions)
	if !reflect.DeepEqual(actions.Bindings(), replacement) {
		t.Fatal("replacement bindings not applied")
	}
	if err := prefs.Update(func(p input.Preferences) input.Preferences {
		p.Bindings["move"] = []input.Input{input.Chord(input.KeyInput{Key: input.KeyA})}
		return p
	}); err == nil {
		t.Fatal("invalid rebind succeeded")
	}
	value, err := prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(value.Bindings, replacement) || !reflect.DeepEqual(actions.Bindings(), replacement) {
		t.Fatal("invalid rebind changed preferences or action map")
	}
	replacement["move"][0] = input.KeyInput{Key: input.KeyB}
	if reflect.DeepEqual(actions.Bindings(), replacement) {
		t.Fatal("action map shares caller's bindings")
	}
}

func TestExplicitControlsApplicationPreservesUnchangedChordState(t *testing.T) {
	bindings := samplePreferenceBindings()
	prefs, err := input.NewPreferences(bindings)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := input.New(bindings, 0)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &input.Snapshot{}
	snapshot.Keys[input.KeyA] = input.State{Held: true}
	snapshot.Keys[input.KeyB] = input.State{Held: true}
	actions.Update(snapshot, 0)
	if !actions.JustPressed("jump") {
		t.Fatal("chord did not start")
	}
	applyInputPreferences(t, prefs, actions)
	actions.Update(snapshot, 0)
	if actions.JustPressed("jump") {
		t.Fatal("applying identical bindings restarted chord")
	}
}
