package audio

import (
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
)

func TestSourceValidate(t *testing.T) {
	tests := []struct {
		name    string
		source  Source
		wantErr string
	}{
		{
			name:   "minimal sfx valid",
			source: Source{Audio: "sfx/step.wav", Volume: 1},
		},
		{
			name: "streamed music valid",
			source: Source{
				Audio:  "music/theme.ogg",
				Volume: 0.8,
				Group:  GroupMusic,
				Loop:   LoopForever,
				Pause:  PauseContinues,
				Load:   LoadStream,
			},
		},
		{
			name:    "empty audio",
			source:  Source{Volume: 1},
			wantErr: "audio ID must not be empty",
		},
		{
			name:    "zero volume",
			source:  Source{Audio: "sfx/step.wav"},
			wantErr: "volume",
		},
		{
			name:    "volume above unity",
			source:  Source{Audio: "sfx/step.wav", Volume: 1.5},
			wantErr: "volume",
		},
		{
			name:   "zero multiplier is the normal rate",
			source: Source{Audio: "sfx/step.wav", Volume: 1, PlaybackMultiplier: 2},
		},
		{
			name:    "negative multiplier",
			source:  Source{Audio: "sfx/step.wav", Volume: 1, PlaybackMultiplier: -1},
			wantErr: "multiplier",
		},
		{
			name:    "invalid loop mode",
			source:  Source{Audio: "sfx/step.wav", Volume: 1, Loop: LoopMode(7)},
			wantErr: "loop mode",
		},
		{
			name:    "invalid pause mode",
			source:  Source{Audio: "sfx/step.wav", Volume: 1, Pause: PauseMode(7)},
			wantErr: "pause mode",
		},
		{
			name:    "invalid load mode",
			source:  Source{Audio: "sfx/step.wav", Volume: 1, Load: LoadMode(7)},
			wantErr: "load mode",
		},
		{
			name:    "invalid group",
			source:  Source{Audio: "sfx/step.wav", Volume: 1, Group: Group(7)},
			wantErr: "group",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.source.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

// The Validatable hook runs at the storage boundary: an invalid
// Source must fail the spawn itself, not surface later at playback.
func TestSourceValidatedAtSpawn(t *testing.T) {
	w := core.NewWorld()
	if _, err := w.NewEntity(Source{Audio: "sfx/step.wav", Volume: 1}); err != nil {
		t.Fatalf("spawn valid source: %v", err)
	}
	_, err := w.NewEntity(Source{Audio: "sfx/step.wav"})
	if err == nil || !strings.Contains(err.Error(), "volume") {
		t.Fatalf("spawn invalid source: err = %v, want validation error", err)
	}
}

func TestSourceRestart(t *testing.T) {
	s := Source{Audio: "sfx/step.wav", Volume: 1}

	// Pausing and resuming are plain Paused writes.
	s.Paused = true
	if !s.Paused {
		t.Fatal("Paused write did not hold")
	}

	// A finished play: Restart clears Completed, resumes, and bumps
	// the retrigger edge the reconcile system diffs.
	s.Completed = true
	s.Paused = true
	before := s.restarts
	s.Restart()
	if s.Completed {
		t.Fatal("Restart() left Completed set")
	}
	if s.Paused {
		t.Fatal("Restart() left Paused set")
	}
	if s.restarts != before+1 {
		t.Fatalf("restarts = %d, want %d", s.restarts, before+1)
	}
	s.Restart()
	if s.restarts != before+2 {
		t.Fatalf("second Restart: restarts = %d, want %d", s.restarts, before+2)
	}
}

func TestMixerDefaultsClampingAndPause(t *testing.T) {
	m := NewMixer()
	if m.Master() != 1 {
		t.Fatalf("Master() = %v, want 1 (unity default)", m.Master())
	}
	if m.GroupVolume(GroupMusic) != 1 {
		t.Fatalf("GroupVolume(music) = %v, want 1 (unity default)", m.GroupVolume(GroupMusic))
	}

	m.SetMaster(1.5)
	if m.Master() != 1 {
		t.Fatalf("Master() after 1.5 = %v, want clamped to 1", m.Master())
	}
	m.SetMaster(-1)
	if m.Master() != 0 {
		t.Fatalf("Master() after -1 = %v, want clamped to 0", m.Master())
	}
	m.SetMaster(0.25)
	if m.Master() != 0.25 {
		t.Fatalf("Master() = %v, want 0.25", m.Master())
	}

	m.SetGroupVolume(GroupSFX, 2)
	if m.GroupVolume(GroupSFX) != 1 {
		t.Fatalf("GroupVolume(sfx) after 2 = %v, want clamped to 1", m.GroupVolume(GroupSFX))
	}
	m.SetGroupVolume(GroupSFX, 0.25)
	if m.GroupVolume(GroupSFX) != 0.25 {
		t.Fatalf("GroupVolume(sfx) = %v, want 0.25", m.GroupVolume(GroupSFX))
	}
	if m.GroupVolume(GroupMusic) != 1 {
		t.Fatalf("GroupVolume(music) = %v, want 1 (unoverridden group)", m.GroupVolume(GroupMusic))
	}

	if m.Paused() {
		t.Fatal("Paused() = true, want false at construction")
	}
	m.PauseAll()
	if !m.Paused() {
		t.Fatal("Paused() after PauseAll = false, want true")
	}
	m.ResumeAll()
	if m.Paused() {
		t.Fatal("Paused() after ResumeAll = true, want false")
	}
}
