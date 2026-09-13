package assets

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	pathpkg "path"
	"reflect"
	"strings"

	"github.com/leonard-atorough/castrum/ecs"
	internalassets "github.com/leonard-atorough/castrum/internal/assets"
	"go.yaml.in/yaml/v3"
)

// LoadResult holds the result of an async load operation.
type LoadResult struct {
	Value any
	Err   error
}

type TextureData struct {
	Image  image.Image
	Width  int
	Height int
}

// Blueprint represents a reusable template for creating entities with predefined components.
type Blueprint struct {
	Name       string          `yaml:"name"`
	Components []ComponentData `yaml:"components"`
	Version    string          `yaml:"version"`
}

// ComponentData represents the data required to instantiate a component for an entity.
type ComponentData struct {
	Type       string         `yaml:"type"`
	Properties map[string]any `yaml:"properties"`
}

// // loadRequest wraps an asset path with its result channel.
// type loadRequest struct {
// 	path     string
// 	resultCh chan<- LoadResult
// }

// const (
// 	defaultWorkerCount  = 2
// 	defaultJobQueueSize = 64
// )

type Assets struct {
	loader *Loader
	saver  *Saver
}

func NewAssets(filesystem fs.FS) *Assets {
	service := newAssetService(filesystem)
	return &Assets{
		loader: newLoader(service),
		saver:  newSaver(service),
	}
}

func (a *Assets) AssetLoader() *Loader {
	return a.loader
}

func (a *Assets) AssetSaver() *Saver {
	return a.saver
}

func registerDefaultDecoders(s *internalassets.Service) {
	registerDefaultDecoder(s, FormatYAML, decodeBlueprintYAML)
	registerDefaultDecoder(s, FormatYML, decodeBlueprintYAML)
	registerDefaultDecoder(s, FormatPNG, decodeTexture)
	registerDefaultDecoder(s, FormatJPG, decodeTexture)
	registerDefaultDecoder(s, FormatJPEG, decodeTexture)
}

func registerDefaultEncoders(s *internalassets.Service) {
	registerDefaultEncoder(s, FormatYAML, encodeBlueprintYAML)
	registerDefaultEncoder(s, FormatYML, encodeBlueprintYAML)
	registerDefaultEncoder(s, FormatPNG, encodeTexturePNG)
	registerDefaultEncoder(s, FormatJPG, encodeTextureJPEG)
	registerDefaultEncoder(s, FormatJPEG, encodeTextureJPEG)
}

func newAssetService(filesystem fs.FS) *internalassets.Service {
	service := internalassets.NewService(filesystem)
	registerDefaultDecoders(service)
	registerDefaultEncoders(service)
	return service
}

func registerDefaultDecoder[T any](service *internalassets.Service, format Format, decoder Decoder[T]) {
	// Built-in registrations are fixed and validated; a failure is an internal
	// wiring error, while the public constructor intentionally has no error.
	_ = service.RegisterDecoder(
		reflect.TypeFor[T](),
		string(format),
		func(ctx context.Context, reader io.Reader) (any, error) {
			return decoder(ctx, reader)
		},
		false,
	)
}

func registerDefaultEncoder[T any](service *internalassets.Service, format Format, encoder Encoder[T]) {
	_ = service.RegisterEncoder(
		reflect.TypeFor[T](),
		string(format),
		func(ctx context.Context, writer io.Writer, value any) error {
			return encoder(ctx, writer, value.(T))
		},
		false,
	)
}

func decodeBlueprintYAML(_ context.Context, reader io.Reader) (Blueprint, error) {
	var blueprint Blueprint
	if err := yaml.NewDecoder(reader).Decode(&blueprint); err != nil {
		return Blueprint{}, err
	}
	return blueprint, nil
}

func encodeBlueprintYAML(_ context.Context, writer io.Writer, blueprint Blueprint) error {
	return yaml.NewEncoder(writer).Encode(blueprint)
}

func decodeTexture(_ context.Context, reader io.Reader) (TextureData, error) {
	decoded, _, err := image.Decode(reader)
	if err != nil {
		return TextureData{}, err
	}
	bounds := decoded.Bounds()
	return TextureData{Image: decoded, Width: bounds.Dx(), Height: bounds.Dy()}, nil
}

func encodeTexturePNG(_ context.Context, writer io.Writer, texture TextureData) error {
	if texture.Image == nil {
		return fmt.Errorf("texture image is nil")
	}
	return png.Encode(writer, texture.Image)
}

func encodeTextureJPEG(_ context.Context, writer io.Writer, texture TextureData) error {
	if texture.Image == nil {
		return fmt.Errorf("texture image is nil")
	}
	return jpeg.Encode(writer, texture.Image, &jpeg.Options{Quality: 90})
}

// // LoadAsync loads an asset asynchronously via the worker pool.
// // Returns immediately with a channel that will receive the result.
// // The channel is closed after the result is sent.
// // Respects context cancellation; if the context is cancelled before the load
// // completes, the result will contain the context error.
// //
// // Example:
// //
// //	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// //	defer cancel()
// //	result := <-assets.LoadAsync(ctx, "sprites/player.png")
// //	if result.Err != nil {
// //		log.Fatal(result.Err)
// //	}
// func (a *AssetLoader) LoadAsync(ctx context.Context, path string) <-chan LoadResult {
// 	resultCh := make(chan LoadResult, 1)

// 	// Check if context is already cancelled to avoid race in select.
// 	// This ensures cancellation errors are handled immediately.
// 	select {
// 	case <-ctx.Done():
// 		resultCh <- LoadResult{Err: ctx.Err()}
// 		close(resultCh)
// 		return resultCh
// 	default:
// 	}

// 	go func() {
// 		select {
// 		case a.jobs <- loadRequest{path: path, resultCh: resultCh}:
// 			// Job submitted to queue
// 		case <-ctx.Done():
// 			resultCh <- LoadResult{Err: ctx.Err()}
// 			close(resultCh)
// 		}
// 	}()

// 	return resultCh
// }

// // LoadBatch loads multiple assets concurrently with a single context.
// // Blocks until all assets are loaded or the context is cancelled.
// // Returns a slice of LoadResult corresponding to the requested paths.
// // Errors for individual loads are contained within each LoadResult.
// //
// // Example:
// //
// //	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// //	defer cancel()
// //	results, err := assets.LoadBatch(ctx, []string{
// //		"sprites/player.png",
// //		"sprites/enemy.png",
// //		"data/level.yaml",
// //	})
// //	if err != nil {
// //		log.Fatal(err)
// //	}
// func (a *AssetLoader) LoadBatch(ctx context.Context, paths []string) ([]LoadResult, error) {
// 	results := make([]LoadResult, len(paths))
// 	channels := make([]<-chan LoadResult, len(paths))

// 	// Submit all jobs
// 	for i, path := range paths {
// 		ch := make(chan LoadResult, 1)
// 		channels[i] = ch
// 		select {
// 		case a.jobs <- loadRequest{path: path, resultCh: ch}:
// 		case <-ctx.Done():
// 			return nil, ctx.Err()
// 		}
// 	}

// 	// Collect results
// 	for i, ch := range channels {
// 		select {
// 		case result := <-ch:
// 			results[i] = result
// 		case <-ctx.Done():
// 			return nil, ctx.Err()
// 		}
// 	}

// 	return results, nil
// }

// // worker is run by each worker goroutine to process load jobs from the queue.
// func (a *AssetLoader) worker() {
// 	for job := range a.jobs {
// 		res, err := a.LoadSync(job.path)
// 		job.resultCh <- LoadResult{Value: res, Err: err}
// 		close(job.resultCh)
// 	}
// }

// CreateFromBlueprint constructs and creates an entity from blueprint data.
func CreateFromBlueprint(world *ecs.World, registry *ComponentRegistry, bp *Blueprint) (*ecs.Entity, error) {
	components := make([]ecs.Component, len(bp.Components))
	for i, comp := range bp.Components {
		instance, err := registry.Resolve(comp.Type, comp.Properties)
		if err != nil {
			return nil, err
		}
		components[i] = instance
	}

	return world.CreateWithComponents(bp.Name, components...)
}

// AssetError represents an error that occurred during the loading or saving of an asset.
type AssetError struct {
	Message string
	Err     error
	Source  string
}

func (e *AssetError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v (source: %s)", e.Message, e.Err, e.Source)
	}
	return fmt.Sprintf("%s (source: %s)", e.Message, e.Source)
}

func (e *AssetError) Unwrap() error {
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
