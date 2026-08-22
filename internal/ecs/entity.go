package ecs

type EntityID uint64

type entity struct {
	id       EntityID
	template string
	alive    bool
	version  uint32
}

func NewEntity(id EntityID, template string) *entity {
	return &entity{
		id:       id,
		template: template,
		alive:    true,
		version:  0,
	}
}

// ID returns the unique identifier of the entity.
func (e *entity) ID() EntityID { return e.id }

// Template returns the template name of the entity.
func (e *entity) Template() string { return e.template }

// IsAlive returns true if the entity is currently alive.
func (e *entity) IsAlive() bool { return e.alive }

// Destroy marks the entity as no longer alive.
func (e *entity) Destroy() { e.alive = false }

func (e *entity) Version() uint32 { return e.version }

func (e *entity) Clone(newID EntityID) *entity {
	return &entity{
		id:       newID,
		template: e.template,
		alive:    e.alive,
		version:  e.version,
	}
}
