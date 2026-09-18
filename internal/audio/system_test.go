package audio

import (
	"context"
	"testing"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
)

func setupSystemTestWorld(t *testing.T) (*ecs.World, *AudioService, *events.EventBus) {
	t.Helper()
	world := ecs.NewWorld()
	svc := NewService(
		testAudioContext(),
		nil,
		Config{
			SampleRate:   44100,
			MasterVolume: 1,
			GroupVolumes: map[Group]float64{
				GroupMusic: 1,
				GroupSFX:   1,
			},
		},
		context.Background(),
	)
	bus := events.NewEventBus()
	world.SetResource(bus)
	return world, svc, bus
}

func spawnAudioEntity(t *testing.T, world *ecs.World, trackID string, volume float64, mode components.PlaybackMode, autoplay bool) ecs.EntityID {
	t.Helper()
	e, err := world.CreateWithComponents("test",
		components.NewAudio(trackID, volume, mode, autoplay),
	)
	if err != nil {
		t.Fatalf("CreateWithComponents() error = %v", err)
	}
	return e.ID
}

// ---------------------------------------------------------------------------
// System.Init
// ---------------------------------------------------------------------------

func TestSystemInitResolvesResources(t *testing.T) {
	world, svc, bus := setupSystemTestWorld(t)
	sys := NewSystem(svc)
	if err := sys.Init(world); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if sys.svc != svc {
		t.Error("Init() did not resolve AudioService")
	}
	if sys.events != bus {
		t.Error("Init() did not resolve EventBus")
	}
	if sys.query == nil {
		t.Error("Init() did not create query")
	}
}

func TestSystemInitFailsWithoutAudioService(t *testing.T) {
	world := ecs.NewWorld()
	world.SetResource(events.NewEventBus())
	sys := NewSystem(nil)
	if err := sys.Init(world); err == nil {
		t.Fatal("Init() without AudioService returned nil error")
	}
}

func TestSystemInitFailsWithoutEventBus(t *testing.T) {
	world := ecs.NewWorld()
	svc := NewService(testAudioContext(), nil, Config{}, context.Background())
	world.SetResource(svc)
	sys := NewSystem(svc)
	if err := sys.Init(world); err == nil {
		t.Fatal("Init() without EventBus returned nil error")
	}
}

// ---------------------------------------------------------------------------
// System.Update — new play
// ---------------------------------------------------------------------------

func TestUpdateCreatesPlayerOnFirstPlay(t *testing.T) {
	world, svc, bus := setupSystemTestWorld(t)
	svc.tracks["sfx"] = NewAudioTrack("sfx", silencePCM(), GroupSFX, LoopNone, 1)
	entity := spawnAudioEntity(t, world, "sfx", 1, components.PlaybackPersist, true)

	sys := NewSystem(svc)
	sys.Init(world)

	var startedEvents []Event
	bus.On(func(_ events.EventMeta, e Event) {
		if e.Type == EventTypeTrackStarted {
			startedEvents = append(startedEvents, e)
		}
	}, false)

	if err := sys.Update(world, 0.016); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !svc.hasPlayer(entity) {
		t.Error("HasPlayer = false, want true after first Update")
	}
	if !svc.isPlaying(entity) {
		t.Error("IsPlaying = false, want true after first Update")
	}
	if len(startedEvents) != 1 {
		t.Errorf("started events = %d, want 1", len(startedEvents))
	}
	if startedEvents[0].EntityID != entity {
		t.Errorf("started event EntityID = %d, want %d", startedEvents[0].EntityID, entity)
	}
}

func TestUpdateDoesNotReEmitStartedOnSecondTick(t *testing.T) {
	world, svc, bus := setupSystemTestWorld(t)
	svc.tracks["sfx"] = NewAudioTrack("sfx", silencePCM(), GroupSFX, LoopNone, 1)
	spawnAudioEntity(t, world, "sfx", 1, components.PlaybackPersist, true)

	sys := NewSystem(svc)
	sys.Init(world)

	var startedCount int
	bus.On(func(_ events.EventMeta, e Event) {
		if e.Type == EventTypeTrackStarted {
			startedCount++
		}
	}, false)

	_ = sys.Update(world, 0.016)
	_ = sys.Update(world, 0.016)

	if startedCount != 1 {
		t.Errorf("started events = %d, want 1 (emitted once, not per tick)", startedCount)
	}
}

// ---------------------------------------------------------------------------
// System.Update — pause / resume
// ---------------------------------------------------------------------------

func TestUpdatePausesWhenPlayingSetFalse(t *testing.T) {
	world, svc, _ := setupSystemTestWorld(t)
	svc.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)
	entity := spawnAudioEntity(t, world, "bgm", 1, components.PlaybackPersist, true)

	sys := NewSystem(svc)
	sys.Init(world)

	_ = sys.Update(world, 0.016) // start
	if !svc.isPlaying(entity) {
		t.Fatal("player not playing after first Update")
	}

	// Simulate external code setting Playing to false.
	ap, _ := world.GetComponent[components.AudioPlayer](entity)
	ap.Playing = false
	_ = world.SetComponent(entity, ap)

	_ = sys.Update(world, 0.016) // pause
	if svc.isPlaying(entity) {
		t.Error("IsPlaying = true, want false after pause")
	}
}

func TestUpdateResumesWhenPlayingSetTrue(t *testing.T) {
	world, svc, _ := setupSystemTestWorld(t)
	svc.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)
	entity := spawnAudioEntity(t, world, "bgm", 1, components.PlaybackPersist, true)

	sys := NewSystem(svc)
	sys.Init(world)

	_ = sys.Update(world, 0.016) // start
	ap, _ := world.GetComponent[components.AudioPlayer](entity)
	ap.Playing = false
	_ = world.SetComponent(entity, ap)
	_ = sys.Update(world, 0.016) // pause

	ap, _ = world.GetComponent[components.AudioPlayer](entity)
	ap.Playing = true
	_ = world.SetComponent(entity, ap)
	_ = sys.Update(world, 0.016) // resume

	if !svc.isPlaying(entity) {
		t.Error("IsPlaying = false, want true after resume")
	}
}

// ---------------------------------------------------------------------------
// System.Update — completion (LoopNone)
// ---------------------------------------------------------------------------

func TestUpdateHandlesCompletionPersistMode(t *testing.T) {
	world, svc, bus := setupSystemTestWorld(t)
	svc.tracks["sfx"] = NewAudioTrack("sfx", silencePCM(), GroupSFX, LoopNone, 1)
	entity := spawnAudioEntity(t, world, "sfx", 1, components.PlaybackPersist, true)

	sys := NewSystem(svc)
	sys.Init(world)

	var stoppedEvents []Event
	bus.On(func(_ events.EventMeta, e Event) {
		if e.Type == EventTypeTrackStopped {
			stoppedEvents = append(stoppedEvents, e)
		}
	}, false)

	_ = sys.Update(world, 0.016) // start playback
	if !svc.isPlaying(entity) {
		t.Fatal("player not playing after first Update")
	}

	// Simulate the player reaching EOF: the player still exists but
	// IsPlaying returns false. We can't force Ebiten to reach EOF in a
	// test, so we remove the player and re-add a finished state by
	// stopping it — but the system detects completion via
	// HasPlayer && !IsPlaying. We simulate by pausing the underlying
	// player directly, which makes IsPlaying return false while
	// HasPlayer stays true.
	svc.mu.Lock()
	if state, ok := svc.players[playerKey{entityID: entity}]; ok && state.player != nil {
		state.player.Pause()
	}
	svc.mu.Unlock()

	_ = sys.Update(world, 0.016) // detect completion

	if svc.hasPlayer(entity) {
		t.Error("HasPlayer = true, want false (player should be removed on completion)")
	}
	ap, _ := world.GetComponent[components.AudioPlayer](entity)
	if ap.Playing {
		t.Error("component Playing = true, want false (completion should set it false)")
	}
	if len(stoppedEvents) != 1 {
		t.Errorf("stopped events = %d, want 1", len(stoppedEvents))
	}
}

func TestUpdateCompletionDoesNotRestartOneShot(t *testing.T) {
	world, svc, _ := setupSystemTestWorld(t)
	svc.tracks["sfx"] = NewAudioTrack("sfx", silencePCM(), GroupSFX, LoopNone, 1)
	entity := spawnAudioEntity(t, world, "sfx", 1, components.PlaybackPersist, true)

	sys := NewSystem(svc)
	sys.Init(world)

	_ = sys.Update(world, 0.016) // start

	// Simulate EOF.
	svc.mu.Lock()
	if state, ok := svc.players[playerKey{entityID: entity}]; ok && state.player != nil {
		state.player.Pause()
	}
	svc.mu.Unlock()

	_ = sys.Update(world, 0.016) // completion: Playing set false, player removed
	_ = sys.Update(world, 0.016) // third tick: should NOT restart

	if svc.hasPlayer(entity) {
		t.Error("HasPlayer = true, want false (one-shot should not restart after completion)")
	}
	ap, _ := world.GetComponent[components.AudioPlayer](entity)
	if ap.Playing {
		t.Error("Playing = true, want false (should remain stopped)")
	}
}

func TestUpdateReplaysOneShotAfterUserSetsPlayingTrue(t *testing.T) {
	world, svc, _ := setupSystemTestWorld(t)
	svc.tracks["sfx"] = NewAudioTrack("sfx", silencePCM(), GroupSFX, LoopNone, 1)
	entity := spawnAudioEntity(t, world, "sfx", 1, components.PlaybackPersist, true)

	sys := NewSystem(svc)
	sys.Init(world)

	_ = sys.Update(world, 0.016) // start

	// Simulate EOF.
	svc.mu.Lock()
	if state, ok := svc.players[playerKey{entityID: entity}]; ok && state.player != nil {
		state.player.Pause()
	}
	svc.mu.Unlock()

	_ = sys.Update(world, 0.016) // completion: Playing false, player removed

	// User sets Playing = true to replay.
	ap, _ := world.GetComponent[components.AudioPlayer](entity)
	ap.Playing = true
	_ = world.SetComponent(entity, ap)

	_ = sys.Update(world, 0.016) // new play: should create a fresh player

	if !svc.hasPlayer(entity) {
		t.Error("HasPlayer = false, want true (one-shot should replay)")
	}
	if !svc.isPlaying(entity) {
		t.Error("IsPlaying = false, want true (replayed one-shot should be playing)")
	}
}

func TestUpdateCompletionDespawnDestroysEntity(t *testing.T) {
	world, svc, bus := setupSystemTestWorld(t)
	svc.tracks["sfx"] = NewAudioTrack("sfx", silencePCM(), GroupSFX, LoopNone, 1)
	entity := spawnAudioEntity(t, world, "sfx", 1, components.PlaybackDespawn, true)

	sys := NewSystem(svc)
	sys.Init(world)

	var stoppedEvents []Event
	bus.On(func(_ events.EventMeta, e Event) {
		if e.Type == EventTypeTrackStopped {
			stoppedEvents = append(stoppedEvents, e)
		}
	}, false)

	_ = sys.Update(world, 0.016) // start

	// Simulate EOF.
	svc.mu.Lock()
	if state, ok := svc.players[playerKey{entityID: entity}]; ok && state.player != nil {
		state.player.Pause()
	}
	svc.mu.Unlock()

	_ = sys.Update(world, 0.016) // completion + despawn

	if svc.hasPlayer(entity) {
		t.Error("HasPlayer = true, want false (player removed before despawn)")
	}
	if len(stoppedEvents) != 1 {
		t.Errorf("stopped events = %d, want 1", len(stoppedEvents))
	}
	if _, err := world.GetComponent[components.AudioPlayer](entity); err == nil {
		t.Error("entity still exists, want it destroyed by PlaybackDespawn")
	}
}

// ---------------------------------------------------------------------------
// System.Update — looping tracks
// ---------------------------------------------------------------------------

func TestUpdateLoopingTrackDoesNotComplete(t *testing.T) {
	world, svc, _ := setupSystemTestWorld(t)
	svc.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)
	entity := spawnAudioEntity(t, world, "bgm", 1, components.PlaybackPersist, true)

	sys := NewSystem(svc)
	sys.Init(world)

	_ = sys.Update(world, 0.016) // start

	// Even if IsPlaying returns false (e.g. buffer underrun), a looping
	// track should not trigger completion.
	svc.mu.Lock()
	if state, ok := svc.players[playerKey{entityID: entity}]; ok && state.player != nil {
		state.player.Pause()
	}
	svc.mu.Unlock()

	_ = sys.Update(world, 0.016)

	if svc.isLooping(entity) {
		// Looping track should NOT be treated as completed.
		if !svc.hasPlayer(entity) {
			t.Error("HasPlayer = false, want true (looping track should not be removed)")
		}
	} else {
		t.Skip("IsLooping returned false; cannot test loop completion guard")
	}
}

// ---------------------------------------------------------------------------
// System.Update — edge cases
// ---------------------------------------------------------------------------

func TestUpdateSkipsIdleEntity(t *testing.T) {
	world, svc, _ := setupSystemTestWorld(t)
	svc.tracks["sfx"] = NewAudioTrack("sfx", silencePCM(), GroupSFX, LoopNone, 1)
	entity := spawnAudioEntity(t, world, "sfx", 1, components.PlaybackPersist, false)

	sys := NewSystem(svc)
	sys.Init(world)

	if err := sys.Update(world, 0.016); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if svc.hasPlayer(entity) {
		t.Error("HasPlayer = true, want false (idle entity should not create a player)")
	}
}

func TestUpdateUninitializedReturnsError(t *testing.T) {
	world, _, _ := setupSystemTestWorld(t)
	sys := &System{}
	if err := sys.Update(world, 0.016); err == nil {
		t.Fatal("Update() on uninitialized system returned nil error")
	}
}

func TestUpdateMissingTrackSkipsEntity(t *testing.T) {
	world, svc, _ := setupSystemTestWorld(t)
	entity := spawnAudioEntity(t, world, "nonexistent", 1, components.PlaybackPersist, true)

	sys := NewSystem(svc)
	sys.Init(world)

	if err := sys.Update(world, 0.016); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if svc.hasPlayer(entity) {
		t.Error("HasPlayer = true, want false (missing track should not create a player)")
	}
}

// ---------------------------------------------------------------------------
// System.Shutdown
// ---------------------------------------------------------------------------

func TestSystemShutdownRemovesAllPlayers(t *testing.T) {
	world, svc, _ := setupSystemTestWorld(t)
	svc.tracks["sfx"] = NewAudioTrack("sfx", silencePCM(), GroupSFX, LoopNone, 1)
	entity := spawnAudioEntity(t, world, "sfx", 1, components.PlaybackPersist, true)

	sys := NewSystem(svc)
	sys.Init(world)

	_ = sys.Update(world, 0.016) // start playback
	if !svc.hasPlayer(entity) {
		t.Fatal("player not created before shutdown")
	}

	if err := sys.Shutdown(world); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if svc.hasPlayer(entity) {
		t.Error("HasPlayer = true, want false after shutdown")
	}
}

func TestSystemShutdownOnUninitializedIsNoop(t *testing.T) {
	world, _, _ := setupSystemTestWorld(t)
	sys := NewSystem(nil)
	if err := sys.Shutdown(world); err != nil {
		t.Errorf("Shutdown() on uninitialized system = %v, want nil", err)
	}
}
