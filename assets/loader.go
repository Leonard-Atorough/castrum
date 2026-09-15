package assets

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync"

	internalassets "github.com/leonard-atorough/castrum/internal/assets"
)

// ID represents the unique identifier for an asset.
// It is used for caching and retrieval purposes within the loader.
type ID string

// Format represents the format of an asset.
// It is used to determine the appropriate decoder for the asset.
type Format string

// Format constants identify the encoding of an asset. They determine which
// decoder or encoder is selected. New formats can be introduced by passing a
// Format value to [Loader.RegisterDecoder] or [Saver.RegisterEncoder].
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

// WithID specifies a unique identifier to be used when loading an asset.
// Be careful reusing the same ID for different assets, as it may lead to cache collisions and canonicalization issues.
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

// CachePolicy controls whether a loaded asset is cached by the [Loader].
const (
	// CachePolicyNone disables caching; the asset is decoded on every Load.
	CachePolicyNone CachePolicy = iota
	// CachePolicyDefault caches the asset by ID and type after the first
	// decode. This is the default when no cache policy is specified.
	CachePolicyDefault
)

// Decoder is a function type that defines how to decode an asset from an io.Reader.
type Decoder[T any] func(context.Context, io.Reader) (T, error)

// Loader is responsible for loading assets from the filesystem.
type Loader struct {
	service    *internalassets.Service
	listenerMu sync.RWMutex
	listeners  []func(ID)
}

func newLoader(service *internalassets.Service) *Loader {
	return &Loader{service: service}
}

// Load loads an asset from the specified path using the provided options.
// It first checks the cache based on the ID and cache policy.
// If the asset is not cached, it reads the asset from the filesystem and decodes it using the appropriate decoder.
// Concurrent calls for the same asset are deduplicated so the decode and
// filesystem read happen at most once while the first call is in flight.
func (l *Loader) Load[T any](ctx context.Context, path string, options ...LoadOption) (T, error) {
	var zero T
	typ := reflect.TypeFor[T]()
	path = normalizeAssetPath(path)

	opts := resolveLoadOptions(path, options...)

	if opts.CachePolicy != CachePolicyNone {
		if cached, ok := l.service.Cached(string(opts.ID), typ, string(opts.Format)); ok {
			result, ok := cached.(T)
			if ok {
				return result, nil
			}
			return result, &AssetError{Message: fmt.Sprintf("cached asset has unexpected type for %s", opts.ID), Source: "Load"}
		}
	}

	result, err := l.service.LoadDedup(string(opts.ID), typ, string(opts.Format), func() (any, error) {
		if opts.CachePolicy != CachePolicyNone {
			if cached, ok := l.service.Cached(string(opts.ID), typ, string(opts.Format)); ok {
				result, ok := cached.(T)
				if ok {
					return result, nil
				}
				return result, &AssetError{Message: fmt.Sprintf("cached asset has unexpected type for %s", opts.ID), Source: "Load"}
			}
		}

		reader, err := l.service.Filesystem().Open(path)
		if err != nil {
			return nil, &AssetError{
				Message: fmt.Sprintf("failed to open file %s", path),
				Err:     err,
				Source:  "Load",
			}
		}
		defer reader.Close()

		val, err := l.service.Decode(ctx, typ, string(opts.Format), reader)
		if err != nil {
			return nil, &AssetError{
				Message: fmt.Sprintf("failed to decode asset from file %s", path),
				Err:     err,
				Source:  "Load",
			}
		}
		if opts.CachePolicy != CachePolicyNone {
			l.service.Cache(string(opts.ID), typ, string(opts.Format), val)
		}
		return val, nil
	})
	if err != nil {
		return zero, err
	}
	if result == nil {
		return zero, nil
	}
	return result.(T), nil
}

// LoadReader loads an asset from the provided reader using the specified options.
// Since the reader has no stable identity, caching by ID is not applicable.
// The asset is decoded directly from the reader without caching.
func (l *Loader) LoadReader[T any](ctx context.Context, reader io.Reader, options ...LoadOption) (T, error) {
	var result T
	typ := reflect.TypeFor[T]()
	opts := resolveLoadOptions("", options...)

	res, err := l.service.Decode(ctx, typ, string(opts.Format), reader)
	if err != nil {
		return result, &AssetError{
			Message: "failed to decode asset from reader",
			Err:     err,
			Source:  "LoadReader",
		}
	}
	result, ok := res.(T)
	if !ok {
		return result, &AssetError{Message: "decoder returned an unexpected type", Source: "LoadReader"}
	}
	return result, nil
}

// RegisterDecoder registers a decoder for the specified format.
// The decoder will be used to decode assets of the given type from readers.
// The override parameter determines whether an existing decoder for the same type and format should be replaced.
func (l *Loader) RegisterDecoder[T any](format Format, decoder Decoder[T], override bool) error {
	if decoder == nil {
		return &AssetError{
			Message: fmt.Sprintf("decoder for format %s must not be nil", format),
			Source:  "RegisterDecoder",
		}
	}
	typ := reflect.TypeFor[T]()
	err := l.service.RegisterDecoder(typ, string(format), func(ctx context.Context, r io.Reader) (any, error) {
		return decoder(ctx, r)
	}, override)
	if err != nil {
		return &AssetError{
			Message: fmt.Sprintf("failed to register decoder for format %s", format),
			Err:     err,
			Source:  "RegisterDecoder",
		}
	}
	return nil
}

// Invalidate removes the asset with the given ID from the cache and notifies
// all registered invalidation listeners. A zero ID is a no-op for individual
// invalidation but is used by [Loader.ClearCache] to signal a full flush.
func (l *Loader) Invalidate(id ID) {
	l.service.Invalidate(string(id))

	l.listenerMu.RLock()
	listeners := append([]func(ID){}, l.listeners...)
	l.listenerMu.RUnlock()
	for _, listener := range listeners {
		listener(id)
	}
}

// RegisterInvalidationListener registers a callback notified after an asset
// ID is removed from the decoded asset cache. The callback should be quick;
// listeners are invoked synchronously by Invalidate.
func (l *Loader) RegisterInvalidationListener(listener func(ID)) {
	if listener == nil {
		return
	}
	l.listenerMu.Lock()
	l.listeners = append(l.listeners, listener)
	l.listenerMu.Unlock()
}

// ClearCache clears the entire asset cache and notifies all registered invalidation listeners.
func (l *Loader) ClearCache() {
	l.service.ClearCache()
	l.listenerMu.RLock()
	listeners := append([]func(ID){}, l.listeners...)
	l.listenerMu.RUnlock()
	for _, listener := range listeners {
		// Notify listeners that all assets have been invalidated.
		// Using an empty ID to indicate a full cache clear.
		listener("")
	}
}
