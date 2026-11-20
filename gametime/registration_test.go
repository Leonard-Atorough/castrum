package gametime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/serialization"

	"github.com/Leonard-Atorough/castrum/core"
)

func newRegistry(t *testing.T) *serialization.Registry {
	t.Helper()
	world := core.NewWorld()
	if err := world.Provide(func(*core.World) (*asset.Codecs, error) {
		return asset.NewCodecs(), nil
	}); err != nil {
		t.Fatalf("provide codecs: %v", err)
	}
	return serialization.NewRegistry(world)
}

func TestRegisterBlueprints(t *testing.T) {
	reg := newRegistry(t)
	if err := RegisterBlueprints(reg); err != nil {
		t.Fatalf("RegisterBlueprints: %v", err)
	}
	// Registration is idempotent-hostile: a second call collides.
	if err := RegisterBlueprints(reg); err == nil {
		t.Fatal("duplicate engine registration accepted")
	}
}

func TestGroupsJSONRoundTrip(t *testing.T) {
	g := NewGroups()
	g.write("world").paused = true
	g.write("dialogue").scale = 0.5

	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("marshal groups: %v", err)
	}
	// Deterministic wire form: name-ordered.
	want := `[{"name":"dialogue","paused":false,"scale":0.5},{"name":"world","paused":true,"scale":1}]`
	if string(data) != want {
		t.Fatalf("wire form = %s, want %s", data, want)
	}

	restored := NewGroups()
	if err := json.Unmarshal(data, restored); err != nil {
		t.Fatalf("unmarshal groups: %v", err)
	}
	restoredData, err := json.Marshal(restored)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	if string(restoredData) != want {
		t.Fatalf("round trip = %s, want %s", restoredData, want)
	}
}

func TestGroupsJSONValidation(t *testing.T) {
	for name, raw := range map[string]string{
		"empty name":     `[{"name":"","paused":false,"scale":1}]`,
		"zero scale":     `[{"name":"world","paused":false,"scale":0}]`,
		"negative scale": `[{"name":"world","paused":false,"scale":-1}]`,
	} {
		g := NewGroups()
		if err := json.Unmarshal([]byte(raw), g); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestSimulationRealJSONRoundTrip(t *testing.T) {
	sim := NewSimulation(250 * time.Millisecond)
	sim.Tick = 41
	sim.Pause()
	if err := sim.SetScale(0.5); err != nil {
		t.Fatalf("SetScale: %v", err)
	}
	data, err := json.Marshal(sim)
	if err != nil {
		t.Fatalf("marshal simulation: %v", err)
	}
	restored := &Simulation{}
	if err := json.Unmarshal(data, restored); err != nil {
		t.Fatalf("unmarshal simulation: %v", err)
	}
	if restored.Tick != 41 || !restored.Paused || restored.Scale != 0.5 || restored.Delta != 250*time.Millisecond {
		t.Fatalf("restored = %+v, want the snapshotted state", *restored)
	}

	real := NewReal(time.Second)
	real.Frame = 7
	real.Elapsed = 3 * time.Second
	data, err = json.Marshal(real)
	if err != nil {
		t.Fatalf("marshal real: %v", err)
	}
	restoredReal := &Real{}
	if err := json.Unmarshal(data, restoredReal); err != nil {
		t.Fatalf("unmarshal real: %v", err)
	}
	if restoredReal.Frame != 7 || restoredReal.Elapsed != 3*time.Second || restoredReal.MaxFrameTime != time.Second {
		t.Fatalf("restored real = %+v, want the snapshotted state", *restoredReal)
	}
}
