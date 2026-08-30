package core

import (
	"fmt"
)

type Registry struct {
	constructors map[string]func(properties map[string]any) (Component, error)
}

func NewRegistry() *Registry {
	return &Registry{
		constructors: make(map[string]func(properties map[string]any) (Component, error)),
	}
}

func (r *Registry) Register(typeName string, constructor func(properties map[string]any) (Component, error)) {
	if _, exists := r.constructors[typeName]; exists {
		panic(fmt.Sprintf("component type %s is already registered", typeName))
	}
	r.constructors[typeName] = constructor
}

func (r *Registry) Resolve(typeName string, properties map[string]any) (Component, error) {
	constructor, exists := r.constructors[typeName]
	if !exists {
		return nil, fmt.Errorf("component type %s is not registered", typeName)
	}
	return constructor(properties)
}

func (r *Registry) List() []string {
	keys := make([]string, 0, len(r.constructors))
	for key := range r.constructors {
		keys = append(keys, key)
	}
	return keys
}

func (r *Registry) Get(typeName string) (func(properties map[string]any) (Component, error), bool) {
	constructor, exists := r.constructors[typeName]
	return constructor, exists
}
