package scene

import (
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/core"
)

// Director queues scene requests but does not execute them immediately.
// It manages scene instances and their lifecycle.
type Director struct {
	nextID    InstanceID
	pending   []request
	instances map[InstanceID]*instance
	roots     []InstanceID
}

type request struct {
	kind     requestKind
	id       InstanceID
	name     string
	nameSet  bool
	hooks    Hooks
	hooksSet bool
	scene    Scene
	props    any
	tree     Node
}

type requestKind uint8

const (
	requestSwitch requestKind = iota
	requestLoad
	requestUpdate
	requestUnload
)

func (k requestKind) needsScene() bool { return k != requestUnload }

// NewDirector creates an empty [Director].
func NewDirector() *Director {
	return &Director{
		nextID:    1,
		instances: make(map[InstanceID]*instance),
	}
}

// Switch builds and validates scene and queues a switch request.
func (d *Director) Switch(scene Scene, props any, opts ...Option) (InstanceID, error) {
	return d.queue(0, scene, props, requestSwitch, "switch", opts...)
}

// Load builds and validates scene and queues a load request.
func (d *Director) Load(scene Scene, props any, opts ...Option) (InstanceID, error) {
	return d.queue(0, scene, props, requestLoad, "load", opts...)
}

// Update builds and validates scene and queues a request to rebuild the
// instance id from props when the request executes. It returns an error if
// id is zero or the tree is invalid.
func (d *Director) Update(id InstanceID, scene Scene, props any, opts ...Option) error {
	if id == 0 {
		return fmt.Errorf("scene: update: invalid instance id 0")
	}
	_, err := d.queue(id, scene, props, requestUpdate, "update", opts...)
	return err
}

// Remove queues a request to unload the specified instance. It returns an
// error if id is zero.
func (d *Director) Remove(id InstanceID) error {
	if id == 0 {
		return fmt.Errorf("scene: remove: invalid instance id 0")
	}
	_, err := d.queue(id, nil, nil, requestUnload, "remove")
	return err
}

// Spawn creates an entity owned by the instance owner. The entity is
// destroyed along with the instance when it unloads. It returns an error if
// ctx is nil, owner does not identify a live instance, or the entity cannot
// be created.
func (d *Director) Spawn(ctx *core.Context, owner InstanceID, components ...any) (core.Entity, error) {
	if ctx == nil || ctx.World == nil {
		return core.Entity{}, fmt.Errorf("scene: spawn: context and world must not be nil")
	}
	inst, ok := d.instances[owner]
	if !ok {
		return core.Entity{}, fmt.Errorf("scene: spawn: invalid owner instance id %d", owner)
	}
	if !inst.live {
		return core.Entity{}, fmt.Errorf("scene: spawn: owner instance id %d is not live", owner)
	}
	ent, err := ctx.World.NewEntity(components...)
	if err != nil {
		return core.Entity{}, err
	}
	inst.entities = append(inst.entities, ent)
	return ent, nil
}

// Live reports whether id identifies a live instance.
func (d *Director) Live(id InstanceID) bool {
	inst, ok := d.instances[id]
	return ok && inst.live
}

// Name returns the name of the instance id. It reports false if id is unknown.
func (d *Director) Name(id InstanceID) (string, bool) {
	inst, ok := d.instances[id]
	if !ok {
		return "", false
	}
	return inst.name, true
}

// Entities returns a copy of the entities owned by the instance id, or nil
// if id is unknown.
func (d *Director) Entities(id InstanceID) []core.Entity {
	inst, ok := d.instances[id]
	if !ok {
		return nil
	}
	return slices.Clone(inst.entities)
}

func (d *Director) queue(target InstanceID, scene Scene, props any, kind requestKind, operation string, opts ...Option) (InstanceID, error) {
	req := request{}
	for _, opt := range opts {
		opt(&req)
	}
	if kind.needsScene() {
		if scene == nil {
			return 0, fmt.Errorf("scene: %s %q: nil scene", operation, req.name)
		}
		tree, err := validate(scene, props)
		if err != nil {
			return 0, err
		}
		req.tree = cloneTree(tree)
	}

	req.kind = kind
	req.scene = scene
	req.props = props
	if target == 0 {
		target = d.nextID
		d.nextID++
	}
	req.id = target

	if !d.coalesce(req) {
		d.pending = append(d.pending, req)
	}
	return req.id, nil
}

// coalesce keeps pending as the batch's end state: one pending request per
// id and at most one switch, so the last request for an id wins. Slots are
// updated in place to keep execution order stable. It reports whether req
// was absorbed into a pending request and must not be queued.
func (d *Director) coalesce(req request) bool {
	if req.kind == requestLoad {
		return false
	}
	for i := range d.pending {
		p := &d.pending[i]
		switch {
		case req.kind == requestSwitch && p.kind == requestSwitch:
			*p = req
			return true
		case req.kind == requestSwitch || p.id != req.id:
			continue
		case p.kind == requestUnload && req.kind == requestUnload:
			return true
		case req.kind == requestUnload && p.kind != requestUpdate:
			// a load or switch that has not entered has nothing to unload,
			// so it is cancelled instead
			d.pending = append(d.pending[:i], d.pending[i+1:]...)
			return true
		default:
			// an update replaces queued content, and update and exit flip
			// one slot; both keep any name and hooks earlier requests set
			if req.nameSet {
				p.name, p.nameSet = req.name, true
			}
			if req.hooksSet {
				p.hooks, p.hooksSet = req.hooks, true
			}
			p.scene, p.props, p.tree = req.scene, req.props, req.tree
			if p.kind != requestLoad && p.kind != requestSwitch {
				p.kind = req.kind
			}
			return true
		}
	}
	return false
}

// Option configures a scene request before it is queued.
type Option func(*request)

// WithName sets a scene request's name.
func WithName(name string) Option {
	return func(r *request) {
		r.name = name
		r.nameSet = true
	}
}

// WithHooks sets a scene request's lifecycle callbacks.
func WithHooks(h Hooks) Option {
	return func(r *request) {
		r.hooks = h
		r.hooksSet = true
	}
}
