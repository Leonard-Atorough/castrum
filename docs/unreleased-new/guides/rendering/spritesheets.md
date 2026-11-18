# Spritesheets and animation

A spritesheet lets several pictures share one texture. Instead of loading a separate file for each pose of a character, you can give each rectangle in the texture a name, draw one by name, and change the name when the character moves.

This guide explains how to define a sheet, draw a region, and play a sequence of regions as an animation. If you already have a sheet, jump to [drawing a region](#draw-one-region) or [playing frames](#play-the-frames). Later sections cover [playback controls](#pause-resume-and-restart), [startup checks](#check-the-sheet-before-opening-the-window), and [creating regions in Go](#create-regions-in-go). The [animate example](../../../../examples/animate/main.go) brings these ideas together in a running game with a flickering torch.

You'll need a game created with `castrum.New` and a runner to display it. The snippets belong in an error-returning setup function with access to `g`; they use the development API. For the basics of displaying an entity, see [Draw and position an image](images.md).

## Make a sheet

Imagine a character image with two 32-by-32 frames side by side. Put the PNG at `assets/walk.png`, then save this document as `assets/walk.sheet.json`:

```json
{
  "texture": "assets/walk.png",
  "regions": {
    "walk_0": { "x": 0, "y": 0, "w": 32, "h": 32 },
    "walk_1": { "x": 32, "y": 0, "w": 32, "h": 32 }
  }
}
```

The rectangles are measured in image pixels: `walk_0` starts at the top-left, and `walk_1` starts 32 pixels to its right. The image must be at least 64 by 32 pixels. The JSON describes the regions; it does not contain the image or decide which frame plays next.

Both `assets/walk.sheet.json` and the texture path inside it are relative to the game's asset filesystem root. The texture path is **not** relative to the JSON file. With the default filesystem, run the game from the directory containing `assets`; see [Load game assets](../assets/loading.md) if you use a different root.

## Draw one region

First, make a reference to the document and use it as the drawable for a sprite:

```go
source := asset.FileSheet("assets/walk.sheet.json")

hero, err := g.World().NewEntity(
	core.Transform{},
	render.Sprite{Drawable: render.SheetSource{
		Sheet:  source,
		Region: "walk_0",
	}},
)
if err != nil {
	return fmt.Errorf("create hero: %w", err)
}
```

Import `fmt` and the `asset`, `core`, and `render` packages from Castrum. The default camera looks at the world origin, so the first frame appears in the center of the window. Keep `source` and `hero` for the animation below. If you already have another sprite at the origin, move or remove it so the two don't overlap.

`FileSheet` makes a reference; it does not open the document. When the renderer collects the sprite, it loads the sheet and texture, finds `walk_0`, and checks that the rectangle fits the image. A wrong path or region can therefore fail while the game is drawing rather than when the entity is created.

You can change a still sprite to another region by updating its `Drawable`. For repeated changes, an animation is more convenient: you describe the order once and let the animation system advance it.

## Play the frames

The sheet knows where each picture is, but it doesn't know their playback order. A **clip** supplies that order and a rate; an **animation component** keeps track of one entity's playback.

Import `github.com/Leonard-Atorough/castrum/animation`, then register a clip using the same `source`:

```go
clip := animation.Clip{
	Source: source,
	Frames: []string{"walk_0", "walk_1"},
	FPS:    6,
}
clips := g.World().MustResource[*animation.ClipStore]()
if err := clips.Add("walk", clip); err != nil {
	return fmt.Errorf("register walk animation: %w", err)
}
```

The order in `Frames` is the order you'll see. A sheet's regions are a map, so don't depend on map iteration to arrange an animation. Several clips can share the same sheet: for example, one could use the idle frames and another the walking frames.

Give the hero an animation component after creating it:

```go
if err := hero.AddComponent(g.World(), animation.Animation{
	Clip: "walk",
	Loop: animation.LoopForever,
}); err != nil {
	return fmt.Errorf("start walk animation: %w", err)
}
```

Run the game from your asset root. The two regions should alternate six times per second while the game is advancing. `castrum.New` already registers the animation system; it updates entities that have both an `Animation` and a `Sprite`. The initial drawable gives the hero a picture before the first animation update.

The clip is shared, but the playback state belongs to each entity. Two heroes using `"walk"` can be paused or restarted independently. Without `LoopForever`, the animation plays once and holds its final frame.

## Pause, resume, and restart

Update the animation component from a system just as you would any other component. For example, this toggle pauses a hero if it is playing and resumes it if it is paused:

```go
if err := hero.Update(ctx.World, func(a *animation.Animation) {
	a.Paused = !a.Paused
}); err != nil {
	return fmt.Errorf("toggle walk animation: %w", err)
}
```

This snippet assumes a system with `ctx` and the `hero` handle. To restart from the beginning, call `a.Restart()` inside the update instead. `PlaybackMultiplier` changes the rate: `2` plays twice as fast, while `0` and `1` both mean the clip's normal rate. Use `Paused` rather than a zero multiplier to stop playback.

For a one-shot animation, `HasCompleted()` stays true after playback finishes; `JustCompleted(ctx.Tick)` identifies the fixed tick on which it finished. A looping clip can complete more than once, so use `JustCompleted` to observe each loop. The engine also emits an `animation.Completed` signal. The [animate example](../../../../examples/animate/main.go) demonstrates pause, restart, loop selection, and a completion readout.

## Check the sheet before opening the window

The short path above deliberately lets the renderer load the sheet when it needs it. For required artwork, you may prefer to find problems during setup. Resolve the reference, load its texture, and check its bounds:

```go
loader := g.World().MustResource[*asset.Loader]()
sheet, err := source.Resolve(loader)
if err != nil {
	return fmt.Errorf("load walk sheet: %w", err)
}
texture, err := loader.Load[asset.TextureData](string(sheet.Texture))
if err != nil {
	return fmt.Errorf("load walk texture: %w", err)
}
if err := sheet.ValidateBounds(texture.Image.Bounds()); err != nil {
	return fmt.Errorf("validate walk sheet: %w", err)
}
```

This example uses the default texture decoder. `Resolve` loads the JSON through the game's loader; the default decoder checks the sheet's metadata, but it doesn't load the image. `ValidateBounds` checks every rectangle against the actual image, so it catches an otherwise valid rectangle that extends past the texture.

There is one more check for animations: registering a clip doesn't read the sheet or verify its frame names. After resolving the sheet, check the names in the clip you intend to play:

```go
for _, name := range clip.Frames {
	if _, err := sheet.Region(name); err != nil {
		return fmt.Errorf("find animation frame: %w", err)
	}
}
```

Run these checks before starting the runner if the game shouldn't open with missing artwork. If a custom texture decoder is used, it must provide a non-nil `Image` for the bounds check.

## Create regions in Go

You don't have to keep the sheet as a JSON file. If your frames form a regular grid, `spritesheet.Grid` can name them for you. It needs the image's bounds, the cell dimensions, and a name prefix:

```go
texture, err := loader.Load[asset.TextureData]("assets/walk.png")
if err != nil {
	return fmt.Errorf("load walk texture: %w", err)
}
sheet, err := spritesheet.Grid(
	"assets/walk.png", texture.Image.Bounds(), 32, 32, "walk",
)
if err != nil {
	return fmt.Errorf("slice walk sheet: %w", err)
}
source := asset.MemorySheet(sheet)
```

Import `github.com/Leonard-Atorough/castrum/spritesheet` and use the game's loader as in the previous section. This is an alternative to `FileSheet`, not an additional step after it. Use the resulting `source` in the same sprite and clip code above.

`Grid` names cells from left to right, then top to bottom: `walk_0`, `walk_1`, and so on. The supplied bounds must divide evenly into positive-size cells. It describes rectangles; it doesn't split or modify the image. If your artwork has irregularly placed regions, use `asset.NewSheet` with a map of names to `asset.SheetRegion` values instead.

`MemorySheet` retains the sheet you pass it. Keep the sheet and its region map read-only while sprites or clips share it. The sheet is in memory, but its texture path still has to resolve through the game's loader.

## Try the complete example

In the [development workspace](../../../../CONTRIBUTING.md#develop-examples-against-the-checkout), run `go run .` from `examples/animate`. The example embeds its artwork, checks the sheet during setup, and plays a torch animation. SPACE pauses it, R restarts it, and 1 or 2 selects play-once or looping playback. It can also generate a sheet document with its `-author` option.

If a frame doesn't appear or an asset fails to load, start with [Diagnose an asset or rendering failure](../assets/troubleshooting.md). For storage and asset roots, see [Load game assets](../assets/loading.md).
