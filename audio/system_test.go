package audio

import (
	"errors"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
)

func TestSystemInitialSyncAndSelection(t *testing.T) {
	h := newAudioHarness(t)
	entity := h.spawn(Tracks("same", "same"), DefaultSettings())
	if err := entity.Update(h.world, func(p *Player) { p.Paused = true }); err != nil {
		t.Fatal(err)
	}
	h.update()
	if got := h.lastSync(entity.ID()); got.source != "same" || !got.want.Restart || got.want.Playing {
		t.Fatalf("initial paused sync = %+v", got)
	}
	h.update()
	if got := h.lastSync(entity.ID()); got.want.Restart || got.want.Playing {
		t.Fatalf("unchanged paused sync = %+v", got)
	}
	if err := entity.Update(h.world, func(p *Player) { p.Next() }); err != nil {
		t.Fatal(err)
	}
	h.update()
	if got := h.lastSync(entity.ID()); !got.want.Restart || got.want.Playing {
		t.Fatalf("same-asset selection = %+v", got)
	}
	h.update()
	if got := h.lastSync(entity.ID()); got.want.Restart {
		t.Fatalf("selection restarted twice: %+v", got)
	}
}

func TestSystemTrackEditsPreserveOrRestartSelection(t *testing.T) {
	h := newAudioHarness(t)
	entity := h.spawn(Tracks("same", "same"), DefaultSettings())
	h.update()
	var editErr error
	if err := entity.Update(h.world, func(p *Player) { editErr = p.Insert(0, "before") }); err != nil {
		t.Fatal(err)
	}
	if editErr != nil {
		t.Fatal(editErr)
	}
	h.update()
	if got := h.lastSync(entity.ID()); got.source != "same" || got.want.Restart {
		t.Fatalf("insert before selection = %+v", got)
	}
	if err := entity.Update(h.world, func(p *Player) { editErr = p.Remove(p.Index()) }); err != nil {
		t.Fatal(err)
	}
	if editErr != nil {
		t.Fatal(editErr)
	}
	h.update()
	if got := h.lastSync(entity.ID()); got.source != "same" || !got.want.Restart {
		t.Fatalf("remove selected duplicate = %+v", got)
	}
}

func TestSystemMixAndPause(t *testing.T) {
	h := newAudioHarness(t)
	settings := Settings{Volume: 0.8, Group: GroupMusic, Balance: 0.25}
	held := h.spawn(Track("held"), settings)
	settings.Pause = PauseContinues
	continues := h.spawn(Track("continues"), settings)
	if err := h.mixer.SetMaster(0.5); err != nil {
		t.Fatal(err)
	}
	if err := h.mixer.SetGroupVolume(GroupMusic, 0.25); err != nil {
		t.Fatal(err)
	}
	h.mixer.PauseAll()
	h.update()
	for _, tc := range []struct {
		entity  *core.Entity
		playing bool
	}{
		{held, false},
		{continues, true},
	} {
		got := h.lastSync(tc.entity.ID()).want
		if got.Playing != tc.playing || got.Volume != 0.1 || got.Rate != 1 || got.Balance != 0.25 {
			t.Errorf("playback for %d = %+v", tc.entity.ID(), got)
		}
	}
}

func TestSystemCompletionAndRestart(t *testing.T) {
	h := newAudioHarness(t)
	entity := h.spawn(Track("sound"), DefaultSettings())
	reader := h.hub.NewReader[Finished]()
	h.update()
	h.controller.setLive(entity.ID(), false)
	h.update()
	if !h.player(entity).Completed() {
		t.Fatal("finished player was not marked completed")
	}
	if got, ok := reader.Next(); !ok || got != (Finished{Entity: entity.ID(), Audio: "sound"}) {
		t.Fatalf("completion signal = %+v, %v", got, ok)
	}
	if _, ok := reader.Next(); ok {
		t.Fatal("unexpected second completion signal")
	}
	h.update()
	if _, ok := reader.Next(); ok {
		t.Fatal("completed player emitted another signal")
	}
	if err := entity.Update(h.world, func(p *Player) { p.Restart() }); err != nil {
		t.Fatal(err)
	}
	h.controller.setLive(entity.ID(), true)
	h.update()
	if got := h.lastSync(entity.ID()).want; !got.Restart || !got.Playing || h.player(entity).Completed() {
		t.Fatalf("restart playback = %+v, completed = %t", got, h.player(entity).Completed())
	}
}

func TestSystemSequenceModes(t *testing.T) {
	for _, mode := range []LoopMode{LoopNone, LoopAll} {
		t.Run(map[LoopMode]string{LoopNone: "once", LoopAll: "all"}[mode], func(t *testing.T) {
			h := newAudioHarness(t)
			settings := DefaultSettings()
			settings.Loop = mode
			entity := h.spawn(Tracks("a", "b"), settings)
			reader := h.hub.NewReader[Finished]()
			h.update()
			if got := h.lastSync(entity.ID()); got.source != "a" || got.want.Loop != LoopNone {
				t.Fatalf("first track = %+v", got)
			}
			h.controller.setLive(entity.ID(), false)
			h.update()
			if got, ok := reader.Next(); !ok || got != (Finished{Entity: entity.ID(), Audio: "a"}) {
				t.Fatalf("first completion = %+v, %v", got, ok)
			}
			if _, ok := reader.Next(); ok {
				t.Fatal("unexpected extra completion")
			}
			if player := h.player(entity); player.Index() != 1 || player.Completed() {
				t.Fatalf("after first track = %+v", player)
			}
			h.controller.setLive(entity.ID(), true)
			h.update()
			if got := h.lastSync(entity.ID()); got.source != "b" || !got.want.Restart || got.want.Loop != LoopNone {
				t.Fatalf("second track = %+v", got)
			}
			h.controller.setLive(entity.ID(), false)
			h.update()
			if got, ok := reader.Next(); !ok || got != (Finished{Entity: entity.ID(), Audio: "b"}) {
				t.Fatalf("second completion = %+v, %v", got, ok)
			}
			player := h.player(entity)
			if mode == LoopNone {
				if !player.Completed() || player.Index() != 1 {
					t.Fatalf("exhausted sequence = %+v", player)
				}
			} else {
				if player.Completed() || player.Index() != 0 {
					t.Fatalf("wrapped sequence = %+v", player)
				}
				h.controller.setLive(entity.ID(), true)
				h.update()
				if got := h.lastSync(entity.ID()); got.source != "a" || !got.want.Restart {
					t.Fatalf("wrapped track = %+v", got)
				}
			}
		})
	}

	t.Run("track", func(t *testing.T) {
		h := newAudioHarness(t)
		settings := DefaultSettings()
		settings.Loop = LoopTrack
		entity := h.spawn(Tracks("a", "b"), settings)
		reader := h.hub.NewReader[Finished]()
		h.update()
		h.update()
		if got := h.lastSync(entity.ID()); got.source != "a" || got.want.Loop != LoopTrack || got.want.Restart {
			t.Fatalf("looping track = %+v", got)
		}
		if h.player(entity).Index() != 0 {
			t.Fatal("looping track advanced")
		}
		if _, ok := reader.Next(); ok {
			t.Fatal("looping track emitted completion")
		}
	})
}

func TestSystemOneShotLifecycle(t *testing.T) {
	h := newAudioHarness(t)
	shot, err := OneShot(h.world, Tracks("a", "b"), DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	persistent := h.spawn(Track("a"), DefaultSettings())
	h.update()
	h.controller.setLive(shot.ID(), false)
	h.update()
	if !shot.IsAlive() || h.player(shot).Index() != 1 {
		t.Fatal("one-shot did not advance to its second track")
	}
	h.controller.setLive(shot.ID(), true)
	h.update()
	h.controller.setLive(shot.ID(), false)
	h.update()
	if shot.IsAlive() {
		t.Fatal("finished one-shot was not destroyed")
	}
	if !persistent.IsAlive() {
		t.Fatal("completion destroyed another player")
	}
}

func TestSystemRetriesFailedSync(t *testing.T) {
	h := newAudioHarness(t)
	entity := h.spawn(Track("sound"), DefaultSettings())
	h.update()
	if err := entity.Update(h.world, func(p *Player) { p.Restart() }); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("sync failed")
	h.controller.err = wantErr
	if err := h.step(); !errors.Is(err, wantErr) {
		t.Fatalf("Update() = %v, want %v", err, wantErr)
	}
	if got := h.lastSync(entity.ID()).want; !got.Restart {
		t.Fatalf("failed sync did not request restart: %+v", got)
	}
	h.controller.err = nil
	h.update()
	if got := h.lastSync(entity.ID()).want; !got.Restart {
		t.Fatalf("retry lost restart: %+v", got)
	}
	h.update()
	if got := h.lastSync(entity.ID()).want; got.Restart {
		t.Fatalf("successful restart repeated: %+v", got)
	}
}
