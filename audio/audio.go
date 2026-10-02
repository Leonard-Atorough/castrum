// Package audio defines audio playback: the [Source] component holds
// an audio file by [asset.ID] plus per-play settings and state, the
// [Mixer] resource holds the global volume buses and pause state,
// and the [Controller] seam hands resolved [Playback] state to a
// runner's players. [OneShot] is the fire-and-forget verb: it plays
// a Source once on an engine-owned entity and reclaims it when the
// play finishes. The package is backend-free: a Source references
// its audio file by asset ID, and the runner decodes and plays it.
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
	// LoadStream keeps the audio in the original source and streams it to each player.
	LoadStream
)

// Group selects the volume bus a play mixes through.
type Group int

const (
	// GroupSFX is the sound effects bus; most spawns are effects.
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
	// PauseHolds means the play holds its position when the mixer is paused.
	PauseHolds PauseMode = iota
	// PauseContinues means the play keeps playing even when the mixer is paused.
	PauseContinues
	// PauseDucks: reserved — arrives with transition curves
)

// Source is the per-entity audio component and the playback handle:
// an audio file reference plus per-play settings and playback state.
// The reconcile system makes the runner's players match it, so a game
// plays, pauses, or retargets audio by writing fields.
//
// Source has no valid zero value — an empty Audio is a spawn error —
// so every real spawn states its settings explicitly.
type Source struct {
	// Audio is the asset ID of the audio file to play.
	Audio asset.ID
	// Group is the volume bus the play mixes through.
	Group Group
	// Volume is the per-play level in (0, 1]; 0 is rejected at
	// spawn. Muting belongs to the buses, not the play.
	Volume float64
	// Loop is the repeat behavior after the last sample.
	Loop LoopMode
	// Pause is the behavior when the mixer is globally paused.
	Pause PauseMode
	// Load selects how the audio reaches players.
	Load LoadMode
	// PlaybackMultiplier scales the playback rate. 0 — the zero
	// value — is the normal rate; 2 is double speed.
	PlaybackMultiplier float64
	// Paused holds the play's position without losing it; write the
	// field directly. It composes with the mixer's global pause
	// inside the reconcile fold, respecting [PauseMode].
	Paused bool
	// Completed reports a finished [LoopNone] play. It is written by
	// the reconcile system; a restart — an Audio change or a
	// [Source.Restart] call — is the only thing that clears it.
	Completed bool
	// restarts is the retrigger edge: Restart bumps it, and the
	// system restarts the player when it changes.
	restarts uint64
	// syncedRestarts and syncedAudio hold the last edge the system
	// synced to the provider, so it can detect restarts. Written by
	// the reconcile system, never by games; they die with the entity.
	syncedRestarts uint64
	syncedAudio    asset.ID
}

// Validate implements the Validatable contract.
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

// NewSource returns a Source for the audio file, with Volume at
// unity: the result is a valid play with every other setting at its
// zero value. Adjust fields for anything beyond the defaults; the
// spawn-time validation still applies.
func NewSource(audio asset.ID) Source {
	return Source{Audio: audio, Volume: 1}
}

// Restart rewinds to the beginning and resumes: a finished play
// becomes a playing one. It clears [Source.Completed] and bumps the
// retrigger edge the reconcile system diffs. Pausing and resuming are
// plain [Source.Paused] writes.
func (s *Source) Restart() {
	s.Completed = false
	s.Paused = false
	s.restarts++
}

// Mixer is the game's global audio bus state, provided as a world
// resource by castrum.New. It owns the master and group volume levels
// and the global pause; a runner applies them to its players. Levels
// are runtime state with 1 as unity.
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

// Playback is the resolved desired state for one play: everything
// the runner needs to make its player match, and nothing it should
// read from the world itself. The reconcile system folds it from a
// [Source] and the [Mixer]; the [Controller] consumes it. Playback is
// a resolved output, not an authored input: its zeros mean what they
// say (Volume 0 is silence), and games never construct one.
type Playback struct {
	// Playing is whether the play should be audible and advancing.
	// False holds the position without losing it.
	Playing bool
	// Restart demands a fresh player from the beginning: an Audio
	// change or a [Source.Restart] bump. It is true on a play's
	// first sync and set by the system, never by the fold.
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

// Controller is the seam over a runner's actual players: the engine's
// audio system drives it, and a runner implements it over its backend
// and provides the implementation as a world resource.
type Controller interface {
	// Sync makes the playback for entityID match want. live reports
	// whether the play has not finished — a paused play is held,
	// not finished, and reports live; a finished [LoopNone] play
	// reports false and must not implicitly restart: only
	// want.Restart restarts. The reconciler calls Sync once per run
	// for each live Source. Errors name the source.
	Sync(entityID core.EntityID, source asset.ID, want Playback) (live bool, err error)
	// Sweep releases the players of entities not synced this run;
	// the reconciler calls it after each iteration.
	Sweep() error
}

// NewAudioSystem returns the engine's audio system: it reconciles
// the world's [Source] components with the given controller,
// applying the [Mixer]'s volumes and global pause, and reclaims
// finished [OneShot] entities. The runner registers it with its own
// controller; a game without a runner simply has no audio system.
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

// oneshot marks an entity spawned by [OneShot]: the reconciler
// destroys it once its play finishes. It carries the entity's own
// handle so destruction needs no by-ID lookup.
type oneshot struct {
	self *core.Entity
}

// OneShot plays src once, fire-and-forget: it spawns an
// engine-owned entity and returns its handle. The reconciler
// destroys the entity when the play finishes — the engine destroys
// only entities spawned here, never Sources attached to game
// entities. Overlapping plays are separate entities and layer
// freely; retarget or stop early through the returned handle.
//
// A one-shot must not loop: a [LoopForever] play never finishes, so
// its entity would never be reclaimed. Use a Source on a game
// entity for looping audio.
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

// fold resolves a Source and the mixer's state into the Playback the
// provider should match. Pure — the headless-tested heart of the
// engine side. Restart is not folded: the system owns the edge.
func fold(s Source, mixer *Mixer) Playback {
	return Playback{
		Playing: !s.Paused && (!mixer.Paused() || s.Pause == PauseContinues),
		Volume:  s.Volume * mixer.Master() * mixer.GroupVolume(s.Group),
		Rate:    resolveMultiplier(s.PlaybackMultiplier),
		Loop:    s.Loop,
		Load:    s.Load,
	}
}

// resolveMultiplier maps the authored zero value (0 = normal rate,
// the Animation pattern) to the resolved 1.
func resolveMultiplier(multiplier float64) float64 {
	if multiplier == 0 {
		return 1
	}
	return multiplier
}
