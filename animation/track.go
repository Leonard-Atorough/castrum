package animation

import (
	"fmt"
	"math"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
)

// Track is one layer of an [Entry]: a named sequence of sheet regions at a
// fixed rate. Tracks in one entry play together (layers).
type Track struct {
	// Name is the track's name.
	Name string `json:"name"`
	// Source is the sheet document ID.
	Source asset.ID `json:"source"`
	// Frames are the sheet region names in playback order.
	Frames []string `json:"frames"`
	// FPS is the playback rate in frames per second.
	FPS float64 `json:"fps"`
}

// SheetRef returns the track's sheet as a file reference.
func (t Track) SheetRef() asset.SheetRef {
	return asset.FileSheet(t.Source)
}

// Validate returns an error if the track has an invalid name, source, frames,
// or playback rate. FPS must be finite, positive, and at most 60.
func (t Track) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("animation: track must have a name")
	}
	if err := asset.FileSheet(t.Source).Validate(); err != nil {
		return fmt.Errorf("animation: track %q source: %w", t.Name, err)
	}
	if len(t.Frames) == 0 {
		return fmt.Errorf("animation: track %q must have at least one frame", t.Name)
	}
	for i, frame := range t.Frames {
		if frame == "" {
			return fmt.Errorf("animation: track %q frame %d must have a name", t.Name, i)
		}
	}
	if !isFinite(t.FPS) || t.FPS <= 0 {
		return fmt.Errorf("animation: track %q fps must be finite and positive, got %v", t.Name, t.FPS)
	}
	if t.FPS > 60 {
		return fmt.Errorf("animation: track %q fps must not exceed 60, got %v", t.Name, t.FPS)
	}
	return nil
}

func (t Track) clone() Track {
	out := t
	out.Frames = slices.Clone(t.Frames)
	return out
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
