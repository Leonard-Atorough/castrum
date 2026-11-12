package asset

import (
	"fmt"
	"io"
	"maps"
	"reflect"
	"sync"
)

// Decoder decodes a value of type T from a reader without closing it.
// The returned value must not retain the reader.
type Decoder[T any] func(io.Reader) (T, error)

// Codecs configures typed decoders. Its zero value is empty.
// Loaders copy its registrations at construction; later changes
// do not affect existing loaders. Copies of registrations share their closures,
// which must support concurrent calls if used concurrently.
//
// Registration is safe for concurrent use. Codecs must not be copied.
type Codecs struct {
	mu       sync.RWMutex
	decoders map[codecKey]any
}

// NewCodecs returns codecs with backend-free texture and font decoders and
// JSON decoding for [AtlasMeta]. Audio decoders are runner-owned.
func NewCodecs() *Codecs {
	c := &Codecs{}
	c.registerDefaults()
	return c
}

// SetDecoder installs or replaces d for T and format. Nil decoders and
// empty formats are errors. Existing loaders are unaffected.
func (c *Codecs) SetDecoder[T any](format Format, d Decoder[T]) error {
	if c == nil {
		return fmt.Errorf("codecs cannot be nil")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.setDecoder(codecKey{reflect.TypeFor[T](), format}, d)
}

// The caller holds c.mu when updating the table.
func (c *Codecs) setDecoder[T any](key codecKey, d Decoder[T]) error {
	if d == nil {
		return fmt.Errorf("decoder cannot be nil")
	}
	if key.format == "" {
		return fmt.Errorf("format cannot be empty")
	}
	if c.decoders == nil {
		c.decoders = make(map[codecKey]any)
	}
	c.decoders[key] = d
	return nil
}

func (c *Codecs) snapshot() *Codecs {
	if c == nil {
		return NewCodecs()
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &Codecs{decoders: maps.Clone(c.decoders)}
}

func (c *Codecs) decoder[T any](format Format) (Decoder[T], error) {
	c.mu.RLock()
	d, ok := c.decoders[codecKey{reflect.TypeFor[T](), format}]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no decoder registered for type %v and format %q", reflect.TypeFor[T](), format)
	}
	return d.(Decoder[T]), nil
}

type codecKey struct {
	typ    reflect.Type
	format Format
}

func isNilPointer(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	return v.Kind() == reflect.Pointer && v.IsNil()
}
