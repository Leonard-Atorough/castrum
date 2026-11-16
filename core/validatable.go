package core

// Validatable is the opt-in contract for component validation. Implement
// it on components with value constraints; the engine calls Validate
// before a component enters storage through the entity API -
// [World.NewEntity], [Entity.AddComponent], [Entity.SetComponent], and
// [Entity.Update]. Query entry writes ([Entry.SetComponent] and
// [Entry.Update]) bypass validation so iteration stays allocation- and
// error-free; a system that consumes a component which other systems may
// write mid-loop is responsible for calling Validate itself before use.
type Validatable interface {
	// Validate checks that the component's values satisfy its
	// constraints.
	Validate() error
}
