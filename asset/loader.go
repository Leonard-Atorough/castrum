package asset

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// ID identifies an asset by its slash-separated source path.
type ID string

// Format identifies an asset's encoding and selects its decoder or encoder.
// It defaults to the lowercased file extension.
type Format string

// Format constants identify common asset encodings, not installed codecs.
// Register typed encoders and decoders with [Codecs].
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

// CachePolicy controls decoded-value caching and in-flight load sharing.
type CachePolicy int

const (
	// CacheReuse reuses successful loads and shares in-flight reads.
	CacheReuse CachePolicy = iota
	// CacheBypass reads independently without consuming or populating the cache.
	CacheBypass
)

func (p CachePolicy) validate() error {
	if p != CacheReuse && p != CacheBypass {
		return fmt.Errorf("invalid cache policy %d", p)
	}
	return nil
}

// Loader loads assets from a filesystem, decodes them with registered
// decoders, and caches the results.
type Loader struct {
	fs          fs.FS
	codecs      *Codecs
	cache       *cache
	cachePolicy CachePolicy
}

// LoaderOptions configures a loader at construction.
type LoaderOptions struct {
	// Codecs supplies shared registrations, read on each call. Nil selects the defaults.
	Codecs *Codecs
	// CachePolicy is the default for path loads. Zero selects CacheReuse.
	CachePolicy CachePolicy
}

// NewLoader creates a Loader backed by filesystem. If filesystem is nil, assets are
// read from the current working directory. Loading is lazy and begins on the
// first call to [Loader.Load]. A filesystem containing a nil pointer returns
// an error, as does an invalid cache policy.
func NewLoader(filesystem fs.FS, opts LoaderOptions) (*Loader, error) {
	if err := opts.CachePolicy.validate(); err != nil {
		return nil, fmt.Errorf("asset: NewLoader: %w", err)
	}
	if filesystem == nil {
		filesystem = os.DirFS(".")
	} else if isNilPointer(filesystem) {
		return nil, fmt.Errorf("asset: NewLoader: filesystem cannot be typed nil")
	}
	codecs := opts.Codecs
	if codecs == nil {
		codecs = NewCodecs()
	}
	a := &Loader{
		fs:          filesystem,
		codecs:      codecs,
		cache:       newCache(),
		cachePolicy: opts.CachePolicy,
	}
	return a, nil
}

// Load reads and decodes the asset at fpath as T. Fpath is a slash-separated
// path relative to the loader's filesystem. The format is inferred from its
// extension unless overridden by [WithFormat].
//
// The loader's [CachePolicy] applies unless overridden by [WithCachePolicy].
// Cached values are shared by cleaned path, exact Go type, and format.
//
// Load closes the file it opens and returns the zero value of T on failure.
// A value is cached only after decoding and closing both succeed.
func (l *Loader) Load[T any](fpath string, opts ...loadOption) (T, error) {
	var zero T
	fpath = normalizeAssetPath(fpath)
	lo, err := resolveLoad(fpath, l.cachePolicy, opts...)
	if err != nil {
		return zero, &AssetError{Op: "Load", ID: ID(fpath), Err: err}
	}
	if lo.cachePolicy == CacheBypass {
		return l.read[T](fpath, lo.format)
	}
	key := newLoadKey(ID(fpath), reflect.TypeFor[T](), lo.format)

	if value, ok := l.cache.get(key); ok {
		return value.(codecValue[T]).value, nil
	}
	if owner, ok := l.cache.decodingOwner(key); ok && owner == goID() {
		// A decoder on this goroutine is recursively loading its own key:
		// waiting on the in-flight call would deadlock, because only this
		// load can complete it. Fail fast instead.
		return zero, &AssetError{Op: "Load", ID: ID(fpath), Err: fmt.Errorf("recursive load: the decoder for %q is loading its own asset", fpath)}
	}

	value, err := l.cache.deduplicate(key, func() (any, error) {
		if value, ok := l.cache.get(key); ok {
			return value, nil // a peer load may have finished while we waited
		}

		value, err := l.read[T](fpath, lo.format)
		if err != nil {
			return nil, err
		}
		box := codecValue[T]{value}
		l.cache.put(key, box)
		return box, nil
	})
	if err != nil {
		return zero, err
	}
	return value.(codecValue[T]).value, nil
}

func (l *Loader) read[T any](name string, format Format) (value T, err error) {
	decoder, err := l.codecs.decoder[T](format)
	if err != nil {
		return value, &AssetError{Op: "Load", ID: ID(name), Err: err}
	}
	file, err := l.fs.Open(name)
	if err != nil {
		return value, &AssetError{Op: "Load", ID: ID(name), Err: err}
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			var zero T
			value = zero
			err = errors.Join(err, &AssetError{Op: "Close", ID: ID(name), Err: closeErr})
		}
	}()
	value, err = decoder(file)
	if err != nil {
		var zero T
		return zero, &AssetError{Op: "Decode", ID: ID(name), Err: err}
	}
	return value, nil
}

// Open opens the asset at name as a raw file and returns its inferred format.
// Unlike [Loader.Load], it does not decode or cache the asset. The caller
// owns the file and must close it.
//
// Consumers that need seeking for looping or position resets must verify that
// the returned file supports it.
func (l *Loader) Open(fpath string) (fs.File, Format, error) {
	if fpath == "" {
		return nil, "", &AssetError{Op: "Open", ID: ID(fpath),
			Err: fmt.Errorf("asset path must not be empty")}
	}
	fpath = normalizeAssetPath(fpath)
	format := resolveFormat(fpath)
	file, err := l.fs.Open(fpath)
	if err != nil {
		return nil, "", &AssetError{Op: "Open", ID: ID(fpath), Err: err}
	}
	return file, format, nil
}

// LoadReader decodes a value of type T from reader using the decoder
// registered for format. The format must be supplied explicitly.
//
// Unlike [Loader.Load], LoadReader does not cache and does not
// deduplicate concurrent decodes. It does not close reader; the caller owns it.
func (l *Loader) LoadReader[T any](reader io.Reader, format Format) (T, error) {
	var zero T
	if isNilPointer(reader) {
		return zero, &AssetError{Op: "Decode", Err: fmt.Errorf("reader cannot be nil")}
	}
	decoder, err := l.codecs.decoder[T](format)
	if err != nil {
		return zero, &AssetError{Op: "Decode", Err: err}
	}

	value, err := decoder(reader)
	if err != nil {
		return zero, &AssetError{Op: "Decode", Err: err}
	}
	return value, nil
}

// WithCachePolicy overrides the loader's default cache policy for one Load.
// CacheBypass does not refresh or invalidate an existing cached value.
func WithCachePolicy(policy CachePolicy) loadOption {
	return loadOptionFunc(func(opts *loadOptions) {
		opts.cachePolicy = policy
	})
}

// WithFormat overrides the format inferred from the extension for a path load
// or save.
func WithFormat(format Format) formatOption { return formatOption{format} }

type formatOption struct{ format Format }

func (o formatOption) applyLoad(opts *loadOptions) { opts.format = o.format }
func (o formatOption) applySave(opts *saveOptions) { opts.format = o.format }

// AssetError identifies a failed asset operation and its path, if known.
type AssetError struct {
	// Op is the failing operation: "Load", "Open", "Decode", "Close", "Save", or "Encode".
	Op string
	// ID is the normalized path of the asset, if known.
	ID ID
	// Err is the underlying cause; Unwrap preserves it for errors.Is.
	Err error
}

// Error returns the operation, asset path, and underlying error.
func (e AssetError) Error() string {
	if e.ID == "" {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("%s %s: %v", e.Op, e.ID, e.Err)
}

// Unwrap returns the underlying error.
func (e AssetError) Unwrap() error {
	return e.Err
}

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

// The box preserves static T, including a nil interface, in the cache and flights.
type codecValue[T any] struct {
	value T
}

type cache struct {
	mu      sync.RWMutex
	entries map[loadKey]any
	pending map[loadKey]*loadCall
	// decoding records the goroutine ID of the owner currently decoding each
	// key, so a recursive same-key Load on that goroutine can be detected and
	// rejected instead of deadlocking on the in-flight call.
	decoding map[loadKey]uint64
}

type loadCall struct {
	done   chan struct{}
	result func() (any, error)
}

func newCache() *cache {
	return &cache{
		entries:  make(map[loadKey]any),
		pending:  make(map[loadKey]*loadCall),
		decoding: make(map[loadKey]uint64),
	}
}

// decodingOwner returns the goroutine ID of the owner currently decoding key.
func (c *cache) decodingOwner(key loadKey) (uint64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.decoding[key]
	return id, ok
}

// setDecoding records key's mid-decode owner; owner 0 means "unknown ID"
// (parse failure), which still compares equal to itself on the same goroutine.
func (c *cache) setDecoding(key loadKey, owner uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.decoding[key] = owner
}

// clearDecoding removes key's mid-decode owner.
func (c *cache) clearDecoding(key loadKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.decoding, key)
}

func goID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	fields := strings.Fields(string(buf[:n]))
	if len(fields) < 2 || fields[0] != "goroutine" {
		return 0
	}
	id, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return id
}

func (c *cache) get(key loadKey) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.entries[key]
	return value, ok
}

func (c *cache) put(key loadKey, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = value
}

func (c *cache) deduplicate(key loadKey, load func() (any, error)) (any, error) {
	c.mu.Lock()
	if call, ok := c.pending[key]; ok {
		c.mu.Unlock()
		<-call.done
		return call.result()
	}
	// Only the owner executes load; OnceValues replays its result or panic.
	call := &loadCall{done: make(chan struct{}), result: sync.OnceValues(load)}
	c.pending[key] = call
	c.mu.Unlock()
	// Mark the key with the owner's goroutine before decoding, so a recursive
	// same-key Load on this goroutine is rejected in Load instead of blocking
	// on this call's done channel, which only this owner closes.
	c.setDecoding(key, goID())
	defer func() {
		c.clearDecoding(key)
		c.mu.Lock()
		delete(c.pending, key)
		close(call.done)
		c.mu.Unlock()
	}()
	return call.result()
}

type loadOption interface {
	applyLoad(*loadOptions)
}

type loadOptions struct {
	format      Format
	cachePolicy CachePolicy
}

type loadOptionFunc func(*loadOptions)

func (f loadOptionFunc) applyLoad(opts *loadOptions) {
	f(opts)
}

func resolveLoad(name string, policy CachePolicy, opts ...loadOption) (loadOptions, error) {
	lo := loadOptions{cachePolicy: policy}
	for _, opt := range opts {
		if isNilPointer(opt) {
			return lo, fmt.Errorf("option cannot be nil")
		}
		opt.applyLoad(&lo)
	}
	if err := lo.cachePolicy.validate(); err != nil {
		return lo, err
	}
	if lo.format == "" {
		lo.format = resolveFormat(name)
	}
	return lo, nil
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
