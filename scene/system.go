package scene

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/input"
)

const (
	// StartupSystemName is the scene system's registration name in [core.PhaseStartup].
	StartupSystemName = "scene-startup"
	// FrameSystemName is the scene system's registration name in [core.PhasePreFrame].
	FrameSystemName = "scene"
)

type loaderSource func(ctx *core.Context, tree Node) (*asset.Loader, error)

// NewSystem returns a system that processes queued requests for director.
// Scenes that declare assets require an [asset.Loader] resource in the world.
func NewSystem(director *Director) core.System {
	return &system{director: director}
}

type system struct {
	director *Director
	loader   *asset.Loader
}

func (s *system) Update(ctx *core.Context) error {
	return s.director.execute(ctx, s.loaderFor)
}

func (s *system) loaderFor(ctx *core.Context, tree Node) (*asset.Loader, error) {
	if !hasAssets(tree) {
		return nil, nil
	}
	if s.loader == nil {
		loader, err := ctx.World.Resource[*asset.Loader]()
		if err != nil {
			return nil, fmt.Errorf("scene: system: %w", err)
		}
		s.loader = loader
	}
	return s.loader, nil
}

func hasAssets(n Node) bool {
	if len(n.Assets) > 0 {
		return true
	}
	for i := range n.Children {
		if hasAssets(n.Children[i]) {
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
