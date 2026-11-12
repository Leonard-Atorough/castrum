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

	"golang.org/x/sync/singleflight"
)

// ID identifies an asset in the decode cache. It defaults to the cleaned
// asset path; [WithID] overrides it. Cache entries are keyed by ID, decoded
// type, and format, so reusing all three for different files returns the
// first cached value.
type ID string

// Format identifies an asset's encoding and selects its decoder.
// It defaults to the lowercased file extension.
type Format string

// Format constants identify common asset encodings, not installed codecs.
// Configure typed decoders with [Codecs].
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

// Loader loads assets from a filesystem, decodes them with registered
// decoders, and caches the results. It also owns the atlas registry.
type Loader struct {
	fs        fs.FS
	codecs    *Codecs
	cache     *cache
	loadGroup *singleflight.Group
	atlases   *atlasRegistry
}

// LoaderOptions configures a loader at construction.
type LoaderOptions struct {
	// Codecs supplies registrations to snapshot. Nil selects the defaults.
	Codecs *Codecs
}

// NewLoader creates a Loader backed by filesystem. If filesystem is nil, assets are
// read from the current working directory. Loading is lazy and begins on the
// first call to [Loader.Load]. A filesystem containing a nil pointer returns
// an error. Codec registrations are private to the returned loader.
func NewLoader(filesystem fs.FS, opts LoaderOptions) (*Loader, error) {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	} else if isNilPointer(filesystem) {
		return nil, fmt.Errorf("asset: NewLoader: filesystem cannot be typed nil")
	}
	a := &Loader{
		fs:        filesystem,
		codecs:    opts.Codecs.snapshot(),
		cache:     newCache(),
		loadGroup: &singleflight.Group{},
		atlases:   newAtlasRegistry(),
	}
	return a, nil
}

// Load opens name, decodes it as T, and caches the result. Concurrent loads
// with the same cache key share the in-flight read and decode.
//
// The format defaults to the lowercased file extension; [WithFormat] and
// [WithID] override the format and cache identity, respectively.
func (l *Loader) Load[T any](fpath string, opts ...loadOption) (T, error) {
	var zero T
	fpath = normalizeAssetPath(fpath)
	typ := reflect.TypeFor[T]()
	lo := resolveLoad(fpath, opts...)
	key := newLoadKey(lo.id, typ, lo.format)

	if value, ok := l.cache.get(key); ok {
		return value.(codecValue[T]).value, nil
	}

	value, err, _ := l.loadGroup.Do(key.String(), func() (any, error) {
		if value, ok := l.cache.get(key); ok {
			return value, nil // a peer load may have finished while we waited
		}

		decoder, err := l.codecs.decoder[T](lo.format)
		if err != nil {
			return nil, &AssetError{Op: "Load", ID: ID(fpath), Err: err}
		}

		file, err := l.fs.Open(fpath)
		if err != nil {
			return nil, &AssetError{Op: "Load", ID: ID(fpath), Err: err}
		}
		defer file.Close()

		value, err := decoder(file)
		if err != nil {
			return nil, &AssetError{Op: "Decode", ID: ID(fpath), Err: err}
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

// RegisterDecoder registers d for values of type T in format. It returns an
// error if d is nil, format is empty, or that pair is already registered.
// Registrations affect only this loader. Configure replacements with
// [Codecs.SetDecoder] before constructing it.
func (l *Loader) RegisterDecoder[T any](format Format, d Decoder[T]) error {
	c := l.codecs
	c.mu.Lock()
	defer c.mu.Unlock()
	key := codecKey{reflect.TypeFor[T](), format}
	if _, exists := c.decoders[key]; exists {
		return fmt.Errorf("decoder for type %v and format %q already exists", key.typ, format)
	}
	return c.setDecoder(key, d)
}

// WithID overrides the asset's default cache identity, its cleaned path.
func WithID(id ID) loadOption {
	return loadOptionFunc(func(opts *loadOptions) {
		opts.id = id
	})
}

// WithFormat overrides the format inferred from the extension for a path load.
func WithFormat(format Format) formatOption { return formatOption{format} }

type formatOption struct{ format Format }

func (o formatOption) applyLoad(opts *loadOptions) { opts.format = o.format }

// AssetError identifies a failed asset operation and its path, if known.
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
	if e.ID == "" {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
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

// The box preserves static T, including a nil interface, in the cache and flights.
type codecValue[T any] struct {
	value T
}

type cache struct {
	mu      sync.RWMutex
	entries map[loadKey]any
}

func newCache() *cache {
	return &cache{
		entries: make(map[loadKey]any),
	}
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

type loadOption interface {
	applyLoad(*loadOptions)
}

type loadOptions struct {
	id     ID
	format Format
}

type loadOptionFunc func(*loadOptions)

func (f loadOptionFunc) applyLoad(opts *loadOptions) {
	f(opts)
}

func resolveLoad(name string, opts ...loadOption) *loadOptions {
	lo := &loadOptions{}
	for _, opt := range opts {
		opt.applyLoad(lo)
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
