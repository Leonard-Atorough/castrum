package audio

import (
	"math"
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/signal"
)

type fakeSync struct {
	entity core.EntityID
	source asset.ID
	want   Playback
}

type fakeController struct {
	syncs        []fakeSync
	live         bool
	liveByEntity map[core.EntityID]bool
	err          error
	sweepErr     error
	sweeps       int
}

func (f *fakeController) Sync(id core.EntityID, source asset.ID, want Playback) (bool, error) {
	f.syncs = append(f.syncs, fakeSync{entity: id, source: source, want: want})
	if live, ok := f.liveByEntity[id]; ok {
		return live, f.err
	}
	return f.live, f.err
}

func (f *fakeController) setLive(id core.EntityID, live bool) {
	if f.liveByEntity == nil {
		f.liveByEntity = make(map[core.EntityID]bool)
	}
	f.liveByEntity[id] = live
}

func (f *fakeController) Sweep() error {
	f.sweeps++
	return f.sweepErr
}

type harness struct {
	t          *testing.T
	world      *core.World
	hub        *signal.Hub
	mixer      *Mixer
	listener   *Listener
	controller *fakeController
	ctx        *core.Context
	system     core.SystemFunc
	syncStart  int
}

func newAudioHarness(t *testing.T) *harness {
	t.Helper()
	world := core.NewWorld()
	h := newAudioHarnessWithWorld(t, world)
	listener := &Listener{}
	if err := world.Provide(func(*core.World) (*Listener, error) { return listener, nil }); err != nil {
		t.Fatalf("provide listener: %v", err)
	}
	h.listener = listener
	return h
}

func newAudioHarnessWithWorld(t *testing.T, world *core.World) *harness {
	t.Helper()
	hub := signal.NewHub()
	if err := world.Provide(func(*core.World) (*signal.Hub, error) { return hub, nil }); err != nil {
		t.Fatalf("provide signal hub: %v", err)
	}
	mixer := NewMixer()
	controller := &fakeController{live: true}
	return &harness{
		t:          t,
		world:      world,
		hub:        hub,
		mixer:      mixer,
		controller: controller,
		ctx:        &core.Context{World: world},
		system:     NewAudioSystem(controller, mixer),
	}
}

func (h *harness) spawn(source Source, settings Settings) core.Entity {
	h.t.Helper()
	player, err := NewPlayer(source, settings)
	if err != nil {
		h.t.Fatalf("new player: %v", err)
	}
	entity, err := h.world.NewEntity(player)
	if err != nil {
		h.t.Fatalf("spawn player: %v", err)
	}
	return entity
}

func (h *harness) step() error {
	h.t.Helper()
	h.syncStart = len(h.controller.syncs)
	return h.system(h.ctx)
}

func (h *harness) update() {
	h.t.Helper()
	if err := h.step(); err != nil {
		h.t.Fatalf("audio update: %v", err)
	}
}

func (h *harness) player(entity core.Entity) Player {
	h.t.Helper()
	player, ok := entity.Component[Player](h.world)
	if !ok {
		h.t.Fatal("player component missing")
	}
	return player
}

func (h *harness) lastSync(id core.EntityID) fakeSync {
	h.t.Helper()
	for i := len(h.controller.syncs) - 1; i >= h.syncStart; i-- {
		if call := h.controller.syncs[i]; call.entity == id {
			return call
		}
	}
	h.t.Fatalf("entity %d was not synced in the last update", id)
	return fakeSync{}
}

func TestNewPlayerValidation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		source   Source
		settings Settings
		wantErr  bool
	}{
		{name: "default settings", source: Track("sound"), settings: DefaultSettings()},
		{name: "empty source", settings: DefaultSettings(), wantErr: true},
		{name: "empty track ID", source: Tracks("a", ""), settings: DefaultSettings(), wantErr: true},
		{name: "zero volume", source: Track("sound"), wantErr: true},
		{name: "nan volume", source: Track("sound"), settings: Settings{Volume: math.NaN()}, wantErr: true},
		{name: "infinite rate", source: Track("sound"), settings: Settings{Volume: 1, PlaybackMultiplier: math.Inf(1)}, wantErr: true},
		{name: "invalid loop", source: Track("sound"), settings: Settings{Volume: 1, Loop: LoopMode(99)}, wantErr: true},
		{name: "invalid balance", source: Track("sound"), settings: Settings{Volume: 1, Balance: 2}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPlayer(tc.source, tc.settings)
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewPlayer() error = %v, want error %t", err, tc.wantErr)
			}
		})
	}
}

func TestPlayerSelection(t *testing.T) {
	player, err := NewPlayer(Tracks("a", "b"), DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	if player.Current() != "a" || player.Index() != 0 {
		t.Fatalf("initial selection = %q at %d", player.Current(), player.Index())
	}
	player.Prev()
	if player.Current() != "b" || player.Index() != 1 {
		t.Fatalf("previous selection = %q at %d", player.Current(), player.Index())
	}
	player.Next()
	if player.Current() != "a" || player.Index() != 0 {
		t.Fatalf("next selection = %q at %d", player.Current(), player.Index())
	}
	if err := player.Seek(1); err != nil {
		t.Fatal(err)
	}
	if err := player.Seek(2); err == nil || player.Current() != "b" || player.Index() != 1 {
		t.Fatalf("invalid seek: selection = %q at %d, error = %v", player.Current(), player.Index(), err)
	}
	if err := player.SetSource(Source{}); err == nil || player.Current() != "b" {
		t.Fatalf("invalid source replacement: selection = %q, error = %v", player.Current(), err)
	}
	if err := player.SetSource(Track("c")); err != nil {
		t.Fatal(err)
	}
	if player.Current() != "c" || player.Index() != 0 {
		t.Fatalf("replacement selection = %q at %d", player.Current(), player.Index())
	}
}

func TestPlayerTrackEdits(t *testing.T) {
	source := Tracks("a", "b", "c")
	player, err := NewPlayer(source, DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	other := player
	if err := player.Seek(1); err != nil {
		t.Fatal(err)
	}
	if err := player.Append(""); err == nil {
		t.Fatal("append accepted empty track")
	}
	if err := player.Insert(0, ""); err == nil {
		t.Fatal("insert accepted empty track")
	}
	if player.Source().Len() != 3 || player.Current() != "b" || player.Index() != 1 {
		t.Fatalf("invalid track edits changed selection or source: %+v", player)
	}
	if err := player.Insert(0, "x"); err != nil {
		t.Fatal(err)
	}
	if player.Current() != "b" || player.Index() != 2 {
		t.Fatalf("insert before selection = %q at %d", player.Current(), player.Index())
	}
	if err := player.Remove(0); err != nil {
		t.Fatal(err)
	}
	if player.Current() != "b" || player.Index() != 1 {
		t.Fatalf("remove before selection = %q at %d", player.Current(), player.Index())
	}
	if err := player.Remove(1); err != nil {
		t.Fatal(err)
	}
	if player.Current() != "c" || player.Index() != 1 {
		t.Fatalf("remove current selection = %q at %d", player.Current(), player.Index())
	}
	if err := player.Remove(1); err != nil {
		t.Fatal(err)
	}
	if player.Current() != "a" || player.Index() != 0 {
		t.Fatalf("remove last selection = %q at %d", player.Current(), player.Index())
	}
	if err := player.Remove(0); err == nil || player.Source().Len() != 1 {
		t.Fatalf("remove final track: length = %d, error = %v", player.Source().Len(), err)
	}
	if source.Len() != 3 || source.At(0) != "a" || source.At(1) != "b" || source.At(2) != "c" ||
		other.Source().At(0) != "a" || other.Source().At(1) != "b" || other.Source().At(2) != "c" {
		t.Fatal("editing player changed another source or player")
	}
}

func TestPlayerAppendDoesNotChangeOtherPlayers(t *testing.T) {
	tracks := make([]asset.ID, 2, 8)
	copy(tracks, []asset.ID{"a", "b"})
	first, err := NewPlayer(Source{tracks: tracks}, DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	second := first
	if err := first.Append("first"); err != nil {
		t.Fatal(err)
	}
	if err := second.Append("second"); err != nil {
		t.Fatal(err)
	}
	if first.Source().At(2) != "first" || second.Source().At(2) != "second" {
		t.Fatal("appending to one player changed another player's source")
	}
}

func TestOneShotValidation(t *testing.T) {
	for _, mode := range []LoopMode{LoopTrack, LoopAll} {
		settings := DefaultSettings()
		settings.Loop = mode
		if _, err := OneShot(core.NewWorld(), Track("sound"), settings); err == nil {
			t.Errorf("OneShot accepted loop mode %v", mode)
		}
	}
	world := core.NewWorld()
	settings := Settings{Volume: 1, Spatial: true}
	if entity, err := OneShot(world, Track("sound"), settings); err == nil || entity != (core.Entity{}) {
		t.Fatalf("OneShot() = %v, %v; want spatial error", entity, err)
	}
	pos := geom.Vector2{X: 25, Y: 40}
	entity, err := OneShotAt(world, Track("sound"), settings, pos)
	if err != nil {
		t.Fatal(err)
	}
	transform, ok := entity.Component[core.Transform](world)
	if !ok || transform.Position != pos {
		t.Fatalf("transform = %+v, exists = %t", transform, ok)
	}
}
