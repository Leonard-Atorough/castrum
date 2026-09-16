package ecs

type Component any

type ComponentHooks interface {
	OnCreate(entityID EntityID)
	OnDestroy(entityID EntityID)
}

type Serializable[T any] interface {
	Serialize() (map[string]any, error)
	Deserialize(map[string]any) (T, error)
}

type Validatable interface {
	Validate() error
}
