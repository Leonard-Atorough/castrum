package input

import "testing"

// mustMap builds an ActionMap or fails the test.
func mustMap(t *testing.T, bindings Bindings, deadzone float64) *ActionMap {
	t.Helper()
	am, err := New(bindings, deadzone)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return am
}

// The resolution basics: each input kind folds into the action's
// tri-state, multiple bound inputs OR together, and duration
// accumulates while held and resets when it is not.
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

// The tick latch, end to end: a press lands in a frame where no tick
// runs, Pressed reports it for exactly one tick, and a tap that
// starts and finishes between ticks delivers both edges together.
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

// The axis crossing: values below the deadzone read as zero, crossing
// in is a press, crossing out is a release, and the edges land in the
// tick view like any other.
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

// The strongest axis wins, never a sum: a keyboard pair and a stick
// bound to the same action must not add up.
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

// The chord contract: the key fires only while every modifier is held,
// and a modifier dropped mid-hold stops the level without firing a
// release.
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

// Player addressing: Pad 0 folds across every connected pad, a player
// number addresses exactly that pad, and an out-of-range player reads
// nothing. Axis addressing follows the same rule, taking the
// strongest value among the addressed pads.
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

// The ownership contract: the ActionManager copies the bindings, so
// the caller's slices and the returned copies are all independent.
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

// A nil ActionMap reads zero on every query: the context publishes
// nil when no bindings were configured, and systems run without
// guards either way.
func TestNilActionMapReadsZero(t *testing.T) {
	var am *ActionMap
	if am.JustPressed("jump") || am.JustReleased("jump") || am.Held("jump") ||
		am.Pressed("jump") || am.Released("jump") ||
		am.Duration("jump") != 0 || am.Axis("move_x") != 0 {
		t.Error("a nil ActionMap should read zero on every query")
	}
}

// Direction-gated axis bindings: stick-up and stick-down drive
// different actions bound to the same axis, one per half, with the
// crossing edges following each action's own half.
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
