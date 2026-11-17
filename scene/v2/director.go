package scenev2

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
)

// Director queues scene requests but does not yet execute them.
//
// [Director.Switch] and [Director.Load] currently return the next ID, not
// the ID reserved for the queued request.
type Director struct {
	nextID    InstanceID
	pending   []request
	instances map[InstanceID]*instance
	roots     []InstanceID
}

type request struct {
	kind  requestKind
	id    InstanceID
	name  string
	scene Scene
	props any
	hooks Hooks
	tree  Node
}

type requestKind uint8

const (
	requestSwitch requestKind = iota
	requestLoad
	requestUpdate
	requestUnload
)

// NewDirector creates an empty [Director].
func NewDirector() *Director {
	return &Director{
		nextID:    1,
		instances: make(map[InstanceID]*instance),
	}
}

// Switch builds and validates scene and queues a switch request.
func (d *Director) Switch(scene Scene, props any, opts ...Option) (InstanceID, error) {
	return d.queue(scene, props, requestSwitch, "switch", opts...)
}

// Load builds and validates scene and queues a load request.
func (d *Director) Load(scene Scene, props any, opts ...Option) (InstanceID, error) {
	return d.queue(scene, props, requestLoad, "load", opts...)
}

// Update builds and validates scene and queues an update request.
//
// The id parameter is not yet used.
func (d *Director) Update(id InstanceID, scene Scene, props any, opts ...Option) error {
	_, err := d.queue(scene, props, requestUpdate, "update", opts...)
	return err
}

// Remove is reserved for removing an instance. It is not yet implemented.
func (d *Director) Remove(id InstanceID) error

// Spawn is reserved for creating an instance-owned entity. It is not yet implemented.
func (d *Director) Spawn(ctx *core.Context, owner InstanceID, components ...any) (core.Entity, error)

func (d *Director) queue(scene Scene, props any, kind requestKind, operation string, opts ...Option) (InstanceID, error) {
	req := request{}
	for _, opt := range opts {
		opt(&req)
	}
	if scene == nil {
		return 0, fmt.Errorf("%s: load %q: nil scene", operation, req.name)
	}
	tree, err := validate(scene, props)
	if err != nil {
		return 0, err
	}

	req.kind = kind
	req.scene = scene
	req.props = props
	req.tree = tree
	req.id = d.nextID

	d.nextID++
	d.pending = append(d.pending, req)
	return d.nextID, nil
}

// Option configures a scene request before it is queued.
type Option func(*request)

// WithName sets a scene request's name.
func WithName(name string) Option {
	return func(r *request) {
		r.name = name
	}
}

// WithHooks sets a scene request's lifecycle callbacks.
func WithHooks(h Hooks) Option {
	return func(r *request) {
		r.hooks = h
	}
}
