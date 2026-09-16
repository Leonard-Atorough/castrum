package assets

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/leonard-atorough/castrum/ecs"
)

// ComponentFactory constructs a component from blueprint properties.
type ComponentFactory func(properties map[string]any) (ecs.Component, error)

// ComponentDefinition describes a component type available to blueprints.
type ComponentDefinition struct {
	Name    string
	Type    reflect.Type
	Size    uintptr
	Factory ComponentFactory
}

// ComponentRegistry maps blueprint component names to construction factories.
type ComponentRegistry struct {
	byName map[string]ComponentDefinition
	byType map[reflect.Type]string
}

// NewComponentRegistry creates an empty blueprint component registry.
func NewComponentRegistry() *ComponentRegistry {
	return &ComponentRegistry{
		byName: make(map[string]ComponentDefinition),
		byType: make(map[reflect.Type]string),
	}
}

// RegisterComponent registers a component type for blueprint construction.
// The optional factory overrides the default property decoder.
func (r *ComponentRegistry) RegisterComponent[T ecs.Component](name string, factories ...func(map[string]any) (T, error)) error {
	typ := reflect.TypeFor[T]()
	factory := func(properties map[string]any) (ecs.Component, error) {
		if len(factories) > 0 && factories[0] != nil {
			return factories[0](properties)
		}

		// If T implements Serializable[T], use the typed Deserialize path.
		// Value receivers on both methods mean the zero value satisfies the
		// interface directly — no reflection or pointer needed.
		var zero T
		if serializable, ok := any(zero).(ecs.Serializable[T]); ok {
			result, err := serializable.Deserialize(properties)
			if err != nil {
				return nil, err
			}
			return result, nil
		}

		// Fallback: set fields by name via reflection.
		value := reflect.New(typ).Elem()
		for key, raw := range properties {
			field := value.FieldByName(key)
			if !field.IsValid() || !field.CanSet() {
				continue
			}
			property := reflect.ValueOf(raw)
			if property.IsValid() && property.Type().AssignableTo(field.Type()) {
				field.Set(property)
			}
		}
		return value.Interface(), nil
	}
	return r.Register(name, typ, factory)
}

// Register associates a blueprint name and component type with a factory.
func (r *ComponentRegistry) Register(name string, typ reflect.Type, factory ComponentFactory) error {
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

	r.byName[name] = ComponentDefinition{
		Name:    name,
		Type:    typ,
		Size:    typ.Size(),
		Factory: factory,
	}
	r.byType[typ] = name
	return nil
}

// Resolve constructs a component from a registered blueprint name.
func (r *ComponentRegistry) Resolve(name string, properties map[string]any) (ecs.Component, error) {
	definition, ok := r.byName[name]
	if !ok {
		return nil, fmt.Errorf("component definition not found for name: %s", name)
	}
	return definition.Factory(properties)
}

// LookupByType returns the definition registered for a component type.
func (r *ComponentRegistry) LookupByType(typ reflect.Type) (ComponentDefinition, bool) {
	name, ok := r.byType[typ]
	if !ok {
		return ComponentDefinition{}, false
	}
	definition, ok := r.byName[name]
	return definition, ok
}

// LookupByName returns the definition registered under name.
func (r *ComponentRegistry) LookupByName(name string) (ComponentDefinition, bool) {
	definition, ok := r.byName[name]
	return definition, ok
}

// Has reports whether name is registered.
func (r *ComponentRegistry) Has(name string) bool {
	_, ok := r.byName[name]
	return ok
}
