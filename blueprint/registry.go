// Package blueprint joins data and entities: a registry of the component
// and resource types that participate in data blueprints and saves. Types
// are registered explicitly - registered means participating, and data
// referencing an unknown key fails loud with the key named.
//
// The registry resolves the game's asset codecs from the world lazily, so
// codec configuration installed after the registry is created participates
// from that point on. Registering a component also installs JSON for it on
// the game's codecs, making the type loadable and saveable through the
// asset system wherever the codecs are used.
//
// The registry is not safe for concurrent use; register during
// construction and use from the game loop, matching the signal hub's
// contract.
package blueprint

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
type Registry struct {
	world      *core.World
	codecs     *asset.Codecs
	components map[string]*componentEntry
	resources  map[string]*resourceEntry
}

// componentEntry erases T behind closures created at registration time,
// when the type is still static. The codecs instance is captured once: the
// resource is stable for the life of the game.
type componentEntry struct {
	key    string
	typ    reflect.Type
	decode func(rd io.Reader) (any, error)
	encode func(w io.Writer, value any) error
}

type resourceEntry struct {
	key string
	typ reflect.Type // the value type T; the world stores *T
}

// NewRegistry returns an empty registry bound to world. The game's asset
// codecs are resolved from the world on first use.
func NewRegistry(world *core.World) *Registry {
	return &Registry{
		world:      world,
		components: make(map[string]*componentEntry),
		resources:  make(map[string]*resourceEntry),
	}
}

// codecsOrError resolves the game's codecs from the world lazily and caches
// the resource.
func (r *Registry) codecsOrError() (*asset.Codecs, error) {
	if r.codecs != nil {
		return r.codecs, nil
	}
	codecs, err := r.world.Resource[*asset.Codecs]()
	if err != nil {
		return nil, fmt.Errorf("blueprint: resolve asset codecs: %w", err)
	}
	r.codecs = codecs
	return codecs, nil
}

// Component registers T as spawnable from data under key. JSON is installed
// for T on the game's codecs; replace it through the codecs resource for a
// custom format. The key is the stable wire identity; renaming it is a
// breaking change.
func (r *Registry) Component[T any](key string) error {
	if key == "" {
		return fmt.Errorf("blueprint: component key must not be empty")
	}
	if _, dup := r.components[key]; dup {
		return fmt.Errorf("blueprint: component key %q already registered", key)
	}
	codecs, err := r.codecsOrError()
	if err != nil {
		return err
	}
	if err := codecs.SetJSON[T](); err != nil {
		return fmt.Errorf("blueprint: component %q: %w", key, err)
	}
	r.components[key] = &componentEntry{
		key: key,
		typ: reflect.TypeFor[T](),
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
			return codecs.Encode[T](w, v, asset.FormatJSON)
		},
	}
	return nil
}

// Resource registers the resource with value type T as saveable under key.
// The world must store the resource as *T - the engine convention. T is
// encoded through the game's codecs; a T that is not a plain value
// implements json.Marshaler and json.Unmarshaler to own its wire format.
func (r *Registry) Resource[T any](key string) error {
	if key == "" {
		return fmt.Errorf("blueprint: resource key must not be empty")
	}
	if _, dup := r.resources[key]; dup {
		return fmt.Errorf("blueprint: resource key %q already registered", key)
	}
	codecs, err := r.codecsOrError()
	if err != nil {
		return err
	}
	if err := codecs.SetJSON[T](); err != nil {
		return fmt.Errorf("blueprint: resource %q: %w", key, err)
	}
	r.resources[key] = &resourceEntry{key: key, typ: reflect.TypeFor[T]()}
	return nil
}

// DecodeComponent decodes raw JSON into a new component value for key. The
// value is validated when the component implements [core.Validatable].
func (r *Registry) DecodeComponent(key string, raw []byte) (any, error) {
	entry, ok := r.components[key]
	if !ok {
		return nil, fmt.Errorf("blueprint: unknown component key %q", key)
	}
	value, err := entry.decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("blueprint: component %q: %w", key, err)
	}
	if v, ok := value.(core.Validatable); ok {
		if err := v.Validate(); err != nil {
			return nil, fmt.Errorf("blueprint: component %q: %w", key, err)
		}
	}
	return value, nil
}

// EncodeComponent writes the component value for key as JSON. The value
// must have the registered concrete type.
func (r *Registry) EncodeComponent(key string, value any, w io.Writer) error {
	entry, ok := r.components[key]
	if !ok {
		return fmt.Errorf("blueprint: unknown component key %q", key)
	}
	if err := entry.encode(w, value); err != nil {
		return fmt.Errorf("blueprint: component %q: %w", key, err)
	}
	return nil
}
