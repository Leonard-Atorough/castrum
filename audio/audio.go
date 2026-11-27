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
	// Bus selects the mixer volume bus; the zero value selects the empty-named bus.
	Bus BusName
	// Volume is the finite per-player volume in (0, 1].
	Volume float64
	// Loop selects how the player advances and repeats tracks. The ebitrun
	// runner restarts the current track when its track-repeat setting changes.
	Loop LoopMode
	// Pause selects how the player responds to a mixer-wide pause.
	Pause PauseMode
	// Load selects eager decoding or streaming. The ebitrun runner restarts
	// the current track when this changes.
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
	// Paused holds playback independently of the mixer's pause.
	Paused bool

	completed        bool
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

// Completed reports whether the sequence has finished without repeating.
// Restart clears it immediately; selecting a track clears it on the next sync.
func (p Player) Completed() bool { return p.completed }

// Current returns the selected audio ID, whether or not it has been synced.
func (p Player) Current() asset.ID { return p.source.At(p.current) }

// Index returns the selected track's index in the source.
func (p Player) Index() int { return p.current }

// Restart selects the first track, clears completion and the player's pause,
// and requests a restart on the next audio system update. A mixer-wide pause
// can still hold playback.
func (p *Player) Restart() {
	p.completed = false
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

// Append adds a track after the existing sequence. It returns an error
// without changing the player if id is empty.
func (p *Player) Append(id asset.ID) error {
	if id == "" {
		return fmt.Errorf("audio: track is empty")
	}
	p.source.tracks = append(slices.Clone(p.source.tracks), id)
	return nil
}

// Insert adds a track at i, preserving the current selection. It returns an
// error without changing the player if id is empty or i is negative or greater
// than the source length.
func (p *Player) Insert(i int, id asset.ID) error {
	if i < 0 || i > len(p.source.tracks) {
		return fmt.Errorf("audio: index out of range")
	}
	if id == "" {
		return fmt.Errorf("audio: track is empty")
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

type oneshot struct{}

// OneShot creates a temporary entity for a non-looping play. The audio system
// destroys the entity when the sequence finishes. It returns an error if the
// source or settings is invalid or settings requests looping or spatial
// playback. Use [OneShotAt] for spatial playback.
func OneShot(world *core.World, src Source, s Settings) (core.Entity, error) {
	if s.Spatial {
		return core.Entity{}, fmt.Errorf("audio: spatial one-shot requires OneShotAt")
	}
	return oneShot(world, src, s, geom.Vector2{})
}

// OneShotAt creates a temporary entity at pos. Set [Settings.Spatial] to enable
// listener-relative playback. It returns an error if the source or settings is
// invalid or settings requests looping.
func OneShotAt(world *core.World, src Source, s Settings, pos geom.Vector2) (core.Entity, error) {
	return oneShot(world, src, s, pos)
}

func oneShot(world *core.World, src Source, s Settings, pos geom.Vector2) (core.Entity, error) {
	if s.Loop == LoopTrack || s.Loop == LoopAll {
		return core.Entity{}, fmt.Errorf("audio: one-shot must not loop")
	}
	player, err := NewPlayer(src, s)
	if err != nil {
		return core.Entity{}, err
	}
	return world.NewEntity(player, core.Transform{Position: pos}, oneshot{})
}
