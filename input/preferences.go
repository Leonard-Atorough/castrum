package input

import "github.com/Leonard-Atorough/castrum/settings"

// Preferences holds JSON-serializable desired action bindings and deadzone.
type Preferences struct {
	// Bindings maps actions to physical inputs.
	Bindings Bindings `json:"bindings"`
	// Deadzone is the map-wide gamepad axis threshold. Zero selects
	// [DefaultDeadzone]; nonzero binding deadzones override it.
	Deadzone float64 `json:"deadzone"`
}

// NewPreferences returns a group containing a validated copy of bindings
// and [DefaultDeadzone].
// It returns an error for nil or invalid bindings.
func NewPreferences(bindings Bindings) (*settings.Group[Preferences], error) {
	return settings.New(Preferences{Bindings: bindings, Deadzone: DefaultDeadzone}, settings.Policy[Preferences]{
		Validator: settings.ValidatorFunc[Preferences](Preferences.Validate),
		Cloner:    settings.ClonerFunc[Preferences](Preferences.Clone),
	})
}

// Validate reports an error if the bindings are nil or invalid, or the
// deadzone is not finite and in [0, 1].
func (p Preferences) Validate() error {
	if err := validateBindings(p.Bindings); err != nil {
		return err
	}
	return validateDeadzone(p.Deadzone)
}

// Clone returns a copy of the preferences, including binding payloads and
// chord members.
func (p Preferences) Clone() Preferences {
	if p.Bindings != nil {
		p.Bindings = cloneBindings(p.Bindings)
	}
	return p
}
