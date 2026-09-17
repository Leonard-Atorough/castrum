package audio

import (
	"testing"
)

func TestNewAudioTrack(t *testing.T) {
	track := NewAudioTrack("shoot", "audio/sfx/shoot.wav", GroupSFX, LoopNone, 0.8)

	if track.ID() != "shoot" {
		t.Errorf("ID() = %q, want %q", track.ID(), "shoot")
	}
	if track.DataID() != "audio/sfx/shoot.wav" {
		t.Errorf("DataID() = %q, want %q", track.DataID(), "audio/sfx/shoot.wav")
	}
	if track.Group() != GroupSFX {
		t.Errorf("Group() = %v, want %v", track.Group(), GroupSFX)
	}
	if track.Loop() != LoopNone {
		t.Errorf("Loop() = %v, want %v", track.Loop(), LoopNone)
	}
	if track.Volume() != 0.8 {
		t.Errorf("Volume() = %v, want %v", track.Volume(), 0.8)
	}
}

func TestNewAudioTrackAllFields(t *testing.T) {
	track := NewAudioTrack("bgm", "audio/music/loop.ogg", GroupMusic, LoopSingle, 1.0)

	if track.ID() != "bgm" {
		t.Errorf("ID() = %q, want %q", track.ID(), "bgm")
	}
	if track.Group() != GroupMusic {
		t.Errorf("Group() = %v, want %v", track.Group(), GroupMusic)
	}
	if track.Loop() != LoopSingle {
		t.Errorf("Loop() = %v, want %v", track.Loop(), LoopSingle)
	}
	if track.Volume() != 1.0 {
		t.Errorf("Volume() = %v, want %v", track.Volume(), 1.0)
	}
}

func TestNewAudioTrackClampsNegativeVolume(t *testing.T) {
	track := NewAudioTrack("quiet", "data", GroupSFX, LoopNone, -0.5)

	if track.Volume() != 0 {
		t.Errorf("Volume() = %v, want 0 (clamped from -0.5)", track.Volume())
	}
}

func TestNewAudioTrackClampsVolumeAboveOne(t *testing.T) {
	track := NewAudioTrack("loud", "data", GroupSFX, LoopNone, 2.0)

	if track.Volume() != 1 {
		t.Errorf("Volume() = %v, want 1 (clamped from 2.0)", track.Volume())
	}
}

func TestNewAudioTrackZeroVolume(t *testing.T) {
	track := NewAudioTrack("silent", "data", GroupSFX, LoopNone, 0)

	if track.Volume() != 0 {
		t.Errorf("Volume() = %v, want 0", track.Volume())
	}
}

func TestNewAudioTrackReturnsNonNil(t *testing.T) {
	track := NewAudioTrack("id", "data", GroupMusic, LoopNone, 0.5)

	if track == nil {
		t.Fatal("NewAudioTrack() returned nil")
	}
}

func TestNewAudioStore(t *testing.T) {
	store := NewAudioStore()

	if store == nil {
		t.Fatal("NewAudioStore() returned nil")
	}
	if store.HasTrack("anything") {
		t.Error("NewAudioStore() should be empty")
	}
}

func TestAudioStoreAddAndGet(t *testing.T) {
	store := NewAudioStore()
	track := NewAudioTrack("shoot", "audio/sfx/shoot.wav", GroupSFX, LoopNone, 0.8)
	store.AddTrack(track)

	got, ok := store.GetTrack("shoot")
	if !ok {
		t.Fatal("GetTrack() returned !ok after AddTrack()")
	}
	if got != track {
		t.Error("GetTrack() returned a different track than the one added")
	}
}

func TestAudioStoreGetMissingReturnsFalse(t *testing.T) {
	store := NewAudioStore()

	track, ok := store.GetTrack("nonexistent")
	if ok {
		t.Error("GetTrack() for missing track returned ok=true, want false")
	}
	if track != nil {
		t.Error("GetTrack() for missing track returned non-nil track")
	}
}

func TestAudioStoreHasTrack(t *testing.T) {
	store := NewAudioStore()
	store.AddTrack(NewAudioTrack("bgm", "data", GroupMusic, LoopSingle, 1.0))

	if !store.HasTrack("bgm") {
		t.Error("HasTrack() returned false for registered track")
	}
	if store.HasTrack("missing") {
		t.Error("HasTrack() returned true for unregistered track")
	}
}

func TestAudioStoreRemoveTrack(t *testing.T) {
	store := NewAudioStore()
	store.AddTrack(NewAudioTrack("temp", "data", GroupSFX, LoopNone, 0.5))

	store.RemoveTrack("temp")

	if store.HasTrack("temp") {
		t.Error("HasTrack() returned true after RemoveTrack()")
	}
}

func TestAudioStoreRemoveTrackMissingIsNoOp(t *testing.T) {
	store := NewAudioStore()

	store.RemoveTrack("does-not-exist")

	if store.HasTrack("does-not-exist") {
		t.Error("HasTrack() returned true after removing non-existent track")
	}
}

func TestAudioStoreAddTrackOverwritesExisting(t *testing.T) {
	store := NewAudioStore()
	original := NewAudioTrack("track", "old.wav", GroupMusic, LoopNone, 0.5)
	replacement := NewAudioTrack("track", "new.wav", GroupSFX, LoopSingle, 1.0)

	store.AddTrack(original)
	store.AddTrack(replacement)

	got, ok := store.GetTrack("track")
	if !ok {
		t.Fatal("GetTrack() returned !ok after overwrite")
	}
	if got.DataID() != "new.wav" {
		t.Errorf("DataID() = %q, want %q (overwritten track)", got.DataID(), "new.wav")
	}
	if got.Group() != GroupSFX {
		t.Errorf("Group() = %v, want %v (overwritten track)", got.Group(), GroupSFX)
	}
}

func TestAudioStoreListTracksEmpty(t *testing.T) {
	store := NewAudioStore()

	tracks := store.ListTracks()
	if len(tracks) != 0 {
		t.Errorf("ListTracks() on empty store returned %d tracks, want 0", len(tracks))
	}
}

func TestAudioStoreListTracks(t *testing.T) {
	store := NewAudioStore()
	t1 := NewAudioTrack("a", "data1", GroupMusic, LoopNone, 0.5)
	t2 := NewAudioTrack("b", "data2", GroupSFX, LoopSingle, 1.0)
	store.AddTrack(t1)
	store.AddTrack(t2)

	tracks := store.ListTracks()
	if len(tracks) != 2 {
		t.Fatalf("ListTracks() returned %d tracks, want 2", len(tracks))
	}

	// Order is non-deterministic; check by ID set.
	ids := map[ID]bool{tracks[0].ID(): true, tracks[1].ID(): true}
	if !ids["a"] || !ids["b"] {
		t.Errorf("ListTracks() returned IDs %v, want {a, b}", ids)
	}
}

func TestAudioStoreListTracksReturnsCopy(t *testing.T) {
	store := NewAudioStore()
	store.AddTrack(NewAudioTrack("a", "data", GroupSFX, LoopNone, 0.5))

	tracks := store.ListTracks()
	tracks[0] = nil // mutate the returned slice

	// Store should be unaffected.
	got, ok := store.GetTrack("a")
	if !ok || got == nil {
		t.Error("mutating ListTracks() result affected the store")
	}
}
