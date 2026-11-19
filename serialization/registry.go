package serialization

import (
	"bytes"
	"fmt"
	"io"
	"reflect"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
)

// Registry maps stable keys to the component and resource types that cross
// the data boundary. Keys are wire identities: do not rename them across
// releases.
//
// Registration installs or replaces the JSON codec for the type using
// [asset.Codecs.SetJSON]. To customize JSON encoding or decoding, replace
// the corresponding codec after registering the type.
//
// A Registry is not safe for concurrent use. Register during construction,
// then use it from the game loop.
type Registry struct {
	world          *core.World
	codecs         *asset.Codecs
	components     map[string]*componentEntry
	componentTypes map[reflect.Type]*componentEntry
	resources      map[string]*resourceEntry
	resourceTypes  map[reflect.Type]*resourceEntry
}

var validatableType = reflect.TypeFor[core.Validatable]()

type componentEntry struct {
	key    string
	typ    reflect.Type
	decode func(rd io.Reader) (any, error)
	encode func(w io.Writer, value any) error
}

type resourceEntry struct {
	key    string
	typ    reflect.Type
	decode func(rd io.Reader) (any, error)
	encode func(w io.Writer, value any) error
}

// NewRegistry returns an empty registry bound to world.
//
// The registry looks up [asset.Codecs] in world when registration first
// needs it, then keeps using that instance.
func NewRegistry(world *core.World) *Registry {
	return &Registry{
		world:          world,
		components:     make(map[string]*componentEntry),
		componentTypes: make(map[reflect.Type]*componentEntry),
		resources:      make(map[string]*resourceEntry),
		resourceTypes:  make(map[reflect.Type]*resourceEntry),
	}
}

func (r *Registry) codecsOrError() (*asset.Codecs, error) {
	if r.codecs != nil {
		return r.codecs, nil
	}
	if r.world == nil {
		return nil, fmt.Errorf("serialization: resolve asset codecs: registry has no world")
	}
	codecs, err := r.world.Resource[*asset.Codecs]()
	if err != nil {
		return nil, fmt.Errorf("serialization: resolve asset codecs: %w", err)
	}
	r.codecs = codecs
	return codecs, nil
}

// Component registers T under key for spawning components from data.
//
// It returns an error if key is empty or already registered as a component,
// T is a pointer type or is already registered as a component under
// another key, only *T implements [core.Validatable], or the codecs
// resource cannot be resolved.
func (r *Registry) Component[T any](key string) error {
	if key == "" {
		return fmt.Errorf("serialization: component key must not be empty")
	}
	if _, dup := r.components[key]; dup {
		return fmt.Errorf("serialization: component key %q already registered", key)
	}

	typ := reflect.TypeFor[T]()
	if typ.Kind() == reflect.Pointer {
		return fmt.Errorf("serialization: component %q must not be a pointer type", key)
	}
	if existing, dup := r.componentTypes[typ]; dup {
		return fmt.Errorf("serialization: component %q: type %v already registered as %q", key, typ, existing.key)
	}
	// DecodeComponent validates the boxed value T, so a pointer-receiver
	// Validate would be skipped silently. Reject it instead.
	if reflect.PointerTo(typ).Implements(validatableType) && !typ.Implements(validatableType) {
		return fmt.Errorf("serialization: component %q: Validate must have a value receiver", key)
	}
	codecs, err := r.codecsOrError()
	if err != nil {
		return err
	}
	if err := codecs.SetJSON[T](); err != nil {
		return fmt.Errorf("serialization: component %q: %w", key, err)
	}
	entry := &componentEntry{
		key: key,
		typ: typ,
		decode: func(rd io.Reader) (any, error) {
			value, err := codecs.Decode[T](rd, asset.FormatJSON)
			if err != nil {
				return nil, err
			}
			return value, nil
		},
		encode: func(w io.Writer, value any) error {
			v, ok := value.(T)
			if !ok {
				return fmt.Errorf("value type %T, want %v", value, reflect.TypeFor[T]())
			}
			return codecs.Encode(w, v, asset.FormatJSON)
		},
	}
	r.components[key] = entry
	r.componentTypes[typ] = entry
	return nil
}

// Resource registers T under key for resource serialization.
//
// It returns an error if key is empty or already registered as a resource,
// T is a pointer type or is already registered as a resource under
// another key, or the codecs resource cannot be resolved.
func (r *Registry) Resource[T any](key string) error {
	if key == "" {
		return fmt.Errorf("serialization: resource key must not be empty")
	}
	if _, dup := r.resources[key]; dup {
		return fmt.Errorf("serialization: resource key %q already registered", key)
	}

	typ := reflect.TypeFor[T]()
	if typ.Kind() == reflect.Pointer {
		return fmt.Errorf("serialization: resource %q must not be a pointer type", key)
	}
	if existing, dup := r.resourceTypes[typ]; dup {
		return fmt.Errorf("serialization: resource %q: type %v already registered as %q", key, typ, existing.key)
	}

	codecs, err := r.codecsOrError()
	if err != nil {
		return err
	}
	if err := codecs.SetJSON[T](); err != nil {
		return fmt.Errorf("serialization: resource %q: %w", key, err)
	}
	entry := &resourceEntry{
		key: key,
		typ: typ,
		decode: func(rd io.Reader) (any, error) {
			value, err := codecs.Decode[T](rd, asset.FormatJSON)
			if err != nil {
				return nil, err
			}
			return value, nil
		},
		encode: func(w io.Writer, value any) error {
			v, ok := value.(T)
			if !ok {
				return fmt.Errorf("value type %T, want %v", value, reflect.TypeFor[T]())
			}
			return codecs.Encode(w, v, asset.FormatJSON)
		},
	}
	r.resources[key] = entry
	r.resourceTypes[typ] = entry
	return nil
}

// DecodeResource decodes raw JSON into a value of the resource's registered
// type.
//
// It validates values that implement [core.Validatable]. A Validate method
// defined only on a pointer to the registered type is not called. If the key
// is unknown or decoding or validation fails, the error names the key.
func (r *Registry) DecodeResource(key string, raw []byte) (any, error) {
	entry, ok := r.resources[key]
	if !ok {
		return nil, fmt.Errorf("serialization: unknown resource key %q", key)
	}
	value, err := entry.decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("serialization: resource %q: %w", key, err)
	}
	if v, ok := value.(core.Validatable); ok {
		if err := v.Validate(); err != nil {
			return nil, fmt.Errorf("serialization: resource %q: %w", key, err)
		}
	}
	return value, nil
}

// EncodeResource writes the resource value for key as JSON. The value must
// have the registered concrete type.
func (r *Registry) EncodeResource(key string, value any, w io.Writer) error {
	entry, ok := r.resources[key]
	if !ok {
		return fmt.Errorf("serialization: unknown resource key %q", key)
	}
	if err := entry.encode(w, value); err != nil {
		return fmt.Errorf("serialization: resource %q: %w", key, err)
	}
	return nil
}

// DecodeComponent decodes raw JSON into a component value for key.
//
// It validates values that implement [core.Validatable]. If the key is
// unknown or decoding or validation fails, the error names the key.
func (r *Registry) DecodeComponent(key string, raw []byte) (any, error) {
	entry, ok := r.components[key]
	if !ok {
		return nil, fmt.Errorf("serialization: unknown component key %q", key)
	}
	value, err := entry.decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("serialization: component %q: %w", key, err)
	}
	if v, ok := value.(core.Validatable); ok {
		if err := v.Validate(); err != nil {
			return nil, fmt.Errorf("serialization: component %q: %w", key, err)
		}
	}
	return value, nil
}

// EncodeComponent writes the component value for key as JSON. The value
// must have the registered concrete type.
func (r *Registry) EncodeComponent(key string, value any, w io.Writer) error {
	entry, ok := r.components[key]
	if !ok {
		return fmt.Errorf("serialization: unknown component key %q", key)
	}
	if err := entry.encode(w, value); err != nil {
		return fmt.Errorf("serialization: component %q: %w", key, err)
	}
	return nil
}

// ComponentType returns the value type registered under key and whether the
// key is registered.
func (r *Registry) ComponentType(key string) (reflect.Type, bool) {
	entry, ok := r.components[key]
	if !ok {
		return nil, false
	}
	return entry.typ, true
}

// KeyForType returns the key registered for the component type typ and
// whether typ is a registered component.
func (r *Registry) KeyForType(typ reflect.Type) (string, bool) {
	entry, ok := r.componentTypes[typ]
	if !ok {
		return "", false
	}
	return entry.key, true
}
