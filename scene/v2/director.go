package scenev2

import "fmt"

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

func NewDirector() *Director {
	return &Director{
		nextID:    1,
		instances: make(map[InstanceID]*instance),
	}
}

func (d *Director) Switch(scene Scene, props any, opts ...Option) (InstanceID, error) {
	req := request{}
	for _, opt := range opts {
		opt(&req)
	}
	if scene == nil {
		return 0, fmt.Errorf("scene: switch %q: nil scene", req.name)
	}

	tree, err := validate(scene, props)
	if err != nil {
		return 0, fmt.Errorf("scene: switch %q: %w", req.name, err)
	}

	req.kind = requestSwitch
	req.id = d.nextID
	req.scene = scene
	req.props = props
	req.tree = tree

	d.nextID++
	d.pending = append(d.pending, req)
	return req.id, nil
}

type Option func(*request)

func WithName(name string) Option {
	return func(r *request) {
		r.name = name
	}
}

func WithHooks(h Hooks) Option {
	return func(r *request) {
		r.hooks = h
	}
}
