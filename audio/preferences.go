package audio

import (
	"fmt"
	"math"

	"github.com/Leonard-Atorough/castrum/settings"
)

// Preferences holds desired mixer levels. It does not apply them to a [Mixer].
type Preferences struct {
	// Master is the master volume in [0, 1].
	Master float64
	// Music is the music bus volume in [0, 1].
	Music float64
	// SFX is the sound-effects bus volume in [0, 1].
	SFX float64
}

// NewPreferences returns a validated group with unity mixer levels.
func NewPreferences() (*settings.Group[Preferences], error) {
	return settings.New(Preferences{Master: 1, Music: 1, SFX: 1}, settings.Policy[Preferences]{
		Validator: settings.ValidatorFunc[Preferences](Preferences.Validate),
	})
}

// Validate reports an error if a mixer level is not finite or outside [0, 1].
func (value Preferences) Validate() error {
	for _, field := range []struct {
		name  string
		value float64
	}{{"master", value.Master}, {"music", value.Music}, {"sfx", value.SFX}} {
		if math.IsNaN(field.value) || math.IsInf(field.value, 0) || field.value < 0 || field.value > 1 {
			return fmt.Errorf("audio: %s volume must be finite and in [0, 1], got %v", field.name, field.value)
		}
	}
	return nil
}
