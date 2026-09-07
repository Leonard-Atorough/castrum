package core

// QueryFor returns all entities that have at least a component of type T.
func QueryFor[T Component](w *World) []EntityID {
	var zero T
	return w.NewQuery().WithRequiredComponents(zero).EntityIDs()
}
