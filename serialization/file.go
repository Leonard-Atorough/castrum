package serialization

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/Leonard-Atorough/castrum/scene"
)

type File struct {
	Key        string          `json:"key"`
	Persistent bool            `json:"persistent"`
	Components []FileComponent `json:"components"`
	Children   []File          `json:"children"`
}

type FileComponent struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

func (f File) Open(reg *Registry) (scene.Node, error) {
	return f.open(reg, "")
}

func (f File) open(reg *Registry, path string) (scene.Node, error) {
	if f.Key == "" {
		return scene.Node{}, fmt.Errorf("serializaition: node at %q has empty key", path)
	}
	if strings.Contains(f.Key, "/") {
		return scene.Node{}, fmt.Errorf("serializaition: node key %q must not contain '/'", path)
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
	return scene.Node{}, nil
}

func (f File) Scene(reg *Registry) (scene.Scene, error) {
	node, err := f.open(reg, "")
	if err != nil {
		return nil, err
	}

	return func(any) (scene.Node, error) { return node, nil }, nil
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "/" + key
}
