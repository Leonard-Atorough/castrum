package audio

import (
	"bytes"
	"testing"
)

func TestNewAudioTrack(t *testing.T) {
	track := NewAudioTrack("shoot", []byte{}, GroupSFX, LoopNone, 0.8)

	if track.ID() != "shoot" {
		t.Errorf("ID() = %q, want %q", track.ID(), "shoot")
	}
	if track.Data() == nil {
		t.Errorf("Data() = nil, want non-nil")
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
	track := NewAudioTrack("bgm", []byte{}, GroupMusic, LoopForever, 1.0)

	if track.ID() != "bgm" {
		t.Errorf("ID() = %q, want %q", track.ID(), "bgm")
	}
	if track.Group() != GroupMusic {
		t.Errorf("Group() = %v, want %v", track.Group(), GroupMusic)
	}
	if track.Loop() != LoopForever {
		t.Errorf("Loop() = %v, want %v", track.Loop(), LoopForever)
	}
	if track.Volume() != 1.0 {
		t.Errorf("Volume() = %v, want %v", track.Volume(), 1.0)
	}
}

func TestNewAudioTrackClampsNegativeVolume(t *testing.T) {
	track := NewAudioTrack("quiet", []byte{}, GroupSFX, LoopNone, -0.5)

	if track.Volume() != 0 {
		t.Errorf("Volume() = %v, want 0 (clamped from -0.5)", track.Volume())
	}
}

func TestNewAudioTrackClampsVolumeAboveOne(t *testing.T) {
	track := NewAudioTrack("loud", []byte{}, GroupSFX, LoopNone, 2.0)

	if track.Volume() != 1 {
		t.Errorf("Volume() = %v, want 1 (clamped from 2.0)", track.Volume())
	}
}

func TestNewAudioTrackZeroVolume(t *testing.T) {
	track := NewAudioTrack("silent", []byte{}, GroupSFX, LoopNone, 0)

	if track.Volume() != 0 {
		t.Errorf("Volume() = %v, want 0", track.Volume())
	}
}

func TestNewAudioTrackReturnsNonNil(t *testing.T) {
	track := NewAudioTrack("id", []byte{}, GroupMusic, LoopNone, 0.5)

	if track == nil {
		t.Fatal("NewAudioTrack() returned nil")
	}
}

func TestAudioTrackDataReturnsPassedPCM(t *testing.T) {
	pcm := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	track := NewAudioTrack("sfx", pcm, GroupSFX, LoopNone, 1)

	if got := track.Data(); !bytes.Equal(got, pcm) {
		t.Errorf("Data() = %v, want %v", got, pcm)
	}
}

func TestAudioTrackDataReturnsNilForNilInput(t *testing.T) {
	track := NewAudioTrack("silent", nil, GroupSFX, LoopNone, 1)

	if got := track.Data(); got != nil {
		t.Errorf("Data() = %v, want nil", got)
	}
}

func TestNewAudioTrackDoesNotClampBelowOne(t *testing.T) {
	track := NewAudioTrack("id", []byte{}, GroupSFX, LoopNone, 0.9)

	if track.Volume() != 0.9 {
		t.Errorf("Volume() = %v, want 0.9 (not clamped)", track.Volume())
	}
}

func TestGroupConstantValues(t *testing.T) {
	if GroupMusic != 0 {
		t.Errorf("GroupMusic = %d, want 0", GroupMusic)
	}
	if GroupSFX != 1 {
		t.Errorf("GroupSFX = %d, want 1", GroupSFX)
	}
}

func TestLoopModeConstantValues(t *testing.T) {
	if LoopNone != 0 {
		t.Errorf("LoopNone = %d, want 0", LoopNone)
	}
	if LoopForever != 1 {
		t.Errorf("LoopForever = %d, want 1", LoopForever)
	}
}
