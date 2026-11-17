// Package scenev2 provides scene builders and node trees.
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
type Scene func(props any) Node

// Of wraps a typed builder as a [Scene].
//
// The returned scene panics if its props cannot be asserted to P.
func Of[P any](build func(P) Node) Scene {
	return func(props any) Node {
		p, ok := props.(P)
		if !ok {
			panic(fmt.Sprintf("scene: props type %T, want %v", props, reflect.TypeFor[P]()))
		}
		return build(p)
	}
}

type Preload interface {
	warm(loader *asset.Loader) error
	id() asset.ID
}

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

// Assets is a placeholder for scene asset preloading, which is not yet implemented.
type Assets struct{ Preload []Preload }

// Node describes an entity and its children in a scene tree.
type Node struct {
	// Key identifies the node among its siblings; it must be nonempty and contain no "/".
	Key string
	// Assets lists preloaded assets.
	Assets Assets
	// Components are the non-nil, non-pointer values attached to the entity.
	Components []any
	// Children are the node's child entities.
	Children []Node
	// Scenes lists nested scene builders; they are not yet applied or validated.
	Scenes []Scene
	// Persistent marks the node for persistence; it currently has no effect.
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
	tree := scene(props)
	return tree, validateNode(tree.Key, tree, map[string]bool{})
}

// InstanceID identifies a scene instance.
type InstanceID uint64

type instance struct {
	id       InstanceID
	name     string
	tree     Node
	entities map[string]core.Entity
	children InstanceID
	live     bool
}

func enter(world *core.World, loader *asset.Loader, inst *instance, tree Node) error {
	if err := validateNode(tree.Key, tree, map[string]bool{}); err != nil {
		return err
	}
	if err := preload(loader, tree); err != nil {
		return err
	}

	entities := make(map[string]core.Entity)
	if err := spawnTree(world, entities, "", tree); err != nil {
		return errors.Join(err, destroy(world, entities))
	}
	inst.tree = tree
	inst.entities = entities
	inst.live = true
	return nil
}

func spawnTree(world *core.World, entities map[string]core.Entity, path string, n Node) error {
	entity, err := world.NewEntity(n.Components...)
	if err != nil {
		return fmt.Errorf("scene: spawn node %q: %w", join(path, n.Key), err)
	}
	entities[join(path, n.Key)] = entity
	for _, c := range n.Children {
		if err := spawnTree(world, entities, join(path, n.Key), c); err != nil {
			return err
		}
	}
	return nil
}

func exit(world *core.World, inst *instance) error {
	errs := errors.Join(destroy(world, inst.entities))
	inst.live = false
	inst.entities = nil
	inst.tree = Node{}
	return errs
}

func destroy(world *core.World, entities map[string]core.Entity) error {
	var errs []error
	for path, entity := range entities {
		if err := world.DestroyEntity(entity); err != nil {
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

var ErrAssetsNotReady = errors.New("scene: assets not ready")

func preload(loader *asset.Loader, tree Node) error {
	return walkAssets(tree, func(p Preload) error {
		if err := p.warm(loader); err != nil {
			return fmt.Errorf("%w: %s: %w", ErrAssetsNotReady, p.id(), err)
		}
		return nil
	})
}

func walkAssets(n Node, visit func(Preload) error) error {
	for _, p := range n.Assets.Preload {
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

type Handle struct {
	id       InstanceID
	director *Director
}

type Hooks struct {
	Enter func(ctx *core.Context, handle *Handle)
	Exit  func(ctx *core.Context, handle *Handle)
	Ready func(ctx *core.Context, handle *Handle)
}
