package ecs

import (
	"fmt"
	"reflect"
	"strings"
)

type ComponentFactory func(props map[string]any) (any, error)

type Definition struct {
	Name    string
	Type    reflect.Type
	Size    uintptr
	Factory ComponentFactory
}

type Registry struct {
	byName map[string]Definition
	byType map[reflect.Type]string
}

func NewRegistry() *Registry {
	return &Registry{
		byName: make(map[string]Definition),
		byType: make(map[reflect.Type]string),
	}
}

func (r *Registry) Register(name string, typ reflect.Type, factory ComponentFactory) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("component name must not be empty")
	}
	if typ == nil {
		return fmt.Errorf("component %q has nil type", name)
	}
	if factory == nil {
		return fmt.Errorf("component %q has nil factory", name)
	}
	if existing, ok := r.byName[name]; ok {
		if existing.Type == typ && existing.Factory != nil {
			return nil
		}
		return fmt.Errorf("component name %q is already registered", name)
	}
	if existingName, ok := r.byType[typ]; ok && existingName != name {
		return fmt.Errorf("component type %s is already registered as %q", typ, existingName)
	}

	r.byName[name] = Definition{
		Name:    name,
		Type:    typ,
		Size:    typ.Size(),
		Factory: factory,
	}
	r.byType[typ] = name
	return nil
}

func (r *Registry) Resolve(name string, props map[string]any) (any, error) {
	def, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("component definition not found for name: %s", name)
	}
	if def.Factory == nil {
		return nil, fmt.Errorf("component definition %q has nil factory", name)
	}
	return def.Factory(props)
}

func (r *Registry) LookupByType(typ reflect.Type) (Definition, bool) {
	name, ok := r.byType[typ]
	if !ok {
		return Definition{}, false
	}
	def, ok := r.byName[name]
	return def, ok
}

func (r *Registry) LookupByName(name string) (Definition, bool) {
	def, ok := r.byName[name]
	return def, ok
}

func (r *Registry) Has(name string) bool {
	_, ok := r.byName[name]
	return ok
}
