// Package scenev2 provides scene builders, node trees, and a director for
// queuing scene requests.
package scenev2

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
)

// Scene builds a [Node] tree from props.
type Scene func(props any) (Node, error)

// Of wraps a typed builder as a [Scene].
//
// The returned scene reports an error if props cannot be asserted to P.
func Of[P any](build func(P) Node) Scene {
	return func(props any) (Node, error) {
		p, ok := props.(P)
		if !ok {
			return Node{}, fmt.Errorf("scene: props type %T, want %v", props, reflect.TypeFor[P]())
		}
		return build(p), nil
	}
}

// Preload describes a typed asset to load before entering a scene.
// Use [Warm] to create one.
type Preload interface {
	warm(loader *asset.Loader) error
	id() asset.ID
}

// Warm declares that the asset at id should be loaded as T before scene entry.
func Warm[T any](id asset.ID) Preload {
	return warm[T]{path: id}
}

type warm[T any] struct{ path asset.ID }

func (w warm[T]) id() asset.ID { return w.path }

func (w warm[T]) warm(loader *asset.Loader) error {
	if _, err := loader.Load[T](string(w.path)); err != nil {
		return err
	}
	return nil
}

// Node describes an entity in a scene tree.
type Node struct {
	// Key identifies the node among its siblings. It must be nonempty and contain no "/".
	Key string
	// Assets lists assets to load before the scene enters.
	Assets []Preload
	// Components are the non-nil, non-pointer values attached to the entity.
	// Component-specific validation runs when the entity is created.
	Components []any
	// Children are nested scene nodes, not a transform hierarchy.
	Children []Node
	// Persistent marks the node for persistence. It currently has no effect.
	Persistent bool
}

func validateNode(path string, n Node, seen map[string]bool) error {
	if n.Key == "" {
		return fmt.Errorf("scene: node at %q has an empty key", path)
	}
	if strings.Contains(n.Key, "/") {
		return fmt.Errorf("scene: node key %q must not contain '/'", n.Key)
	}
	if seen[n.Key] {
		return fmt.Errorf("scene: duplicate sibling key %q", n.Key)
	}
	seen[n.Key] = true
	for i, c := range n.Components {
		if c == nil {
			return fmt.Errorf("scene: node %q: component at index %d is nil", n.Key, i)
		}
		if reflect.TypeOf(c).Kind() == reflect.Pointer {
			return fmt.Errorf("scene: node %q: component at index %d (%T) is a pointer", n.Key, i, c)
		}
	}
	childSeen := make(map[string]bool, len(n.Children))
	for _, child := range n.Children {
		if err := validateNode(path+"/"+child.Key, child, childSeen); err != nil {
			return err
		}
	}
	return nil
}

func validate(scene Scene, props any) (Node, error) {
	tree, err := scene(props)
	if err != nil {
		return Node{}, err
	}
	return tree, validateNode(tree.Key, tree, map[string]bool{})
}

// InstanceID identifies a scene instance within a [Director].
type InstanceID uint64

type instance struct {
	id       InstanceID
	name     string
	tree     Node
	entities map[string]core.Entity
	order    []string
	children []InstanceID
	live     bool
}

func enter(world *core.World, loader *asset.Loader, inst *instance, tree Node) error {
	if inst.live {
		panic("scene: enter called on already live instance")
	}
	if err := preload(loader, tree); err != nil {
		return err
	}

	entities := make(map[string]core.Entity)
	order, err := spawnTree(world, entities, nil, "", tree)
	if err != nil {
		return errors.Join(err, destroy(world, entities, order))
	}
	inst.tree = tree
	inst.entities = entities
	inst.order = order
	inst.live = true
	return nil
}

func spawnTree(world *core.World, entities map[string]core.Entity, order []string, path string, n Node) ([]string, error) {
	entity, err := world.NewEntity(n.Components...)
	if err != nil {
		return order, fmt.Errorf("scene: spawn node %q: %w", join(path, n.Key), err)
	}
	nodePath := join(path, n.Key)
	entities[nodePath] = entity
	order = append(order, nodePath)
	for _, c := range n.Children {
		if order, err = spawnTree(world, entities, order, nodePath, c); err != nil {
			return order, err
		}
	}
	return order, nil
}

func exit(world *core.World, inst *instance) error {
	errs := errors.Join(destroy(world, inst.entities, inst.order))
	inst.live = false
	inst.entities = nil
	inst.order = nil
	inst.tree = Node{}
	return errs
}

func destroy(world *core.World, entities map[string]core.Entity, order []string) error {
	var errs []error
	for _, path := range order {
		if err := world.DestroyEntity(entities[path]); err != nil {
			errs = append(errs, fmt.Errorf("scene: destroy node %q: %w", path, err))
		}
	}
	return errors.Join(errs...)
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "/" + key
}

// ErrAssetLoadFailed identifies a failure to preload a scene asset.
var ErrAssetLoadFailed = errors.New("scene: asset load failed")

func preload(loader *asset.Loader, tree Node) error {
	return walkAssets(tree, func(p Preload) error {
		if err := p.warm(loader); err != nil {
			return fmt.Errorf("%w: %s: %w", ErrAssetLoadFailed, p.id(), err)
		}
		return nil
	})
}

func walkAssets(n Node, visit func(Preload) error) error {
	for _, p := range n.Assets {
		if err := visit(p); err != nil {
			return err
		}
	}
	for _, child := range n.Children {
		if err := walkAssets(child, visit); err != nil {
			return err
		}
	}
	return nil
}

// Handle identifies a scene instance in lifecycle callbacks.
type Handle struct {
	id       InstanceID
	director *Director
}

// Hooks contains scene lifecycle callbacks. The director does not yet invoke them.
type Hooks struct {
	// Enter is the callback for scene entry.
	Enter func(ctx *core.Context, handle *Handle)
	// Exit is the callback for scene exit.
	Exit func(ctx *core.Context, handle *Handle)
	// Ready is the callback for scene readiness.
	Ready func(ctx *core.Context, handle *Handle)
}
