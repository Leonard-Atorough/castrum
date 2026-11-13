package castrum

import (
	"fmt"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/settings"
)

func TestGameInputPreferenceCodecDefaults(t *testing.T) {
	files := fstest.MapFS{
		"controls.json": {Data: []byte(`{"bindings":{"jump":[{"pad_button":{"button":0}}]},"deadzone":0.25}`)},
	}
	for _, codecs := range []*asset.Codecs{nil, NewAssetCodecs()} {
		game, err := New(WithFilesystem(files), WithAssetCodecs(codecs))
		if err != nil {
			t.Fatal(err)
		}
		loader := game.World().MustResource[*asset.Loader]()
		controls, err := loader.Load[input.Preferences]("controls.json")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := input.NewPreferences(controls.Bindings); err != nil {
			t.Fatal(err)
		}
		if got := controls.Bindings["jump"]; len(got) != 1 || got[0].PadButton == nil || got[0].PadButton.Button != input.PadSouth || controls.Deadzone != 0.25 {
			t.Fatalf("restored controls=%+v", controls)
		}
	}
}

func applyInputPreferences(t *testing.T, prefs *settings.Group[input.Preferences], actions *input.ActionMap) {
	t.Helper()
	value, err := prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	if err := actions.SetPreferences(value); err != nil {
		t.Fatal(err)
	}
}

func TestInputPreferencesPersistenceAndStartup(t *testing.T) {
	want := input.Preferences{Deadzone: 0.6, Bindings: input.Bindings{
		"jump": {
			input.KeyInput{Key: input.KeySpace, Modifiers: [3]input.Key{input.KeyControlLeft}}.Binding(),
			input.MouseButtonInput{Button: input.MouseButtonLeft}.Binding(),
			input.PadButtonInput{Pad: 2, Button: input.PadSouth}.Binding(),
		},
		"move": {
			input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD}.Binding(),
			input.PadAxisInput{Pad: 1, Axis: input.PadLeftStickX, Direction: -1, Deadzone: 0.05}.Binding(),
		},
		"global": {input.PadAxisInput{Axis: input.PadLeftStickX}.Binding()},
		"dash": {input.Chord(
			input.KeyInput{Key: input.KeyShiftLeft}.Binding(),
			input.PadAxisInput{Axis: input.PadRightTrigger, Deadzone: 0.25}.Binding(),
		)},
	}}
	document := fmt.Sprintf(`{"deadzone":0.6,"bindings":{
		"jump":[{"key":{"key":%d,"modifiers":[%d,0,0]}},{"mouse_button":{"button":%d}},{"pad_button":{"pad":2,"button":%d}}],
		"move":[{"key_pair":{"negative":%d,"positive":%d}},{"pad_axis":{"pad":1,"axis":%d,"direction":-1,"deadzone":0.05}}],
		"global":[{"pad_axis":{"axis":%d}}],
		"dash":[{"chord":{"inputs":[{"key":{"key":%d}},{"pad_axis":{"axis":%d,"deadzone":0.25}}]}}]
	}}`, input.KeySpace, input.KeyControlLeft, input.MouseButtonLeft, input.PadSouth,
		input.KeyA, input.KeyD, input.PadLeftStickX, input.PadLeftStickX, input.KeyShiftLeft, input.PadRightTrigger)
	loaded := roundTripPreferences(t, want, document)
	prefs, err := input.NewPreferences(loaded.Bindings)
	if err != nil {
		t.Fatal(err)
	}
	if err := prefs.Set(loaded); err != nil {
		t.Fatal(err)
	}
	value, err := prefs.Value()
	if err != nil {
		t.Fatal(err)
	}
	game, err := New(WithInputPreferences(value))
	if err != nil {
		t.Fatal(err)
	}
	actions := game.Context().Actions
	snapshot := &input.Snapshot{}
	snapshot.Keys[input.KeySpace] = input.State{Pressed: true, Held: true}
	snapshot.Keys[input.KeyControlLeft].Held = true
	snapshot.Keys[input.KeyShiftLeft].Held = true
	snapshot.Pads[0].Connected = true
	snapshot.Pads[0].Axes[input.PadLeftStickX] = -0.5
	snapshot.Pads[0].Axes[input.PadRightTrigger] = 0.6
	actions.Update(snapshot, 0.1)
	if !actions.JustPressed("jump") || actions.Axis("move") != -0.5 || !actions.Held("dash") || actions.Axis("global") != 0 {
		t.Fatal("restored bindings and deadzones did not resolve their physical inputs")
	}
}

func TestInputPreferencesStartupValidationAndOptionOrder(t *testing.T) {
	for _, invalid := range []input.Preferences{
		{},
		{Bindings: input.Bindings{}, Deadzone: -0.1},
		{Bindings: input.Bindings{}, Deadzone: 1.1},
	} {
		if game, err := New(WithInputPreferences(invalid)); err == nil || game != nil {
			t.Fatal("invalid startup preferences accepted")
		}
	}
	p := input.Preferences{Bindings: samplePreferenceBindings(), Deadzone: 0.3}
	replacement := input.Bindings{"move": {input.PadAxisInput{Axis: input.PadLeftStickX}.Binding()}}
	game, err := New(WithInputPreferences(p), WithBindings(replacement))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := &input.Snapshot{}
	snapshot.Pads[0].Connected = true
	snapshot.Pads[0].Axes[input.PadLeftStickX] = 0.2
	game.Context().Actions.Update(snapshot, 0)
	if !reflect.DeepEqual(game.Context().Actions.Bindings(), replacement) || game.Context().Actions.Axis("move") != 0 {
		t.Fatal("later bindings did not preserve the configured deadzone")
	}
	game, err = New(WithInputPreferences(p), WithBindings(nil))
	if err != nil || game.Context().Actions != nil {
		t.Fatal("later nil bindings did not disable the action map")
	}
}

func samplePreferenceBindings() input.Bindings {
	return input.Bindings{
		"jump": {input.Chord(input.KeyInput{Key: input.KeyA}.Binding(), input.KeyInput{Key: input.KeyB}.Binding())},
	}
}

func TestExplicitControlsRemovalReleasesHeldAction(t *testing.T) {
	bindings := input.Bindings{"move": {input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD}.Binding()}}
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
	replacement := input.Bindings{"move": {input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD}.Binding()}}
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
		p.Bindings["move"] = []input.Input{input.Chord(input.KeyInput{Key: input.KeyA}.Binding())}
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
	replacement["move"][0] = input.KeyInput{Key: input.KeyB}.Binding()
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
