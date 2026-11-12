package asset

import (
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"strings"
)

// SaveStore publishes encoded documents.
//
// Saver supplies encode bound to the selected encoder and value; the store
// need not know their types or formats. Write calls encode synchronously at
// most once and must call it before reporting success. It must not retain
// encode or publish incomplete output on an encoding failure. The store owns
// its writer and reports write, close, and publication errors. Publication
// guarantees depend on the implementation.
type SaveStore interface {
	Write(name string, encode func(io.Writer) error) error
}

// Saver encodes values with private codec registrations. Path saves delegate
// publication to a borrowed [SaveStore]; Saver never closes the store.
// Concurrent calls require concurrency-safe encoders and storage. Callers must
// order saves to the same destination when the newest value must win.
type Saver struct {
	store  SaveStore
	codecs *Codecs
}

// SaverOptions configures a saver at construction.
type SaverOptions struct {
	// Codecs supplies registrations to snapshot. Nil selects the defaults.
	Codecs *Codecs
}

// NewSaver creates a saver with a snapshot of the configured codecs.
// A nil store permits [Saver.SaveWriter] only. A store containing a nil
// pointer returns an error.
func NewSaver(store SaveStore, opts SaverOptions) (*Saver, error) {
	if store != nil && isNilPointer(store) {
		return nil, fmt.Errorf("asset: NewSaver: store cannot be typed nil")
	}
	return &Saver{store: store, codecs: opts.Codecs.snapshot()}, nil
}

// Save encodes value and publishes it at name through the store. The format
// defaults to the lowercased extension; [WithFormat] overrides it. Names are
// cleaned slash-separated paths relative to the store. Missing encoders and
// invalid names fail before the store is called. Save does not invalidate
// loader caches.
func (s *Saver) Save[T any](name string, value T, opts ...saveOption) error {
	if name == "" {
		return &AssetError{Op: "Save", Err: fmt.Errorf("asset path must not be empty")}
	}
	name = normalizeAssetPath(name)
	if name == "." || !fs.ValidPath(name) || strings.Contains(name, "\\") {
		return &AssetError{Op: "Save", ID: ID(name), Err: fmt.Errorf("invalid asset path")}
	}
	options := saveOptions{format: resolveFormat(name)}
	for _, opt := range opts {
		if isNilPointer(opt) {
			return &AssetError{Op: "Save", ID: ID(name), Err: fmt.Errorf("option cannot be nil")}
		}
		opt.applySave(&options)
	}
	encoder, err := s.codecs.encoder[T](options.format)
	if err != nil {
		return &AssetError{Op: "Save", ID: ID(name), Err: err}
	}
	if s.store == nil {
		return &AssetError{Op: "Save", ID: ID(name), Err: fmt.Errorf("no save store configured")}
	}
	if err := s.store.Write(name, func(writer io.Writer) error {
		if isNilPointer(writer) {
			return &AssetError{Op: "Encode", ID: ID(name), Err: fmt.Errorf("writer cannot be nil")}
		}
		if err := encoder(writer, value); err != nil {
			return &AssetError{Op: "Encode", ID: ID(name), Err: err}
		}
		return nil
	}); err != nil {
		return &AssetError{Op: "Save", ID: ID(name), Err: err}
	}
	return nil
}

// SaveWriter encodes value to writer in format without closing the writer
// or using the store. Nil writers and missing encoders return errors.
func (s *Saver) SaveWriter[T any](writer io.Writer, value T, format Format) error {
	if isNilPointer(writer) {
		return &AssetError{Op: "Encode", Err: fmt.Errorf("writer cannot be nil")}
	}
	encoder, err := s.codecs.encoder[T](format)
	if err != nil {
		return &AssetError{Op: "Encode", Err: err}
	}
	if err := encoder(writer, value); err != nil {
		return &AssetError{Op: "Encode", Err: err}
	}
	return nil
}

// RegisterEncoder adds e for T and format to this saver only. Nil encoders,
// empty formats, and duplicate registrations are errors. Configure replacements
// with [Codecs.SetEncoder] before constructing the saver.
func (s *Saver) RegisterEncoder[T any](format Format, e Encoder[T]) error {
	c := s.codecs
	c.mu.Lock()
	defer c.mu.Unlock()
	key := codecKey{reflect.TypeFor[T](), format}
	if _, exists := c.encoders[key]; exists {
		return fmt.Errorf("encoder for type %v and format %q already exists", key.typ, format)
	}
	return c.setEncoder(key, e)
}

type saveOption interface {
	applySave(*saveOptions)
}

type saveOptions struct {
	format Format
}
