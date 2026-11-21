package input

import (
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/core"
)

func TestFrameSystemResolvesSnapshotResource(t *testing.T) {
	am, err := New(Bindings{
		"jump": {KeyInput{Key: KeySpace}.Binding()},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	world := core.NewWorld()
	snapshot := &Snapshot{}
	snapshot.Keys[KeySpace] = State{Pressed: true, Held: true}
	if err := world.Provide(func(*core.World) (*Snapshot, error) {
		return snapshot, nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx := &core.Context{World: world, DeltaTime: time.Second}
	if err := NewFrameSystem(am).Update(ctx); err != nil {
		t.Fatal(err)
	}
	if !am.JustPressed("jump") {
		t.Fatal("frame system did not resolve the snapshot resource into actions")
	}
}

func TestFrameSystemToleratesMissingSnapshot(t *testing.T) {
	am, err := New(Bindings{
		"jump": {KeyInput{Key: KeySpace}.Binding()},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &core.Context{World: core.NewWorld(), DeltaTime: time.Second}
	if err := NewFrameSystem(am).Update(ctx); err != nil {
		t.Fatalf("missing snapshot resource: %v", err)
	}
	if am.Held("jump") {
		t.Fatal("missing snapshot read as input")
	}
}

func TestTickSystemDeliversLatchedEdges(t *testing.T) {
	am, err := New(Bindings{
		"jump": {KeyInput{Key: KeySpace}.Binding()},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	world := core.NewWorld()
	snapshot := &Snapshot{}
	snapshot.Keys[KeySpace] = State{Pressed: true, Held: true}
	if err := world.Provide(func(*core.World) (*Snapshot, error) {
		return snapshot, nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx := &core.Context{World: world, DeltaTime: time.Second}
	if err := NewFrameSystem(am).Update(ctx); err != nil {
		t.Fatal(err)
	}
	if am.Pressed("jump") {
		t.Fatal("press latched before a tick")
	}
	if err := NewTickSystem(am).Update(ctx); err != nil {
		t.Fatal(err)
	}
	if !am.Pressed("jump") {
		t.Fatal("tick system did not latch the accumulated press")
	}
	if err := NewTickSystem(am).Update(ctx); err != nil {
		t.Fatal(err)
	}
	if am.Pressed("jump") {
		t.Fatal("tick edge survived a second tick")
	}
}
