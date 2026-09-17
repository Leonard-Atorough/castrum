// Package audio provides the runtime definition and registry for audio
// playback tracks.
//
// An AudioTrack is an immutable definition created via [NewAudioTrack] and
// registered with an [AudioStore]. Tracks are referenced by [ID] from the
// public [components.Audio] component — the component carries only the track
// ID and per-entity playback state; the track definition (asset data
// reference, group, loop mode, volume) lives here.
//
// The [AudioStore] is registered as a world resource by [castrum.NewGame] and
// resolved by the audio system during [System.Init]. It follows the same
// pattern as [internal/animation.ClipStore]: a concurrent map guarded by an
// RWMutex, with no service indirection layer.
//
// Audio data (raw PCM bytes decoded from WAV, MP3, or OGG) is loaded and cached
// separately by the [assets.Loader]. The track's DataID field is the
// [assets.ID] used to retrieve the cached [assets.AudioData] at playback time.
package audio

import "sync"

// ID identifies an audio track within an [AudioStore].
type ID string

// Group selects which volume bus applies to a track at mix time. The audio
// system reads this to apply group-level volume (e.g. music volume vs SFX
// volume) independently.
type Group int

const (
	// GroupMusic is for background music and long-running ambient tracks.
	GroupMusic Group = iota
	// GroupSFX is for short sound effects triggered by gameplay events.
	GroupSFX
)

// LoopMode controls how a track repeats after reaching the end of its PCM data.
type LoopMode int

const (
	// LoopNone plays the track once and stops.
	LoopNone LoopMode = iota
	// LoopSingle repeats the track from the beginning after it ends.
	LoopSingle
	// LoopAll is reserved for future playlist-style looping across multiple
	// tracks. Currently behaves the same as LoopSingle.
	LoopAll
)

// AudioTrack is an immutable audio definition: a reference to cached audio
// data plus playback configuration. Tracks are created via [NewAudioTrack]
// and registered with an [AudioStore]. All fields are private and read via
// accessors to prevent mutation after construction.
//
// The DataID field is the [assets.ID] of the cached [assets.AudioData]; the
// audio system resolves it at playback time to create an Ebiten audio player.
type AudioTrack struct {
	id     ID
	dataID string
	group  Group
	loop   LoopMode
	volume float64
}

// NewAudioTrack creates an AudioTrack with the given definition. Volume is
// clamped to [0, 1].
func NewAudioTrack(id ID, dataID string, group Group, loop LoopMode, volume float64) *AudioTrack {
	if volume < 0 {
		volume = 0
	} else if volume > 1 {
		volume = 1
	}
	return &AudioTrack{
		id:     id,
		dataID: dataID,
		group:  group,
		loop:   loop,
		volume: volume,
	}
}

// ID returns the track identifier used to look up the track in an [AudioStore].
func (t *AudioTrack) ID() ID {
	return t.id
}

// DataID returns the assets.ID of the cached audio data the track plays.
func (t *AudioTrack) DataID() string {
	return t.dataID
}

// Group returns the volume bus the track belongs to (music or SFX).
func (t *AudioTrack) Group() Group {
	return t.group
}

// Loop returns the loop mode controlling how the track repeats.
func (t *AudioTrack) Loop() LoopMode {
	return t.loop
}

// Volume returns the track's default volume in [0, 1].
func (t *AudioTrack) Volume() float64 {
	return t.volume
}

// AudioStore is the concurrent in-memory registry for audio tracks. It is
// registered as a world resource by [castrum.NewGame] and resolved by the
// audio system during Init.
type AudioStore struct {
	tracks map[ID]*AudioTrack
	mu     sync.RWMutex
}

// NewAudioStore creates an empty audio store.
func NewAudioStore() *AudioStore {
	return &AudioStore{
		tracks: make(map[ID]*AudioTrack),
	}
}

// AddTrack registers a track in the store, keyed by the track's ID. If a
// track with the same ID already exists it is replaced.
func (s *AudioStore) AddTrack(track *AudioTrack) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tracks[track.ID()] = track
}

// GetTrack returns the track registered under id, or ok=false if no such
// track exists.
func (s *AudioStore) GetTrack(id ID) (*AudioTrack, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	track, exists := s.tracks[id]
	return track, exists
}

// HasTrack reports whether a track is registered under id.
func (s *AudioStore) HasTrack(id ID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.tracks[id]
	return exists
}

// RemoveTrack removes the track registered under id. No-op if the ID is
// not registered.
func (s *AudioStore) RemoveTrack(id ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tracks, id)
}

// ListTracks returns all registered tracks. The order is non-deterministic
// (map iteration order). The returned slice is a copy; mutating it does not
// affect the store.
func (s *AudioStore) ListTracks() []*AudioTrack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tracks := make([]*AudioTrack, 0, len(s.tracks))
	for _, track := range s.tracks {
		tracks = append(tracks, track)
	}
	return tracks
}
