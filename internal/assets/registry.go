package assets

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync"
)

// decoderFunc and encoderFunc are type-erased at the backend boundary. The
// public assets package adapts typed callbacks before registering them here.
type decoderFunc func(context.Context, io.Reader) (any, error)
type encoderFunc func(context.Context, io.Writer, any) error

type codecKey struct {
	typ    reflect.Type
	format string
}

type decoderRegistry struct {
	mu       sync.RWMutex
	decoders map[codecKey]decoderFunc
}

type encoderRegistry struct {
	mu       sync.RWMutex
	encoders map[codecKey]encoderFunc
}

func newDecoderRegistry() decoderRegistry {
	return decoderRegistry{decoders: make(map[codecKey]decoderFunc)}
}

func newEncoderRegistry() encoderRegistry {
	return encoderRegistry{encoders: make(map[codecKey]encoderFunc)}
}

func (r *decoderRegistry) register(typ reflect.Type, format string, decoder decoderFunc) error {
	if typ == nil {
		return fmt.Errorf("decoder type must not be nil")
	}
	if format == "" {
		return fmt.Errorf("decoder format must not be empty")
	}
	if decoder == nil {
		return fmt.Errorf("decoder for %s/%s must not be nil", typ, format)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	key := codecKey{typ: typ, format: format}
	if _, exists := r.decoders[key]; exists {
		return fmt.Errorf("decoder for %s/%s is already registered", typ, format)
	}
	r.decoders[key] = decoder
	return nil
}

func (r *decoderRegistry) lookup(typ reflect.Type, format string) (decoderFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	decoder, ok := r.decoders[codecKey{typ: typ, format: format}]
	return decoder, ok
}

func (r *encoderRegistry) register(typ reflect.Type, format string, encoder encoderFunc) error {
	if typ == nil {
		return fmt.Errorf("encoder type must not be nil")
	}
	if format == "" {
		return fmt.Errorf("encoder format must not be empty")
	}
	if encoder == nil {
		return fmt.Errorf("encoder for %s/%s must not be nil", typ, format)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	key := codecKey{typ: typ, format: format}
	if _, exists := r.encoders[key]; exists {
		return fmt.Errorf("encoder for %s/%s is already registered", typ, format)
	}
	r.encoders[key] = encoder
	return nil
}

func (r *encoderRegistry) lookup(typ reflect.Type, format string) (encoderFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	encoder, ok := r.encoders[codecKey{typ: typ, format: format}]
	return encoder, ok
}
