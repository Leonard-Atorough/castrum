package components

import "fmt"

// PlaybackMode controls the entity lifecycle when a track finishes playing.
// The audio system reads this to decide whether to despawn the entity or keep
// it with Playing set to false.
type PlaybackMode int

const (
	// PlaybackPersist keeps the entity after playback completes. The audio
	// system sets Playing to false. Use for tracks you want to restart or
	// reconfigure at runtime, such as background music.
	PlaybackPersist PlaybackMode = iota
	// PlaybackDespawn removes the entity when playback completes. Use for
	// one-shot sound effects spawned as transient entities.
	PlaybackDespawn
)

// AudioPlayer is the per-entity component for audio playback. It references a
// track by ID in the audio system's AudioService. The track definition (asset
// data, group, loop mode, default volume) lives on the track, not here.
//
// Field roles:
//   - TrackID, Volume, Mode, and Autoplay are configuration set at creation time.
//   - Playing is runtime state mutated by the audio system.
type AudioPlayer struct {
	TrackID  string
	Volume   float64
	Mode     PlaybackMode
	Playing  bool
	Autoplay bool
}

// NewAudio creates an AudioPlayer for the given track ID. Volume is clamped
// to [0, 1]. When autoplay is true, Playing is set to true so the audio system
// begins playback immediately on the first update tick.
func NewAudio(trackID string, volume float64, mode PlaybackMode, autoplay bool) AudioPlayer {
	if volume < 0 {
		volume = 0
	} else if volume > 1 {
		volume = 1
	}
	return AudioPlayer{
		TrackID:  trackID,
		Volume:   volume,
		Mode:     mode,
		Playing:  autoplay,
		Autoplay: autoplay,
	}
}

// Validate checks that the Audio has a non-empty TrackID.
func (a AudioPlayer) Validate() error {
	if a.TrackID == "" {
		return fmt.Errorf("TrackID cannot be empty")
	}
	return nil
}

// Serialize converts the Audio to a map suitable for blueprint serialization
// or save-game storage. All fields are included.
func (a AudioPlayer) Serialize() (map[string]any, error) {
	return map[string]any{
		"trackID":  a.TrackID,
		"volume":   a.Volume,
		"mode":     float64(a.Mode),
		"playing":  a.Playing,
		"autoplay": a.Autoplay,
	}, nil
}

// Deserialize populates an Audio from a serialized map, returning the
// reconstructed component. The result is validated before returning.
func (a AudioPlayer) Deserialize(data map[string]any) (AudioPlayer, error) {
	if v, ok := data["trackID"].(string); ok {
		a.TrackID = v
	}
	if v, ok := data["volume"].(float64); ok {
		a.Volume = v
	}
	if v, ok := data["mode"].(float64); ok {
		a.Mode = PlaybackMode(int(v))
	}
	if v, ok := data["playing"].(bool); ok {
		a.Playing = v
	}
	if v, ok := data["autoplay"].(bool); ok {
		a.Autoplay = v
	}
	return a, a.Validate()
}
