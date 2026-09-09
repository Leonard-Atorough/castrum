package assets

import (
	"context"
	"image"
	"io/fs"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type fileExtension string

var (
	FileExtensionTexture   []fileExtension = []fileExtension{".png", ".jpg", ".jpeg"}
	FileExtensionBlueprint fileExtension   = ".yaml"
)

// LoadResult holds the result of an async load operation.
type LoadResult struct {
	Value any
	Err   error
}

// loadRequest wraps an asset path with its result channel.
type loadRequest struct {
	path     string
	resultCh chan<- LoadResult
}

// Assets manages loading and caching of game resources (textures, blueprints, and atlases).
// Supports both synchronous and asynchronous loading with concurrency control via an
// internal worker pool. LoadSync is blocking and ideal for initialization; LoadAsync and
// LoadBatch use workers for concurrent loading with context cancellation support.
//
// Animation clips are not loaded from disk; they are created programmatically via
// Animation clips and atlases are not loaded from disk; they are created
// programmatically via the animation.Manager and atlas.Manager.
type Assets struct {
	Textures   *textureStore
	Blueprints *blueprintStore
	jobs       chan loadRequest
}

const defaultWorkerCount = 4

func NewAssets(filesystem fs.FS) *Assets {
	if filesystem == nil {
		filesystem = os.DirFS(".")
	}

	a := &Assets{
		Textures:   newTextureStore(filesystem),
		Blueprints: newBlueprintStore(filesystem),
		jobs:       make(chan loadRequest, 100),
	}

	// Start worker goroutines for async loading
	for range defaultWorkerCount {
		go a.worker()
	}

	return a
}

// LoadSync is the gateway for synchronous asset loading (blocking).
// It infers the asset kind from the path's extension, routes to the matching store,
// and returns the cached result. Results are cached after first load, so repeated
// calls for the same path are fast.
//
// Use LoadSync during initialization, scene setup, or when you need immediate results.
// For loading during gameplay, prefer LoadAsync or LoadBatch.
//
// For known types, the specific stores can be used directly:
//   - Assets.Textures.Load(path) for image assets
//   - Assets.Blueprints.Load(path) for entity blueprints
//
// Texture atlases are created programmatically via atlas.Manager.
// Animation clips are created programmatically via animation.Manager.
func (a *Assets) LoadSync(path string) (res any, err error) {
	switch {
	case hasTextureExtension(path):
		res, err = a.Textures.Load(path)
	case hasBlueprintExtension(path):
		res, err = a.Blueprints.Load(path)
	default:
		return nil, nil
	}
	return res, err
}

// LoadAsync loads an asset asynchronously via the worker pool.
// Returns immediately with a channel that will receive the result.
// The channel is closed after the result is sent.
// Respects context cancellation; if the context is cancelled before the load
// completes, the result will contain the context error.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	result := <-assets.LoadAsync(ctx, "sprites/player.png")
//	if result.Err != nil {
//		log.Fatal(result.Err)
//	}
func (a *Assets) LoadAsync(ctx context.Context, path string) <-chan LoadResult {
	resultCh := make(chan LoadResult, 1)

	// Check if context is already cancelled to avoid race in select.
	// This ensures cancellation errors are handled immediately.
	select {
	case <-ctx.Done():
		resultCh <- LoadResult{Err: ctx.Err()}
		close(resultCh)
		return resultCh
	default:
	}

	go func() {
		select {
		case a.jobs <- loadRequest{path: path, resultCh: resultCh}:
			// Job submitted to queue
		case <-ctx.Done():
			resultCh <- LoadResult{Err: ctx.Err()}
			close(resultCh)
		}
	}()

	return resultCh
}

// LoadBatch loads multiple assets concurrently with a single context.
// Blocks until all assets are loaded or the context is cancelled.
// If any load fails or the context is cancelled, the error is returned immediately
// but in-flight loads may continue.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	results, err := assets.LoadBatch(ctx, []string{
//		"sprites/player.png",
//		"sprites/enemy.png",
//		"data/level.yaml",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
func (a *Assets) LoadBatch(ctx context.Context, paths []string) ([]LoadResult, error) {
	results := make([]LoadResult, len(paths))
	channels := make([]<-chan LoadResult, len(paths))

	// Submit all jobs
	for i, path := range paths {
		ch := make(chan LoadResult, 1)
		channels[i] = ch
		select {
		case a.jobs <- loadRequest{path: path, resultCh: ch}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Collect results
	for i, ch := range channels {
		select {
		case result := <-ch:
			results[i] = result
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return results, nil
}

// worker is run by each worker goroutine to process load jobs from the queue.
func (a *Assets) worker() {
	for job := range a.jobs {
		res, err := a.LoadSync(job.path)
		job.resultCh <- LoadResult{Value: res, Err: err}
		close(job.resultCh)
	}
}

func hasTextureExtension(path string) bool {
	for _, ext := range FileExtensionTexture {
		if len(path) >= len(ext) && path[len(path)-len(ext):] == string(ext) {
			return true
		}
	}
	return false
}

func hasBlueprintExtension(path string) bool {
	ext := FileExtensionBlueprint
	return len(path) >= len(ext) && path[len(path)-len(ext):] == string(ext)
}

func loadImageFromFS(fs fs.FS, path string) (*ebiten.Image, image.Image, error) {
	file, err := fs.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	img, generic, err := ebitenutil.NewImageFromFileSystem(fs, path)
	if err != nil {
		return nil, nil, err
	}
	return img, generic, nil
}
