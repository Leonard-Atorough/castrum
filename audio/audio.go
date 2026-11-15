// Package audio defines backend-independent audio playback. [Source]
// components describe individual plays, [Mixer] holds global volume and
// pause state, and [NewAudioSystem] reconciles them through a runner's
// [Controller], emitting a [Finished] signal when a play completes.
// [OneShot] starts a play on an engine-owned entity and reclaims
// it when playback finishes. Audio files are identified by [asset.ID].
package audio

import (
	"fmt"
	"math"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
)

// LoadMode selects how a play's audio reaches players.
type LoadMode int

const (
	// LoadEager decodes the audio once and caches it for all players.
	LoadEager LoadMode = iota
	// LoadStream decodes the audio from its asset source for each player.
	LoadStream
)

// Group selects the volume bus a play mixes through.
type Group int

const (
	// GroupSFX is the sound-effects bus.
	GroupSFX Group = iota
	// GroupMusic is the music bus.
	GroupMusic
)

// LoopMode controls how a play repeats after its last sample.
type LoopMode int

const (
	// LoopNone means the play does not repeat.
	LoopNone LoopMode = iota
	// LoopForever means the play repeats indefinitely.
	LoopForever
)

// PauseMode controls what a global [Mixer] pause does to a play.
type PauseMode int

const (
	// PauseHolds holds the play's position during a mixer-wide pause.
	PauseHolds PauseMode = iota
	// PauseContinues excludes the play from mixer-wide pauses.
	PauseContinues
)

// Source describes an audio play attached to a [core.Entity]: the tracks it
// plays and the mix settings the audio system applies to a runner's player.
// A source with only [Source.Track] set plays a single clip; adding entries to
// [Source.More] turns it into a sequence. [NewSource] creates a single-track
// source and [NewPlaylistSource] creates a multi-track source, both at unity
// volume.
//
// The zero value is invalid: Volume must be in (0, 1]. A source with no tracks
// is valid but silent: the audio system releases its player until tracks are
// added again.
type Source struct {
	// Group is the volume bus the play mixes through.
	Group Group
	// Volume is the finite per-play level in (0, 1]; use a bus to mute its plays.
	Volume float64
	// Loop is the repeat behavior after the last sample.
	Loop LoopMode
	// Pause is the behavior when the mixer is globally paused.
	Pause PauseMode
	// Load selects how the audio reaches players.
	Load LoadMode
	// PlaybackMultiplier scales the playback rate; zero means normal speed.
	PlaybackMultiplier float64
	// Paused holds the play's position. It composes with the mixer's global
	// pause according to [Source.Pause].
	Paused bool
	// Completed reports whether a [LoopNone] play has finished. The audio
	// system owns this field: it sets it on completion and clears it on
	// restart, including a clip or playlist selection change.
	Completed bool
	// Balance attenuates the opposite stereo channel: -1 mutes the right,
	// 1 mutes the left, and zero preserves the original mix.
	// Values must be finite and in [-1, 1].
	Balance float64
	// Spatial enables position-derived volume and balance using the entity's
	// [core.Transform] and the world's [Listener].
	Spatial bool
	// Attenuation maps normalized distance to attenuation. Nil means linear.
	// A custom curve must be pure, map 0 to 0 and 1 to 1, and return finite
	// values in [0, 1].
	Attenuation geom.Ease
	// Track is the asset ID of the first track in the play's sequence. It is
	// the only track unless [Source.More] names follow-on tracks.
	Track asset.ID
	// More lists the tracks that follow [Source.Track], in play order.
	// A nil slice means the source plays [Source.Track] alone.
	More []asset.ID
	// Repeat controls whether the sequence restarts after its last track
	// finishes. It has no effect on a single-track source's own looping,
	// which [Source.Loop] governs.
	Repeat RepeatMode

	index            int
	selectionVersion uint64
	restarts         uint64
	syncedRestarts   uint64
	syncedAudio      asset.ID
	syncedVersion    uint64
}

// NewSource creates a source that plays track at unity volume with all other
// settings at their zero values.
func NewSource(track asset.ID) Source {
	return Source{Volume: 1, Track: track}
}

// NewPlaylistSource creates a source that plays tracks in order at unity
// volume with all other settings at their zero values. With no tracks it
// returns a source that fails [Source.Validate] until tracks are added.
func NewPlaylistSource(tracks ...asset.ID) Source {
	if len(tracks) == 0 {
		return Source{Volume: 1}
	}
	return Source{Volume: 1, Track: tracks[0], More: slices.Clone(tracks[1:])}
}

// Validate reports an error if Volume is non-finite or outside (0, 1],
// PlaybackMultiplier is non-finite or negative, Balance is non-finite or
// outside [-1, 1], Group, Loop, Pause, or Load has an unsupported value, a
// track ID is empty, or Repeat has an unsupported value.
func (s Source) Validate() error {
	if math.IsNaN(s.Volume) || s.Volume <= 0 || s.Volume > 1 {
		return fmt.Errorf("audio: source volume must be finite and in (0, 1], got %v", s.Volume)
	}
	if math.IsNaN(s.PlaybackMultiplier) || math.IsInf(s.PlaybackMultiplier, 0) || s.PlaybackMultiplier < 0 {
		return fmt.Errorf("audio: source playback multiplier must be finite and not negative (0 is the normal rate), got %v", s.PlaybackMultiplier)
	}
	if s.Loop != LoopNone && s.Loop != LoopForever {
		return fmt.Errorf("audio: invalid loop mode %d", s.Loop)
	}
	if s.Pause != PauseHolds && s.Pause != PauseContinues {
		return fmt.Errorf("audio: invalid pause mode %d", s.Pause)
	}
	if s.Load != LoadEager && s.Load != LoadStream {
		return fmt.Errorf("audio: invalid load mode %d", s.Load)
	}
	if s.Group != GroupSFX && s.Group != GroupMusic {
		return fmt.Errorf("audio: invalid group %d", s.Group)
	}
	if math.IsNaN(s.Balance) || s.Balance < -1 || s.Balance > 1 {
		return fmt.Errorf("audio: source balance must be finite and in [-1, 1], got %v", s.Balance)
	}
	if s.Track == "" && len(s.More) > 0 {
		return fmt.Errorf("audio: source must name a first track before follow-on tracks")
	}
	for i, track := range s.More {
		if track == "" {
			return fmt.Errorf("audio: source track %d must not be empty", i+1)
		}
	}
	if s.Repeat != RepeatNone && s.Repeat != RepeatAll {
		return fmt.Errorf("audio: invalid repeat mode")
	}
	return nil
}

// Restart requests playback from the beginning, clears [Source.Completed],
// and resumes the source.
func (s *Source) Restart() {
	s.Completed = false
	s.Paused = false
	s.restarts++
}

// Mixer holds the master and group volume levels and the global pause state
// used by [NewAudioSystem].
type Mixer struct {
	master float64
	groups map[Group]float64
	paused bool
}

// NewMixer returns a mixer at unity master volume with no group
// overrides and not paused.
func NewMixer() *Mixer {
	return &Mixer{master: 1, groups: make(map[Group]float64)}
}

// Master returns the master volume, from 0 (silence) to 1
// (unity).
func (m *Mixer) Master() float64 {
	return m.master
}

// SetMaster sets the master volume, clamped to [0, 1].
//
// It returns an error for NaN without changing the volume.
func (m *Mixer) SetMaster(volume float64) error {
	if math.IsNaN(volume) {
		return fmt.Errorf("audio: master volume must not be NaN")
	}
	m.master = clamp01(volume)
	return nil
}

// GroupVolume returns the group's volume, from 0 (silence) to 1
// (unity). A group without an override reads unity.
func (m *Mixer) GroupVolume(group Group) float64 {
	if volume, ok := m.groups[group]; ok {
		return volume
	}
	return 1
}

// SetGroupVolume sets the group's volume, clamped to [0, 1].
//
// It returns an error for NaN without changing the volume.
func (m *Mixer) SetGroupVolume(group Group, volume float64) error {
	if math.IsNaN(volume) {
		return fmt.Errorf("audio: group volume must not be NaN")
	}
	m.groups[group] = clamp01(volume)
	return nil
}

// Paused reports whether the mixer is globally paused. Each play's
// [PauseMode] decides what a pause does to its playback.
func (m *Mixer) Paused() bool {
	return m.paused
}

// PauseAll pauses the mixer globally.
func (m *Mixer) PauseAll() {
	m.paused = true
}

// ResumeAll clears the global pause.
func (m *Mixer) ResumeAll() {
	m.paused = false
}

// TogglePause flips the global pause and reports the new state.
// Each play's [PauseMode] decides what the new state does to its
// playback.
func (m *Mixer) TogglePause() bool {
	m.paused = !m.paused
	return m.paused
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// oneshot retains the handle so cleanup destroys only the entity spawned by
// [OneShot].
type oneshot struct {
	self *core.Entity
}

// OneShot starts src playing track on a new engine-owned entity and returns
// its handle. The audio system destroys the entity when playback finishes; it
// never destroys entities that carry ordinary [Source] components.
//
// The source must not loop: a [LoopForever] play never finishes and its
// entity would not be reclaimed. Use a regular [Source] for looping audio.
// OneShot returns an error for a looping, spatial, or invalid source.
func OneShot(world *core.World, src Source, track asset.ID) (*core.Entity, error) {
	if src.Spatial {
		return nil, fmt.Errorf("audio: one-shot %q must not be spatial: use OneShotAt", track)
	}
	return oneShot(world, src, track, core.Transform{})
}

// OneShotAt starts track at position on a new engine-owned entity and returns
// its handle. It enables [Source.Spatial] and sets the entity's
// [core.Transform] position. The world must provide a [Listener]; spatial
// adjustment is applied only when it is enabled.
//
// The source must not loop: a [LoopForever] play never finishes and its entity
// would not be reclaimed. Use a regular [Source] for looping audio.
func OneShotAt(world *core.World, src Source, track asset.ID, position geom.Vector2) (*core.Entity, error) {
	src.Spatial = true
	return oneShot(world, src, track, core.Transform{Position: position})
}

func oneShot(world *core.World, src Source, track asset.ID, transform core.Transform) (*core.Entity, error) {
	if src.Loop == LoopForever {
		return nil, fmt.Errorf("audio: one-shot %q must not loop: use a Source on a game entity", track)
	}
	src.Track = track
	src.index = 0
	entity, err := world.NewEntity(src, transform)
	if err != nil {
		return nil, err
	}
	if err := entity.AddComponent(world, oneshot{self: entity}); err != nil {
		return nil, err
	}
	return entity, nil
}
