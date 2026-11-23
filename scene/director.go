package scene

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
)

// Director queues scene requests but does not execute them immediately.
// It manages scene instances and their lifecycle.
type Director struct {
	nextID    InstanceID
	pending   []request
	instances map[InstanceID]*instance
	roots     []InstanceID

	loader *asset.Loader
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

type loaderSource func(ctx *core.Context, tree Node) (*asset.Loader, error)

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

// Roots returns a copy of the live root instance ids, in entry order.
func (d *Director) Roots() []InstanceID {
	return slices.Clone(d.roots)
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

// execute drains one request batch in FIFO order. Requests queued from
// hooks while a batch runs land in the next batch, and one failed request
// does not stop the rest.
func (d *Director) execute(ctx *core.Context, loaderFor loaderSource) error {
	pending := d.pending
	d.pending = nil
	if len(pending) == 0 {
		return nil
	}
	var errs []error
	for _, req := range pending {
		if err := d.apply(ctx, loaderFor, req); err != nil {
			errs = append(errs, err)
		}
	}
	if ctx.World != nil {
		if am, err := ctx.World.Resource[input.Edges](); err == nil {
			am.FlushEdges()
		}
	}
	return errors.Join(errs...)
}

func (d *Director) apply(ctx *core.Context, loaderFor loaderSource, req request) error {
	switch req.kind {
	case requestSwitch:
		return d.applySwitch(ctx, loaderFor, req)
	case requestLoad:
		return d.applyLoad(ctx, loaderFor, req)
	case requestUpdate:
		return d.applyUpdate(ctx, loaderFor, req)
	case requestUnload:
		return d.applyUnload(ctx, req)
	default:
		return fmt.Errorf("scene: apply: unknown request kind %v", req.kind)
	}
}

func (d *Director) applySwitch(ctx *core.Context, loaderFor loaderSource, req request) error {
	if err := d.enterInstance(ctx, loaderFor, req); err != nil {
		return err
	}
	errs := d.exitAll(ctx)
	d.roots = []InstanceID{req.id}
	return errors.Join(errs...)
}

func (d *Director) applyLoad(ctx *core.Context, loaderFor loaderSource, req request) error {
	if err := d.enterInstance(ctx, loaderFor, req); err != nil {
		return err
	}
	d.roots = append(d.roots, req.id)
	return nil
}

func (d *Director) applyUpdate(ctx *core.Context, loaderFor loaderSource, req request) error {
	inst, ok := d.instances[req.id]
	if !ok {
		return fmt.Errorf("scene: update: instance id %d not found", req.id)
	}
	loader, err := loaderFor(ctx, req.tree)
	if err != nil {
		return err
	}
	pre := len(inst.entities)
	staged := &instance{id: inst.id, name: inst.name, hooks: inst.hooks}
	if req.nameSet {
		staged.name = req.name
	}
	if req.hooksSet {
		staged.hooks = req.hooks
	}
	if err := enter(ctx.World, loader, staged, req.tree); err != nil {
		return fmt.Errorf("scene: update %q: rebuild: %w", inst.name, err)
	}
	handle := &Handle{id: inst.id, director: d}
	if staged.hooks.Enter != nil {
		if err := staged.hooks.Enter(ctx, handle); err != nil {
			if rewind := exit(ctx.World, staged); rewind != nil {
				err = errors.Join(err, rewind)
			}
			return fmt.Errorf("scene: update %q: hook: %w", inst.name, err)
		}
	}
	atEnterEnd := inst.entities
	old := slices.Clone(atEnterEnd[:pre])
	enterBorn := slices.Clone(atEnterEnd[pre:])
	oldHooks := inst.hooks
	var errs []error
	if oldHooks.Exit != nil {
		if err := oldHooks.Exit(ctx, handle); err != nil {
			errs = append(errs, fmt.Errorf("scene: exit %q: hook: %w", inst.name, err))
		}
	}
	exitBorn := slices.Clone(inst.entities[len(atEnterEnd):])

	inst.name, inst.hooks, inst.tree, inst.live = staged.name, staged.hooks, staged.tree, true
	inst.entities = append(staged.entities, enterBorn...)
	if err := destroy(ctx.World, append(old, exitBorn...)); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (d *Director) applyUnload(ctx *core.Context, req request) error {
	inst, ok := d.instances[req.id]
	if !ok {
		return fmt.Errorf("scene: remove: instance id %d not found", req.id)
	}
	var errs []error
	if err := d.exitInstance(ctx, inst); err != nil {
		errs = append(errs, err)
	}
	delete(d.instances, req.id)
	d.roots = slices.DeleteFunc(d.roots, func(id InstanceID) bool { return id == req.id })
	return errors.Join(errs...)
}

func (d *Director) exitAll(ctx *core.Context) []error {
	var errs []error
	for i := len(d.roots) - 1; i >= 0; i-- {
		id := d.roots[i]
		inst, ok := d.instances[id]
		if !ok {
			continue
		}
		if err := d.exitInstance(ctx, inst); err != nil {
			errs = append(errs, err)
		}
		delete(d.instances, id)
	}
	d.roots = d.roots[:0]
	return errs
}

func (d *Director) enterInstance(ctx *core.Context, loaderFor loaderSource, req request) error {
	loader, err := loaderFor(ctx, req.tree)
	if err != nil {
		return err
	}
	inst := &instance{id: req.id, name: req.name, hooks: req.hooks}
	if err := enter(ctx.World, loader, inst, req.tree); err != nil {
		return fmt.Errorf("scene: enter %q: %w", inst.name, err)
	}
	d.instances[req.id] = inst
	handle := &Handle{id: inst.id, director: d}
	if inst.hooks.Enter != nil {
		if err := inst.hooks.Enter(ctx, handle); err != nil {
			delete(d.instances, req.id)
			if rewind := exit(ctx.World, inst); rewind != nil {
				err = errors.Join(err, rewind)
			}
			return fmt.Errorf("scene: enter %q: hook: %w", inst.name, err)
		}
	}
	return nil
}

func (d *Director) exitInstance(ctx *core.Context, inst *instance) error {
	var errs []error
	if inst.hooks.Exit != nil {
		if err := inst.hooks.Exit(ctx, &Handle{id: inst.id, director: d}); err != nil {
			errs = append(errs, fmt.Errorf("scene: exit %q: hook: %w", inst.name, err))
		}
	}
	if err := exit(ctx.World, inst); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (d *Director) loaderFor(ctx *core.Context, tree Node) (*asset.Loader, error) {
	if !hasAssets(tree) {
		return nil, nil
	}
	if d.loader == nil {
		loader, err := ctx.World.Resource[*asset.Loader]()
		if err != nil {
			return nil, fmt.Errorf("scene: system: %w", err)
		}
		d.loader = loader
	}
	return d.loader, nil
}

func hasAssets(n Node) bool {
	if len(n.Assets) > 0 {
		return true
	}
	return slices.ContainsFunc(n.Children, hasAssets)
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

const (
	// StartupSystemName is the scene system's registration name in [core.PhaseStartup].
	StartupSystemName = "scene-startup"
	// FrameSystemName is the scene system's registration name in [core.PhasePreFrame].
	FrameSystemName = "scene"
)

// System returns the system that processes queued requests for the director.
// Scenes that declare assets require an [asset.Loader] resource in the world.
//
// Register it in [core.PhaseStartup] as [StartupSystemName] and in
// [core.PhasePreFrame] as [FrameSystemName].
func (d *Director) System() core.SystemFunc {
	return func(ctx *core.Context) error {
		return d.execute(ctx, d.loaderFor)
	}
}
