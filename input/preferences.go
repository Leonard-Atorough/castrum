package input

import "github.com/Leonard-Atorough/castrum/settings"

// Preferences holds desired action bindings.
type Preferences struct {
	// Bindings maps actions to physical inputs.
	Bindings Bindings
}

// NewPreferences returns a group containing a validated copy of bindings.
// It returns an error for nil or invalid bindings.
func NewPreferences(bindings Bindings) (*settings.Group[Preferences], error) {
	return settings.New(Preferences{Bindings: bindings}, settings.Policy[Preferences]{
		Validator: settings.ValidatorFunc[Preferences](Preferences.Validate),
		Cloner:    settings.ClonerFunc[Preferences](Preferences.Clone),
	})
}

// Validate reports an error if the action bindings are nil or invalid.
func (p Preferences) Validate() error {
	return validateBindings(p.Bindings)
}

// Clone returns a copy of the preferences, including binding and chord slices.
func (p Preferences) Clone() Preferences {
	if p.Bindings != nil {
		p.Bindings = cloneBindings(p.Bindings)
	}
	return p
}
