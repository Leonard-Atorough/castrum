package assets

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"reflect"
	"strings"

	internalassets "github.com/leonard-atorough/castrum/internal/assets"
)

type Assets struct {
	Loader *Loader
	Saver  *Saver
}

func NewAssets(filesystem fs.FS) *Assets {
	service := internalassets.NewService(filesystem)
	return &Assets{
		Loader: newLoader(service),
		Saver:  newSaver(service),
	}
}

// ID represents the unique identifier for an asset.
// It is used for caching and retrieval purposes within the loader.
type ID string

// Format represents the format of an asset.
// It is used to determine the appropriate decoder for the asset.
type Format string

const (
	FormatJSON Format = "json"
	FormatXML  Format = "xml"
	FormatYAML Format = "yaml"
	FormatYML  Format = "yml"
	FormatTOML Format = "toml"
	FormatPNG  Format = "png"
	FormatJPG  Format = "jpg"
	FormatJPEG Format = "jpeg"
	FormatWAV  Format = "wav"
	FormatMP3  Format = "mp3"
	FormatOGG  Format = "ogg"
)

// LoadOption represents an option that can be applied when loading an asset.
// It allows configuring the loading behavior, such as specifying the format, ID, or cache policy.
type LoadOption interface {
	applyLoad(*LoadOptions)
}

// LoadOptions represents the options that can be applied when loading an asset.
// It includes the format, ID, and cache policy for the asset.
type LoadOptions struct {
	Format      Format
	ID          ID
	CachePolicy CachePolicy
}

// WithFormat specifies the format to be used when loading an asset.
func WithFormat(format Format) LoadOption {
	return loadOptionFunc(func(opts *LoadOptions) {
		opts.Format = format
	})
}

// WithID specifies the ID to be used when loading an asset.
func WithID(id ID) LoadOption {
	return loadOptionFunc(func(opts *LoadOptions) {
		opts.ID = id
	})
}
// WithCache specifies the cache policy to be used when loading an asset.
func WithCache(policy CachePolicy) LoadOption {
	return loadOptionFunc(func(opts *LoadOptions) {
		opts.CachePolicy = policy
	})
}

type loadOptionFunc func(*LoadOptions)

func (f loadOptionFunc) applyLoad(opts *LoadOptions) {
	f(opts)
}

// CachePolicy represents the caching policy for an asset.
// It determines whether the asset should be cached and how it should be retrieved from the cache.
type CachePolicy int

const (
	CachePolicyNone CachePolicy = iota
	CachePolicyDefault
)

// Decoder is a function type that defines how to decode an asset from an io.Reader.
type Decoder[T any] func(context.Context, io.Reader) (T, error)

// Loader is responsible for loading assets from the filesystem.
type Loader struct {
	service *internalassets.Service
}

func NewLoader(filesystem fs.FS) *Loader {
	return &Loader{service: internalassets.NewService(filesystem)}
}

func newLoader(service *internalassets.Service) *Loader {
	return &Loader{service: service}
}

// Load loads an asset from the specified path using the provided options.
// It first checks the cache based on the ID and cache policy.
// If the asset is not cached, it reads the asset from the filesystem and decodes it using the appropriate decoder.
func (l *Loader) Load[T any](ctx context.Context, path string, options ...LoadOption) (T, error) {
	var result T
	typ := reflect.TypeFor[T]()

	opts := &LoadOptions{CachePolicy: CachePolicyDefault}
	for _, option := range options {
		option.applyLoad(opts)
	}

	format := resolveFormat(path, opts.Format)
	id := opts.ID
	if id == "" {
		id = ID(pathpkg.Clean(path))
	}

	if opts.CachePolicy != CachePolicyNone {
		if cached, ok := l.service.Cached(string(id), typ, string(format)); ok {
			result, ok := cached.(T)
			if ok {
				return result, nil
			}
			return result, &ResourceError{Message: fmt.Sprintf("cached asset has unexpected type for %s", id), Source: "Load"}
		}
	}

	fs := l.service.Filesystem()
	sr, err := fs.Open(path)
	if err != nil {
		return result, &ResourceError{
			Message: fmt.Sprintf("failed to open file %s", path),
			Err:     err,
			Source:  "Load",
		}
	}
	defer sr.Close()

	res, err := l.service.Decode(ctx, typ, string(format), sr)
	if err != nil {
		return result, &ResourceError{
			Message: fmt.Sprintf("failed to decode asset from file %s", path),
			Err:     err,
			Source:  "Load",
		}
	}
	result, ok := res.(T)
	if !ok {
		return result, &ResourceError{Message: fmt.Sprintf("decoder returned an unexpected type for %s", id), Source: "Load"}
	}
	if opts.CachePolicy != CachePolicyNone {
		l.service.Cache(string(id), typ, string(format), result)
	}
	return result, nil
}

// LoadReader loads an asset from the provided reader using the specified options.
// Since the reader has no stable identity, caching by ID is not applicable.
// The asset is decoded directly from the reader without caching.
func (l *Loader) LoadReader[T any](ctx context.Context, reader io.Reader, options ...LoadOption) (T, error) {
	var result T
	typ := reflect.TypeFor[T]()
	opts := &LoadOptions{}
	for _, option := range options {
		option.applyLoad(opts)
	}

	res, err := l.service.Decode(ctx, typ, string(opts.Format), reader)
	if err != nil {
		return result, &ResourceError{
			Message: "failed to decode asset from reader",
			Err:     err,
			Source:  "LoadReader",
		}
	}
	result, ok := res.(T)
	if !ok {
		return result, &ResourceError{Message: "decoder returned an unexpected type", Source: "LoadReader"}
	}
	return result, nil
}

// RegisterDecoder registers a decoder for the specified format.
// The decoder will be used to decode assets of the given type from readers.
// The override parameter determines whether an existing decoder for the same type and format should be replaced.
func (l *Loader) RegisterDecoder[T any](format Format, decoder Decoder[T], override bool) error {
	typ := reflect.TypeFor[T]()
	err := l.service.RegisterDecoder(typ, string(format), func(ctx context.Context, r io.Reader) (any, error) {
		return decoder(ctx, r)
	}, override)
	if err != nil {
		return &ResourceError{
			Message: fmt.Sprintf("failed to register decoder for format %s", format),
			Err:     err,
			Source:  "RegisterDecoder",
		}
	}
	return nil
}

func (l *Loader) Invalidate(id ID) {
	l.service.Invalidate(string(id))
}

func (l *Loader) ClearCache() {
	l.service.ClearCache()
}

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
	service *internalassets.Service
}

func NewSaver(filesystem fs.FS) *Saver {
	return &Saver{service: internalassets.NewService(filesystem)}
}

func newSaver(service *internalassets.Service) *Saver {
	return &Saver{service: service}
}

func (s *Saver) SavePath[T any](ctx context.Context, path string, value T, options ...SaveOption) error {
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
			return &ResourceError{
				Message: fmt.Sprintf("failed to create directory %s", dir),
				Err:     err,
				Source:  "SavePath",
			}
		}
	}

	if !opts.AtomicWrite {
		writer, err := os.Create(path)
		if err != nil {
			return &ResourceError{
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
		return &ResourceError{
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
		return &ResourceError{
			Message: fmt.Sprintf("failed to replace file %s", path),
			Err:     err,
			Source:  "SavePath",
		}
	}
	return nil
}

func (s *Saver) Save[T any](ctx context.Context, writer io.Writer, value T, options ...SaveOption) error {
	opts := &SaveOptions{CreateDir: true, AtomicWrite: true}
	for _, option := range options {
		option.applySave(opts)
	}
	if opts.Format == "" {
		return &ResourceError{
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
		return &ResourceError{
			Message: fmt.Sprintf("failed to register encoder for format %s", format),
			Err:     err,
			Source:  "RegisterEncoder",
		}
	}
	return nil
}

// ResourceError represents an error that occurred during the loading or saving of an asset.
type ResourceError struct {
	Message string
	Err     error
	Source  string
}

func (e *ResourceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v (source: %s)", e.Message, e.Err, e.Source)
	}
	return fmt.Sprintf("%s (source: %s)", e.Message, e.Source)
}

func (e *ResourceError) Unwrap() error {
	return e.Err
}

func resolveFormat(assetPath string, explicit Format) Format {
	if explicit != "" {
		return explicit
	}
	switch strings.TrimPrefix(strings.ToLower(pathpkg.Ext(assetPath)), ".") {
	case "yml":
		return FormatYML
	case "jpeg":
		return FormatJPEG
	default:
		return Format(strings.TrimPrefix(strings.ToLower(pathpkg.Ext(assetPath)), "."))
	}
}

func (s *Saver) saveToFile[T any](ctx context.Context, writer *os.File, path string, value T, opts SaveOptions) error {
	if err := s.save(ctx, writer, value, opts, "SavePath"); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return &ResourceError{
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
		return &ResourceError{
			Message: fmt.Sprintf("failed to encode asset to format %s", opts.Format),
			Err:     err,
			Source:  source,
		}
	}
	return nil
}
