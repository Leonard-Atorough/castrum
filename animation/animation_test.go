package animation

import (
	"strings"
	"testing"
)

// One row per validation rule; the input is mutated relative to testSets.
func TestValidationRules(t *testing.T) {
	validTrack := Track{Name: "body", Source: "atlas.json", Frames: []string{"a"}, FPS: 10}
	validState := func() State {
		return State{Name: "idle", Loop: Loop, Tracks: []Track{validTrack}}
	}
	validSet := func() Set {
		return Set{Name: "hero", States: map[string]State{"idle": validState()}}
	}

	tests := []struct {
		name    string
		build   func() (Sets, error)
		wantErr string
	}{
		{"empty sets", func() (Sets, error) { return NewSets(map[string]Set{}) }, "at least one set"},
		{"key mismatch", func() (Sets, error) {
			return NewSets(map[string]Set{"hero": {Name: "other", States: validSet().States}})
		}, "mismatched name"},
		{"empty set name", func() (Sets, error) {
			return NewSets(map[string]Set{"": {Name: "", States: validSet().States}})
		}, "set name cannot be empty"},
		{"no states", func() (Sets, error) {
			return NewSets(map[string]Set{"hero": {Name: "hero"}})
		}, "at least one state"},
		{"state key mismatch", func() (Sets, error) {
			s := validSet()
			s.States["walk"] = validState() // Name is "idle", key is "walk"
			return NewSets(map[string]Set{"hero": s})
		}, "mismatched name"},
		{"dangling next", func() (Sets, error) {
			s := validSet()
			st := validState()
			st.Next = "gone"
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "unknown state"},
		{"no tracks", func() (Sets, error) {
			s := validSet()
			s.States["idle"] = State{Name: "idle"}
			return NewSets(map[string]Set{"hero": s})
		}, "at least one track"},
		{"duplicate track names", func() (Sets, error) {
			s := validSet()
			st := validState()
			st.Tracks = []Track{validTrack, validTrack}
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "duplicate track name"},
		{"bad loop", func() (Sets, error) {
			s := validSet()
			st := validState()
			st.Loop = LoopMode(99)
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "invalid loop mode"},
		{"negative blend", func() (Sets, error) {
			s := validSet()
			st := validState()
			st.EnterBlend = -0.1
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "invalid enter blend"},
		{"unknown ease", func() (Sets, error) {
			s := validSet()
			st := validState()
			st.BlendEase = "warp"
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "unknown blend ease"},
		{"track no name", func() (Sets, error) {
			s := validSet()
			st := validState()
			tr := validTrack
			tr.Name = ""
			st.Tracks = []Track{tr}
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "track must have a name"},
		{"track bad source", func() (Sets, error) {
			s := validSet()
			st := validState()
			tr := validTrack
			tr.Source = ""
			st.Tracks = []Track{tr}
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "source"},
		{"track no frames", func() (Sets, error) {
			s := validSet()
			st := validState()
			tr := validTrack
			tr.Frames = nil
			st.Tracks = []Track{tr}
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "at least one frame"},
		{"track empty frame name", func() (Sets, error) {
			s := validSet()
			st := validState()
			tr := validTrack
			tr.Frames = []string{"a", ""}
			st.Tracks = []Track{tr}
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "must have a name"},
		{"track zero fps", func() (Sets, error) {
			s := validSet()
			st := validState()
			tr := validTrack
			tr.FPS = 0
			st.Tracks = []Track{tr}
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "fps"},
		{"track fps over cap", func() (Sets, error) {
			s := validSet()
			st := validState()
			tr := validTrack
			tr.FPS = 61
			st.Tracks = []Track{tr}
			s.States["idle"] = st
			return NewSets(map[string]Set{"hero": s})
		}, "must not exceed 60"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.build()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestNewSetsDeepCopiesInput(t *testing.T) {
	input := map[string]Set{
		"hero": {Name: "hero", States: map[string]State{
			"idle": {Name: "idle", Loop: Loop, Tracks: []Track{{Name: "body", Source: "a.json", Frames: []string{"a"}, FPS: 10}}},
		}},
	}
	sets, err := NewSets(input)
	if err != nil {
		t.Fatal(err)
	}
	input["hero"].States["idle"].Tracks[0].Frames[0] = "mutated"
	input["hero"].States["extra"] = State{Name: "extra"}
	got, _ := sets.Get("hero")
	st := got.States["idle"]
	if st.Tracks[0].Frames[0] != "a" {
		t.Fatalf("frames = %v, want deep-copied a", st.Tracks[0].Frames)
	}
	if _, ok := got.States["extra"]; ok {
		t.Fatal("added state leaked into Sets")
	}
}

func TestAnimatorValidateAndFrames(t *testing.T) {
	sets := testSets(t)

	if _, err := NewAnimator("", "idle"); err == nil {
		t.Fatal("NewAnimator accepted an empty set")
	}
	if _, err := NewAnimator("hero", ""); err == nil {
		t.Fatal("NewAnimator accepted an empty state")
	}
	a, err := NewAnimator("hero", "idle")
	if err != nil {
		t.Fatal(err)
	}
	if a.Current() != "" {
		t.Fatalf("Current() = %q before start, want empty", a.Current())
	}
	// Not started: no frames, no error.
	if frames, err := a.Frames(sets); err != nil || frames != nil {
		t.Fatalf("Frames before start = %v, %v; want nil, nil", frames, err)
	}
	// Started manually (in-package): one layer, full weight.
	a.current = "idle"
	a.tracks = []trackState{{frame: 0}}
	frames, err := a.Frames(sets)
	if err != nil || len(frames) != 1 || frames[0].Track != "body" || frames[0].Region != "a" || frames[0].Weight != 1 {
		t.Fatalf("Frames = %+v, %v; want body:a:1", frames, err)
	}
	// Unknown set errors.
	a.Set = "ghost"
	if _, err := a.Frames(sets); err == nil {
		t.Fatal("Frames accepted an unknown set")
	}
}
