// Package asset loads, decodes, and caches typed assets from an [fs.FS].
// Paths are slash-separated and relative to the filesystem root. The package
// stays backend-free: images decode to [image.Image], leaving backend-specific
// conversion to runners.
package asset

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// ID identifies an asset in the decode cache. It defaults to the cleaned
// asset path; [WithID] overrides it. Cache entries are keyed by ID, decoded
// type, and format, so reusing all three for different files returns the
// first cached value.
type ID string

// Format identifies an asset's encoding and selects the decoder for it.
// It defaults to the lowercased file extension.
type Format string

// Format constants identify common asset encodings. Register a decoder for a
// format with [Server.RegisterDecoder].
const (
	// FormatJSON is the JSON format.
	FormatJSON Format = "json"
	// FormatYAML is the YAML format.
	FormatYAML Format = "yaml"
	// FormatYML is the YML format.
	FormatYML Format = "yml"
	// FormatPNG is the PNG format.
	FormatPNG Format = "png"
	// FormatJPG is the JPG format.
	FormatJPG Format = "jpg"
	// FormatJPEG is the JPEG format.
	FormatJPEG Format = "jpeg"
	// FormatWAV is the WAV format.
	FormatWAV Format = "wav"
	// FormatMP3 is the MP3 format.
	FormatMP3 Format = "mp3"
	// FormatOGG is the OGG format.
	FormatOGG Format = "ogg"
	// FormatTTF is the TTF format.
	FormatTTF Format = "ttf"
	// FormatOTF is the OTF format.
	FormatOTF Format = "otf"
)

// Decoder decodes a value of type T from a reader.
type Decoder[T any] func(reader io.Reader) (T, error)

// Server loads assets from a filesystem, decodes them with registered
// decoders, and caches the results. It also owns the atlas registry.
type Server struct {
	fs        fs.FS
	mu        sync.RWMutex // guards decoders
	decoders  map[codecKey]decoderFunc
	cache     *cache
	loadGroup *singleflight.Group
	atlases   *atlasRegistry
}

// New creates a Server backed by filesystem. If filesystem is nil, assets are
// read from the current working directory. Loading is lazy and begins on the
// first call to [Server.Load].
func New(filesystem fs.FS) *Server {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}
	a := &Server{
		fs:        filesystem,
		decoders:  make(map[codecKey]decoderFunc),
		cache:     newCache(),
		loadGroup: &singleflight.Group{},
		atlases:   newAtlasRegistry(),
	}
	a.registerDefaults()
	return a
}

// Load opens name, decodes it as T, and caches the result. Concurrent loads
// with the same cache key share the in-flight read and decode.
//
// The format defaults to the lowercased file extension; [WithFormat] and
// [WithID] override the format and cache identity, respectively.
func (s *Server) Load[T any](fpath string, opts ...loadOption) (T, error) {
	var zero T
	fpath = normalizeAssetPath(fpath)
	typ := reflect.TypeFor[T]()
	lo := resolveLoad(fpath, opts...)
	key := newLoadKey(lo.id, typ, lo.format)

	if value, ok := s.cache.get(key); ok {
		if value == nil {
			return zero, nil // only a pointer decoder can cache nil
		}
		return value.(T), nil
	}

	value, err, _ := s.loadGroup.Do(key.String(), func() (any, error) {
		if value, ok := s.cache.get(key); ok {
			return value, nil // a peer load may have finished while we waited
		}

		file, err := s.fs.Open(fpath)
		if err != nil {
			return nil, &AssetError{Op: "Load", ID: ID(fpath), Err: err}
		}
		defer file.Close()

		s.mu.RLock()
		decoder, ok := s.decoders[codecKey{typ: typ, format: lo.format}]
		s.mu.RUnlock()
		if !ok {
			return nil, &AssetError{Op: "Load", ID: ID(fpath),
				Err: fmt.Errorf("no decoder registered for type %v and format %q; decoders register at engine or runner construction - loading audio before the runner exists is the common cause", typ, lo.format)}
		}

		value, err := decoder(file)
		if err != nil {
			return nil, &AssetError{Op: "Decode", ID: ID(fpath), Err: err}
		}
		s.cache.put(key, value)
		return value, nil
	})
	if err != nil {
		return zero, err
	}
	if value == nil {
		return zero, nil
	}
	return value.(T), nil // safe: same construction as the cache-hit assertion
}

// Open opens the asset at name as a raw file and returns its inferred format.
// Unlike [Server.Load], it does not decode or cache the asset. The caller
// owns the file and must close it.
//
// Consumers that need seeking for looping or position resets must verify that
// the returned file supports it.
func (s *Server) Open(fpath string) (fs.File, Format, error) {
	if fpath == "" {
		return nil, "", &AssetError{Op: "Open", ID: ID(fpath),
			Err: fmt.Errorf("asset path must not be empty")}
	}
	fpath = normalizeAssetPath(fpath)
	format := resolveFormat(fpath)
	file, err := s.fs.Open(fpath)
	if err != nil {
		return nil, "", &AssetError{Op: "Open", ID: ID(fpath), Err: err}
	}
	return file, format, nil
}

// LoadReader decodes a value of type T from reader using the decoder
// registered for format. The format must be supplied explicitly.
//
// Unlike [Server.Load], LoadReader does not cache and does not
// deduplicate concurrent decodes. It does not close reader; the caller owns it.
func (s *Server) LoadReader[T any](reader io.Reader, format Format) (T, error) {
	var zero T
	typ := reflect.TypeFor[T]()

	s.mu.RLock()
	decoder, ok := s.decoders[codecKey{typ: typ, format: format}]
	s.mu.RUnlock()
	if !ok {
		return zero, fmt.Errorf("no decoder registered for type %v and format %q; decoders register at engine or runner construction - loading audio before the runner exists is the common cause", typ, format)
	}

	value, err := decoder(reader)
	if err != nil {
		return zero, err
	}
	if value == nil {
		return zero, nil // only a pointer or interface decoder can produce nil
	}
	return value.(T), nil
}

// RegisterDecoder registers d for values of type T in format. It returns an
// error if d is nil or format is empty. If a decoder already exists for that
// type and format, override must be true to replace it.
func (s *Server) RegisterDecoder[T any](format Format, d Decoder[T], override bool) error {
	if d == nil {
		return fmt.Errorf("decoder cannot be nil")
	}
	if format == "" {
		return fmt.Errorf("format cannot be empty")
	}
	typ := reflect.TypeFor[T]()
	key := codecKey{typ: typ, format: format}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.decoders[key]; exists && !override {
		return fmt.Errorf("decoder for type %v and format %q already exists", typ, format)
	}
	s.decoders[key] = func(reader io.Reader) (any, error) {
		value, err := d(reader)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
	return nil
}

// WithID overrides the asset's default cache identity, its cleaned path.
func WithID(id ID) loadOption {
	return loadOptionFunc(func(opts *loadOptions) {
		opts.id = id
	})
}

// WithFormat overrides the format inferred from the lowercased file extension.
func WithFormat(format Format) loadOption {
	return loadOptionFunc(func(opts *loadOptions) {
		opts.format = format
	})
}

// AssetError identifies the operation and asset path associated with a load
// or decode failure.
type AssetError struct {
	// Op is the failing operation: "Load", "Open", or "Decode".
	Op string
	// ID is the normalized path of the asset, if known.
	ID ID
	// Err is the underlying cause; Unwrap preserves it for errors.Is.
	Err error
}

// Error returns the operation, asset path, and underlying error.
func (e AssetError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.ID, e.Err)
}

// Unwrap returns the underlying error.
func (e AssetError) Unwrap() error {
	return e.Err
}

// loadKey identifies one decoded representation of an asset. Type and
// format are part of the identity because one ID may legitimately have
// multiple decoded representations.
type loadKey struct {
	id     ID
	typ    reflect.Type
	format Format
}

func newLoadKey(id ID, typ reflect.Type, format Format) loadKey {
	return loadKey{
		id:     id,
		typ:    typ,
		format: format,
	}
}

// String renders a collision-free composite key for the singleflight
// group: length-prefixed parts, so no separator can be smuggled.
func (k loadKey) String() string {
	typName := "<nil>"
	if k.typ != nil {
		typName = k.typ.PkgPath() + "." + k.typ.String()
	}
	return fmt.Sprintf(
		"%d:%s%d:%s%d:%s",
		len(k.id), k.id,
		len(typName), typName,
		len(k.format), k.format,
	)
}

type cacheEntry struct {
	value    any
	hits     atomic.Uint64 // incremented lock-free in get; input for a future eviction policy
	loadedAt time.Time
}

// cache stores decoded values, keyed by asset identity. Thread safety is
// the invariant that earns this type its existence: get is shared-lock
// with a lock-free hit counter; put is exclusive.
type cache struct {
	mu      sync.RWMutex
	entries map[loadKey]*cacheEntry
}

func newCache() *cache {
	return &cache{
		entries: make(map[loadKey]*cacheEntry),
	}
}

func (c *cache) get(key loadKey) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	entry.hits.Add(1)
	return entry.value, true
}

func (c *cache) put(key loadKey, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &cacheEntry{value: value, loadedAt: time.Now()}
}

type codecKey struct {
	typ    reflect.Type
	format Format
}

// decoderFunc is the type-erased form a registered Decoder[T] is stored
// as. Only [Server.RegisterDecoder] builds one, boxing the value as T.
type decoderFunc func(reader io.Reader) (any, error)

type loadOption interface {
	apply(*loadOptions)
}

type loadOptions struct {
	id     ID
	format Format
}

type loadOptionFunc func(*loadOptions)

func (f loadOptionFunc) apply(opts *loadOptions) {
	f(opts)
}

func resolveLoad(name string, opts ...loadOption) *loadOptions {
	lo := &loadOptions{}
	for _, opt := range opts {
		opt.apply(lo)
	}
	if lo.id == "" {
		lo.id = ID(name)
	}
	if lo.format == "" {
		lo.format = resolveFormat(name)
	}
	return lo
}

// normalizeAssetPath returns the canonical slash-separated, cleaned path
// fs.FS expects.
func normalizeAssetPath(name string) string {
	return path.Clean(name)
}

func resolveFormat(name string) Format {
	ext := path.Ext(name)
	if ext == "" {
		return ""
	}
	return Format(strings.ToLower(ext[1:])) // drop the dot, case-insensitive
}
