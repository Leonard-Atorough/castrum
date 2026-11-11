package audio

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/signal"
)

func TestSourceValidate(t *testing.T) {
	tests := []struct {
		name    string
		source  Source
		wantErr string
	}{
		{
			name:   "minimal sfx valid",
			source: Source{Volume: 1},
		},
		{
			name: "streamed music valid",
			source: Source{
				Volume: 0.8,
				Group:  GroupMusic,
				Loop:   LoopForever,
				Pause:  PauseContinues,
				Load:   LoadStream,
			},
		},
		{
			name:    "zero volume",
			source:  Source{},
			wantErr: "volume",
		},
		{
			name:    "volume above unity",
			source:  Source{Volume: 1.5},
			wantErr: "volume",
		},
		{
			name:    "nan volume",
			source:  Source{Volume: math.NaN()},
			wantErr: "volume",
		},
		{
			name:   "positive multiplier",
			source: Source{Volume: 1, PlaybackMultiplier: 2},
		},
		{
			name:    "negative multiplier",
			source:  Source{Volume: 1, PlaybackMultiplier: -1},
			wantErr: "multiplier",
		},
		{
			name:    "nan multiplier",
			source:  Source{Volume: 1, PlaybackMultiplier: math.NaN()},
			wantErr: "multiplier",
		},
		{
			name:    "infinite multiplier",
			source:  Source{Volume: 1, PlaybackMultiplier: math.Inf(1)},
			wantErr: "multiplier",
		},
		{
			name:    "invalid loop mode",
			source:  Source{Volume: 1, Loop: LoopMode(7)},
			wantErr: "loop mode",
		},
		{
			name:    "invalid pause mode",
			source:  Source{Volume: 1, Pause: PauseMode(7)},
			wantErr: "pause mode",
		},
		{
			name:    "invalid load mode",
			source:  Source{Volume: 1, Load: LoadMode(7)},
			wantErr: "load mode",
		},
		{
			name:    "invalid group",
			source:  Source{Volume: 1, Group: Group(7)},
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

func TestClipValidatedAtSpawn(t *testing.T) {
	w := newWorldWithHub(t)
	if _, err := w.NewEntity(Source{Volume: 1}, Clip{Audio: "sfx/step.wav"}); err != nil {
		t.Fatalf("spawn valid clip: %v", err)
	}
	_, err := w.NewEntity(Source{Volume: 1}, Clip{})
	if err == nil || !strings.Contains(err.Error(), "clip audio") {
		t.Fatalf("spawn invalid clip: err = %v, want validation error", err)
	}
}

func TestSourceBalanceValidation(t *testing.T) {
	for _, tt := range []struct {
		name    string
		balance float64
		valid   bool
	}{
		{"center", 0, true},
		{"full left", -1, true},
		{"full right", 1, true},
		{"partial left", -0.5, true},
		{"partial right", 0.5, true},
		{"below range", math.Nextafter(-1, math.Inf(-1)), false},
		{"above range", math.Nextafter(1, math.Inf(1)), false},
		{"NaN", math.NaN(), false},
		{"negative infinity", math.Inf(-1), false},
		{"positive infinity", math.Inf(1), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := NewSource()
			source.Balance = tt.balance
			err := source.Validate()
			if tt.valid {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "balance") {
				t.Fatalf("Validate() = %v, want balance error", err)
			}
		})
	}
}

func TestSourceBalanceValidatedAtSpawnAndUpdate(t *testing.T) {
	world := newWorldWithHub(t)
	source := NewSource()
	source.Balance = math.NaN()
	if _, err := world.NewEntity(source); err == nil || !strings.Contains(err.Error(), "balance") {
		t.Fatalf("spawn error = %v, want balance error", err)
	}
	source.Balance = 0.5
	entity, err := world.NewEntity(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := entity.Update(world, func(s *Source) { s.Balance = math.NaN() }); err == nil || !strings.Contains(err.Error(), "balance") {
		t.Fatalf("update error = %v, want balance error", err)
	}
	if got, ok := entity.Component[Source](world); !ok || got.Balance != 0.5 {
		t.Fatalf("invalid update changed stored source: %+v", got)
	}
}

func TestSourceRestart(t *testing.T) {
	s := Source{Volume: 1, Balance: -0.5}

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
	if s.Balance != -0.5 {
		t.Fatalf("Restart() changed balance to %v, want -0.5", s.Balance)
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

	if err := m.SetMaster(1.5); err != nil {
		t.Fatal(err)
	}
	if m.Master() != 1 {
		t.Fatalf("Master() after 1.5 = %v, want clamped to 1", m.Master())
	}
	if err := m.SetMaster(-1); err != nil {
		t.Fatal(err)
	}
	if m.Master() != 0 {
		t.Fatalf("Master() after -1 = %v, want clamped to 0", m.Master())
	}
	if err := m.SetMaster(0.25); err != nil {
		t.Fatal(err)
	}
	if m.Master() != 0.25 {
		t.Fatalf("Master() = %v, want 0.25", m.Master())
	}

	if err := m.SetGroupVolume(GroupSFX, 2); err != nil {
		t.Fatal(err)
	}
	if m.GroupVolume(GroupSFX) != 1 {
		t.Fatalf("GroupVolume(sfx) after 2 = %v, want clamped to 1", m.GroupVolume(GroupSFX))
	}
	if err := m.SetGroupVolume(GroupSFX, 0.25); err != nil {
		t.Fatal(err)
	}
	if m.GroupVolume(GroupSFX) != 0.25 {
		t.Fatalf("GroupVolume(sfx) = %v, want 0.25", m.GroupVolume(GroupSFX))
	}
	if m.GroupVolume(GroupMusic) != 1 {
		t.Fatalf("GroupVolume(music) = %v, want 1 (unoverridden group)", m.GroupVolume(GroupMusic))
	}
	if err := m.SetMaster(math.NaN()); err == nil || m.Master() != 0.25 {
		t.Fatalf("NaN master: error=%v volume=%v, want error and unchanged volume", err, m.Master())
	}
	if err := m.SetGroupVolume(GroupSFX, math.NaN()); err == nil || m.GroupVolume(GroupSFX) != 0.25 {
		t.Fatalf("NaN group: error=%v volume=%v, want error and unchanged volume", err, m.GroupVolume(GroupSFX))
	}
	if err := m.SetGroupVolume(GroupMusic, math.NaN()); err == nil || m.GroupVolume(GroupMusic) != 1 {
		t.Fatalf("NaN default group: error=%v volume=%v, want error and unity volume", err, m.GroupVolume(GroupMusic))
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
		mixer  func(*Mixer) error
		want   Playback
	}{
		{
			name:   "defaults: playing at the authored mix",
			source: Source{Volume: 1},
			want:   Playback{Playing: true, Volume: 1, Rate: 1},
		},
		{
			name:   "mix is multiplicative",
			source: Source{Volume: 0.5, Group: GroupSFX},
			mixer: func(m *Mixer) error {
				if err := m.SetMaster(0.5); err != nil {
					return err
				}
				return m.SetGroupVolume(GroupSFX, 0.5)
			},
			want: Playback{Playing: true, Volume: 0.125, Rate: 1},
		},
		{
			name:   "unoverridden group reads unity",
			source: Source{Volume: 1, Group: GroupMusic},
			mixer: func(m *Mixer) error {
				return m.SetGroupVolume(GroupSFX, 0.25)
			},
			want: Playback{Playing: true, Volume: 1, Rate: 1},
		},
		{
			name:   "source pause holds",
			source: Source{Volume: 1, Paused: true},
			want:   Playback{Playing: false, Volume: 1, Rate: 1},
		},
		{
			name:   "mixer pause holds a PauseHolds play",
			source: Source{Volume: 1, Pause: PauseHolds},
			mixer:  func(m *Mixer) error { m.PauseAll(); return nil },
			want:   Playback{Playing: false, Volume: 1, Rate: 1},
		},
		{
			name:   "mixer pause does not hold a PauseContinues play",
			source: Source{Volume: 1, Pause: PauseContinues},
			mixer:  func(m *Mixer) error { m.PauseAll(); return nil },
			want:   Playback{Playing: true, Volume: 1, Rate: 1},
		},
		{
			name:   "source pause overrides PauseContinues",
			source: Source{Volume: 1, Pause: PauseContinues, Paused: true},
			mixer:  func(m *Mixer) error { m.PauseAll(); return nil },
			want:   Playback{Playing: false, Volume: 1, Rate: 1},
		},
		{
			name:   "explicit multiplier passes through",
			source: Source{Volume: 1, PlaybackMultiplier: 2},
			want:   Playback{Playing: true, Volume: 1, Rate: 2},
		},
		{
			name: "loop and load pass through",
			source: Source{
				Volume: 1,
				Group:  GroupMusic, Loop: LoopForever, Load: LoadStream,
			},
			want: Playback{Playing: true, Volume: 1, Rate: 1, Loop: LoopForever, Load: LoadStream},
		},
		{
			name:   "balance is independent of mixed volume",
			source: Source{Volume: 0.5, Balance: -0.5},
			mixer: func(m *Mixer) error {
				if err := m.SetMaster(0.5); err != nil {
					return err
				}
				return m.SetGroupVolume(GroupSFX, 0.5)
			},
			want: Playback{Playing: true, Volume: 0.125, Rate: 1, Balance: -0.5},
		},
		{
			name:   "balance survives source pause",
			source: Source{Volume: 1, Balance: 0.5, Paused: true},
			want:   Playback{Volume: 1, Rate: 1, Balance: 0.5},
		},
		{
			name:   "balance survives mixer pause and mute",
			source: Source{Volume: 1, Balance: -1},
			mixer: func(m *Mixer) error {
				m.PauseAll()
				return m.SetMaster(0)
			},
			want: Playback{Rate: 1, Balance: -1},
		},
		{
			name:   "mute does not pause",
			source: Source{Volume: 1},
			mixer:  func(m *Mixer) error { return m.SetMaster(0) },
			want:   Playback{Playing: true, Rate: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMixer()
			if tt.mixer != nil {
				if err := tt.mixer(m); err != nil {
					t.Fatal(err)
				}
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

func newAudioWorld(t *testing.T, src Source, audioID asset.ID) (*core.Entity, *core.Context) {
	t.Helper()
	w := newWorldWithHub(t)
	entity, err := w.NewEntity(src, Clip{Audio: audioID})
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
	entity, ctx := newAudioWorld(t, Source{Volume: 1}, "sfx/step.wav")
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

func TestPlaylistSystemAdvancesAfterCompletion(t *testing.T) {
	w := newWorldWithHub(t)
	listener := &Listener{Enabled: true, Mode: FixedPosition, Range: 100}
	if err := w.Provide(func(*core.World) (*Listener, error) { return listener, nil }); err != nil {
		t.Fatal(err)
	}
	entity, err := w.NewEntity(Source{Volume: 1, Spatial: true}, core.Transform{Position: geom.Vector2{X: 50}}, NewPlaylist([]asset.ID{"music/one.ogg", "music/two.ogg"}))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &core.Context{World: w}
	fake := &fakeController{live: true}
	sys := NewAudioSystem(fake, NewMixer())

	runAudio(t, sys, ctx)
	if len(fake.syncs) != 1 || fake.syncs[0].source != "music/one.ogg" || !fake.syncs[0].want.Restart {
		t.Fatalf("initial playlist syncs = %+v, want first track with restart", fake.syncs)
	}
	if got := fake.syncs[0].want; got.Volume != 0.5 || got.Balance != 0.5 {
		t.Fatalf("spatial playlist playback = %+v, want volume and balance 0.5", got)
	}

	if err := entity.Update(w, func(tr *core.Transform) { tr.Position.X = 150 }); err != nil {
		t.Fatal(err)
	}
	fake.live = false
	runAudio(t, sys, ctx)
	if got := fake.syncs[len(fake.syncs)-1].want; got.Volume != 0 || !got.Playing {
		t.Fatalf("out-of-range playlist playback = %+v, want silent playback", got)
	}
	playlist, ok := entity.Component[Playlist](w)
	if !ok || playlist.Current() != "music/two.ogg" {
		t.Fatalf("playlist after completion = %+v, want music/two.ogg", playlist)
	}
	if source := readSource(t, entity, ctx); source.Completed {
		t.Fatal("advancing to the next track should clear completion")
	}

	if err := entity.Update(w, func(tr *core.Transform) { tr.Position.X = 50 }); err != nil {
		t.Fatal(err)
	}
	fake.live = true
	runAudio(t, sys, ctx)
	last := fake.syncs[len(fake.syncs)-1]
	if last.source != "music/two.ogg" || !last.want.Restart {
		t.Fatalf("next track sync = %+v, want music/two.ogg with restart", last)
	}
	if last.want.Volume != 0.5 || last.want.Balance != 0.5 {
		t.Fatalf("next track lost spatialization: %+v", last.want)
	}
}

func TestPlaylistSystemRestartsRepeatedAssetSelection(t *testing.T) {
	w := newWorldWithHub(t)
	entity, err := w.NewEntity(Source{Volume: 1}, NewPlaylist([]asset.ID{"music/same.ogg", "music/same.ogg"}))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &core.Context{World: w}
	fake := &fakeController{live: true}
	sys := NewAudioSystem(fake, NewMixer())
	runAudio(t, sys, ctx)

	if err := entity.Update(w, func(p *Playlist) { p.Next() }); err != nil {
		t.Fatal(err)
	}
	runAudio(t, sys, ctx)
	last := fake.syncs[len(fake.syncs)-1]
	if last.source != "music/same.ogg" || !last.want.Restart {
		t.Fatalf("repeated asset sync = %+v, want same asset with restart", last)
	}
}

func TestPlaylistSystemRepeatsAll(t *testing.T) {
	w := newWorldWithHub(t)
	playlist := NewPlaylist([]asset.ID{"music/loop.ogg"})
	playlist.Repeat = RepeatAll
	entity, err := w.NewEntity(Source{Volume: 1}, playlist)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &core.Context{World: w}
	fake := &fakeController{live: true}
	sys := NewAudioSystem(fake, NewMixer())
	runAudio(t, sys, ctx)

	fake.live = false
	runAudio(t, sys, ctx)
	got, ok := entity.Component[Playlist](w)
	if !ok || got.Current() != "music/loop.ogg" {
		t.Fatalf("repeated playlist = %+v, want its only track selected", got)
	}
	if source := readSource(t, entity, ctx); source.Completed {
		t.Fatal("repeat-all should clear completion when wrapping")
	}

	fake.live = true
	runAudio(t, sys, ctx)
	if last := fake.syncs[len(fake.syncs)-1]; !last.want.Restart {
		t.Fatal("repeat-all should restart the wrapped track")
	}
}

func TestEmptyPlaylistValidation(t *testing.T) {
	w := newWorldWithHub(t)
	entity, err := w.NewEntity(Source{Volume: 1}, NewPlaylist(nil))
	if err != nil {
		t.Fatalf("spawn empty playlist: %v", err)
	}
	if err := entity.Update(w, func(p *Playlist) { p.Repeat = RepeatMode(9) }); err == nil {
		t.Fatal("empty playlist accepted unsupported repeat mode")
	}
	if _, err := w.NewEntity(Source{Volume: 1}, NewPlaylist([]asset.ID{""})); err == nil {
		t.Fatal("playlist accepted an empty track ID")
	}
}

func TestAudioSystemRejectsBothSelectors(t *testing.T) {
	w := newWorldWithHub(t)
	if _, err := w.NewEntity(Source{Volume: 1}, Clip{Audio: "music/clip.ogg"}, NewPlaylist([]asset.ID{"music/list.ogg"})); err != nil {
		t.Fatal(err)
	}
	sys := NewAudioSystem(&fakeController{}, NewMixer())
	err := sys.Update(&core.Context{World: w})
	if err == nil || !strings.Contains(err.Error(), "both Clip and Playlist") {
		t.Fatalf("Update() = %v, want ambiguous selector error", err)
	}
}

func TestSystemCompletesAndHolds(t *testing.T) {
	entity, ctx := newAudioWorld(t, Source{Volume: 1}, "sfx/step.wav")
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
	entity, ctx := newAudioWorld(t, Source{Volume: 1}, "sfx/step.wav")
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
	entity, ctx := newAudioWorld(t, Source{Volume: 1, Balance: 0.5}, "sfx/step.wav")
	fake := &fakeController{live: false}
	sys := NewAudioSystem(fake, NewMixer())

	// Complete the play, then retrigger the same source: a real
	// provider reports a fresh restart as live.
	runAudio(t, sys, ctx)
	if s := readSource(t, entity, ctx); !s.Completed {
		t.Fatal("finished play must be completed")
	}
	if err := entity.Update(ctx.World, func(s *Source) { s.Restart() }); err != nil {
		t.Fatal(err)
	}
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
	if s.Balance != 0.5 || last.want.Balance != 0.5 {
		t.Fatalf("restart balance: source=%v playback=%v, want 0.5", s.Balance, last.want.Balance)
	}

	// Complete again, then retarget to another source.
	fake.live = false
	runAudio(t, sys, ctx)
	if s := readSource(t, entity, ctx); !s.Completed {
		t.Fatal("finished play must be completed")
	}
	if err := entity.Update(ctx.World, func(c *Clip) { c.Audio = "sfx/other.wav" }); err != nil {
		t.Fatal(err)
	}
	fake.live = true
	runAudio(t, sys, ctx)
	s = readSource(t, entity, ctx)
	if s.Completed {
		t.Fatal("a Clip change must clear Completed")
	}
	last = fake.syncs[len(fake.syncs)-1]
	if !last.want.Restart || last.source != "sfx/other.wav" {
		t.Fatalf("Clip change must reach the provider as a restart, got %+v", last)
	}
	if s.Balance != 0.5 || last.want.Balance != 0.5 {
		t.Fatalf("audio change balance: source=%v playback=%v, want 0.5", s.Balance, last.want.Balance)
	}
}

func TestSystemSyncErrorPropagates(t *testing.T) {
	_, ctx := newAudioWorld(t, Source{Volume: 1}, "sfx/step.wav")
	fake := &fakeController{live: true, err: errors.New("boom")}
	sys := NewAudioSystem(fake, NewMixer())
	err := sys.Update(ctx)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Update() = %v, want wrapped controller error", err)
	}
}

func TestOneShotPlaysAndIsReclaimed(t *testing.T) {
	w := newWorldWithHub(t)
	if err := w.Provide(func(*core.World) (*Listener, error) {
		return &Listener{Enabled: true, Mode: FixedPosition, Range: 100}, nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx := &core.Context{World: w}
	fake := &fakeController{live: true}
	sys := NewAudioSystem(fake, NewMixer())

	entity, err := OneShotAt(w, Source{Volume: 1}, Clip{Audio: "sfx/step.wav"}, geom.Vector2{X: 150})
	if err != nil {
		t.Fatalf("OneShotAt: %v", err)
	}
	if !entity.IsAlive() {
		t.Fatal("one-shot must be alive at spawn")
	}

	runAudio(t, sys, ctx) // plays: live
	if got := fake.syncs[0].want; !got.Playing || got.Volume != 0 || got.Balance != 1 {
		t.Fatalf("out-of-range one-shot = %+v, want silent playback panned hard right", got)
	}
	if !entity.IsAlive() {
		t.Fatal("a live one-shot must not be destroyed")
	}
	if _, ok := entity.Component[Source](w); !ok {
		t.Fatal("a live one-shot must remain in world storage")
	}

	fake.live = false // the play runs out
	runAudio(t, sys, ctx)
	if entity.IsAlive() {
		t.Fatal("a finished one-shot must be destroyed by the system")
	}
	for e := range core.NewQuery(w).With(Source{}).Execute() {
		if e.ID() == entity.ID() {
			t.Fatal("finished one-shot remains in world storage")
		}
	}
}

func TestOneShotDoesNotTouchUserSources(t *testing.T) {
	w := newWorldWithHub(t)
	ctx := &core.Context{World: w}
	fake := &fakeController{live: false}
	sys := NewAudioSystem(fake, NewMixer())

	user, err := w.NewEntity(Source{Volume: 1}, Clip{Audio: "sfx/step.wav"})
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

func TestOneShotRejectsInvalidSources(t *testing.T) {
	w := newWorldWithHub(t)
	if _, err := OneShot(w, Source{Volume: 1, Loop: LoopForever}, Clip{Audio: "ambience.ogg"}); err == nil {
		t.Fatal("a looping one-shot must be rejected")
	}
	if _, err := OneShot(w, Source{Volume: 1, Spatial: true}, Clip{Audio: "sfx/step.wav"}); err == nil {
		t.Fatal("a spatial one-shot must be rejected")
	}
	if _, err := OneShotAt(w, Source{Volume: 1, Loop: LoopForever}, Clip{Audio: "ambience.ogg"}, geom.Vector2{}); err == nil {
		t.Fatal("a looping positional one-shot must be rejected")
	}
}

func TestNewSourceDefaults(t *testing.T) {
	s := NewSource()
	if s.Volume != 1 || s.Balance != 0 {
		t.Fatalf("NewSource() = %+v, want unity volume and zero balance", s)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil: the constructor result must be a valid play as-is", err)
	}
}

// newWorldWithHub returns a world with the signal hub the audio system
// resolves.
func newWorldWithHub(t *testing.T) *core.World {
	t.Helper()
	w := core.NewWorld()
	if err := w.Provide(func(*core.World) (*signal.Hub, error) {
		return signal.NewHub(), nil
	}); err != nil {
		t.Fatalf("provide signal hub: %v", err)
	}
	return w
}

func TestFinishedSignalOnCompletion(t *testing.T) {
	entity, ctx := newAudioWorld(t, Source{Volume: 1}, "sfx/step.wav")
	fake := &fakeController{live: true}
	sys := NewAudioSystem(fake, NewMixer())
	hub := ctx.World.MustResource[*signal.Hub]()
	finished := hub.NewReader[Finished]()

	runAudio(t, sys, ctx) // plays: live, no completion

	fake.live = false // the play runs out
	runAudio(t, sys, ctx)

	var got []Finished
	for event, ok := finished.Next(); ok; event, ok = finished.Next() {
		got = append(got, event)
	}
	if len(got) != 1 || got[0] != (Finished{Entity: entity.ID(), Audio: "sfx/step.wav"}) {
		t.Fatalf("finished signals = %v, want one for entity %d", got, entity.ID())
	}

	// Completion is a transition: a later update does not emit it again.
	runAudio(t, sys, ctx)
	if event, ok := finished.Next(); ok {
		t.Fatalf("unexpected repeated finish signal: %+v", event)
	}
}
