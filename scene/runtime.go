package scene

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Leonard-Atorough/castrum/core"
)

func (d *Director) collectSubtree(inst *instance, order *[]*instance) {
	for _, id := range slices.Backward(inst.children) {
		d.collectSubtree(d.instances[id], order)
	}
	*order = append(*order, inst)
}

const (
	// StartupSystemName is the registered name for the scene startup system.
	StartupSystemName = "scene-startup"
	// FrameSystemName is the registered name for the scene frame system.
	FrameSystemName = "scene"
)

// NewStartupSystem returns the system that applies queued scene requests during
// startup. If a request fails, the system returns its error and consumes that
// request; later requests remain queued.
func NewStartupSystem(d *Director) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		return processRequests(d, ctx)
	})
}

// NewFrameSystem returns the system that applies queued scene requests during
// frame updates. If a request fails, the system returns its error and consumes
// that request; later requests remain queued.
func NewFrameSystem(d *Director) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		return processRequests(d, ctx)
	})
}

func processRequests(d *Director, ctx *core.Context) error {
	reqs := d.take()
	if len(reqs) == 0 {
		return nil
	}
	for i, req := range reqs {
		if err := apply(d, ctx, req); err != nil {
			d.pending = append(reqs[i+1:], d.pending...)
			return err
		}
	}
	if ctx.Actions != nil {
		ctx.Actions.FlushEdges()
	}
	return nil
}

func apply(d *Director, ctx *core.Context, req request) error {
	switch req.kind {
	case requestSwitch:
		var errs []error
		for id := InstanceID(1); id < d.next; id++ {
			if inst, ok := d.instances[id]; ok && inst.live && inst.parent == 0 {
				if err := exit(d, ctx, inst); err != nil {
					errs = append(errs, err)
				}
			}
		}
		if err := enter(d, ctx, req.id, req.blueprint); err != nil {
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	case requestLoad:
		return enter(d, ctx, req.id, req.blueprint)
	case requestUpdate:
		inst, ok := d.findInstance(req.id)
		if !ok {
			return fmt.Errorf("scene: update: instance %d not found", req.id)
		}
		if !inst.live {
			return fmt.Errorf("scene: update: instance %d is pending; update live instances only", req.id)
		}
		return rebuild(d, ctx, inst, req.blueprint)
	case requestUnload:
		inst, ok := d.findInstance(req.id)
		if !ok {
			return fmt.Errorf("scene: remove: instance %d not found", req.id)
		}
		if inst.live {
			return exit(d, ctx, inst)
		}
		purgeRequests(d, req.id)
		delete(d.instances, req.id)
		return nil
	default:
		return fmt.Errorf("scene: apply: unknown request kind %v", req.kind)
	}
}

func purgeRequests(d *Director, id InstanceID) {
	d.pending = slices.DeleteFunc(d.pending, func(r request) bool {
		return r.id == id
	})
}

func enter(d *Director, ctx *core.Context, id InstanceID, bp *Blueprint) error {
	inst, ok := d.findInstance(id)
	if !ok {
		return fmt.Errorf("scene: enter: instance %d not found", id)
	}
	if inst.live {
		return fmt.Errorf("scene: enter: instance %d is already live", inst.id)
	}
	spawned := []*core.Entity{}
	for _, spec := range bp.EntityList {
		if err := spawnSpec(ctx.World, spec, inst.id, &spawned); err != nil {
			rollbackEnter(d, ctx.World, inst, spawned)
			return fmt.Errorf("scene: enter %q: %w", inst.name, err)
		}
	}
	inst.live = true
	if bp.Hooks.Enter != nil {
		if err := bp.Hooks.Enter(ctx, inst.view()); err != nil {
			rollbackEnter(d, ctx.World, inst, spawned)
			return fmt.Errorf("scene: enter %q: hook: %w", inst.name, err)
		}
	}

	entered := []*instance{}
	if err := enterSubscenes(d, ctx, inst, bp.EntityList, &entered); err != nil {
		errs := []error{err}
		for _, child := range slices.Backward(entered) {
			if exitErr := exit(d, ctx, child); exitErr != nil {
				errs = append(errs, exitErr)
			}
		}
		rollbackEnter(d, ctx.World, inst, spawned)
		return errors.Join(errs...)
	}
	return nil
}

func enterSubscenes(d *Director, ctx *core.Context, inst *instance, specs []EntitySpec, entered *[]*instance) error {
	for i := range specs {
		for j := range specs[i].SubScenes {
			sub := specs[i].SubScenes[j]
			child := d.reserve(&sub, inst.id)
			inst.children = append(inst.children, child.id)
			if err := enter(d, ctx, child.id, &sub); err != nil {
				inst.children = inst.children[:len(inst.children)-1]
				return fmt.Errorf("scene: enter %q: subscene %q: %w", inst.name, sub.Name, err)
			}
			*entered = append(*entered, child)
		}
		if err := enterSubscenes(d, ctx, inst, specs[i].Children, entered); err != nil {
			return err
		}
	}
	return nil
}

func spawnSpec(world *core.World, spec EntitySpec, owner InstanceID, spawned *[]*core.Entity) error {
	entity, err := newOwnedEntity(world, owner, spec.Components...)
	if err != nil {
		return fmt.Errorf("entity %q: %w", spec.Name, err)
	}
	*spawned = append(*spawned, entity)
	for _, child := range spec.Children {
		if err := spawnSpec(world, child, owner, spawned); err != nil {
			return err
		}
	}
	return nil
}

func newOwnedEntity(world *core.World, owner InstanceID, components ...any) (*core.Entity, error) {
	ownedComponents := make([]any, 0, len(components)+1)
	ownedComponents = append(ownedComponents, OwnershipRecord{Instance: owner})
	ownedComponents = append(ownedComponents, components...)
	return world.NewEntity(ownedComponents...)
}

func rollbackEnter(d *Director, world *core.World, inst *instance, spawned []*core.Entity) {
	for _, s := range slices.Backward(spawned) {
		world.DestroyEntity(s)
	}
	destroyOwnedEntities(world, []InstanceID{inst.id})
	delete(d.instances, inst.id)
}

func exit(d *Director, ctx *core.Context, inst *instance) error {
	return teardown(d, ctx, inst, false)
}

func teardown(d *Director, ctx *core.Context, inst *instance, keepRoot bool) error {
	order := make([]*instance, 0, len(inst.children)+1)
	d.collectSubtree(inst, &order)

	// Phase 1: every Exit hook runs bottom-up while all entities still
	// exist, so a parent's hook can read its children's state.
	var hookErrs []error
	for _, item := range order {
		if item.blueprint.Hooks.Exit == nil {
			continue
		}
		if err := item.blueprint.Hooks.Exit(ctx, item.view()); err != nil {
			hookErrs = append(hookErrs, fmt.Errorf("scene: exit hook for instance %d (%q): %w", item.id, item.name, err))
		}
	}

	// Phase 2: irreversible work. Collect handles before destroying;
	// destroying during a query pass invalidates the iteration.
	ids := make([]InstanceID, 0, len(order))
	for _, item := range order {
		ids = append(ids, item.id)
	}
	destroyOwnedEntities(ctx.World, ids)

	// Records go children-first; order already is.
	for _, item := range order {
		if keepRoot && item == inst {
			inst.children = nil
			continue
		}
		delete(d.instances, item.id)
	}
	if !keepRoot {
		unlinkFromParent(d, inst)
	}
	if len(hookErrs) > 0 {
		return fmt.Errorf("scene: teardown committed despite exit hook errors: %w", errors.Join(hookErrs...))
	}
	return nil
}

func destroyOwnedEntities(world *core.World, owners []InstanceID) {
	ids := make(map[InstanceID]struct{}, len(owners))
	for _, owner := range owners {
		ids[owner] = struct{}{}
	}
	doomed := []*core.Entity{}
	q := core.NewQuery(world).With(OwnershipRecord{}).Where(func(e core.Entry) bool {
		rec, ok := e.Component[OwnershipRecord]()
		if !ok {
			return false
		}
		_, owned := ids[rec.Instance]
		return owned
	})
	for e := range q.Execute() {
		doomed = append(doomed, core.NewEntity(e.ID()))
	}
	for _, e := range doomed {
		world.DestroyEntity(e) // best-effort, like rollback
	}
}

func unlinkFromParent(d *Director, inst *instance) {
	if inst.parent != 0 {
		if parent, ok := d.findInstance(inst.parent); ok {
			parent.children = slices.DeleteFunc(parent.children, func(id InstanceID) bool {
				return id == inst.id
			})
		}
	}
}

func rebuild(d *Director, ctx *core.Context, inst *instance, bp *Blueprint) error {
	teardownErr := teardown(d, ctx, inst, true)
	inst.blueprint = bp
	inst.live = false
	enterErr := enter(d, ctx, inst.id, bp)
	if enterErr != nil {
		unlinkFromParent(d, inst)
	}
	return errors.Join(teardownErr, enterErr)
}
