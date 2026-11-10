package gametime

import (
	"math"
	"testing"
)

func TestNewSimulationDefaults(t *testing.T) {
	s := NewSimulation()
	if s.Tick != 0 {
		t.Fatalf("Tick = %d, want 0", s.Tick)
	}
	if s.Paused {
		t.Fatal("new simulation is paused")
	}
	if s.Scale != 1 {
		t.Fatalf("Scale = %v, want 1", s.Scale)
	}
}

func TestSimulationPauseResume(t *testing.T) {
	s := NewSimulation()
	s.Pause()
	if !s.Paused {
		t.Fatal("Pause did not pause")
	}
	s.Resume()
	if s.Paused {
		t.Fatal("Resume did not resume")
	}
}

func TestSimulationSetScale(t *testing.T) {
	s := NewSimulation()
	for _, scale := range []float64{0.5, 1, 2, 100} {
		if err := s.SetScale(scale); err != nil {
			t.Fatalf("SetScale(%v): %v", scale, err)
		}
		if s.Scale != scale {
			t.Fatalf("Scale = %v, want %v", s.Scale, scale)
		}
	}

	for name, scale := range map[string]float64{
		"zero":     0,
		"negative": -1,
		"NaN":      math.NaN(),
		"infinite": math.Inf(1),
	} {
		before := s.Scale
		if err := s.SetScale(scale); err == nil {
			t.Fatalf("%s scale accepted", name)
		}
		if s.Scale != before {
			t.Fatalf("%s scale changed Scale to %v", name, s.Scale)
		}
	}
}

func TestSimulationSnapshotRestore(t *testing.T) {
	s := NewSimulation()
	s.Tick = 41
	s.Pause()
	if err := s.SetScale(0.5); err != nil {
		t.Fatalf("SetScale: %v", err)
	}

	restored := &Simulation{Tick: s.Tick, Paused: s.Paused, Scale: s.Scale}
	if restored.Tick != 41 || !restored.Paused || restored.Scale != 0.5 {
		t.Fatalf("restored = %+v, want the snapshotted state", *restored)
	}
	restored.Resume()
	if restored.Paused {
		t.Fatal("restored simulation cannot resume")
	}
}
