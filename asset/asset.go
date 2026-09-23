// Package asset loads, decodes, and caches game assets from a filesystem.
//
// Assets are identified by fs.FS paths — slash-separated and relative to
// the filesystem root — and by the Go type they decode into. The package
// is backend-free: images decode to image.Image, never a GPU texture;
// backend-specific conversion belongs to a runner.
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

// ID is the cache identity of an asset. It defaults to the cleaned asset
// path; [WithID] overrides it. The decode cache keys on ID, type, and
// format together, so one ID may hold several types or formats — but two
// different files loaded under one ID, type, and format collide: the
// first load wins.
type ID string

// Format identifies an asset's encoding and selects the decoder for it.
// It defaults to the lowercased file extension.
type Format string

// Asset formats with built-in constants. Decoders for them are not
// registered by default; the engine or runner registers the set it
// supports. New formats are introduced by [Asset.RegisterDecoder].
const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
	FormatYML  Format = "yml"
	FormatPNG  Format = "png"
	FormatJPG  Format = "jpg"
	FormatJPEG Format = "jpeg"
	FormatWAV  Format = "wav"
	FormatMP3  Format = "mp3"
	FormatOGG  Format = "ogg"
)

// Decoder decodes a value of type T from a reader.
type Decoder[T any] func(reader io.Reader) (T, error)

// Asset loads and caches decoded assets from a filesystem. It is the
// single owner of the load flow — cache check, deduplication, open,
// decode, cache put — and the pieces of that flow are not callable
// separately.
type Asset struct {
	fs        fs.FS
	mu        sync.RWMutex // guards decoders
	decoders  map[codecKey]decoderFunc
	cache     *cache
	loadGroup *singleflight.Group
}

// New creates an Asset over filesystem. A nil filesystem defaults to
// os.DirFS("."), the game's working directory: asset names resolve
// relative to it, wherever the files live. Nothing is read at
// construction; loading is lazy and starts at the first [Asset.Load].
func New(filesystem fs.FS) *Asset {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}
	return &Asset{
		fs:        filesystem,
		decoders:  make(map[codecKey]decoderFunc),
		cache:     newCache(),
		loadGroup: &singleflight.Group{},
	}
}

// Load reads and decodes the asset at name as T, serving repeat loads
// from the cache. Concurrent loads of the same asset are deduplicated:
// the read and decode happen at most once while the first call is in
// flight.
//
// The format defaults to the lowercased file extension; [WithFormat] and
// [WithID] override it and the cache identity.
func (a *Asset) Load[T any](fpath string, opts ...loadOption) (T, error) {
	var zero T
	fpath = normalizeAssetPath(fpath)
	typ := reflect.TypeFor[T]()
	lo := resolveLoad(fpath, opts...)
	key := newLoadKey(lo.id, typ, lo.format)

	if value, ok := a.cache.get(key); ok {
		if value == nil {
			return zero, nil // only a pointer decoder can cache nil
		}
		return value.(T), nil
	}

	value, err, _ := a.loadGroup.Do(key.String(), func() (any, error) {
		if value, ok := a.cache.get(key); ok {
			return value, nil // a peer load may have finished while we waited
		}

		file, err := a.fs.Open(fpath)
		if err != nil {
			return nil, &AssetError{Op: "Load", ID: ID(fpath), Err: err}
		}
		defer file.Close()

		a.mu.RLock()
		decoder, ok := a.decoders[codecKey{typ: typ, format: lo.format}]
		a.mu.RUnlock()
		if !ok {
			return nil, &AssetError{Op: "Load", ID: ID(fpath),
				Err: fmt.Errorf("no decoder registered for type %v and format %q", typ, lo.format)}
		}

		value, err := decoder(file)
		if err != nil {
			return nil, &AssetError{Op: "Decode", ID: ID(fpath), Err: err}
		}
		a.cache.put(key, value)
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

// LoadReader decodes a value of type T from the given reader using the
// registered decoder for format. A reader has no extension to infer the
// format from, so it must be given explicitly.
//
// Unlike [Asset.Load], LoadReader does not cache and does not
// deduplicate concurrent decodes. The reader is not closed; the caller
// owns it.
func (a *Asset) LoadReader[T any](reader io.Reader, format Format) (T, error) {
	var zero T
	typ := reflect.TypeFor[T]()

	a.mu.RLock()
	decoder, ok := a.decoders[codecKey{typ: typ, format: format}]
	a.mu.RUnlock()
	if !ok {
		return zero, fmt.Errorf("no decoder registered for type %v and format %q", typ, format)
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

// RegisterDecoder registers d as the decoder for values of type T in the
// given format. override replaces an existing decoder for the same type
// and format; without it, a duplicate is an error.
//
// This is the only way into the codec registry: the wrapper boxes the
// decoded value as T, so the type assertions in [Asset.Load] and
// [Asset.LoadReader] are safe by construction. A panic from either means
// a registration bypassed this wrapper — an engine-internal invariant
// breach, not user error.
func (a *Asset) RegisterDecoder[T any](format Format, d Decoder[T], override bool) error {
	if d == nil {
		return fmt.Errorf("decoder cannot be nil")
	}
	if format == "" {
		return fmt.Errorf("format cannot be empty")
	}
	typ := reflect.TypeFor[T]()
	key := codecKey{typ: typ, format: format}

	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.decoders[key]; exists && !override {
		return fmt.Errorf("decoder for type %v and format %q already exists", typ, format)
	}
	a.decoders[key] = func(reader io.Reader) (any, error) {
		value, err := d(reader)
		if err != nil {
			return nil, err
		}
		return value, nil
	}
	return nil
}

// WithID overrides the cache identity of an asset. The default is the
// cleaned asset path. See [ID] for the collision rules.
func WithID(id ID) loadOption {
	return loadOptionFunc(func(opts *loadOptions) {
		opts.id = id
	})
}

// WithFormat overrides the asset's format. The default is inferred from
// the lowercased file extension.
func WithFormat(format Format) loadOption {
	return loadOptionFunc(func(opts *loadOptions) {
		opts.format = format
	})
}

// AssetError wraps a failure that occurred while loading or decoding an
// asset, naming the operation and the asset involved.
type AssetError struct {
	// Op is the failing operation: "Load" or "Decode".
	Op string
	// ID is the asset's cache identity, if one applies.
	ID ID
	// Err is the underlying cause; Unwrap preserves it for errors.Is.
	Err error
}

func (e AssetError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.Op, e.ID, e.Err)
}

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
// as. Only [Asset.RegisterDecoder] builds one, boxing the value as T.
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
