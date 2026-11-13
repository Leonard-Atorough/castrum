package input

import "testing"

func TestRemovedBindingReleasesAndPreservesPendingEdges(t *testing.T) {
	am := mustMap(t, Bindings{
		"move": {KeyPairInput{Negative: KeyA, Positive: KeyD}.Binding()},
		"jump": {KeyInput{Key: KeySpace}.Binding()},
	}, 0)
	snapshot := &Snapshot{}
	snapshot.Keys[KeyD] = State{Pressed: true, Held: true}
	snapshot.Keys[KeySpace] = State{Pressed: true, Held: true}
	am.Update(snapshot, 0.1)
	if !am.Held("move") || !am.JustPressed("move") || am.Axis("move") != 1 {
		t.Fatal("move did not become active")
	}
	if err := am.SetBindings(Bindings{"jump": {KeyInput{Key: KeySpace}.Binding()}}); err != nil {
		t.Fatal(err)
	}
	if !am.Held("move") || am.Axis("move") != 1 {
		t.Fatal("replacement changed the current frame before Update")
	}
	am.Update(snapshot, 0.1)
	if am.Held("move") || am.Axis("move") != 0 || am.Duration("move") != 0 ||
		am.JustPressed("move") || !am.JustReleased("move") {
		t.Fatal("removed action did not release in the next frame")
	}
	if !am.Held("jump") || am.Duration("jump") != 0.2 {
		t.Fatal("removing move disturbed the retained action")
	}
	am.Tick()
	if !am.Pressed("move") || !am.Released("move") || !am.Pressed("jump") {
		t.Fatal("tick did not receive pending press and synthesized release")
	}
	am.Update(snapshot, 0.1)
	if am.JustReleased("move") {
		t.Fatal("removed action released in more than one frame")
	}
	if !am.Pressed("move") || !am.Released("move") {
		t.Fatal("frame update cleared the current tick's edges")
	}
	am.Tick()
	if am.Pressed("move") || am.Released("move") {
		t.Fatal("removed action's edges survived a second tick")
	}
	if _, exists := am.states["move"]; exists {
		t.Fatal("inactive removed action was not reclaimed")
	}
}

func TestRemovedBindingDeliversEdgesAcrossDelayedTicksAndRebinding(t *testing.T) {
	am := mustMap(t, Bindings{"jump": {KeyInput{Key: KeySpace}.Binding()}}, 0)
	snapshot := &Snapshot{}
	snapshot.Keys[KeySpace] = State{Pressed: true, Held: true}
	am.Update(snapshot, 0)
	if err := am.SetBindings(Bindings{}); err != nil {
		t.Fatal(err)
	}
	am.Tick()
	if !am.Pressed("jump") || am.Released("jump") {
		t.Fatal("tick before input update did not retain the old binding's press")
	}
	am.Update(nil, 0)
	am.Update(nil, 0)
	if err := am.SetBindings(Bindings{"jump": {KeyInput{Key: KeyA}.Binding()}}); err != nil {
		t.Fatal(err)
	}
	am.Update(nil, 0)
	am.Tick()
	if am.Pressed("jump") || !am.Released("jump") {
		t.Fatal("release was lost when the action was re-added before the tick")
	}
	am.Tick()
	if am.Released("jump") {
		t.Fatal("synthesized release was delivered twice")
	}
}

func TestRemovingInactiveBindingDoesNotSynthesizeRelease(t *testing.T) {
	am := mustMap(t, Bindings{"jump": {KeyInput{Key: KeySpace}.Binding()}}, 0)
	am.Update(nil, 0)
	if err := am.SetBindings(Bindings{}); err != nil {
		t.Fatal(err)
	}
	am.Update(nil, 0)
	if _, exists := am.states["jump"]; exists {
		t.Fatal("inactive removed action was not reclaimed without a tick")
	}
	am.Tick()
	if am.JustReleased("jump") || am.Released("jump") {
		t.Fatal("inactive action produced a release")
	}
}

func TestRemovingBindingPreservesCompletedTap(t *testing.T) {
	am := mustMap(t, Bindings{"jump": {KeyInput{Key: KeySpace}.Binding()}}, 0)
	press := &Snapshot{}
	press.Keys[KeySpace] = State{Pressed: true, Held: true}
	am.Update(press, 0)
	release := &Snapshot{}
	release.Keys[KeySpace] = State{Released: true}
	am.Update(release, 0)
	if err := am.SetBindings(Bindings{}); err != nil {
		t.Fatal(err)
	}
	am.Update(nil, 0)
	if am.JustReleased("jump") {
		t.Fatal("completed tap produced another frame release")
	}
	am.Tick()
	if !am.Pressed("jump") || !am.Released("jump") {
		t.Fatal("completed tap edges were lost")
	}
	am.Tick()
	if am.Pressed("jump") || am.Released("jump") {
		t.Fatal("completed tap edges were delivered twice")
	}
}
