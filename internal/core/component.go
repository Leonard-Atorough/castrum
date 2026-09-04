package core

type Component any

type ComponentHooks interface {
	OnCreate(entityID EntityID)
	OnDestroy(entityID EntityID)
}

type Serializable interface {
	Serialize() (map[string]any, error)
	Deserialize(map[string]any) error
}
