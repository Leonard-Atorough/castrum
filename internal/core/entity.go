// Package ecs provides a lightweight entity component system (ECS) implementation.
// It manages entities, their components, tags, and hierarchical relationships.
package core

type EntityID uint64

// Entity represents a game Entity with an ID, template, and lifecycle state.
// Entities are not directly exposed in the public API; they are accessed through World methods.
type Entity struct {
	ID       EntityID
	template string
	alive    bool
	version  uint32

	archetypeID  uint64 // ID of the archetype this entity belongs to
	archetypeIdx int    // Index of the entity within its archetype's entity slice
}

func NewEntity(id EntityID, template string) *Entity {
	return &Entity{
		ID:       id,
		template: template,
		alive:    true,
		version:  0,
	}
}

// Template returns the template name of the entity.
func (e *Entity) Template() string { return e.template }

// IsAlive returns true if the entity is currently alive.
func (e *Entity) IsAlive() bool { return e.alive }

// Destroy marks the entity as no longer alive.
func (e *Entity) Destroy() { e.alive = false }

func (e *Entity) Version() uint32 { return e.version }

func (e *Entity) Clone(newID EntityID) *Entity {
	return &Entity{
		ID:       newID,
		template: e.template,
		alive:    e.alive,
		version:  e.version,
	}
}
