package core

import (
	"fmt"
	"reflect"
	"sync"
)

type ComponentRegistry struct {
	mu         sync.RWMutex
	types      map[reflect.Type]*ComponentType
	nameToType map[string]reflect.Type
}

type ComponentType struct {
	Type           reflect.Type
	Name           string
	Size           int
	IsSerializable bool
	HasHooks       bool
}

var GlobalRegistry = &ComponentRegistry{
	types:      make(map[reflect.Type]*ComponentType),
	nameToType: make(map[string]reflect.Type),
}

func Register[T any]() *ComponentType {
	typ := reflect.TypeFor[T]()
	GlobalRegistry.mu.Lock()
	defer GlobalRegistry.mu.Unlock()

	if info, exists := GlobalRegistry.types[typ]; exists {
		return info
	}

	info := &ComponentType{
		Type:           typ,
		Name:           typ.Name(),
		Size:           int(typ.Size()),
		IsSerializable: reflect.PointerTo(typ).Implements(reflect.TypeFor[Serializable]()),
		HasHooks:       reflect.PointerTo(typ).Implements(reflect.TypeFor[ComponentHooks]()),
	}
	GlobalRegistry.types[typ] = info
	GlobalRegistry.nameToType[info.Name] = typ
	return info
}

func RegisterMultiple(types ...reflect.Type) {
	for _, typ := range types {
		GlobalRegistry.mu.Lock()
		if _, exists := GlobalRegistry.types[typ]; !exists {
			info := &ComponentType{
				Type:           typ,
				Name:           typ.Name(),
				Size:           int(typ.Size()),
				IsSerializable: reflect.PointerTo(typ).Implements(reflect.TypeFor[Serializable]()),
				HasHooks:       reflect.PointerTo(typ).Implements(reflect.TypeFor[ComponentHooks]()),
			}
			GlobalRegistry.types[typ] = info
			GlobalRegistry.nameToType[info.Name] = typ
		}
		GlobalRegistry.mu.Unlock()
	}
}

func Resolve(name string, props map[string]any) (Component, error) {
	GlobalRegistry.mu.RLock()
	typ, exists := GlobalRegistry.nameToType[name]
	GlobalRegistry.mu.RUnlock()
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

func GetTypeInfo(typ reflect.Type) *ComponentType {
	GlobalRegistry.mu.RLock()
	defer GlobalRegistry.mu.RUnlock()

	return GlobalRegistry.types[typ]
}

func ListTypeInfo() []*ComponentType {
	GlobalRegistry.mu.RLock()
	defer GlobalRegistry.mu.RUnlock()

	list := make([]*ComponentType, 0, len(GlobalRegistry.types))
	for _, info := range GlobalRegistry.types {
		list = append(list, info)
	}
	return list
}
