package audio

import (
	"errors"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
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

	if paused := m.TogglePause(); !paused {
		t.Fatal("TogglePause() = false, want true (the new state)")
	}
	if !m.Paused() {
		t.Fatal("Paused() after TogglePause = false, want true")
	}
	if paused := m.TogglePause(); paused {
		t.Fatal("TogglePause() = true, want false (the new state)")
	}
	if m.Paused() {
		t.Fatal("Paused() after second TogglePause = true, want false")
	}
}

func TestFold(t *testing.T) {
	tests := []struct {
		name   string
		source Source
		mixer  func(*Mixer)
		want   Playback
	}{
		{
			name:   "defaults: playing at the authored mix",
			source: Source{Audio: "sfx/step.wav", Volume: 1},
			want:   Playback{Playing: true, Volume: 1, Rate: 1},
		},
		{
			name:   "mix is multiplicative",
			source: Source{Audio: "sfx/step.wav", Volume: 0.5, Group: GroupSFX},
			mixer: func(m *Mixer) {
				m.SetMaster(0.5)
				m.SetGroupVolume(GroupSFX, 0.5)
			},
			want: Playback{Playing: true, Volume: 0.125, Rate: 1},
		},
		{
			name:   "unoverridden group reads unity",
			source: Source{Audio: "sfx/step.wav", Volume: 1, Group: GroupMusic},
			mixer: func(m *Mixer) {
				m.SetGroupVolume(GroupSFX, 0.25)
			},
			want: Playback{Playing: true, Volume: 1, Rate: 1},
		},
		{
			name:   "source pause holds",
			source: Source{Audio: "sfx/step.wav", Volume: 1, Paused: true},
			want:   Playback{Playing: false, Volume: 1, Rate: 1},
		},
		{
			name:   "mixer pause holds a PauseHolds play",
			source: Source{Audio: "sfx/step.wav", Volume: 1, Pause: PauseHolds},
			mixer:  func(m *Mixer) { m.PauseAll() },
			want:   Playback{Playing: false, Volume: 1, Rate: 1},
		},
		{
			name:   "mixer pause does not hold a PauseContinues play",
			source: Source{Audio: "ui/click.wav", Volume: 1, Pause: PauseContinues},
			mixer:  func(m *Mixer) { m.PauseAll() },
			want:   Playback{Playing: true, Volume: 1, Rate: 1},
		},
		{
			name:   "authored multiplier zero resolves to the normal rate",
			source: Source{Audio: "sfx/step.wav", Volume: 1},
			want:   Playback{Playing: true, Volume: 1, Rate: 1},
		},
		{
			name:   "explicit multiplier passes through",
			source: Source{Audio: "sfx/step.wav", Volume: 1, PlaybackMultiplier: 2},
			want:   Playback{Playing: true, Volume: 1, Rate: 2},
		},
		{
			name: "loop and load pass through",
			source: Source{
				Audio: "music/theme.ogg", Volume: 1,
				Group: GroupMusic, Loop: LoopForever, Load: LoadStream,
			},
			want: Playback{Playing: true, Volume: 1, Rate: 1, Loop: LoopForever, Load: LoadStream},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMixer()
			if tt.mixer != nil {
				tt.mixer(m)
			}
			got := fold(tt.source, m)
			if got != tt.want {
				t.Fatalf("fold() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

type fakeSync struct {
	entity core.EntityID
	source asset.ID
	want   Playback
}

type fakeController struct {
	syncs  []fakeSync
	live   bool // what Sync reports; toggled by tests between runs
	err    error
	sweeps int
}

func (f *fakeController) Sync(id core.EntityID, source asset.ID, want Playback) (bool, error) {
	f.syncs = append(f.syncs, fakeSync{entity: id, source: source, want: want})
	return f.live, f.err
}

func (f *fakeController) Sweep() error {
	f.sweeps++
	return nil
}

func newAudioWorld(t *testing.T, src Source) (*core.Entity, *core.Context) {
	t.Helper()
	w := core.NewWorld()
	entity, err := w.NewEntity(src)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	return entity, &core.Context{World: w}
}

func runAudio(t *testing.T, sys core.System, ctx *core.Context) {
	t.Helper()
	if err := sys.Update(ctx); err != nil {
		t.Fatalf("update: %v", err)
	}
}

func readSource(t *testing.T, e *core.Entity, ctx *core.Context) Source {
	t.Helper()
	s, ok := e.Component[Source](ctx.World)
	if !ok {
		t.Fatal("source component missing")
	}
	return s
}

func TestSystemFoldsAndFlagsRestartOnFirstSync(t *testing.T) {
	entity, ctx := newAudioWorld(t, Source{Audio: "sfx/step.wav", Volume: 1})
	fake := &fakeController{live: true}
	sys := NewAudioSystem(fake, NewMixer())

	runAudio(t, sys, ctx)

	if len(fake.syncs) != 1 {
		t.Fatalf("syncs = %d, want 1", len(fake.syncs))
	}
	call := fake.syncs[0]
	if call.source != "sfx/step.wav" {
		t.Fatalf("source = %q, want sfx/step.wav", call.source)
	}
	if !call.want.Restart {
		t.Fatal("first sync must be a restart: the provider creates its player")
	}
	if !call.want.Playing {
		t.Fatal("folded Playing = false, want true")
	}
	if fake.sweeps != 1 {
		t.Fatalf("sweeps = %d, want 1 per run", fake.sweeps)
	}
	if s := readSource(t, entity, ctx); s.Completed {
		t.Fatal("live play must not be completed")
	}
}

func TestSystemCompletesAndHolds(t *testing.T) {
	entity, ctx := newAudioWorld(t, Source{Audio: "sfx/step.wav", Volume: 1})
	fake := &fakeController{live: true}
	sys := NewAudioSystem(fake, NewMixer())

	runAudio(t, sys, ctx) // create
	fake.live = false
	runAudio(t, sys, ctx) // finish
	if s := readSource(t, entity, ctx); !s.Completed {
		t.Fatal("finished LoopNone play must be completed")
	}

	// Steady state: a held completed play syncs (no restart, still
	// playing per the fold) and writes nothing.
	n := len(fake.syncs)
	runAudio(t, sys, ctx)
	if s := readSource(t, entity, ctx); !s.Completed {
		t.Fatal("held play lost its Completed")
	}
	if got := fake.syncs[n]; got.want.Restart {
		t.Fatal("unchanged play must not restart")
	}
	if got := fake.syncs[n]; got.want.Playing != true {
		t.Fatal("the fold of a completed play still wants playing; the provider holds it")
	}
}

func TestSystemManualCompletedClearIsCorrected(t *testing.T) {
	entity, ctx := newAudioWorld(t, Source{Audio: "sfx/step.wav", Volume: 1})
	fake := &fakeController{live: false}
	sys := NewAudioSystem(fake, NewMixer())

	runAudio(t, sys, ctx)
	if s := readSource(t, entity, ctx); !s.Completed {
		t.Fatal("finished play must be completed")
	}
	if err := entity.Update(ctx.World, func(s *Source) { s.Completed = false }); err != nil {
		t.Fatalf("manual clear: %v", err)
	}
	runAudio(t, sys, ctx) // a manual clear is a no-op by contract
	if s := readSource(t, entity, ctx); !s.Completed {
		t.Fatal("manual Completed clear must be re-set by the system")
	}
}

func TestSystemRestartEdgesClearCompleted(t *testing.T) {
	entity, ctx := newAudioWorld(t, Source{Audio: "sfx/step.wav", Volume: 1})
	fake := &fakeController{live: false}
	sys := NewAudioSystem(fake, NewMixer())

	// Complete the play, then retrigger the same source: a real
	// provider reports a fresh restart as live.
	runAudio(t, sys, ctx)
	if s := readSource(t, entity, ctx); !s.Completed {
		t.Fatal("finished play must be completed")
	}
	entity.Update(ctx.World, func(s *Source) { s.Restart() })
	fake.live = true
	runAudio(t, sys, ctx)
	s := readSource(t, entity, ctx)
	if s.Completed {
		t.Fatal("a restarts bump must clear Completed")
	}
	last := fake.syncs[len(fake.syncs)-1]
	if !last.want.Restart {
		t.Fatal("a restarts bump must reach the provider as a restart")
	}

	// Complete again, then retarget to another source.
	fake.live = false
	runAudio(t, sys, ctx)
	if s := readSource(t, entity, ctx); !s.Completed {
		t.Fatal("finished play must be completed")
	}
	entity.Update(ctx.World, func(s *Source) { s.Audio = "sfx/other.wav" })
	fake.live = true
	runAudio(t, sys, ctx)
	s = readSource(t, entity, ctx)
	if s.Completed {
		t.Fatal("an Audio change must clear Completed")
	}
	last = fake.syncs[len(fake.syncs)-1]
	if !last.want.Restart || last.source != "sfx/other.wav" {
		t.Fatalf("Audio change must reach the provider as a restart, got %+v", last)
	}
}

func TestSystemSyncErrorPropagates(t *testing.T) {
	_, ctx := newAudioWorld(t, Source{Audio: "sfx/step.wav", Volume: 1})
	fake := &fakeController{live: true, err: errors.New("boom")}
	sys := NewAudioSystem(fake, NewMixer())
	err := sys.Update(ctx)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Update() = %v, want wrapped controller error", err)
	}
}

func TestOneShotPlaysAndIsReclaimed(t *testing.T) {
	w := core.NewWorld()
	ctx := &core.Context{World: w}
	fake := &fakeController{live: true}
	sys := NewAudioSystem(fake, NewMixer())

	entity, err := OneShot(w, Source{Audio: "sfx/step.wav", Volume: 1})
	if err != nil {
		t.Fatalf("OneShot: %v", err)
	}
	if !entity.IsAlive() {
		t.Fatal("one-shot must be alive at spawn")
	}

	runAudio(t, sys, ctx) // plays: live
	if !entity.IsAlive() {
		t.Fatal("a live one-shot must not be destroyed")
	}

	fake.live = false // the play runs out
	runAudio(t, sys, ctx)
	if entity.IsAlive() {
		t.Fatal("a finished one-shot must be destroyed by the system")
	}
}

func TestOneShotDoesNotTouchUserSources(t *testing.T) {
	w := core.NewWorld()
	ctx := &core.Context{World: w}
	fake := &fakeController{live: false}
	sys := NewAudioSystem(fake, NewMixer())

	user, err := w.NewEntity(Source{Audio: "sfx/step.wav", Volume: 1})
	if err != nil {
		t.Fatalf("spawn user source: %v", err)
	}
	runAudio(t, sys, ctx)
	if s := readSource(t, user, ctx); !s.Completed {
		t.Fatal("user play must complete")
	}
	if !user.IsAlive() {
		t.Fatal("the system must never destroy a user entity")
	}
}

func TestOneShotRejectsLoopForever(t *testing.T) {
	w := core.NewWorld()
	if _, err := OneShot(w, Source{
		Audio: "ambience.ogg", Volume: 1, Loop: LoopForever,
	}); err == nil {
		t.Fatal("a looping one-shot must be rejected")
	}
}

func TestNewSourceDefaults(t *testing.T) {
	s := NewSource("sfx/step.wav")
	if s.Audio != "sfx/step.wav" || s.Volume != 1 {
		t.Fatalf("NewSource() = %+v, want the audio file at unity volume", s)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil: the constructor result must be a valid play as-is", err)
	}
}
