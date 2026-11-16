// Package audio provides backend-independent playback. [Player] components
// select tracks from a [Source], while a [Mixer] controls global volume and
// pause state. [NewAudioSystem] reconciles them with a runner's [Controller]
// and emits [Finished] signals. [OneShot] creates a temporary player that the
// system destroys when its sequence finishes.
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

// LoopMode controls how a player advances and repeats its tracks.
type LoopMode int

const (
	// LoopNone plays the sequence once.
	LoopNone LoopMode = iota
	// LoopTrack repeats the current track without advancing the sequence.
	LoopTrack
	// LoopAll repeats the sequence after its last track.
	LoopAll
)

// PauseMode controls what a global [Mixer] pause does to a play.
type PauseMode int

const (
	// PauseHolds holds the play's position during a mixer-wide pause.
	PauseHolds PauseMode = iota
	// PauseContinues excludes the play from mixer-wide pauses.
	PauseContinues
)

// Source is a sequence of audio asset IDs for a [Player].
type Source struct{ tracks []asset.ID }

// Track returns a source with the given track ID.
func Track(id asset.ID) Source {
	return Source{tracks: []asset.ID{id}}
}

// Tracks returns a source with a copy of the given track IDs.
func Tracks(ids ...asset.ID) Source {
	cloned := make([]asset.ID, len(ids))
	copy(cloned, ids)

	return Source{tracks: cloned}
}

// Len returns the number of tracks.
func (s Source) Len() int { return len(s.tracks) }

// At returns the track at index i. It panics if i is out of range.
func (s Source) At(i int) asset.ID { return s.tracks[i] }

// Validate reports an error if the source has no tracks or a track ID is
// empty.
func (s Source) Validate() error {
	if len(s.tracks) == 0 {
		return fmt.Errorf("audio: source must have at least one track")
	}
	for i, track := range s.tracks {
		if track == "" {
			return fmt.Errorf("audio: track at index %d is empty", i)
		}
	}
	return nil
}

// Settings controls how a [Player] plays its source.
type Settings struct {
	// Group selects the mixer volume bus.
	Group Group
	// Volume is the finite per-player volume in (0, 1].
	Volume float64
	// Loop selects how the player advances and repeats tracks.
	Loop LoopMode
	// Pause selects how the player responds to a mixer-wide pause.
	Pause PauseMode
	// Load selects eager decoding or streaming.
	Load LoadMode
	// PlaybackMultiplier is the finite non-negative playback rate; zero means
	// normal speed. A runner may reject unsupported rates.
	PlaybackMultiplier float64
	// Balance attenuates the opposite stereo channel; it must be finite and
	// from -1 to 1.
	Balance float64
	// Spatial enables listener-relative volume and balance. It requires a
	// [Listener] resource; an enabled listener also requires a [core.Transform]
	// on the player entity.
	Spatial bool
	// Attenuation maps normalized distance to volume falloff. Nil is linear;
	// a curve must map 0 to 0 and 1 to 1 and return finite values in [0, 1].
	Attenuation geom.Ease
}

// DefaultSettings returns settings with unity volume and zero-valued options.
func DefaultSettings() Settings {
	return Settings{Volume: 1}
}

// Validate reports an error for invalid numeric or enum settings.
func (s Settings) Validate() error {
	if math.IsNaN(s.Volume) || s.Volume <= 0 || s.Volume > 1 {
		return fmt.Errorf("audio: settings volume must be finite and in (0, 1], got %v", s.Volume)
	}
	if math.IsNaN(s.PlaybackMultiplier) || math.IsInf(s.PlaybackMultiplier, 0) || s.PlaybackMultiplier < 0 {
		return fmt.Errorf("audio: settings playback multiplier must be finite and not negative (0 is the normal rate), got %v", s.PlaybackMultiplier)
	}
	if s.Loop != LoopNone && s.Loop != LoopTrack && s.Loop != LoopAll {
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
		return fmt.Errorf("audio: settings balance must be finite and in [-1, 1], got %v", s.Balance)
	}
	return nil
}

// Player describes playback of a [Source] on an entity. The audio system
// reconciles it with a [Controller].
//
// Construct a player with [NewPlayer]; its zero value is not valid.
type Player struct {
	// Settings controls playback and may be updated while the player is active.
	Settings Settings
	// Paused requests that playback hold its position. The mixer can also
	// hold playback.
	Paused bool
	// Completed is set by the audio system when the last track finishes
	// without repeating the sequence. Restart clears it immediately; the
	// audio system clears it when advancing or syncing a selection.
	Completed bool

	source           Source
	current          int
	selectionVersion uint64
	restarts         uint64
	syncedRestarts   uint64
	syncedAudio      asset.ID
	syncedSelection  uint64
	synced           bool
}

// NewPlayer returns a player with the first track selected.
// It returns an error if src or settings is invalid.
func NewPlayer(src Source, settings Settings) (Player, error) {
	p := Player{
		Settings: settings,
		source:   src,
	}
	err := p.Validate()
	if err != nil {
		return Player{}, err
	}
	return p, nil
}

// Validate reports an error if the player's source or settings is invalid.
func (p Player) Validate() error {
	err := p.Settings.Validate()
	if err != nil {
		return err
	}
	err = p.source.Validate()
	if err != nil {
		return err
	}
	return nil
}

// Source returns the player's track sequence.
func (p Player) Source() Source { return p.source }

// Current returns the selected audio ID, whether or not it has been synced.
func (p Player) Current() asset.ID { return p.source.At(p.current) }

// Index returns the selected track's index in the source.
func (p Player) Index() int { return p.current }

// Restart selects the first track, clears completion and the player's pause,
// and requests a restart on the next audio system update. A mixer-wide pause
// can still hold playback.
func (p *Player) Restart() {
	p.Completed = false
	p.Paused = false
	p.current = 0
	p.restarts++
}

// Next selects the next track, wrapping to the first, and requests a restart.
func (p *Player) Next() {
	p.current = (p.current + 1) % len(p.source.tracks)
	p.selectionVersion++
}

// Prev selects the previous track, wrapping to the last, and requests a restart.
func (p *Player) Prev() {
	p.current = (p.current - 1 + len(p.source.tracks)) % len(p.source.tracks)
	p.selectionVersion++
}

// Seek selects track i and requests a restart. The index selects a track, not
// a position within one. It returns an error without changing the selection
// if i is out of range.
func (p *Player) Seek(i int) error {
	if i >= 0 && i < len(p.source.tracks) {
		p.current = i
		p.selectionVersion++
		return nil
	}
	return fmt.Errorf("audio: index out of range")
}

// SetSource replaces the source, selects its first track, and requests a
// restart. It returns an error without changing the player if src is invalid.
func (p *Player) SetSource(src Source) error {
	err := src.Validate()
	if err != nil {
		return err
	}
	p.source = src
	p.current = 0
	p.selectionVersion++
	return nil
}

// Append adds a track after the existing sequence.
func (p *Player) Append(id asset.ID) {
	p.source.tracks = append(slices.Clone(p.source.tracks), id)
}

// Insert adds a track at i, preserving the current selection. It returns an
// error if i is negative or greater than the source length.
func (p *Player) Insert(i int, id asset.ID) error {
	if i < 0 || i > len(p.source.tracks) {
		return fmt.Errorf("audio: index out of range")
	}
	if len(p.source.tracks) > 0 && i <= p.current {
		p.current++
	}
	p.source.tracks = slices.Insert(slices.Clone(p.source.tracks), i, id)
	return nil
}

// Remove removes the track at i. Removing the selected track selects the next
// track, or the previous one at the end, and requests a restart. It returns an
// error if i is out of range or the source has only one track.
func (p *Player) Remove(i int) error {
	if i < 0 || i >= len(p.source.tracks) {
		return fmt.Errorf("audio: index out of range")
	}
	if len(p.source.tracks) == 1 {
		return fmt.Errorf("audio: cannot remove the final track")
	}
	selected := i == p.current
	p.source.tracks = slices.Delete(slices.Clone(p.source.tracks), i, i+1)
	if i < p.current || p.current >= len(p.source.tracks) {
		p.current--
	}
	if selected {
		p.selectionVersion++
	}
	return nil
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

// OneShot creates a temporary entity for a non-looping play. The audio system
// destroys the entity when the sequence finishes. It returns an error if the
// source or settings is invalid or settings requests looping.
func OneShot(world *core.World, src Source, s Settings) (*core.Entity, error) {
	return oneShot(world, src, s, geom.Vector2{})
}

// OneShotAt starts a [OneShot] at pos. Set [Settings.Spatial] to enable
// listener-relative playback.
func OneShotAt(world *core.World, src Source, s Settings, pos geom.Vector2) (*core.Entity, error) {
	return oneShot(world, src, s, pos)
}

func oneShot(world *core.World, src Source, s Settings, pos geom.Vector2) (*core.Entity, error) {
	if s.Loop == LoopTrack || s.Loop == LoopAll {
		return nil, fmt.Errorf("audio: one-shot must not loop")
	}
	player, err := NewPlayer(src, s)
	if err != nil {
		return nil, err
	}
	entity, err := world.NewEntity(player, core.Transform{Position: pos})
	if err != nil {
		return nil, err
	}
	if err := entity.AddComponent(world, oneshot{self: entity}); err != nil {
		_ = world.DestroyEntity(entity)
		return nil, err
	}
	return entity, nil
}
