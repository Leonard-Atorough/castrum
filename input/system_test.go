package input

import (
	"strings"
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
	if err := NewFrameSystem(am)(ctx); err != nil {
		t.Fatal(err)
	}
	if !am.JustPressed("jump") {
		t.Fatal("frame system did not resolve the snapshot resource into actions")
	}
}

func TestFrameSystemMissingSnapshotErrors(t *testing.T) {
	am, err := New(Bindings{
		"jump": {KeyInput{Key: KeySpace}.Binding()},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &core.Context{World: core.NewWorld(), DeltaTime: time.Second}
	err = NewFrameSystem(am)(ctx)
	if err == nil || !strings.Contains(err.Error(), "snapshot") {
		t.Fatalf("missing snapshot resource: err = %v, want a snapshot resolution error", err)
	}
	if am.Held("jump") {
		t.Fatal("failed frame must not read as input")
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
	if err := NewFrameSystem(am)(ctx); err != nil {
		t.Fatal(err)
	}
	if am.Pressed("jump") {
		t.Fatal("press latched before a tick")
	}
	if err := NewTickSystem(am)(ctx); err != nil {
		t.Fatal(err)
	}
	if !am.Pressed("jump") {
		t.Fatal("tick system did not latch the accumulated press")
	}
	if err := NewTickSystem(am)(ctx); err != nil {
		t.Fatal(err)
	}
	if am.Pressed("jump") {
		t.Fatal("tick edge survived a second tick")
	}
}
