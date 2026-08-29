// Package ecs provides a lightweight entity component system (ECS) implementation.
// It manages entities, their components, tags, and hierarchical relationships.
package core

type EntityID uint64

// Entity represents a game Entity with an ID, template, tags, and lifecycle state.
// Entities are not directly exposed in the public API; they are accessed through World methods.
type Entity struct {
	id       EntityID
	template string
	tags     map[string]struct{}
	alive    bool
	version  uint32

	archetypeID  uint64 // ID of the archetype this entity belongs to
	archetypeIdx int    // Index of the entity within its archetype's entity slice
}

func NewEntity(id EntityID, template string) *Entity {
	return &Entity{
		id:       id,
		template: template,
		tags:     make(map[string]struct{}),
		alive:    true,
		version:  0,
	}
}

// ID returns the unique identifier of the entity.
func (e *Entity) ID() EntityID { return e.id }

// Template returns the template name of the entity.
func (e *Entity) Template() string { return e.template }

// IsAlive returns true if the entity is currently alive.
func (e *Entity) IsAlive() bool { return e.alive }

// Tags returns a copy of the entity's current tag set.
func (e *Entity) Tags() []string {
	if len(e.tags) == 0 {
		return nil
	}
	out := make([]string, 0, len(e.tags))
	for tag := range e.tags {
		out = append(out, tag)
	}
	return out
}

// HasTag reports whether the entity currently has the given tag.
func (e *Entity) HasTag(tag string) bool {
	_, ok := e.tags[tag]
	return ok
}

// AddTag adds a tag to the entity if it is not already present.
func (e *Entity) AddTag(tag string) {
	if tag == "" {
		return
	}
	e.tags[tag] = struct{}{}
}

// RemoveTag removes a tag from the entity.
func (e *Entity) RemoveTag(tag string) {
	delete(e.tags, tag)
}

// Destroy marks the entity as no longer alive.
func (e *Entity) Destroy() { e.alive = false }

func (e *Entity) Version() uint32 { return e.version }

func (e *Entity) Clone(newID EntityID) *Entity {
	clone := &Entity{
		id:       newID,
		template: e.template,
		tags:     make(map[string]struct{}, len(e.tags)),
		alive:    e.alive,
		version:  e.version,
	}
	for tag := range e.tags {
		clone.tags[tag] = struct{}{}
	}
	return clone
}
