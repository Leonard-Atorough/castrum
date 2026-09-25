package core

// Validatable is the opt-in contract for spawn-time component
// validation. Implement it on components with value constraints;
// the engine calls Validate whenever the component enters storage.
type Validatable interface {
	// Validate checks that the component's values satisfy its
	// constraints.
	Validate() error
}
