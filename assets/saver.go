package assets

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	internalassets "github.com/leonard-atorough/castrum/internal/assets"
)

// SaveOption configures the behavior of [Saver.Save] and [Saver.SavePath].
type SaveOption interface {
	applySave(*SaveOptions)
}

// SaveOptions controls how an asset is encoded and written. The zero value
// is not used directly; Save and SavePath apply their own defaults before
// applying options.
type SaveOptions struct {
	Format      Format
	AtomicWrite bool
	CreateDir   bool
}

// WithSaveFormat sets the encoding format for the save operation.
func WithSaveFormat(format Format) SaveOption {
	return saveOptionFunc(func(opts *SaveOptions) {
		opts.Format = format
	})
}

// WithSaveAtomicWrite controls whether SavePath writes atomically (write to a
// temp file, then rename). Has no effect on [Saver.Save], which writes to a
// caller-provided writer.
func WithSaveAtomicWrite(atomicWrite bool) SaveOption {
	return saveOptionFunc(func(opts *SaveOptions) {
		opts.AtomicWrite = atomicWrite
	})
}

// WithSaveCreateDir controls whether SavePath creates missing parent
// directories. Has no effect on [Saver.Save] which writes to a caller-provided writer.
func WithSaveCreateDir(createDir bool) SaveOption {
	return saveOptionFunc(func(opts *SaveOptions) {
		opts.CreateDir = createDir
	})
}

func (f saveOptionFunc) applySave(opts *SaveOptions) {
	f(opts)
}

type saveOptionFunc func(*SaveOptions)

// Encoder is a function type that encodes a value of type T to a writer
// in a specific format.
type Encoder[T any] func(context.Context, io.Writer, T) error

// Saver encodes assets to the host filesystem or an [io.Writer] using
// format-specific encoders. A Saver created by [NewAssets] shares an internal
// service with the [Loader], so saving an asset invalidates the loader's
// cached copy.
type Saver struct {
	service    *internalassets.Service
	invalidate func(ID)
}

func newSaver(service *internalassets.Service, invalidate func(ID)) *Saver {
	return &Saver{service: service, invalidate: invalidate}
}

// SavePath encodes value and writes it to path on the host filesystem.
// The format is inferred from the file extension unless [WithSaveFormat]
// is provided. By default, missing directories are created and the write
// is atomic (write to temp, then rename).
func (s *Saver) SavePath[T any](ctx context.Context, path string, value T, options ...SaveOption) error {
	path = normalizeSavePath(path)
	opts := &SaveOptions{CreateDir: true, AtomicWrite: true}
	for _, option := range options {
		option.applySave(opts)
	}
	if opts.Format == "" {
		opts.Format = resolveFormat(path, opts.Format)
	}

	if opts.CreateDir {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return &AssetError{
				Message: fmt.Sprintf("failed to create directory %s", dir),
				Err:     err,
				Source:  "SavePath",
			}
		}
	}

	if !opts.AtomicWrite {
		writer, err := os.Create(path)
		if err != nil {
			return &AssetError{
				Message: fmt.Sprintf("failed to create file %s", path),
				Err:     err,
				Source:  "SavePath",
			}
		}
		if err := s.saveToFile(ctx, writer, path, value, *opts); err != nil {
			return err
		}
		if s.invalidate != nil {
			s.invalidate(assetIDForSavePath(path))
		} else {
			s.service.Invalidate(string(assetIDForSavePath(path)))
		}
		return nil
	}

	dir := filepath.Dir(path)
	temporary, err := os.CreateTemp(dir, ".castrum-*"+filepath.Ext(path))
	if err != nil {
		return &AssetError{
			Message: fmt.Sprintf("failed to create temporary file for %s", path),
			Err:     err,
			Source:  "SavePath",
		}
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := s.saveToFile(ctx, temporary, path, value, *opts); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return &AssetError{
			Message: fmt.Sprintf("failed to replace file %s", path),
			Err:     err,
			Source:  "SavePath",
		}
	}

	if s.invalidate != nil {
		s.invalidate(assetIDForSavePath(path))
	} else {
		s.service.Invalidate(string(assetIDForSavePath(path)))
	}
	return nil
}

// Save encodes value and writes it to writer. The format must be specified
// via [WithSaveFormat] since there is no file extension to infer from.
// [WithSaveAtomicWrite] and [WithSaveCreateDir] are rejected as errors
// since they have no meaning when writing to a writer.
func (s *Saver) Save[T any](ctx context.Context, writer io.Writer, value T, options ...SaveOption) error {
	opts := &SaveOptions{}
	for _, option := range options {
		option.applySave(opts)
	}
	if opts.Format == "" {
		return &AssetError{
			Message: "format is required when saving to a writer",
			Source:  "Save",
		}
	}
	if opts.AtomicWrite || opts.CreateDir {
		return &AssetError{
			Message: "AtomicWrite and CreateDir options are ignored when saving to a writer",
			Source:  "Save",
		}
	}
	return s.save(ctx, writer, value, *opts, "Save")
}

// RegisterEncoder registers an encoder for the given format and type T.
// If override is true, an existing encoder for the same type and format
// is replaced; otherwise an error is returned if one is already registered.
func (s *Saver) RegisterEncoder[T any](format Format, encoder Encoder[T], override bool) error {
	if encoder == nil {
		return &AssetError{
			Message: fmt.Sprintf("encoder for format %s must not be nil", format),
			Source:  "RegisterEncoder",
		}
	}
	typ := reflect.TypeFor[T]()
	err := s.service.RegisterEncoder(typ, string(format), func(ctx context.Context, w io.Writer, v any) error {
		return encoder(ctx, w, v.(T))
	}, override)
	if err != nil {
		return &AssetError{
			Message: fmt.Sprintf("failed to register encoder for format %s", format),
			Err:     err,
			Source:  "RegisterEncoder",
		}
	}
	return nil
}

func (s *Saver) saveToFile[T any](ctx context.Context, writer *os.File, path string, value T, opts SaveOptions) error {
	if err := s.save(ctx, writer, value, opts, "SavePath"); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return &AssetError{
			Message: fmt.Sprintf("failed to close file %s", path),
			Err:     err,
			Source:  "SavePath",
		}
	}
	return nil
}

func (s *Saver) save[T any](ctx context.Context, writer io.Writer, value T, opts SaveOptions, source string) error {
	typ := reflect.TypeFor[T]()
	if err := s.service.Encode(ctx, typ, string(opts.Format), writer, value); err != nil {
		return &AssetError{
			Message: fmt.Sprintf("failed to encode asset to format %s", opts.Format),
			Err:     err,
			Source:  source,
		}
	}
	return nil
}
