// Package scene defines scene blueprints and manages their runtime instances.
package scene

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
)

// Blueprint describes the entities that make up a scene.
//
// [Director.Load], [Director.Switch], and [Director.Update] snapshot the
// blueprint structure and component interface values when they queue a request.
// References inside component values and state captured by hooks remain shared.
type Blueprint struct {
	// Name identifies the blueprint.
	Name string
	// Assets lists asset IDs associated with the blueprint. The scene system
	// does not currently load them.
	Assets []asset.ID
	// EntityList contains the root entity specifications.
	EntityList []EntitySpec
	// Hooks defines callbacks for instance lifecycle transitions.
	Hooks Hooks
}

// Validate reports whether bp has a valid name, assets, and
// entity specifications, including nested subscenes.
func (bp Blueprint) Validate() error {
	if bp.Name == "" {
		return fmt.Errorf("scene: blueprint name must not be empty")
	}
	if len(bp.EntityList) == 0 {
		return fmt.Errorf("scene: blueprint %q has no nodes", bp.Name)
	}
	if slices.Contains(bp.Assets, "") {
		return fmt.Errorf("scene: blueprint %q has an empty asset ID", bp.Name)
	}
	return validateNodes(bp.Name, bp.EntityList)
}

func validateNodes(owner string, nodes []EntitySpec) error {
	seen := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if n.Name == "" {
			return fmt.Errorf("scene: blueprint %q has an unnamed node", owner)
		}
		if seen[n.Name] {
			return fmt.Errorf("scene: blueprint %q has duplicate node name %q", owner, n.Name)
		}
		seen[n.Name] = true
		for i, c := range n.Components {
			if c == nil {
				return fmt.Errorf("scene: blueprint %q node %q: component at index %d is nil", owner, n.Name, i)
			}
			if reflect.TypeOf(c).Kind() == reflect.Pointer {
				return fmt.Errorf("scene: blueprint %q node %q: component at index %d (%T) is a pointer", owner, n.Name, i, c)
			}
			if v, ok := c.(core.Validatable); ok {
				if err := v.Validate(); err != nil {
					return fmt.Errorf("scene: blueprint %q node %q: %w", owner, n.Name, err)
				}
			}
		}
		if err := validateNodes(owner, n.Children); err != nil {
			return err
		}
		for _, sub := range n.SubScenes {
			if err := sub.Validate(); err != nil {
				return fmt.Errorf("scene: subscene %q: %w", sub.Name, err)
			}
		}
	}
	return nil
}

func snapshotBlueprint(bp *Blueprint) *Blueprint {
	snapshot := *bp
	snapshot.Assets = slices.Clone(bp.Assets)
	snapshot.EntityList = snapshotEntitySpecs(bp.EntityList)
	return &snapshot
}

func snapshotEntitySpecs(specs []EntitySpec) []EntitySpec {
	snapshot := slices.Clone(specs)
	for i := range snapshot {
		snapshot[i].Components = slices.Clone(snapshot[i].Components)
		snapshot[i].Children = snapshotEntitySpecs(snapshot[i].Children)
		snapshot[i].SubScenes = slices.Clone(snapshot[i].SubScenes)
		for j := range snapshot[i].SubScenes {
			snapshot[i].SubScenes[j] = *snapshotBlueprint(&snapshot[i].SubScenes[j])
		}
	}
	return snapshot
}

// EntitySpec describes an entity to create in a scene instance.
type EntitySpec struct {
	// Name identifies the entity among its siblings.
	Name string
	// Components contains the non-nil, non-pointer values attached to the entity.
	Components []any
	// Children contains child entities owned by the same scene instance.
	Children []EntitySpec
	// SubScenes contains nested scenes managed as separate instances. For each
	// entity specification, they enter before those on its child specifications.
	SubScenes []Blueprint
}

// Hooks contains callbacks for scene-instance lifecycle transitions.
type Hooks struct {
	// Enter runs after the instance's entities are created,
	// before its nested scenes enter. An error rolls back the entry.
	Enter func(ctx *core.Context, self *Instance) error
	// Exit runs while the instance's entities still exist. Exit hooks run
	// children-first; errors do not stop other hooks or teardown, and are
	// reported after teardown commits.
	Exit func(ctx *core.Context, self *Instance) error
}

// InstanceID identifies an instance managed by a [Director]. IDs increase
// monotonically and are not reused.
type InstanceID uint64

// Instance is a detached view of a scene instance.
type Instance struct {
	// ID uniquely identifies this instance within its Director.
	ID InstanceID
	// Name identifies the instance and is unchanged by blueprint updates.
	Name string
	// Blueprint is a detached structural snapshot of the blueprint assigned to
	// this instance. Mutating its structure does not change the instance; use
	// [Director.Update] to replace its blueprint.
	Blueprint Blueprint
}

// OwnershipRecord identifies the scene instance that owns an entity's lifetime.
type OwnershipRecord struct {
	// Instance is the ID of the owning scene instance.
	Instance InstanceID
}

type instance struct {
	id        InstanceID
	name      string
	blueprint *Blueprint
	parent    InstanceID // 0 for roots
	children  []InstanceID
	live      bool // false while pending, true once applied
}

func (i *instance) snapshot() Instance {
	return Instance{ID: i.id, Name: i.name, Blueprint: *snapshotBlueprint(i.blueprint)}
}

func (i *instance) view() *Instance {
	view := i.snapshot()
	return &view
}

type request struct {
	kind      requestKind
	id        InstanceID // reserved for load/switch; target for update/remove
	blueprint *Blueprint
}

type requestKind uint8

const (
	requestSwitch requestKind = iota
	requestLoad
	requestUpdate
	requestUnload
)

// Director queues scene-instance transitions and tracks their instances.
// It is single-threaded and must not be accessed concurrently.
type Director struct {
	next      InstanceID
	pending   []request
	instances map[InstanceID]*instance
}

// NewDirector returns an empty [Director].
func NewDirector() *Director {
	return &Director{
		next:      1, // 0 is the "no parent" sentinel
		instances: make(map[InstanceID]*instance),
	}
}

// Switch queues removal of live root instances and loads bp as a new root.
// The returned ID is reserved immediately; the instance remains pending until
// the request is applied. The request snapshots bp when queued, so later
// changes to its structure do not affect it.
func (d *Director) Switch(bp *Blueprint) (InstanceID, error) {
	return d.queueRoot(bp, requestSwitch, "Switch")
}

// Load queues bp as a root instance without removing existing instances.
// The returned ID is reserved immediately; the instance remains pending until
// the request is applied. The request snapshots bp when queued, so later
// changes to its structure do not affect it.
func (d *Director) Load(bp *Blueprint) (InstanceID, error) {
	return d.queueRoot(bp, requestLoad, "Load")
}

func (d *Director) queueRoot(bp *Blueprint, kind requestKind, operation string) (InstanceID, error) {
	if bp == nil {
		return 0, fmt.Errorf("scene: %s: blueprint is nil", operation)
	}
	if err := bp.Validate(); err != nil {
		return 0, err
	}
	bp = snapshotBlueprint(bp)
	inst := d.reserve(bp, 0)
	d.pending = append(d.pending, request{kind: kind, id: inst.id, blueprint: bp})
	return inst.id, nil
}

// Get returns a detached view of id, including pending instances. It reports
// false if id is unknown.
func (d *Director) Get(id InstanceID) (Instance, bool) {
	inst, ok := d.findInstance(id)
	if !ok {
		return Instance{}, false
	}
	return inst.snapshot(), true
}

// ByName returns a detached view of the first live or pending instance with
// the given name, ordered by instance ID. It reports false if none exists.
func (d *Director) ByName(name string) (Instance, bool) {
	inst, ok := d.findInstanceByName(name)
	if !ok {
		return Instance{}, false
	}
	return inst.snapshot(), true
}

// Update queues bp as a replacement blueprint for a live instance. It returns
// an error if bp is invalid or id does not identify a live instance. The
// request snapshots bp when queued, so later changes to its structure do not
// affect it.
func (d *Director) Update(id InstanceID, bp *Blueprint) error {
	if bp == nil {
		return fmt.Errorf("scene: Update: blueprint is nil")
	}
	if err := bp.Validate(); err != nil {
		return err
	}
	inst, ok := d.findInstance(id)
	if !ok {
		return fmt.Errorf("scene: Update: instance %d not found", id)
	}
	if !inst.live {
		return fmt.Errorf("scene: Update: instance %d is pending; update live instances only", id)
	}
	bp = snapshotBlueprint(bp)
	d.pending = append(d.pending, request{kind: requestUpdate, id: id, blueprint: bp})
	return nil
}

// Remove queues teardown of the instance subtree rooted at id. It returns an
// error if id does not identify an instance.
func (d *Director) Remove(id InstanceID) error {
	inst, ok := d.findInstance(id)
	if !ok {
		return fmt.Errorf("scene: Remove: instance %d not found", id)
	}
	d.pending = append(d.pending, request{kind: requestUnload, id: inst.id})
	return nil
}

// Spawn creates an entity owned by the live scene instance identified by
// owner. The entity is created in ctx.World and is destroyed when the instance
// is removed or rebuilt. Call this from the engine goroutine, like other world
// operations. It returns an error if ctx or its world is nil, owner is not a
// live instance, or components are invalid. OwnershipRecord is managed by the
// director and must not be supplied by callers.
func (d *Director) Spawn(ctx *core.Context, owner InstanceID, components ...any) (*core.Entity, error) {
	if ctx == nil || ctx.World == nil {
		return nil, fmt.Errorf("scene: Spawn: context and world must not be nil")
	}
	inst, ok := d.findInstance(owner)
	if !ok || !inst.live {
		return nil, fmt.Errorf("scene: Spawn: instance %d is not live", owner)
	}
	for _, component := range components {
		if _, ok := component.(OwnershipRecord); ok {
			return nil, fmt.Errorf("scene: Spawn: OwnershipRecord is managed by the director")
		}
	}
	return newOwnedEntity(ctx.World, owner, components...)
}

func (d *Director) nextID() InstanceID {
	id := d.next
	d.next++
	return id
}

func (d *Director) reserve(bp *Blueprint, parent InstanceID) *instance {
	id := d.nextID()
	inst := &instance{id: id, name: bp.Name, blueprint: bp, parent: parent}
	d.instances[id] = inst
	return inst
}

func (d *Director) findInstance(id InstanceID) (*instance, bool) {
	inst, ok := d.instances[id]
	return inst, ok
}

func (d *Director) findInstanceByName(name string) (*instance, bool) {
	for id := InstanceID(1); id < d.next; id++ {
		if inst, ok := d.instances[id]; ok && inst.name == name {
			return inst, true
		}
	}
	return nil, false
}

func (d *Director) take() []request {
	reqs := append([]request(nil), d.pending...)
	d.pending = nil
	return reqs
}
