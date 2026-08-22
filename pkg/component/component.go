package component

type Component interface {
	Name() string
	Clone() Component
}
