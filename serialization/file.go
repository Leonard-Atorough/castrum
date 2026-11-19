package serialization

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/scene"
)

// File is the JSON form of a scene node. [File.Open] and [File.Scene] build
// engine values from it using a [Registry].
type File struct {
	// Key names the node within its parent. It must not be empty or contain '/'.
	Key string `json:"key"`
	// Persistent flags the node as a candidate for game runtime state
	// serialization; it is copied to [scene.Node.Persistent].
	Persistent bool `json:"persistent"`
	// Components lists the node's components. Each registered type may appear once.
	Components []FileComponent `json:"components"`
	// Children lists the child nodes. Child keys must be unique within the node.
	Children []File `json:"children"`
}

// FileComponent is one component entry in a [File].
type FileComponent struct {
	// Type is the key the component type was registered under with [Registry.Component].
	Type string `json:"type"`
	// Value is the component's JSON, decoded with the type's registered codec.
	Value json.RawMessage `json:"value"`
}

// Open builds the [scene.Node] described by f and its children.
//
// It fails on an invalid or duplicate key, an unknown or repeated component
// type, or a component that fails to decode or validate.
func (f File) Open(reg *Registry) (scene.Node, error) {
	return f.open(reg, "")
}

// Scene returns a [scene.Scene] that yields the node built by [File.Open].
//
// The scene ignores props and returns copies of the same node, sharing its
// component and child slices.
func (f File) Scene(reg *Registry) (scene.Scene, error) {
	node, err := f.open(reg, "")
	if err != nil {
		return nil, err
	}

	return func(any) (scene.Node, error) { return node, nil }, nil
}

// Spawn creates an entity in world from n's components.
//
// An override replaces the node component of the same type, and overrides of
// other types are added. Spawn fails if n has children; load scenes with
// [scene.Director] instead. It ignores the node's key, assets, and persistence.
// Entity creation follows the rules of [core.World.NewEntity].
func Spawn(world *core.World, n scene.Node, overrides ...any) (core.Entity, error) {
	if len(n.Children) != 0 {
		return core.Entity{}, fmt.Errorf("serialization: spawn node %q has children; use scene.Director for loading scenes", n.Key)
	}
	components := make([]any, 0, len(n.Components)+len(overrides))
	for _, c := range n.Components {
		if !replacedBy(c, overrides) {
			components = append(components, c)
		}
	}
	components = append(components, overrides...)
	return world.NewEntity(components...)
}

func replacedBy(c any, overrides []any) bool {
	typ := reflect.TypeOf(c)
	for _, o := range overrides {
		if reflect.TypeOf(o) == typ {
			return true
		}
	}
	return false
}

func (f File) open(reg *Registry, path string) (scene.Node, error) {
	if f.Key == "" {
		return scene.Node{}, fmt.Errorf("serialization: node at %q has empty key", path)
	}
	if strings.Contains(f.Key, "/") {
		return scene.Node{}, fmt.Errorf("serialization: node key %q must not contain '/'", path)
	}

	nodepath := join(path, f.Key)
	n := scene.Node{Key: f.Key, Persistent: f.Persistent}

	types := make(map[reflect.Type]bool, len(f.Components))
	for i, c := range f.Components {
		if _, ok := reg.ComponentType(c.Type); !ok {
			return scene.Node{}, fmt.Errorf("serialization: unknown component type %q at index %d in node %q", c.Type, i, nodepath)
		}

		val, err := reg.DecodeComponent(c.Type, c.Value)
		if err != nil {
			return scene.Node{}, fmt.Errorf("serialization: failed to decode component %q at index %d in node %q: %w", c.Type, i, nodepath, err)
		}

		if a, ok := val.(core.Authored); ok {
			if err := a.Authored(); err != nil {
				return scene.Node{}, fmt.Errorf("serialization: component %q at index %d in node %q: %w", c.Type, i, nodepath, err)
			}
		}

		typ := reflect.TypeOf(val)
		if types[typ] {
			return scene.Node{}, fmt.Errorf("serialization: duplicate component type %q at index %d in node %q", typ, i, nodepath)
		}
		types[typ] = true
		n.Components = append(n.Components, val)
	}

	seen := make(map[string]bool, len(f.Children))
	for _, child := range f.Children {
		if _, ok := seen[child.Key]; ok {
			return scene.Node{}, fmt.Errorf("serialization: duplicate child key %q in node %q", child.Key, nodepath)
		}
		seen[child.Key] = true
		cn, err := child.open(reg, nodepath)
		if err != nil {
			return scene.Node{}, err
		}
		n.Children = append(n.Children, cn)
	}
	return n, nil
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "/" + key
}
