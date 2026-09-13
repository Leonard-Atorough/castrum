package assets

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"

	internalassets "github.com/leonard-atorough/castrum/internal/assets"
)

type SaveOption interface {
	applySave(*SaveOptions)
}

type SaveOptions struct {
	Format      Format
	AtomicWrite bool
	CreateDir   bool
}

func WithSaveFormat(format Format) SaveOption {
	return saveOptionFunc(func(opts *SaveOptions) {
		opts.Format = format
	})
}

func WithSaveAtomicWrite(atomicWrite bool) SaveOption {
	return saveOptionFunc(func(opts *SaveOptions) {
		opts.AtomicWrite = atomicWrite
	})
}

func WithSaveCreateDir(createDir bool) SaveOption {
	return saveOptionFunc(func(opts *SaveOptions) {
		opts.CreateDir = createDir
	})
}

func (f saveOptionFunc) applySave(opts *SaveOptions) {
	f(opts)
}

type saveOptionFunc func(*SaveOptions)

type Encoder[T any] func(context.Context, io.Writer, T) error

type Saver struct {
	service    *internalassets.Service
	invalidate func(ID)
}

func NewSaver(filesystem fs.FS) *Saver {
	return &Saver{service: newAssetService(filesystem)}
}

func newSaver(service *internalassets.Service, invalidate func(ID)) *Saver {
	return &Saver{service: service, invalidate: invalidate}
}

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
		return s.saveToFile(ctx, writer, path, value, *opts)
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

func (s *Saver) Save[T any](ctx context.Context, writer io.Writer, value T, options ...SaveOption) error {
	opts := &SaveOptions{CreateDir: true, AtomicWrite: true}
	for _, option := range options {
		option.applySave(opts)
	}
	if opts.Format == "" {
		return &AssetError{
			Message: "format is required when saving to a writer",
			Source:  "Save",
		}
	}
	return s.save(ctx, writer, value, *opts, "Save")
}

func (s *Saver) RegisterEncoder[T any](format Format, encoder Encoder[T], override bool) error {
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
