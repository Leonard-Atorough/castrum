// Package audio defines backend-independent audio playback. [Source]
// components describe individual plays, [Mixer] holds global volume and
// pause state, and [NewAudioSystem] reconciles them through a runner's
// [Controller]. [OneShot] starts a play on an engine-owned entity and reclaims
// it when playback finishes. Audio files are identified by [asset.ID].
package audio

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
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

// Source describes an audio play attached to a [core.Entity]. The audio system
// applies its settings to a runner's player.
//
// The zero value is invalid. Use [NewSource] to create a source at unity
// volume, and set Audio to a non-empty asset ID.
type Source struct {
	// Audio is the asset ID of the audio file to play.
	Audio asset.ID
	// Group is the volume bus the play mixes through.
	Group Group
	// Volume is the per-play level. [Source.Validate] rejects values less than
	// or equal to zero or greater than one; use a bus to mute its plays.
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
	// restart, including an Audio change.
	Completed      bool
	restarts       uint64
	syncedRestarts uint64
	syncedAudio    asset.ID
}

// Validate reports an error if Audio is empty, Volume is less than or equal
// to zero or greater than one, PlaybackMultiplier is negative, or Group,
// Loop, Pause, or Load has an unsupported value.
func (s Source) Validate() error {
	if s.Audio == "" {
		return fmt.Errorf("audio: source audio ID must not be empty")
	}
	if s.Volume <= 0 || s.Volume > 1 {
		return fmt.Errorf("audio: source volume must be in (0, 1], got %v", s.Volume)
	}
	if s.PlaybackMultiplier < 0 {
		return fmt.Errorf("audio: source playback multiplier must not be negative (0 is the normal rate), got %v", s.PlaybackMultiplier)
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
	return nil
}

// NewSource creates a source at unity volume with all other settings at their
// zero values. The audio ID must be non-empty for the source to pass
// [Source.Validate].
func NewSource(audio asset.ID) Source {
	return Source{Audio: audio, Volume: 1}
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
func (m *Mixer) SetMaster(volume float64) {
	m.master = clamp01(volume)
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
func (m *Mixer) SetGroupVolume(group Group, volume float64) {
	m.groups[group] = clamp01(volume)
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

// Playback is the resolved state that a [Controller] applies to a play.
// [NewAudioSystem] derives it from a [Source] and a [Mixer].
type Playback struct {
	// Playing reports whether playback should advance. False holds its position.
	Playing bool
	// Restart requests a fresh player from the beginning. The audio system
	// sets it on the first sync, an Audio change, or [Source.Restart].
	Restart bool
	// Volume is the final mix — master x group x play — in [0, 1].
	// Zero is silence: runtime fade targets are safe here.
	Volume float64
	// Rate is the resolved playback multiplier; 1 is the normal rate.
	Rate float64
	// Loop is the play's repeat behavior after the last sample.
	Loop LoopMode
	// Load selects eager decode or per-play streaming.
	Load LoadMode
}

// Controller applies resolved [Playback] state to backend players. Runners
// implement it and pass it to [NewAudioSystem].
type Controller interface {
	// Sync applies want to the player for entityID and source. live reports
	// whether the play has not finished, not whether it is currently playing:
	// a paused play is live, while a finished [LoopNone] play is not.
	//
	// A finished play must not resume merely because want.Playing is true;
	// want.Restart requests an explicit restart. The audio system calls Sync
	// for each [Source] on every run.
	Sync(entityID core.EntityID, source asset.ID, want Playback) (live bool, err error)
	// Sweep releases players for entities not passed to Sync since the
	// previous sweep. The audio system calls it after each reconciliation pass.
	Sweep() error
}

// NewAudioSystem returns a [core.System] that reconciles [Source] components
// with controller and mixer, and destroys completed [OneShot] entities.
// Without an audio system, sources do not play.
func NewAudioSystem(controller Controller, mixer *Mixer) core.System {
	var sourceQuery, oneshotQuery *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if sourceQuery == nil {
			sourceQuery = core.NewQuery(ctx.World).With(Source{})
			oneshotQuery = core.NewQuery(ctx.World).With(Source{}, oneshot{})
		}
		for e := range sourceQuery.Execute() {
			s, _ := e.Component[Source]()

			restart := s.syncedRestarts != s.restarts || s.syncedAudio != s.Audio
			if restart {
				s.Completed = false
				s.syncedRestarts = s.restarts
				s.syncedAudio = s.Audio
			}

			want := fold(s, mixer)
			want.Restart = restart
			live, err := controller.Sync(e.ID(), s.Audio, want)
			if err != nil {
				return fmt.Errorf("audio: entity %d: %w", e.ID(), err)
			}

			changed := restart
			if !live && s.Loop == LoopNone && !s.Completed {
				s.Completed = true
				changed = true
			}
			if changed {
				e.SetComponent(s)
			}
		}

		var done []*core.Entity
		for e := range oneshotQuery.Execute() {
			s, _ := e.Component[Source]()
			if !s.Completed {
				continue
			}
			shot, _ := e.Component[oneshot]()
			done = append(done, shot.self)
		}
		for _, entity := range done {
			if err := ctx.World.DestroyEntity(entity); err != nil {
				return fmt.Errorf("audio: destroy one-shot: %w", err)
			}
		}

		if err := controller.Sweep(); err != nil {
			return fmt.Errorf("audio: sweep: %w", err)
		}
		return nil
	})
}

// oneshot retains the handle so cleanup destroys only the entity spawned by
// [OneShot].
type oneshot struct {
	self *core.Entity
}

// OneShot starts src on a new engine-owned entity and returns its handle. The
// audio system destroys the entity when playback finishes; it never destroys
// entities that carry ordinary [Source] components.
//
// The source must not loop: a [LoopForever] play never finishes and its
// entity would not be reclaimed. Use a regular [Source] for looping audio.
// OneShot returns an error for a looping or invalid source.
func OneShot(world *core.World, src Source) (*core.Entity, error) {
	if src.Loop == LoopForever {
		return nil, fmt.Errorf("audio: one-shot %q must not loop: use a Source on a game entity", src.Audio)
	}
	entity, err := world.NewEntity(src)
	if err != nil {
		return nil, err
	}
	if err := entity.AddComponent(world, oneshot{self: entity}); err != nil {
		return nil, err
	}
	return entity, nil
}

func fold(s Source, mixer *Mixer) Playback {
	return Playback{
		Playing: !s.Paused && (!mixer.Paused() || s.Pause == PauseContinues),
		Volume:  s.Volume * mixer.Master() * mixer.GroupVolume(s.Group),
		Rate:    resolveMultiplier(s.PlaybackMultiplier),
		Loop:    s.Loop,
		Load:    s.Load,
	}
}

func resolveMultiplier(multiplier float64) float64 {
	if multiplier == 0 {
		return 1
	}
	return multiplier
}
