package components

import "fmt"

// AudioPlayer is the per-entity component for audio playback. It references a track
// by ID in the audio system's AudioStore. The track definition (asset data,
// group, loop mode, default volume) lives on the track, not here — AudioPlayer
// only carries the runtime playback state.
//
// Field roles:
//   - TrackID and Autoplay are configuration set at creation time.
//   - Playing is runtime state mutated by the audio system.
type AudioPlayer struct {
	TrackID  string
	Playing  bool
	Autoplay bool
}

// NewAudio creates an Audio for the given track ID. When autoplay is true,
// Playing is set to true so the audio system begins playback immediately on
// the first update tick.
func NewAudio(trackID string, autoplay bool) AudioPlayer {
	return AudioPlayer{
		TrackID:  trackID,
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
	if v, ok := data["playing"].(bool); ok {
		a.Playing = v
	}
	if v, ok := data["autoplay"].(bool); ok {
		a.Autoplay = v
	}
	return a, a.Validate()
}
