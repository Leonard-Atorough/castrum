package assets

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
)

type request struct {
	path   string
	result chan result
}

type result struct {
	value any
	err   error
}

type workerGroup struct {
	workers []worker
}

type worker struct {
	id int
}

type Service struct {
	filesystem fs.FS
	decoders   decoderRegistry
	encoders   encoderRegistry
	cache      *cache
	jobs       chan request
	workers    workerGroup
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
	}
}

func (s *Service) Filesystem() fs.FS {
	return s.filesystem
}

func (s *Service) RegisterDecoder(typ reflect.Type, format string, decoder func(context.Context, io.Reader) (any, error)) error {
	return s.decoders.register(typ, format, decoder)
}

func (s *Service) RegisterEncoder(typ reflect.Type, format string, encoder func(context.Context, io.Writer, any) error) error {
	return s.encoders.register(typ, format, encoder)
}

func (s *Service) Decoder(typ reflect.Type, format string) (decoderFunc, error) {
	decoder, ok := s.decoders.lookup(typ, format)
	if !ok {
		return nil, fmt.Errorf("decoder not registered for %s/%s", typ, format)
	}
	return decoder, nil
}

func (s *Service) Encoder(typ reflect.Type, format string) (encoderFunc, error) {
	encoder, ok := s.encoders.lookup(typ, format)
	if !ok {
		return nil, fmt.Errorf("encoder not registered for %s/%s", typ, format)
	}
	return encoder, nil
}

func (s *Service) Cached(id string, typ reflect.Type, format string) (any, bool) {
	return s.cache.get(cacheKey{id: id, typ: typ, format: format})
}

func (s *Service) Cache(id string, typ reflect.Type, format string, value any) {
	s.cache.put(cacheKey{id: id, typ: typ, format: format}, value)
}

// TODO: add path normalization, in-flight load deduplication, and explicit
// worker shutdown once Loader and Saver delegate real I/O here.
