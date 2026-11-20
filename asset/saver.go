package asset

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// SaveStore publishes encoded documents.
type SaveStore interface {
	// Write publishes the output of encode at name and returns any encoding,
	// write, close, or publication error.
	//
	// Write owns the writer and calls encode synchronously at most once.
	// It must call encode before reporting success and must not retain it or
	// publish incomplete output on an encoding failure. Publication guarantees
	// depend on the store.
	Write(name string, encode func(io.Writer) error) error
}

// Saver encodes values with shared codecs. Path saves delegate publication to a
// borrowed [SaveStore]; Saver never closes the store.
// Concurrent calls require concurrency-safe encoders and storage. Callers must
// order saves to the same destination when the newest value must win.
type Saver struct {
	store  SaveStore
	codecs *Codecs
}

// SaverOptions configures a saver at construction.
type SaverOptions struct {
	// Codecs supplies shared registrations, read on each call. Nil selects the defaults.
	Codecs *Codecs
}

// NewSaver creates a saver that reads the configured codecs on each call.
// A nil store permits [Saver.SaveWriter] only. A store containing a nil
// pointer returns an error.
func NewSaver(store SaveStore, opts SaverOptions) (*Saver, error) {
	if store != nil && isNilPointer(store) {
		return nil, fmt.Errorf("asset: NewSaver: store cannot be typed nil")
	}
	codecs := opts.Codecs
	if codecs == nil {
		codecs = NewCodecs()
	}
	return &Saver{store: store, codecs: codecs}, nil
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

type saveOption interface {
	applySave(*saveOptions)
}

type saveOptions struct {
	format Format
}
