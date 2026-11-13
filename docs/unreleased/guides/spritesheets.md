# Spritesheets

A spritesheet gives names to pixel rectangles in one texture. Draw a region by name, or put several names in order to play an animation. You can load the sheet from JSON or build it in Go; both routes use the same document and references.

The sheet describes geometry, not playback. These types connect it to your game:

- `asset.Sheet` owns the JSON-friendly document, validation, loading, and saving.
- `spritesheet.Grid` builds a sheet from known texture bounds without loading, saving, or modifying pixels.
- `asset.SheetRef` identifies a JSON file or retains a sheet in memory.
- `render.SheetSource` selects one region to draw.
- `animation.Clip` orders regions, and an `animation.Animation` component owns one entity's playback.

No sheet registry is required. [Assets](assets.md) explains filesystem roots, codecs, and writable storage; [Rendering](rendering.md) explains cameras and sprite style.

## Quick start: draw a region

Put an image at `assets/sprites/characters.png`, at least 64 by 32 pixels, and save this document as `assets/sprites/characters.sheet.json`:

```json
{
  "texture": "assets/sprites/characters.png",
  "regions": {
    "hero_idle_0": { "x": 0, "y": 0, "w": 32, "h": 32 },
    "hero_idle_1": { "x": 32, "y": 0, "w": 32, "h": 32 }
  }
}
```

Both paths are slash-separated names relative to the game's filesystem root, including on Windows. The texture path is not relative to the JSON file. The default game filesystem reads from the working directory; use [WithFilesystem](assets.md#choose-the-asset-root) to supply another root or embedded content.

In your game setup, create a reference and attach a region to a sprite:

```go
import (
	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/render"
)

g, err := castrum.New()
if err != nil {
	return err
}
source := asset.FileSheet("assets/sprites/characters.sheet.json")
_, err = g.World().NewEntity(
	core.Transform{Position: geom.Vector2{X: 100, Y: 100}},
	render.Sprite{Drawable: render.SheetSource{
		Sheet: source, Region: "hero_idle_0",
	}},
)
if err != nil {
	return err
}
```

These setup snippets assume a function that returns an error and use `g` and `source` from this example. The game provides the loader and the previous-transform state required by rendering. Run it with your platform runner as described in [Your first game](../getting-started/your-first-game.md).

`FileSheet` only creates a reference; it performs no I/O. The collector resolves the document through the world's loader, loads its texture, and checks the selected region before drawing. Missing files and mismatched image bounds can therefore fail on the first collected frame. [Validate during setup](#choose-when-validation-happens) when these failures should prevent launch.

## Choose when validation happens

There are three separate checks:

| Check | What it verifies | When it runs |
| --- | --- | --- |
| Reference validation | A valid file path, or a memory sheet with a valid texture path and non-empty region map | Sprite and clip validation, and `SheetRef.Resolve` |
| Metadata validation | The texture path and every region's name and geometry | `NewSheet`, default JSON decoding, `Sheet.Validate`, and `Sheet.Save` |
| Image-bound validation | Region rectangles fit the actual texture | Selected-region drawing, or an explicit `Sheet.ValidateBounds` call |

Metadata validation rejects empty or escaping texture paths, empty region maps, empty or invalid UTF-8 names, negative coordinates, non-positive sizes, and overflowing rectangles. It does not load the image. The default decoder returns a decode `AssetError` for invalid metadata, caches no value, and retries on the next load. Duplicate JSON map keys follow standard last-key-wins behavior.

Drawing checks only the selected region against the decoded image; it neither scans every region each frame nor silently clips rectangles. To check the whole sheet before launch, resolve the reference and preload its texture during setup:

```go
loader := g.World().MustResource[*asset.Loader]()
document, err := source.Resolve(loader)
if err != nil {
	return err
}
texture, err := loader.Load[asset.TextureData](string(document.Texture))
if err != nil {
	return err
}
if err := document.ValidateBounds(texture.Image.Bounds()); err != nil {
	return err
}
```

This uses the default image decoder. Custom texture decoders must also supply a non-nil `Image`.

Start custom codec configurations with `asset.NewCodecs` to retain the validating sheet decoder. Calling `SetJSON[asset.Sheet]()` explicitly replaces it with ordinary JSON decoding, making metadata validation your responsibility. Memory references also check only the sheet header when resolved: call `Sheet.Validate` or `ValidateBounds` before sharing a manually constructed sheet.

## Author regions in Go

### Slice a regular grid

Import `github.com/Leonard-Atorough/castrum/spritesheet`. Load the texture to obtain its actual bounds, then build a sheet:

```go
loader := g.World().MustResource[*asset.Loader]()
texture, err := loader.Load[asset.TextureData]("assets/sprites/characters.png")
if err != nil {
	return err
}
sheet, err := spritesheet.Grid(
	"assets/sprites/characters.png", texture.Image.Bounds(), 32, 32, "hero_idle",
)
if err != nil {
	return err
}
source = asset.MemorySheet(sheet)
```

The bounds must be non-empty and non-negative. Tile dimensions must be positive and evenly divide the supplied bounds, and the prefix must be non-empty. Names follow row-major order: `hero_idle_0`, `hero_idle_1`, and so on, from left to right and then top to bottom. `Grid` uses the supplied bounds' origin, so you can also slice a rectangular part of an image when it divides evenly. It describes regions; it does not split or repack the pixels.

`MemorySheet` retains the pointer without copying it. It needs no sheet file, JSON codec, or registration, but its texture still resolves through the game's loader. Use this reference wherever the quick start uses `source`.

### Define irregular regions

For individually placed rectangles, use the copying, validating constructor:

```go
sheet, err := asset.NewSheet("assets/sprites/characters.png",
	map[string]asset.SheetRegion{
		"hero_idle_0": {X: 0, Y: 0, W: 32, H: 32},
		"hero_idle_1": {X: 32, Y: 0, W: 32, H: 32},
	})
if err != nil {
	return err
}
source = asset.MemorySheet(sheet)
```

Alternatively, construct an `asset.Sheet` with public `Texture` and `Regions` fields and call `Validate` before sharing it. `NewSheet` copies the supplied map; a struct literal does not. Neither route loads the texture or checks its actual bounds.

Treat cached sheets and retained memory sheets, including their maps, as read-only while shared. `MemorySheet` is not a snapshot, and resolving a reference returns a value that shares the region map. Animation clips retain the same sheet reference.

## Save authored metadata

A sheet does not choose a host directory or filename. Supply a saver backed by a writable store. This helper opens an application-chosen directory, saves the sheet, and reports close errors:

```go
import (
	"errors"
	"os"

	"github.com/Leonard-Atorough/castrum/asset"
)

func saveSheet(root string, sheet *asset.Sheet) (err error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	directory, err := asset.NewDirectory(root)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, directory.Close())
	}()
	saver, err := asset.NewSaver(directory, asset.SaverOptions{})
	if err != nil {
		return err
	}
	return sheet.Save(saver, "assets/sprites/characters.sheet.json")
}
```

For a running game or authoring tool, keep the directory and saver open for their users' lifetime rather than recreating them per save. See [Directory guarantees and limits](assets.md#directory-guarantees-and-limits) for publication, permissions, and concurrency.

`Sheet.Save` validates metadata, then writes the value type `asset.Sheet` using JSON regardless of the filename's extension. Default codecs supply that encoder. The generic alternative is `saver.Save("assets/sprites/characters.sheet.json", *sheet)`; it infers JSON from the extension and does not perform sheet validation automatically.

Both routes use ordinary exported fields and the existing JSON tooling. `encoding/json` sorts map keys, producing stable output without custom marshalers. Saving writes only metadata, not the texture or a clip's playback order.

## Load and reuse a sheet

Load the document directly when setup needs its metadata:

```go
loader := g.World().MustResource[*asset.Loader]()
document, err := loader.Load[asset.Sheet]("assets/sprites/characters.sheet.json")
if err != nil {
	return err
}
region, err := document.Region("hero_idle_0")
if err != nil {
	return err
}
_ = region
```

Load the value type `asset.Sheet`, not `*asset.Sheet`. Direct loads infer the format from the extension; file references select JSON explicitly, even when the filename has another extension. A file reference and a direct JSON load of the same cleaned path share the loader's typed cache. `source.Resolve(loader)` returns the referenced sheet with a cleaned texture path; memory references need no loader to resolve their metadata.

A file reference can find a saved document only if its loader's filesystem can read that destination. Writing to a separate user-data directory does not make the document appear in an embedded content filesystem. Either keep the memory reference or load through a separate loader rooted at the save directory and retain that document with `asset.MemorySheet(&document)`. The document's texture path must still resolve through the game's loader.

Sharing codecs does not connect storage or caches. See [Save and reload a document](assets.md#save-and-reload-a-document) for constructing a loader and saver over the same directory.

Saving does not replace cached documents or GPU textures. File references follow their loader's cache policy; memory references retain their sheet pointer. There is no implicit hot reload. [CacheBypass](assets.md#choose-cached-content-or-fresh-reads) reads fresh metadata without replacing existing cached values, and does not invalidate GPU caches.

## Animate ordered regions

An animation frame is a region name, not a separate image. The sheet owns geometry, the clip owns frame order and FPS, and the component owns playback:

```text
sheet reference + ordered region names -> Clip -> Animation -> Sprite.Drawable
```

One sheet can serve several clips, such as `idle`, `walk`, and `attack`, without duplicating the image. Do not derive playback order from map iteration or sorted region names: `hero_idle_10` sorts before `hero_idle_2`. Always provide an explicitly ordered frame slice.

### Define a clip

A `Clip` retains the file or memory reference, lists its regions in playback order, and sets its rate. Import `github.com/Leonard-Atorough/castrum/animation` and add it to the game's clip store with a stable name:

```go
clip := animation.Clip{
	Source: source,
	Frames: []string{
		"hero_idle_0",
		"hero_idle_1",
	},
	FPS: 10,
}
if err := g.World().MustResource[*animation.ClipStore]().Add("hero_idle", clip); err != nil {
	return err
}
```

`ClipStore.Add` requires a non-empty, unique clip name, a valid `Source` reference, a non-empty frame slice, and finite, positive `FPS`. It performs no file I/O and does not check individual frame names, including whether they are empty or exist in the sheet. The renderer resolves the selected frame when drawing.

For setup-time checks, resolve and validate the sheet as [shown above](#choose-when-validation-happens), then check the clip's ordered names:

```go
for _, name := range clip.Frames {
	if _, err := document.Region(name); err != nil {
		return err
	}
}
```

The store copies the frame slice on `Add`, so later changes to the input slice do not change the registered definition. `ClipStore.Get` returns a shared slice; do not modify it. Register each clip before the first fixed update that uses it. Otherwise the animation system returns an error naming the entity and missing clip.

### Play an animation

Instead of the static entity in the quick start, spawn an entity with a `Sprite`, a `Transform`, and an `Animation` component that names the clip and chooses its loop mode:

```go
hero, err := g.World().NewEntity(
	core.Transform{Position: geom.Vector2{X: 100, Y: 100}},
	render.Sprite{Drawable: render.SheetSource{
		Sheet: source, Region: "hero_idle_0",
	}},
	animation.Animation{
		Clip: "hero_idle",
		Loop: animation.LoopForever,
	},
)
if err != nil {
	return err
}
```

`castrum.New` provides the clip store and registers the animation system automatically; no animation option is needed. The system queries entities with both `Animation` and `Sprite`. During fixed updates it advances unpaused playback and writes the selected `render.SheetSource` to `Sprite.Drawable`, retaining the clip's reference. A bare `render.Sprite{}` also works, but stays blank until an unpaused update supplies its drawable. The initial drawable above gives the entity a picture before that update.

Two entities can share a clip without sharing playback state. Each entity has its own current frame, elapsed time, pause state, playback speed, and loop mode.

An empty `Animation` is not playable. Component validation requires a non-empty `Clip`, a non-negative `Current`, finite non-negative `Elapsed` and `PlaybackMultiplier`, and either `LoopNone` or `LoopForever`. The animation system checks that the clip exists and that `Current` fits its frame slice, even when paused.

### Control playback

Playback controls live on the component. In a gameplay system, query animated sprites, change the component, and write it back. For example, double the playback rate of every matching entity:

```go
animatedSprites := core.NewQuery(g.World()).With(animation.Animation{}, render.Sprite{})
for e := range animatedSprites.Execute() {
	anim, _ := e.Component[animation.Animation]()
	anim.PlaybackMultiplier = 2
	e.SetComponent(anim)
}
```

The fields you will use most often are:

- `Paused` stops advancement and leaves `Sprite.Drawable` unchanged. Setting it back to false resumes from the stored frame and elapsed time.
- `PlaybackMultiplier` scales the clip's rate. Both `0` and `1` mean the authored rate; `2` plays twice as fast. Use `Paused`, not a zero multiplier, to stop playback. Negative values are invalid.
- `Loop` selects `LoopNone` or `LoopForever`.

`Restart` rewinds the frame and elapsed time to zero, resumes playback, and clears the completion state. It preserves the clip, loop mode, and multiplier. Write the changed component back; the drawable updates on the next animation-system update.

### Looping and completion

Looping belongs to the entity, on the `Animation` component:

- `animation.LoopNone` plays the clip once, then holds the last frame paused.
- `animation.LoopForever` wraps from the last frame back to the first.

`LoopNone` is the zero value, so a spawned animation plays once unless it says otherwise.

Completion happens when playback advances past the final frame, not when it first displays that frame. The engine records the fixed tick in `CompletedOn`; zero means no completion has been recorded since initialization or `Restart`.

In a fixed-phase system that runs after animation advancement, read the component and use `JustCompleted(ctx.Tick)` for the current tick's completion edge, or `HasCompleted()` for persistent non-looping completion:

```go
if anim.JustCompleted(ctx.Tick) {
	// React to this completion.
}

if anim.HasCompleted() {
	// Transition out of the completed action.
}
```

Use `JustCompleted` with running fixed ticks, not tick zero during setup: it compares tick values directly. In a frame-phase system, prefer a signal reader when every completion matters, since several fixed ticks can run between display frames.

A completed `LoopNone` animation holds its last frame and remains on the entity. Unpausing it does not replay it: on the next frame crossing, it completes again and pauses. Call `Restart` to replay.

For `LoopForever`, `HasCompleted` is always false, but `CompletedOn` and `JustCompleted` record loop boundaries. If one fixed update crosses several loops, only that tick is recorded and only one signal is emitted.

### Signals

The animation system also emits `animation.Completed` with the entity ID and fixed tick when playback passes the end of a clip. This includes loop wraps and repeat completions caused by unpausing a finished non-looping animation.

Import `github.com/Leonard-Atorough/castrum/signal`. Use a consumer system to react to a particular entity's animation without checking every `Animation` component:

```go
hub := g.World().MustResource[*signal.Hub]()

if err := g.AddSystem(core.PhaseFixed, "game.animation_complete", hub.Consumer(
	func(_ *core.Context, completed animation.Completed) error {
		if completed.Entity == hero.ID() {
			// Trigger the next game action.
		}
		return nil
	},
)); err != nil {
	return err
}
```

Signal handlers run when their consumer system updates, not immediately when animation emits. See [Signals](signals.md#lifetime-the-tick-and-frame-windows) for reader lifetime and delivery across fixed and frame phases.

### Change clips

Changing `Clip` alone preserves the current frame, elapsed time, pause state, and completion record. After registering `"run"`, use a query entry `e` from a query that includes `Animation` to switch clips and start at its first frame:

```go
anim, _ := e.Component[animation.Animation]()
anim.Clip = "run"
anim.Restart()
e.SetComponent(anim)
```

If you deliberately keep playback state, ensure `Current` fits the new clip's frame list. An out-of-range frame produces a system error rather than wrapping or resetting automatically, even when paused.

### Timing

Each clip has one fixed `FPS` value for all of its frames. The engine advances animations during the fixed phase, accumulating elapsed time until one or more frame durations have passed. A slow update can therefore advance several frames in one tick; the remaining partial-frame time is retained.

There are no per-frame durations, per-frame events, blending, reverse playback, or animation timelines in the current API. Completion signals identify the entity and tick, not a particular frame or clip. Coordinate the component from your own systems when the fixed-rate clip model does not supply the behavior you need.

## Troubleshooting

If a sheet or animation does not behave as expected, check the layer that owns the problem:

- **The entity fails to spawn:** `Animation.Clip` is empty, the loop mode is invalid, or another component is invalid.
- **The game loop reports a missing clip:** the component names a clip that was not added to the clip store.
- **The sprite cannot resolve a frame:** check the sheet path, JSON codec, texture path, region name, and actual image bounds.
- **The animation appears frozen:** check that the entity has both `Animation` and `Sprite`, then check `Paused`, `FPS`, and `PlaybackMultiplier`. It may have completed; check `HasCompleted`.
- **A paused animation has no picture:** supply an initial `Sprite.Drawable`; paused playback does not set it.
- **A completed animation resumes oddly:** unpausing a finished animation does not replay it; call `Restart`.
- **A switched clip starts at a strange frame:** call `Restart` when changing `Clip`.
- **Grid construction fails:** check that tile dimensions evenly divide the image bounds and that the prefix is non-empty.
- **A saved sheet cannot be found:** check that the loader reads the saver's destination, and that the texture path is relative to the game's root.
- **An edited sheet still shows old content:** saving does not invalidate decoded or GPU caches; retained memory references also keep their original sheet.

## Where to go next

- [Assets](assets.md) - filesystem roots, codecs, writable storage, and cache policies.
- [Rendering](rendering.md) - sprite style, cameras, ordering, and culling.
- [ECS](ecs.md) - queries and component updates for gameplay control.
- [Signals](signals.md) - independent readers and completion delivery.
- [examples/animate](../../../examples/animate) - runnable grid slicing and frame playback; use the repository's development workspace.
