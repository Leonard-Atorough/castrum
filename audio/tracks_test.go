package audio

import (
	"testing"

	"github.com/Leonard-Atorough/castrum/asset"
)

func TestSourceTrackSequence(t *testing.T) {
	s := NewPlaylistSource("music/one.ogg", "music/two.ogg", "music/three.ogg")
	if s.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", s.Len())
	}
	if got := s.Current(); got != "music/one.ogg" {
		t.Fatalf("Current() = %q, want music/one.ogg", got)
	}
	if got := s.TrackAt(2); got != "music/three.ogg" {
		t.Fatalf("TrackAt(2) = %q, want music/three.ogg", got)
	}
	if got := s.TrackAt(3); got != "" {
		t.Fatalf("TrackAt(3) = %q, want empty for out of range", got)
	}

	s.Next()
	if got := s.Current(); got != "music/two.ogg" {
		t.Fatalf("Current() after Next = %q, want music/two.ogg", got)
	}
	s.Next()
	s.Next()
	if got := s.Current(); got != "music/one.ogg" {
		t.Fatalf("Current() after wrapping Next = %q, want music/one.ogg", got)
	}
	s.Previous()
	if got := s.Current(); got != "music/three.ogg" {
		t.Fatalf("Current() after wrapping Previous = %q, want music/three.ogg", got)
	}
	s.Seek(1)
	if got := s.Current(); got != "music/two.ogg" {
		t.Fatalf("Current() after Seek = %q, want music/two.ogg", got)
	}
	s.Seek(1)
	if got := s.selectionVersion; got == 0 {
		t.Fatal("Seek to the current position must not bump the selection version")
	}
	s.Seek(9)
	if got := s.Current(); got != "music/two.ogg" {
		t.Fatalf("Current() after out-of-range Seek = %q, want unchanged", got)
	}
}

func TestSourceSingleTrackNavigationRestarts(t *testing.T) {
	s := NewSource("music/loop.ogg")
	before := s.selectionVersion
	s.Next()
	if s.selectionVersion == before {
		t.Fatal("Next on a single-track source must bump the selection version")
	}
	if got := s.Current(); got != "music/loop.ogg" {
		t.Fatalf("Current() = %q, want the only track", got)
	}
}

func TestSourceTrackMutation(t *testing.T) {
	s := NewPlaylistSource("music/one.ogg", "music/two.ogg")

	if err := s.AddTrack("music/three.ogg"); err != nil {
		t.Fatalf("AddTrack: %v", err)
	}
	if s.Len() != 3 || s.TrackAt(2) != "music/three.ogg" {
		t.Fatalf("source after AddTrack = %+v, want three tracks", s)
	}

	if err := s.InsertTrack(0, "music/zero.ogg"); err != nil {
		t.Fatalf("InsertTrack: %v", err)
	}
	if s.Track != "music/zero.ogg" || s.TrackAt(1) != "music/one.ogg" || s.Len() != 4 {
		t.Fatalf("source after InsertTrack(0) = %+v, want the new first track", s)
	}

	s.Seek(2)
	if err := s.RemoveTrack(2); err != nil {
		t.Fatalf("RemoveTrack: %v", err)
	}
	if got := s.Current(); got != "music/three.ogg" {
		t.Fatalf("Current() after removing the selected track = %q, want the track that shifted into its place", got)
	}

	if err := s.SetTrack("sfx/jingle.wav"); err != nil {
		t.Fatalf("SetTrack: %v", err)
	}
	if s.Len() != 1 || s.Current() != "sfx/jingle.wav" {
		t.Fatalf("source after SetTrack = %+v, want a single selected track", s)
	}

	s.Clear()
	if s.Len() != 0 || s.Current() != "" {
		t.Fatalf("source after Clear = %+v, want no tracks", s)
	}
	if err := s.AddTrack("music/one.ogg"); err != nil {
		t.Fatalf("AddTrack after Clear: %v", err)
	}
	if s.Current() != "music/one.ogg" {
		t.Fatalf("Current() after re-adding = %q, want music/one.ogg", s.Current())
	}
}

func TestSourceTrackMutationErrors(t *testing.T) {
	s := NewSource("music/one.ogg")
	if err := s.AddTrack(""); err == nil {
		t.Fatal("AddTrack accepted an empty track")
	}
	if err := s.InsertTrack(-1, "music/two.ogg"); err == nil {
		t.Fatal("InsertTrack accepted a negative position")
	}
	if err := s.InsertTrack(0, ""); err == nil {
		t.Fatal("InsertTrack accepted an empty track")
	}
	if err := s.RemoveTrack(1); err == nil {
		t.Fatal("RemoveTrack accepted an out-of-range position")
	}
	if err := s.SetTrack(""); err == nil {
		t.Fatal("SetTrack accepted an empty track")
	}
}

func TestNewPlaylistSourceClonesTracks(t *testing.T) {
	tracks := []asset.ID{"music/one.ogg", "music/two.ogg"}
	s := NewPlaylistSource(tracks...)
	tracks[0] = "music/hacked.ogg"
	if s.Track != "music/one.ogg" {
		t.Fatalf("NewPlaylistSource aliased the caller's slice: %+v", s)
	}
}
