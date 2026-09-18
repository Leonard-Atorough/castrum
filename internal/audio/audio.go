// Package audio defines audio playback tracks and the service that plays them.
//
// An [AudioTrack] is an immutable definition created via [NewAudioTrack]:
// it holds the decoded PCM bytes plus playback configuration (group, loop
// mode, volume). Tracks are registered with an [AudioService] and referenced
// by [ID] from the [components.AudioPlayer] component, which carries only the
// track ID and per-entity runtime state.
//
// [AudioService] owns the track registry and the per-entity players. It
// resolves a track's PCM data at playback time to create a player and applies
// a volume mix of master, group, track, and per-entity volumes.
//
// Audio data (raw PCM bytes decoded from WAV, MP3, or OGG) is loaded and
// cached separately by the [assets.Loader]; the track stores the decoded
// bytes, not the asset reference.
package audio

// ID identifies an audio track within an [AudioService].
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
	// LoopForever repeats the track indefinitely.
	LoopForever
)

// AudioTrack is an immutable audio definition: decoded PCM data plus playback
// configuration. Tracks are created via [NewAudioTrack] and registered with
// an [AudioService]. All fields are private and read via accessors to prevent
// mutation after construction.
type AudioTrack struct {
	id     ID
	pcm    []byte
	group  Group
	loop   LoopMode
	volume float64
}

// NewAudioTrack creates an AudioTrack with the given definition. Volume is
// clamped to [0, 1].
func NewAudioTrack(id ID, pcm []byte, group Group, loop LoopMode, volume float64) *AudioTrack {
	if volume < 0 {
		volume = 0
	} else if volume > 1 {
		volume = 1
	}
	return &AudioTrack{
		id:     id,
		pcm:    pcm,
		group:  group,
		loop:   loop,
		volume: volume,
	}
}

// ID returns the track identifier used to look up the track in an [AudioService].
func (t *AudioTrack) ID() ID {
	return t.id
}

// Data returns the decoded PCM bytes the track plays.
func (t *AudioTrack) Data() []byte {
	return t.pcm
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
