package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/leonard-atorough/castrum"
	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/ecs"
)

const testSampleRate = 44100

// Ebiten's audio context is a process-wide singleton: audio.NewContext panics
// on the second call, so it is created once and shared across all tests.
var (
	testCtxOnce sync.Once
	testCtx     *audio.Context
)

func testAudioContext() *audio.Context {
	testCtxOnce.Do(func() {
		testCtx = audio.NewContext(testSampleRate)
	})
	return testCtx
}

// newTestService builds an AudioService with a real headless Ebiten audio
// context and unit volumes. The loader is nil; tests exercising AddTrack build
// their own service via newTestServiceWithLoader.
func newTestService(t *testing.T) *AudioService {
	t.Helper()
	return NewAudioService(
		testAudioContext(),
		nil,
		castrum.AudioConfig{MasterVolume: 1, MusicVolume: 1, SFXVolume: 1},
		context.Background(),
	)
}

// silencePCM returns a buffer of zero bytes large enough to feed a player
// without immediately draining.
func silencePCM() []byte { return make([]byte, 8192) }

// approxEqual reports whether two float64s are within 1e-9, for volume checks.
func approxEqual(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}

// makeWAV returns a minimal valid RIFF/WAV file: 16-bit mono PCM of silence at
// the given sample rate. numSamples is the number of zero samples in the data
// chunk.
func makeWAV(sampleRate, numSamples int) []byte {
	dataLen := numSamples * 2 // 16-bit mono
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataLen))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16)) // subchunk1 size
	binary.Write(&buf, binary.LittleEndian, uint16(1))  // PCM
	binary.Write(&buf, binary.LittleEndian, uint16(1))  // mono
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate*2)) // byte rate
	binary.Write(&buf, binary.LittleEndian, uint16(2))            // block align
	binary.Write(&buf, binary.LittleEndian, uint16(16))           // bits per sample
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataLen))
	buf.Write(make([]byte, dataLen))
	return buf.Bytes()
}

// newTestServiceWithLoader builds an AudioService whose loader is backed by a
// fstest.MapFS containing a single WAV asset at path, sampled at the context
// rate so resampling is a no-op.
func newTestServiceWithLoader(t *testing.T, path string, wav []byte) *AudioService {
	t.Helper()
	a := assets.NewAssets(fstest.MapFS{
		path: &fstest.MapFile{Data: wav},
	})
	return NewAudioService(
		testAudioContext(),
		a.AssetLoader(),
		castrum.AudioConfig{MasterVolume: 1, MusicVolume: 1, SFXVolume: 1},
		context.Background(),
	)
}

// ---------------------------------------------------------------------------
// Track registry
// ---------------------------------------------------------------------------

func TestGetTrackMissingReturnsFalse(t *testing.T) {
	s := newTestService(t)
	track, ok := s.GetTrack("nope")
	if ok {
		t.Error("GetTrack(missing) returned ok=true, want false")
	}
	if track != nil {
		t.Error("GetTrack(missing) returned non-nil track, want nil")
	}
}

func TestGetTrackReturnsRegistered(t *testing.T) {
	s := newTestService(t)
	want := NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)
	s.tracks["bgm"] = want

	got, ok := s.GetTrack("bgm")
	if !ok {
		t.Fatal("GetTrack(bgm) returned ok=false, want true")
	}
	if got != want {
		t.Error("GetTrack returned a different track pointer")
	}
}

func TestHasTrackReportsRegistration(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if !s.HasTrack("bgm") {
		t.Error("HasTrack(bgm) = false, want true")
	}
	if s.HasTrack("sfx") {
		t.Error("HasTrack(sfx) = true, want false")
	}
}

func TestRemoveTrackUnregisters(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	s.RemoveTrack("bgm")
	if s.HasTrack("bgm") {
		t.Error("RemoveTrack left the track registered")
	}
}

func TestRemoveTrackMissingIsNoop(t *testing.T) {
	s := newTestService(t)
	s.RemoveTrack("ghost") // must not panic
}

// ---------------------------------------------------------------------------
// AddTrack (loader + resample)
// ---------------------------------------------------------------------------

func TestAddTrackRegistersDecodedTrack(t *testing.T) {
	s := newTestServiceWithLoader(t, "shoot.wav", makeWAV(testSampleRate, 256))

	if err := s.AddTrack("shoot", "shoot.wav", GroupSFX, LoopNone, 0.8); err != nil {
		t.Fatalf("AddTrack: %v", err)
	}
	track, ok := s.GetTrack("shoot")
	if !ok {
		t.Fatal("track not registered after AddTrack")
	}
	if track.Group() != GroupSFX {
		t.Errorf("Group() = %v, want GroupSFX", track.Group())
	}
	if track.Volume() != 0.8 {
		t.Errorf("Volume() = %v, want 0.8", track.Volume())
	}
	if len(track.Data()) == 0 {
		t.Error("Data() is empty, want decoded PCM bytes")
	}
}

func TestAddTrackResamplesToContextRate(t *testing.T) {
	// Encode at 22050 Hz; the context runs at 44100 Hz, so the track's PCM
	// must be resampled up. The decoder upmixes mono to stereo 16-bit (256
	// samples -> 1024 bytes), and doubling the sample rate must produce
	// strictly more than that.
	const srcRate = 22050
	s := newTestServiceWithLoader(t, "bgm.wav", makeWAV(srcRate, 256))

	if err := s.AddTrack("bgm", "bgm.wav", GroupMusic, LoopForever, 1); err != nil {
		t.Fatalf("AddTrack: %v", err)
	}
	track, ok := s.GetTrack("bgm")
	if !ok {
		t.Fatal("track not registered")
	}
	if len(track.Data()) <= 1024 {
		t.Errorf("Data() len = %d, want > 1024 (resampled up from %d Hz)",
			len(track.Data()), srcRate)
	}
}

func TestAddTrackSkipsResampleWhenRatesMatch(t *testing.T) {
	s := newTestServiceWithLoader(t, "sfx.wav", makeWAV(testSampleRate, 256))

	if err := s.AddTrack("sfx", "sfx.wav", GroupSFX, LoopNone, 1); err != nil {
		t.Fatalf("AddTrack: %v", err)
	}
	track, _ := s.GetTrack("sfx")
	// With matching sample rates the PCM is returned unchanged from the
	// decoder, which upmixes to stereo 16-bit: 256 samples * 2 channels *
	// 2 bytes = 1024 bytes.
	if len(track.Data()) != 1024 {
		t.Errorf("Data() len = %d, want 1024 (no resampling, stereo upmix)", len(track.Data()))
	}
}

func TestAddTrackMissingAssetReturnsError(t *testing.T) {
	s := newTestServiceWithLoader(t, "x.wav", makeWAV(testSampleRate, 64))

	if err := s.AddTrack("x", "missing.wav", GroupSFX, LoopNone, 1); err == nil {
		t.Fatal("AddTrack with missing asset succeeded, want error")
	}
}

// ---------------------------------------------------------------------------
// SyncPlayer
// ---------------------------------------------------------------------------

func TestSyncPlayerMissingTrackReturnsError(t *testing.T) {
	s := newTestService(t)
	if err := s.SyncPlayer(ecs.EntityID(1), "missing", true, 1); err == nil {
		t.Fatal("SyncPlayer with missing track succeeded, want error")
	}
}

func TestSyncPlayerCreatesAndPlays(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("SyncPlayer play: %v", err)
	}
	state := s.players[playerKey{entityID: 1}]
	if state == nil || state.player == nil {
		t.Fatal("player not created")
	}
	if !state.player.IsPlaying() {
		t.Error("player.IsPlaying() = false, want true after SyncPlayer(true)")
	}
}

func TestSyncPlayerPauses(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("play: %v", err)
	}
	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", false, 1); err != nil {
		t.Fatalf("pause: %v", err)
	}
	state := s.players[playerKey{entityID: 1}]
	if state.player.IsPlaying() {
		t.Error("player.IsPlaying() = true, want false after SyncPlayer(false)")
	}
}

func TestSyncPlayerReusesPlayerAcrossCalls(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("play: %v", err)
	}
	first := s.players[playerKey{entityID: 1}].player

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", false, 1); err != nil {
		t.Fatalf("pause: %v", err)
	}
	second := s.players[playerKey{entityID: 1}].player

	if first != second {
		t.Error("SyncPlayer created a new player instead of reusing the existing one")
	}
}

func TestSyncPlayerAppliesMixedVolume(t *testing.T) {
	// Master=1, music=0.5, track=0.8, component=0.5 → effective 0.2.
	s := newTestService(t)
	s.config = castrum.AudioConfig{MasterVolume: 1, MusicVolume: 0.5, SFXVolume: 1}
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 0.8)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 0.5); err != nil {
		t.Fatalf("SyncPlayer: %v", err)
	}
	const want = 0.2
	if got := s.players[playerKey{entityID: 1}].player.Volume(); !approxEqual(got, want) {
		t.Errorf("player.Volume() = %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// StopPlayer / RemovePlayer
// ---------------------------------------------------------------------------

func TestStopPlayerUnknownEntityIsNoop(t *testing.T) {
	s := newTestService(t)
	if err := s.StopPlayer(ecs.EntityID(1)); err != nil {
		t.Errorf("StopPlayer(unknown) = %v, want nil (no-op)", err)
	}
}

func TestStopPlayerStateWithoutPlayerIsNoop(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)
	// SyncPlayer(false) registers a state but never creates a player.
	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", false, 1); err != nil {
		t.Fatalf("SyncPlayer(false): %v", err)
	}
	if err := s.StopPlayer(ecs.EntityID(1)); err != nil {
		t.Errorf("StopPlayer(state-without-player) = %v, want nil (no-op)", err)
	}
}

func TestStopPlayerPausesAndRetains(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("SyncPlayer: %v", err)
	}
	if err := s.StopPlayer(ecs.EntityID(1)); err != nil {
		t.Fatalf("StopPlayer: %v", err)
	}
	// The player state must still be present so it can be resumed.
	if _, ok := s.players[playerKey{entityID: 1}]; !ok {
		t.Error("StopPlayer removed the player state, want it retained")
	}
}

func TestRemovePlayerUnknownEntityIsNoop(t *testing.T) {
	s := newTestService(t)
	if err := s.RemovePlayer(ecs.EntityID(1)); err != nil {
		t.Errorf("RemovePlayer(unknown) = %v, want nil (no-op)", err)
	}
}

func TestRemovePlayerStateWithoutPlayerDiscardsState(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)
	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", false, 1); err != nil {
		t.Fatalf("SyncPlayer(false): %v", err)
	}
	if err := s.RemovePlayer(ecs.EntityID(1)); err != nil {
		t.Fatalf("RemovePlayer(state-without-player) = %v, want nil", err)
	}
	// No player was ever created, so RemovePlayer just discards the state.
	if _, ok := s.players[playerKey{entityID: 1}]; ok {
		t.Error("RemovePlayer left the player state, want it discarded")
	}
}

func TestRemovePlayerStopsAndDiscards(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("SyncPlayer: %v", err)
	}
	if err := s.RemovePlayer(ecs.EntityID(1)); err != nil {
		t.Fatalf("RemovePlayer: %v", err)
	}
	if _, ok := s.players[playerKey{entityID: 1}]; ok {
		t.Error("RemovePlayer left the player state, want it discarded")
	}
}

// ---------------------------------------------------------------------------
// HasPlayer / IsPlaying / IsLooping
// ---------------------------------------------------------------------------

func TestHasPlayerUnknownEntityReturnsFalse(t *testing.T) {
	s := newTestService(t)
	if s.HasPlayer(ecs.EntityID(1)) {
		t.Error("HasPlayer(unknown) = true, want false")
	}
}

func TestHasPlayerAfterSyncReturnsTrue(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("SyncPlayer: %v", err)
	}
	if !s.HasPlayer(ecs.EntityID(1)) {
		t.Error("HasPlayer(after sync) = false, want true")
	}
}

func TestHasPlayerFalseAfterRemove(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("SyncPlayer: %v", err)
	}
	if err := s.RemovePlayer(ecs.EntityID(1)); err != nil {
		t.Fatalf("RemovePlayer: %v", err)
	}
	if s.HasPlayer(ecs.EntityID(1)) {
		t.Error("HasPlayer(after remove) = true, want false")
	}
}

func TestIsPlayingUnknownEntityReturnsFalse(t *testing.T) {
	s := newTestService(t)
	if s.IsPlaying(ecs.EntityID(1)) {
		t.Error("IsPlaying(unknown) = true, want false")
	}
}

func TestIsPlayingTrueAfterPlay(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("SyncPlayer: %v", err)
	}
	if !s.IsPlaying(ecs.EntityID(1)) {
		t.Error("IsPlaying(after play) = false, want true")
	}
}

func TestIsPlayingFalseAfterPause(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("play: %v", err)
	}
	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", false, 1); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if s.IsPlaying(ecs.EntityID(1)) {
		t.Error("IsPlaying(after pause) = true, want false")
	}
}

func TestIsPlayingFalseWhenStateHasNoPlayer(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	// SyncPlayer(false) creates a state entry but no player.
	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", false, 1); err != nil {
		t.Fatalf("SyncPlayer: %v", err)
	}
	if !s.HasPlayer(ecs.EntityID(1)) {
		t.Fatal("HasPlayer = false, want true (state exists)")
	}
	if s.IsPlaying(ecs.EntityID(1)) {
		t.Error("IsPlaying(state-without-player) = true, want false")
	}
}

func TestIsLoopingUnknownEntityReturnsFalse(t *testing.T) {
	s := newTestService(t)
	if s.IsLooping(ecs.EntityID(1)) {
		t.Error("IsLooping(unknown) = true, want false")
	}
}

func TestIsLoopingTrueAfterSyncLoopForever(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopForever, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("SyncPlayer: %v", err)
	}
	if !s.IsLooping(ecs.EntityID(1)) {
		t.Error("IsLooping(after sync loop forever) = false, want true")
	}
}

func TestIsLoopingFalseAfterSyncLoopNone(t *testing.T) {
	s := newTestService(t)
	s.tracks["bgm"] = NewAudioTrack("bgm", silencePCM(), GroupMusic, LoopNone, 1)

	if err := s.SyncPlayer(ecs.EntityID(1), "bgm", true, 1); err != nil {
		t.Fatalf("SyncPlayer: %v", err)
	}
	if s.IsLooping(ecs.EntityID(1)) {
		t.Error("IsLooping(after sync loop none) = true, want false")
	}
}
