package audio

import (
	"fmt"
	"maps"
	"math"

	"github.com/Leonard-Atorough/castrum/settings"
)

// Preferences holds JSON-serializable desired mixer levels.
//
// It does not apply them to a mixer.
type Preferences struct {
	// Master is the master volume in [0, 1].
	Master float64 `json:"master"`
	// Buses maps bus names to desired volumes in [0, 1].
	Buses map[string]float64 `json:"buses"`
}

// NewPreferences returns a validated group with unity mixer levels.
func NewPreferences() (*settings.Group[Preferences], error) {
	return settings.New(Preferences{Master: 1, Buses: map[string]float64{}}, settings.Policy[Preferences]{
		Validator: settings.ValidatorFunc[Preferences](Preferences.Validate),
		Cloner:    settings.ClonerFunc[Preferences](Preferences.Clone),
	})
}

// Validate reports an error if a mixer level is not finite or outside [0, 1].
func (value Preferences) Validate() error {
	if math.IsNaN(value.Master) || math.IsInf(value.Master, 0) || value.Master < 0 || value.Master > 1 {
		return fmt.Errorf("audio: master volume must be finite and in [0, 1], got %v", value.Master)
	}
	for name, busValue := range value.Buses {
		if math.IsNaN(busValue) || math.IsInf(busValue, 0) || busValue < 0 || busValue > 1 {
			return fmt.Errorf("audio: bus %s volume must be finite and in [0, 1], got %v", name, busValue)
		}
	}
	return nil
}

// Clone returns a copy of the preferences and their bus volumes.
func (value Preferences) Clone() Preferences {
	value.Buses = maps.Clone(value.Buses)
	return value
}
