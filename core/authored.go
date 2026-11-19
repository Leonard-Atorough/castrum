package core

// Authored is the opt-in contract for validating components in authored files.
//
// Its rules may be stricter than [Validatable], allowing runtime zero values
// while rejecting incomplete file content. File loading calls Authored on
// component values; runtime entity creation does not. Implement Authored with
// a value receiver so file loading can detect it.
type Authored interface {
	// Authored returns an error if the component is unsuitable for authored
	// file content.
	Authored() error
}
