// Package audio defines audio playback: the [Source] component holds
// an audio file by [asset.ID] plus per-play settings and state, and
// the [Mixer] resource holds the global volume buses and pause state.
// The package is backend-free: a Source references its audio file by
// asset ID, and the runner decodes and plays it.
package audio

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/asset"
)

// LoadMode selects how a track's audio reaches players.
type LoadMode int

const (
	LoadEager  LoadMode = iota // zero value: decode once, share bytes
	LoadStream                 // opt-in: one open reader per player
)

// Group selects the volume bus a track mixes through.
type Group int

const (
	GroupSFX Group = iota // zero value: most spawns are effects
	GroupMusic
)

// LoopMode controls how a track repeats after its last sample.
type LoopMode int

const (
	LoopNone    LoopMode = iota // zero value: Source holds completed
	LoopForever                 // wraps to the start
)

// PauseMode controls what a global [Mixer] pause does to a play.
type PauseMode int

const (
	PauseHolds     PauseMode = iota // zero value: holds position with the game
	PauseContinues                  // UI/menu audio: keeps playing through pause
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
	// Paused holds this play's position without losing it; write the
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
// and the global pause; a runner applies them to its players. All
// levels are runtime state with 1 as unity, set explicitly or left at
// the [NewMixer] defaults.
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

// Master returns the master volume level, from 0 (silence) to 1
// (unity).
func (m *Mixer) Master() float64 {
	return m.master
}

// SetMaster sets the master volume level, clamped to [0, 1].
func (m *Mixer) SetMaster(volume float64) {
	m.master = clamp01(volume)
}

// GroupVolume returns the group's volume level, from 0 (silence) to 1
// (unity). A group without an override reads unity.
func (m *Mixer) GroupVolume(group Group) float64 {
	if volume, ok := m.groups[group]; ok {
		return volume
	}
	return 1
}

// SetGroupVolume sets the group's volume level, clamped to [0, 1].
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

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
