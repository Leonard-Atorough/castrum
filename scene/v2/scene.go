package scenev2

import (
	"fmt"
	"reflect"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
)

// InstanceID identifies an instance.
type InstanceID uint64

// Scene is a function that builds a Node from props.
type Scene func(props any) Node

// Of wraps a typed builder into Scene.
func Of[P any](build func(P) Node) Scene {
	return func(props any) Node {
		p, ok := props.(P)
		if !ok {
			panic(fmt.Sprintf("scene: props type %T, want %v", props, reflect.TypeFor[P]()))
		}
		return build(p)
	}
}

type Node struct {
	Key        string
	Components []any
	Children   []Node
	Scenes     []Scene
	Persistent bool
}

func validateNode(path string, n Node, seen map[string]bool) error {
	if n.Key == "" {
		return fmt.Errorf("scene: node at %q has an empty key", path)
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

func Validate(scene Scene, props any) error {
	tree := scene(props)
	return validateNode(tree.Key, tree, map[string]bool{tree.Key: true})
}

// instance is a record of a constructed, applied scene. It is created and
// managed by the [Director].
type instance struct {
	id       InstanceID
	name     string
	tree     Node
	entities map[string]core.EntityID
	children InstanceID
	live     bool
}

func enter(world *core.World, loader *asset.Loader, inst *instance, tree Node) error {
	if err := preload(loader, tree); err != nil {
		return err
	}

	spawned := make([]*core.Entity, 0, 8)
	entities := make(map[string]core.EntityID)
	if err := spawnTree(world, entities, &spawned, "", tree); err != nil {
		destroyAll(world, spawned)
		return err
	}
	inst.tree = tree
	inst.entities = entities
	inst.live = true
	return nil
}

func spawnTree(world *core.World, entities map[string]core.EntityID, spawned *[]*core.Entity,
	path string, n Node) error {
	entity, err := world.NewEntity(n.Components...)
	if err != nil {
		return fmt.Errorf("scene: spawn node %q: %w", join(path, n.Key), err)
	}
	*spawned = append(*spawned, entity)
	entities[join(path, n.Key)] = entity.ID()
	for _, c := range n.Children {
		if err := spawnTree(world, entities, spawned, join(path, n.Key), c); err != nil {
			return err
		}
	}
	return nil
}

func destroy(world *core.World, inst *instance) {
	for _, id := range inst.entities {
		world.DestroyEntity(id)
	}
}

func preload(loader *asset.Loader, tree Node) error {
	// Walk the tree collecting Assets.Preload declarations, then resolve
	// each through the loader. Missing or unreadable -> wrap ErrAssetsNotReady
	// with the asset ID named.
	return nil
}

func destroyAll(world *core.World, spawned []*core.Entity) {
	for _, e := range spawned {
		world.DestroyEntity(e)
	}
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "/" + key
}

// type Hooks struct {
// 	Enter func(ctx *core.Context, self *Instance)
// 	Exit  func(ctx *core.Context, self *Instance)
// 	Ready func(ctx *core.Context, self *Instance)
// }
