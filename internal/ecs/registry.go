package ecs

import (
	"fmt"
	"reflect"
	"sync"
)

type componentRegistry struct {
	mu         sync.RWMutex
	types      map[reflect.Type]*componentType
	nameToType map[string]reflect.Type
}

type componentType struct {
	Type           reflect.Type
	Name           string
	Size           int
	IsSerializable bool
	HasHooks       bool
}

var globalRegistry = &componentRegistry{
	types:      make(map[reflect.Type]*componentType),
	nameToType: make(map[string]reflect.Type),
}

func Register[T any]() *componentType {
	typ := reflect.TypeFor[T]()
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if info, exists := globalRegistry.types[typ]; exists {
		return info
	}

	info := &componentType{
		Type:           typ,
		Name:           typ.Name(),
		Size:           int(typ.Size()),
		IsSerializable: reflect.PointerTo(typ).Implements(reflect.TypeFor[Serializable]()),
		HasHooks:       reflect.PointerTo(typ).Implements(reflect.TypeFor[ComponentHooks]()),
	}
	globalRegistry.types[typ] = info
	globalRegistry.nameToType[info.Name] = typ
	return info
}

func Resolve(name string, props map[string]any) (Component, error) {
	globalRegistry.mu.RLock()
	typ, exists := globalRegistry.nameToType[name]
	globalRegistry.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("type %s not registered", name)
	}

	instance := reflect.New(typ).Elem()

	// Deserialize needs a pointer receiver to mutate the instance, so the
	// interface check happens on Addr() - but Resolve still returns the
	// mutated value (not the pointer) to match GetComponent/SetComponent's
	// value semantics used everywhere else in the engine.
	if ser, ok := instance.Addr().Interface().(Serializable); ok {
		if err := ser.Deserialize(props); err != nil {
			return nil, err
		}
		return instance.Interface().(Component), nil
	}

	for key, value := range props {
		field := instance.FieldByName(key)
		if field.IsValid() && field.CanSet() {
			val := reflect.ValueOf(value)
			if val.Type().AssignableTo(field.Type()) {
				field.Set(val)
			}
		}
	}
	return instance.Interface().(Component), nil
}
