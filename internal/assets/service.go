package assets

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"

	"golang.org/x/sync/singleflight"
)

type Service struct {
	filesystem fs.FS
	decoders   decoderRegistry
	encoders   encoderRegistry
	cache      *cache
	loadGroup  *singleflight.Group
}

// NewService creates the backend owner for public loaders and savers.
// Worker startup and shutdown are intentionally deferred until the execution
// semantics of queued requests are finalized.
func NewService(filesystem fs.FS) *Service {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}
	return &Service{
		filesystem: filesystem,
		decoders:   newDecoderRegistry(),
		encoders:   newEncoderRegistry(),
		cache:      newCache(),
		loadGroup:  &singleflight.Group{},
	}
}

func (s *Service) Filesystem() fs.FS {
	return s.filesystem
}

func (s *Service) LoadGroup() *singleflight.Group {
	return s.loadGroup
}

func (s *Service) RegisterDecoder(typ reflect.Type, format string, decoder func(context.Context, io.Reader) (any, error), override bool) error {
	return s.decoders.register(typ, format, decoder, override)
}

func (s *Service) RegisterEncoder(typ reflect.Type, format string, encoder func(context.Context, io.Writer, any) error, override bool) error {
	return s.encoders.register(typ, format, encoder, override)
}

// Decode resolves and executes a decoder. Keeping codec execution here makes
// the service the owner of lookup, invocation, and result validation.
func (s *Service) Decode(ctx context.Context, typ reflect.Type, format string, reader io.Reader) (any, error) {
	decoder, ok := s.decoders.lookup(typ, format)
	if !ok {
		return nil, fmt.Errorf("decoder not registered for %s/%s", typeName(typ), format)
	}
	value, err := decoder(ctx, reader)
	if err != nil {
		return nil, err
	}
	if value == nil && typ != nil && typ.Kind() != reflect.Pointer && typ.Kind() != reflect.Interface {
		return nil, fmt.Errorf("decoder returned nil for %s/%s", typeName(typ), format)
	}
	if value != nil && !reflect.TypeOf(value).AssignableTo(typ) {
		return nil, fmt.Errorf("decoder returned %s, want %s", reflect.TypeOf(value), typeName(typ))
	}
	return value, nil
}

// Encode resolves and executes an encoder.
func (s *Service) Encode(ctx context.Context, typ reflect.Type, format string, writer io.Writer, value any) error {
	encoder, ok := s.encoders.lookup(typ, format)
	if !ok {
		return fmt.Errorf("encoder not registered for %s/%s", typeName(typ), format)
	}
	if value == nil {
		return fmt.Errorf("cannot encode nil as %s", typeName(typ))
	}
	valueType := reflect.TypeOf(value)
	if !valueType.AssignableTo(typ) {
		return fmt.Errorf("cannot encode %s as %s", valueType, typeName(typ))
	}
	return encoder(ctx, writer, value)
}

func (s *Service) Cached(id string, typ reflect.Type, format string) (any, bool) {
	return s.cache.get(NewLoadKey(id, typ, format))
}

func (s *Service) Cache(id string, typ reflect.Type, format string, value any) {
	s.cache.put(NewLoadKey(id, typ, format), value)
}

func (s *Service) Invalidate(id string) {
	s.cache.invalidate(id)
}

func (s *Service) ClearCache() {
	s.cache.clear()
}

func NewLoadKey(id string, typ reflect.Type, format string) loadKey {
	return loadKey{id: id, typ: typ, format: format}
}

func typeName(typ reflect.Type) string {
	if typ == nil {
		return "<nil>"
	}
	return typ.String()
}

// TODO: add explicit worker shutdown if queued execution is introduced.
