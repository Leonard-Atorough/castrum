package input

import (
	"math"
	"testing"
)

func mustMap(t *testing.T, bindings Bindings, deadzone float64) *ActionMap {
	t.Helper()
	am, err := New(bindings, deadzone)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return am
}

func TestUpdateResolvesBindings(t *testing.T) {
	am := mustMap(t, Bindings{
		"jump": []Input{KeyInput{Key: KeySpace}, MouseButtonInput{Button: MouseButtonLeft}},
		"walk": []Input{KeyInput{Key: KeyW}},
	}, 0)

	s := &Snapshot{}
	s.Keys[KeySpace] = State{Pressed: true, Held: true}
	s.Keys[KeyW] = State{Held: true}
	am.Update(s, 0.1)
	am.Update(s, 0.1)

	if !am.JustPressed("jump") || !am.Held("jump") {
		t.Errorf("jump = just pressed %v, held %v, want true/true",
			am.JustPressed("jump"), am.Held("jump"))
	}
	if !am.Held("walk") {
		t.Error("walk should be held")
	}
	if am.Duration("walk") != 0.2 {
		t.Errorf("walk duration = %v, want 0.2", am.Duration("walk"))
	}

	// An input gone reads as not held, and duration resets - even
	// without a release edge.
	empty := &Snapshot{}
	am.Update(empty, 0.1)
	if am.Held("walk") {
		t.Error("walk should not be held on an empty snapshot")
	}
	if am.Duration("walk") != 0 {
		t.Errorf("walk duration = %v, want 0 after reset", am.Duration("walk"))
	}
}

func TestFlushEdgesClearsPendingTickEdgesAndPreservesState(t *testing.T) {
	am := mustMap(t, Bindings{"jump": []Input{KeyInput{Key: KeySpace}}}, 0)
	s := &Snapshot{}
	s.Keys[KeySpace] = State{Pressed: true, Held: true}
	am.Update(s, 0.1)

	am.FlushEdges()

	if am.JustPressed("jump") || am.JustReleased("jump") || am.Pressed("jump") || am.Released("jump") {
		t.Fatal("FlushEdges should clear frame, pending tick, and current tick edges")
	}
	if !am.Held("jump") || math.Abs(am.Duration("jump")-0.1) > 1e-9 {
		t.Fatalf("held = %v, duration = %v; want true and 0.1", am.Held("jump"), am.Duration("jump"))
	}

	am.Tick()
	if am.Pressed("jump") || am.Released("jump") {
		t.Fatal("Tick should not deliver edges that FlushEdges cleared")
	}

	s.Keys[KeySpace] = State{Held: true}
	am.Update(s, 0.1)
	if am.JustPressed("jump") || !am.Held("jump") || math.Abs(am.Duration("jump")-0.2) > 1e-9 {
		t.Fatalf("after held update: just pressed = %v, held = %v, duration = %v; want false, true, and 0.2",
			am.JustPressed("jump"), am.Held("jump"), am.Duration("jump"))
	}
}

func TestLatchDrainsPerTick(t *testing.T) {
	am := mustMap(t, Bindings{
		"jump": []Input{KeyInput{Key: KeySpace}},
	}, 0)

	// Frame 1: the press happens. Frame code sees it; tick code does
	// not yet.
	s := &Snapshot{}
	s.Keys[KeySpace] = State{Pressed: true, Held: true}
	am.Update(s, 0.016)
	if !am.JustPressed("jump") {
		t.Fatal("JustPressed should report the press before any Tick")
	}
	if am.Pressed("jump") {
		t.Fatal("Pressed should not report the press before a Tick delivers it")
	}

	// Tick 1 consumes the press. It reads true for the whole tick.
	am.Tick()
	if !am.Pressed("jump") {
		t.Fatal("Pressed should report the press in the tick that consumed it")
	}

	// Frame 2: nothing new. The tick view is unchanged until the next
	// Tick.
	am.Update(&Snapshot{}, 0.016)
	if !am.Pressed("jump") {
		t.Fatal("Pressed should stay true within the tick that consumed it")
	}

	// Tick 2: no events accumulated, so the tick view clears.
	am.Tick()
	if am.Pressed("jump") {
		t.Fatal("Pressed should be false in the tick after the press was consumed")
	}

	// A tap shorter than a tick interval: press in one frame, release
	// in the next, no tick between. Both edges must reach the tick.
	s = &Snapshot{}
	s.Keys[KeySpace] = State{Pressed: true, Held: true}
	am.Update(s, 0.016)
	s = &Snapshot{}
	s.Keys[KeySpace] = State{Released: true}
	am.Update(s, 0.016)
	am.Tick()
	if !am.Pressed("jump") || !am.Released("jump") {
		t.Errorf("tap between ticks: pressed/released = %v/%v, want true/true",
			am.Pressed("jump"), am.Released("jump"))
	}
	am.Tick()
	if am.Pressed("jump") || am.Released("jump") {
		t.Error("both edges should be consumed after one tick")
	}
}

func TestAxisDeadzoneCrossing(t *testing.T) {
	am := mustMap(t, Bindings{
		"move_x": []Input{PadAxisInput{Axis: PadLeftStickX}},
	}, 0.5)

	s := &Snapshot{}
	s.Pads[0] = PadSnapshot{Connected: true}
	s.Pads[0].Axes[PadLeftStickX] = 0.3
	am.Update(s, 0.016)
	if am.Axis("move_x") != 0 || am.Held("move_x") || am.JustPressed("move_x") {
		t.Errorf("below deadzone: axis %v, held %v, just pressed %v, want zero/false/false",
			am.Axis("move_x"), am.Held("move_x"), am.JustPressed("move_x"))
	}

	s.Pads[0].Axes[PadLeftStickX] = 0.8
	am.Update(s, 0.016)
	if am.Axis("move_x") != 0.8 || !am.Held("move_x") || !am.JustPressed("move_x") {
		t.Errorf("crossing in: axis %v, held %v, just pressed %v, want 0.8/true/true",
			am.Axis("move_x"), am.Held("move_x"), am.JustPressed("move_x"))
	}
	am.Tick()
	if !am.Pressed("move_x") {
		t.Error("the axis crossing press should reach the tick view")
	}

	s.Pads[0].Axes[PadLeftStickX] = 0.2
	am.Update(s, 0.016)
	if am.Held("move_x") || !am.JustReleased("move_x") {
		t.Errorf("crossing out: held %v, just released %v, want false/true",
			am.Held("move_x"), am.JustReleased("move_x"))
	}
	am.Tick()
	if !am.Released("move_x") || am.Pressed("move_x") {
		t.Error("the axis crossing release should reach the tick view, and the press should be gone")
	}
}

func TestAxisStrongestWins(t *testing.T) {
	am := mustMap(t, Bindings{
		"move_x": []Input{KeyPairInput{Negative: KeyA, Positive: KeyD}, PadAxisInput{Axis: PadLeftStickX}},
	}, 0)

	s := &Snapshot{}
	s.Keys[KeyD] = State{Held: true}
	s.Pads[0] = PadSnapshot{Connected: true}
	s.Pads[0].Axes[PadLeftStickX] = -0.4
	am.Update(s, 0.016)
	if am.Axis("move_x") != 1 {
		t.Errorf("axis = %v, want 1 (the pair is stronger than the stick, never a sum)",
			am.Axis("move_x"))
	}

	s = &Snapshot{}
	s.Pads[0] = PadSnapshot{Connected: true}
	s.Pads[0].Axes[PadLeftStickX] = -0.75
	am.Update(s, 0.016)
	if am.Axis("move_x") != -0.75 {
		t.Errorf("axis = %v, want -0.75", am.Axis("move_x"))
	}
}

func TestChordModifiers(t *testing.T) {
	am := mustMap(t, Bindings{
		"save": []Input{KeyInput{Key: KeyS, Modifiers: [3]Key{KeyControlLeft}}},
	}, 0)

	s := &Snapshot{}
	s.Keys[KeyControlLeft] = State{Held: true}
	s.Keys[KeyS] = State{Pressed: true, Held: true}
	am.Update(s, 0.016)
	if !am.JustPressed("save") || !am.Held("save") {
		t.Errorf("with modifiers held: just pressed %v, held %v, want true/true",
			am.JustPressed("save"), am.Held("save"))
	}

	// Same key, no modifier held: the chord does not fire.
	s = &Snapshot{}
	s.Keys[KeyS] = State{Pressed: true, Held: true}
	am.Update(s, 0.016)
	if am.JustPressed("save") || am.Held("save") {
		t.Errorf("without modifiers held: just pressed %v, held %v, want false/false",
			am.JustPressed("save"), am.Held("save"))
	}

	// Mid-hold modifier drop: the level stops, no release edge fires.
	s = &Snapshot{}
	s.Keys[KeyControlLeft] = State{Held: true}
	s.Keys[KeyS] = State{Held: true}
	am.Update(s, 0.016)
	s = &Snapshot{}
	s.Keys[KeyS] = State{Held: true}
	am.Update(s, 0.016)
	if am.Held("save") || am.JustReleased("save") {
		t.Errorf("after modifier drop: held %v, just released %v, want false/false",
			am.Held("save"), am.JustReleased("save"))
	}
}

func TestPadAddressing(t *testing.T) {
	am := mustMap(t, Bindings{
		"p1":    []Input{PadButtonInput{Pad: 1, Button: PadSouth}},
		"p2":    []Input{PadButtonInput{Pad: 2, Button: PadSouth}},
		"any":   []Input{PadButtonInput{Button: PadSouth}},
		"p1_x":  []Input{PadAxisInput{Pad: 1, Axis: PadLeftStickX}},
		"any_x": []Input{PadAxisInput{Axis: PadLeftStickX}},
		"p5":    []Input{PadButtonInput{Pad: 5, Button: PadSouth}},
	}, 0)

	s := &Snapshot{}
	s.Pads[0] = PadSnapshot{Connected: true}
	s.Pads[0].Buttons[PadSouth] = State{Held: true}
	s.Pads[0].Axes[PadLeftStickX] = 0.5
	s.Pads[1] = PadSnapshot{Connected: true}
	s.Pads[1].Axes[PadLeftStickX] = 0.9
	am.Update(s, 0.016)

	if !am.Held("p1") || am.Held("p2") {
		t.Errorf("p1/p2 held = %v/%v, want true/false: player 1 is the first pad",
			am.Held("p1"), am.Held("p2"))
	}
	if !am.Held("any") {
		t.Error("any-pad should fire from the first pad's button")
	}
	if am.Axis("p1_x") != 0.5 {
		t.Errorf("p1_x = %v, want 0.5", am.Axis("p1_x"))
	}
	if am.Axis("any_x") != 0.9 {
		t.Errorf("any_x = %v, want 0.9 (the strongest addressed pad wins)",
			am.Axis("any_x"))
	}
	if am.Held("p5") {
		t.Error("player 5 is out of range and should read nothing")
	}

	// The fold reaches across pads: player 2 alone still drives an
	// any-pad binding.
	s = &Snapshot{}
	s.Pads[1] = PadSnapshot{Connected: true}
	s.Pads[1].Buttons[PadSouth] = State{Held: true}
	am.Update(s, 0.016)
	if !am.Held("any") || am.Held("p1") {
		t.Errorf("player 2 alone: any/p1 held = %v/%v, want true/false",
			am.Held("any"), am.Held("p1"))
	}
}

func TestBindingsOwnership(t *testing.T) {
	caller := Bindings{
		"jump": []Input{KeyInput{Key: KeySpace}},
	}
	am := mustMap(t, caller, 0)

	// Mutating the caller's slice after construction cannot reach the
	// manager.
	caller["jump"][0] = KeyInput{Key: KeyA}
	if got := am.Bindings()["jump"][0]; got != (KeyInput{Key: KeySpace}) {
		t.Errorf("caller slice mutation leaked: binding = %v, want KeySpace", got)
	}

	// The map the manager hands back is the caller's to mutate.
	returned := am.Bindings()
	returned["jump"][0] = KeyInput{Key: KeyEnter}
	returned["extra"] = nil
	if got := am.Bindings()["jump"][0]; got != (KeyInput{Key: KeySpace}) {
		t.Errorf("returned copy mutation leaked: binding = %v, want KeySpace", got)
	}
	if _, ok := am.Bindings()["extra"]; ok {
		t.Error("returned copy map mutation leaked")
	}
}

// A nil snapshot resolves as every input off: no panic, all queries
// read zero, and duration resets.
func TestNilSnapshotResolvesOff(t *testing.T) {
	am := mustMap(t, Bindings{
		"jump": []Input{KeyInput{Key: KeySpace}},
	}, 0)

	s := &Snapshot{}
	s.Keys[KeySpace] = State{Held: true}
	am.Update(s, 0.1)

	am.Update(nil, 0.1)
	if am.JustPressed("jump") || am.Held("jump") || am.Pressed("jump") {
		t.Error("nil snapshot should resolve as every input off")
	}
	if am.Duration("jump") != 0 {
		t.Errorf("duration = %v, want 0 after a nil frame", am.Duration("jump"))
	}

	// Tick on a never-updated action must not panic either.
	am2 := mustMap(t, Bindings{"fire": []Input{KeyInput{Key: KeyNone}}}, 0)
	am2.Tick()
	if am2.Pressed("fire") {
		t.Error("never-updated action should read false after Tick")
	}
}

func TestNilActionMapReadsZero(t *testing.T) {
	var am *ActionMap
	if am.JustPressed("jump") || am.JustReleased("jump") || am.Held("jump") ||
		am.Pressed("jump") || am.Released("jump") ||
		am.Duration("jump") != 0 || am.Axis("move_x") != 0 {
		t.Error("a nil ActionMap should read zero on every query")
	}
}

func TestAxisDirectionGates(t *testing.T) {
	am := mustMap(t, Bindings{
		"forward":  []Input{PadAxisInput{Axis: PadLeftStickY, Direction: -1}},
		"backward": []Input{PadAxisInput{Axis: PadLeftStickY, Direction: 1}},
	}, 0)

	s := &Snapshot{}
	s.Pads[0] = PadSnapshot{Connected: true}
	s.Pads[0].Axes[PadLeftStickY] = -0.8 // screen-up
	am.Update(s, 0.016)
	if !am.Held("forward") || am.Held("backward") {
		t.Errorf("stick up: forward/backward held = %v/%v, want true/false",
			am.Held("forward"), am.Held("backward"))
	}

	if am.Axis("forward") != -0.8 {
		t.Errorf("forward axis = %v, want -0.8", am.Axis("forward"))
	}

	// Flipping the stick across the halves releases forward and
	// presses backward - each action's composite is its own half.
	s.Pads[0].Axes[PadLeftStickY] = 0.8
	am.Update(s, 0.016)
	if !am.JustReleased("forward") || am.Held("forward") {
		t.Errorf("flip to down: forward just released/held = %v/%v, want true/false",
			am.JustReleased("forward"), am.Held("forward"))
	}
	if !am.JustPressed("backward") || !am.Held("backward") {
		t.Errorf("flip to down: backward just pressed/held = %v/%v, want true/true",
			am.JustPressed("backward"), am.Held("backward"))
	}

	// A zero Direction takes either direction: both actions fire on
	// any deflection, the ungated default.
	ungated := mustMap(t, Bindings{
		"either": []Input{PadAxisInput{Axis: PadLeftStickY}},
	}, 0)
	ungated.Update(s, 0.016)
	if !ungated.Held("either") {
		t.Error("ungated axis binding should fire on any deflection")
	}
}

func TestBindingDeadzonesAndLatestAxis(t *testing.T) {
	am := mustMap(t, Bindings{
		"sensitive": []Input{PadAxisInput{Axis: PadLeftStickX, Deadzone: 0.05}},
		"inherited": []Input{PadAxisInput{Axis: PadLeftStickX}},
		"strict":    []Input{PadAxisInput{Axis: PadLeftStickX, Deadzone: 0.8}},
	}, 0.5)
	actions := []Action{"sensitive", "inherited", "strict"}
	previous := [3]bool{}
	for _, tt := range []struct {
		value float64
		want  [3]float64
	}{
		{value: 0.049},
		{value: 0.05, want: [3]float64{0.05, 0, 0}},
		{value: -0.05, want: [3]float64{-0.05, 0, 0}},
		{value: 0.5, want: [3]float64{0.5, 0.5, 0}},
		{value: 0.8, want: [3]float64{0.8, 0.8, 0.8}},
		{},
	} {
		s := &Snapshot{}
		s.Pads[0].Connected = true
		s.Pads[0].Axes[PadLeftStickX] = tt.value
		am.Update(s, 0.1)
		am.Tick()
		for i, action := range actions {
			held := tt.want[i] != 0
			pressed, released := held && !previous[i], !held && previous[i]
			if am.Axis(action) != tt.want[i] || am.Held(action) != held {
				t.Errorf("%s at %v: axis=%v held=%v; want %v, %v",
					action, tt.value, am.Axis(action), am.Held(action), tt.want[i], held)
			}
			if am.JustPressed(action) != pressed || am.JustReleased(action) != released ||
				am.Pressed(action) != pressed || am.Released(action) != released {
				t.Errorf("%s at %v: wrong frame or tick edges", action, tt.value)
			}
			previous[i] = held
		}
		am.Tick()
		for i, action := range actions {
			if am.Axis(action) != tt.want[i] || am.Pressed(action) || am.Released(action) {
				t.Errorf("%s: catch-up tick should retain the axis and clear delivered edges", action)
			}
		}
	}
}

func TestChordTransitionsAndTickLatch(t *testing.T) {
	for _, first := range []Key{KeyA, KeyB} {
		t.Run(first.String(), func(t *testing.T) {
			am := mustMap(t, Bindings{
				"dash": {Chord(KeyInput{Key: KeyA}, KeyInput{Key: KeyB})},
			}, 0)
			s := &Snapshot{}
			s.Keys[first].Held = true
			am.Update(s, 0.1)
			if am.Held("dash") || am.JustPressed("dash") {
				t.Fatal("one member must not activate the chord")
			}
			s.Keys[KeyA].Held, s.Keys[KeyB].Held = true, true
			am.Update(s, 0.1)
			if !am.Held("dash") || !am.JustPressed("dash") || am.Axis("dash") != 0 {
				t.Fatal("completing the chord should press a digital action")
			}
			am.Update(s, 0.1)
			if am.JustPressed("dash") || am.Duration("dash") != 0.2 {
				t.Fatal("a held chord must not repeat its press")
			}
			s.Keys[first].Held = false
			am.Update(s, 0.1)
			if am.Held("dash") || !am.JustReleased("dash") || am.Duration("dash") != 0 {
				t.Fatal("releasing either member must release the chord")
			}
			am.Tick()
			if !am.Pressed("dash") || !am.Released("dash") {
				t.Fatal("a chord tap between ticks must deliver both edges")
			}
			am.Tick()
			if am.Pressed("dash") || am.Released("dash") {
				t.Fatal("the next tick must consume both edges")
			}
			am.Update(nil, 0.1)
			if am.Held("dash") || am.JustReleased("dash") {
				t.Fatal("an inactive chord must stay inactive on a nil snapshot")
			}
		})
	}
}

func TestAxisTapBetweenTicksKeepsLatestLevel(t *testing.T) {
	am := mustMap(t, Bindings{
		"throttle": []Input{PadAxisInput{Axis: PadRightTrigger, Deadzone: 0.05}},
	}, 0)
	s := &Snapshot{}
	s.Pads[0].Connected = true
	s.Pads[0].Axes[PadRightTrigger] = 0.1
	am.Update(s, 0.01)
	am.Update(nil, 0.01)
	am.Tick()
	if am.Axis("throttle") != 0 || am.Held("throttle") || !am.Pressed("throttle") || !am.Released("throttle") {
		t.Fatal("axis tap should deliver both edges with the latest, zero level")
	}
}

func TestChordBindingsKeepIndependentTransitions(t *testing.T) {
	am := mustMap(t, Bindings{
		"dash": {
			KeyInput{Key: KeySpace},
			Chord(KeyInput{Key: KeyA}, KeyInput{Key: KeyB}),
			Chord(KeyInput{Key: KeyC}, KeyInput{Key: KeyD}),
		},
	}, 0)
	s := &Snapshot{}
	s.Keys[KeySpace].Held = true
	am.Update(s, 0.1)
	s.Keys[KeyA].Held, s.Keys[KeyB].Held = true, true
	am.Update(s, 0.1)
	if !am.JustPressed("dash") {
		t.Fatal("another held binding must not suppress a chord press")
	}
	s.Keys[KeyC].Held, s.Keys[KeyD].Held = true, true
	am.Update(s, 0.1)
	if !am.JustPressed("dash") {
		t.Fatal("the second chord must have its own press")
	}
	s.Keys[KeyB].Held = false
	am.Update(s, 0.1)
	if !am.Held("dash") || !am.JustReleased("dash") {
		t.Fatal("binding lists must retain their OR-folded levels and edges")
	}
}

func TestChordMixedDevicesAndGamepadCoherence(t *testing.T) {
	members := []Input{
		KeyInput{Key: KeyShiftLeft},
		MouseButtonInput{Button: MouseButtonLeft},
		PadButtonInput{Button: PadSouth},
		PadAxisInput{Axis: PadRightTrigger, Direction: 1, Deadzone: 0.05},
	}
	am := mustMap(t, Bindings{"fire": {Chord(members...)}}, 0.5)
	s := &Snapshot{}
	s.Keys[KeyShiftLeft].Held = true
	s.Mouse.Buttons[MouseButtonLeft].Held = true
	s.Pads[0].Connected, s.Pads[1].Connected = true, true
	s.Pads[0].Buttons[PadSouth].Held = true
	s.Pads[1].Axes[PadRightTrigger] = 0.1
	am.Update(s, 0.1)
	if am.Held("fire") {
		t.Fatal("any-pad members must not combine different controllers")
	}
	s.Pads[1].Buttons[PadSouth].Held = true
	am.Update(s, 0.1)
	if !am.Held("fire") || !am.JustPressed("fire") {
		t.Fatal("one matching controller should satisfy all any-pad members")
	}
	s.Pads[1].Connected = false
	am.Update(s, 0.1)
	if am.Held("fire") || !am.JustReleased("fire") {
		t.Fatal("disconnecting the matching controller must release the chord")
	}
	s.Pads[1].Connected = true
	members[3] = PadAxisInput{Pad: 2, Axis: PadRightTrigger, Deadzone: 0.05}
	if err := am.SetBindings(Bindings{"fire": {Chord(members...)}}); err != nil {
		t.Fatalf("SetBindings: %v", err)
	}
	s.Pads[1].Buttons[PadSouth].Held = false
	am.Update(s, 0.1)
	if !am.Held("fire") {
		t.Fatal("explicit pad addresses should allow intentional cross-controller chords")
	}
	s.Mouse.Buttons[MouseButtonLeft].Held = false
	am.Update(s, 0.1)
	if am.Held("fire") || !am.JustReleased("fire") {
		t.Fatal("every device member must remain active")
	}
}

func TestChordBindingsOwnershipAndRebinding(t *testing.T) {
	members := []Input{KeyInput{Key: KeyA}, KeyInput{Key: KeyB}}
	am := mustMap(t, Bindings{"dash": []Input{ChordInput{Inputs: members}}}, 0)
	members[0] = KeyInput{Key: KeyC}
	returned := am.Bindings()["dash"][0].(ChordInput)
	returned.Inputs[1] = KeyInput{Key: KeyD}
	s := &Snapshot{}
	s.Keys[KeyA].Held, s.Keys[KeyB].Held = true, true
	am.Update(s, 0.1)
	if !am.Held("dash") {
		t.Fatal("caller and returned member slices must not mutate owned bindings")
	}
	if err := am.SetBindings(Bindings{"dash": []Input{ChordInput{Inputs: members[:1]}}}); err == nil {
		t.Fatal("invalid replacement should fail")
	}
	am.Update(s, 0.1)
	if !am.Held("dash") || am.JustPressed("dash") {
		t.Fatal("failed replacement must preserve chord state")
	}
	replacement := []Input{KeyInput{Key: KeyC}, KeyInput{Key: KeyD}}
	if err := am.SetBindings(Bindings{"dash": []Input{
		KeyInput{Key: KeyEnter}, ChordInput{Inputs: replacement},
	}}); err != nil {
		t.Fatalf("SetBindings: %v", err)
	}
	replacement[0] = KeyInput{Key: KeyE}
	if !am.Held("dash") {
		t.Fatal("replacement must not change published state before Update")
	}
	am.Update(s, 0.1)
	if am.Held("dash") {
		t.Fatal("the old chord must not survive replacement")
	}
	s.Keys[KeyC].Held, s.Keys[KeyD].Held = true, true
	am.Update(s, 0.1)
	if !am.Held("dash") || !am.JustPressed("dash") {
		t.Fatal("replacement members must be copied and get fresh transition state")
	}
}

func TestBindingValidation(t *testing.T) {
	nested := ChordInput{Inputs: []Input{KeyInput{Key: KeyA}, KeyInput{Key: KeyB}}}
	for _, bound := range []Input{
		nil, &KeyInput{Key: KeyA}, ChordInput{},
		ChordInput{Inputs: []Input{KeyInput{Key: KeyA}}},
		ChordInput{Inputs: []Input{nested, KeyInput{Key: KeyC}}},
		ChordInput{Inputs: []Input{KeyInput{Key: KeyA}, nil}},
		PadAxisInput{Deadzone: -0.1}, PadAxisInput{Deadzone: 1.1},
		PadAxisInput{Deadzone: math.NaN()}, PadAxisInput{Deadzone: math.Inf(1)},
		ChordInput{Inputs: []Input{KeyInput{Key: KeyA}, PadAxisInput{Deadzone: math.NaN()}}},
	} {
		bindings := Bindings{"invalid": []Input{bound}}
		if _, err := New(bindings, 0); err == nil {
			t.Errorf("New accepted invalid binding %#v", bound)
		}
		am := mustMap(t, Bindings{}, 0)
		if err := am.SetBindings(bindings); err == nil {
			t.Errorf("SetBindings accepted invalid binding %#v", bound)
		}
	}
	for _, deadzone := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := New(Bindings{}, deadzone); err == nil {
			t.Errorf("New accepted invalid deadzone %v", deadzone)
		}
	}
}

func TestChordUpdateAllocations(t *testing.T) {
	am := mustMap(t, Bindings{
		"dash": {Chord(
			KeyPairInput{Negative: KeyA, Positive: KeyD},
			PadAxisInput{Axis: PadLeftStickX, Direction: -1, Deadzone: 0.05},
		)},
	}, 0)
	s := &Snapshot{}
	s.Keys[KeyD].Held = true
	s.Pads[0].Connected = true
	s.Pads[0].Axes[PadLeftStickX] = -0.1
	am.Update(s, 0.1)
	if !am.Held("dash") {
		t.Fatal("axis and key-pair members should activate the chord")
	}
	if got := testing.AllocsPerRun(100, func() {
		am.Update(s, 0.1)
		am.Tick()
	}); got != 0 {
		t.Errorf("settled Update/Tick allocated %v times, want zero", got)
	}
	s.Keys[KeyA].Held = true
	am.Update(s, 0.1)
	if am.Held("dash") || !am.JustReleased("dash") {
		t.Fatal("opposing key-pair keys should cancel and release the chord")
	}
}
