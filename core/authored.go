package core

// Authored reports whether a component value is acceptable as authored
// file content. File data is written deliberately, so a component that
// would draw nothing - a text with no font, a sprite with no asset -
// is an authoring mistake even though its zero value is valid runtime
// state. Data boundaries reject Authored failures; runtime spawn does
// not.
type Authored interface {
	// Authored reports an error naming the rule when the value would
	// contribute nothing to a file-authored entity.
	Authored() error
}
