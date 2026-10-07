# Animation

Castrum currently supports frame-based sprite animation. An animation clip plays named regions from an atlas in order at a fixed frame rate. The same model works for a spritesheet: register the sheet as a grid atlas, then use its generated regions as clip frames.

The short version is: prepare an atlas, register a clip, spawn an entity with `Animation` and `Sprite`, and let the engine advance the sprite each fixed tick. The renderer does not need to know that animation exists; its drawable simply changes over time.

The runnable version is [examples/animate](../../examples/animate). It embeds a spritesheet, registers it as a grid atlas, creates a flickering-torch clip, and spawns an animated entity.

## The animation model

Animation has three parts:

- An atlas which owns the image and its named regions.
- An `AnimationClip` which owns the ordered frame names, FPS, and loop mode.
- An `Animation` component which owns one entity's current clip and playback state.

The relationship is:

```text
atlas regions -> AnimationClip -> Animation component -> Sprite.Drawable
```

The atlas owns frame geometry, while the clip owns frame order. This keeps one spritesheet useful for several animations, such as `idle`, `walk`, and `attack`, without duplicating the image.

## Prepare the frames

An animation frame is an atlas region. The animation package does not load image files or divide spritesheets itself. Register the atlas through the asset server first.

For a regular spritesheet, use `RegisterGridAtlas`. The tile dimensions must divide the image dimensions evenly, and the generated regions are named in row-major order using the prefix:

```go
if err := g.AssetServer().RegisterGridAtlas(
	"torch_light",
	"torch_light.png",
	16, 28,
	"torch_light",
); err != nil {
	return err
}
```

This creates regions such as `torch_light_0`, `torch_light_1`, and `torch_light_2`. For irregular frame sizes, register an atlas from a JSON sidecar instead. The [assets guide](assets.md) covers both atlas forms.

Register the atlas during setup, before spawning entities that refer to it. Grid dimensions, the prefix, the image, and sidecar rectangles are validated during registration. An unknown atlas or region is reported later when the renderer resolves a sprite.

## Define a clip

An `AnimationClip` names the atlas, lists its regions in playback order, and sets its rate and looping behavior. Add it to the game's clip store with a stable name:

```go
if err := g.Clips().Add("flickering_torch", animation.AnimationClip{
	Source: "torch_light",
	Frames: []string{
		"torch_light_0",
		"torch_light_1",
		"torch_light_2",
		"torch_light_3",
		"torch_light_4",
		"torch_light_5",
	},
	FPS:  10,
	Loop: animation.LoopForever,
}); err != nil {
	return err
}
```

`ClipStore.Add` validates the clip's own fields: the name must be unique, `Source` must not be empty, `Frames` must contain at least one name, and `FPS` must be positive. It does not verify atlas or region names at registration time. Those references are resolved when the sprite is rendered.

The clip store copies the frame slice when you add a clip, so changing the original slice later does not change the registered definition. A clip can be registered before or after the entities that use it, but an entity that reaches the animation system with a missing clip produces an error naming the entity and clip.

## Play an animation

Spawn an entity with a `Sprite`, a `Transform`, and an `Animation` component that names the clip:

```go
_, err := g.World().NewEntity(
	core.Transform{Position: geom.Vector2{X: 100, Y: 100}},
	core.Sprite{},
	animation.Animation{Clip: "flickering_torch"},
)
if err != nil {
	return err
}
```

The animation system is registered automatically by `castrum.New`; you do not register it yourself. During fixed updates it advances elapsed time, selects the current frame, and writes a `core.AtlasSource` to `Sprite.Drawable`.

Two entities can share a clip without sharing playback state. Each entity has its own current frame, elapsed time, pause state, and playback speed.

An empty `Animation` is not playable. Its `Clip` field must name the clip to play, and the component is validated when you spawn or update the entity.

## Control playback

Playback is state on the component. Query the entity, change the component, and write it back:

```go
for e := range torches.Execute() {
	anim, _ := e.Component[animation.Animation]()
	if hit {
		anim.PlaybackMultiplier = 2
		e.SetComponent(anim)
	}
}
```

The fields you will use most often are:

- `Paused` stops advancement while keeping the current frame.
- `PlaybackMultiplier` scales the clip's rate. `0` is the normal rate, `1` is the authored rate, and `2` plays twice as fast.

The component also provides four methods:

- `Pause` pauses at the current frame.
- `Resume` continues from the current frame.
- `Restart` rewinds to frame zero and resumes.
- `Stop` rewinds to frame zero and pauses.

Use `Restart` when an effect should play again from the beginning. Use `Stop` when it should remain at its first frame until something resumes it.

## Looping and completion

Looping belongs to the clip rather than the entity:

- `animation.LoopForever` wraps from the last frame to the first.
- `animation.LoopNone` plays once, holds the last frame, and sets `Paused`.

`LoopNone` is the zero value, so it is the default when you want a one-shot animation. To react to completion, query the component and check `Paused` for a non-looping clip, then remove the entity or trigger the next state.

## Change clips

The current frame and elapsed time belong to the `Animation` component, not the clip. Reset them when switching to a clip whose frame sequence may have a different length or starting pose:

```go
anim, _ := e.Component[animation.Animation]()
anim.Clip = "run"
anim.Current = 0
anim.Elapsed = 0
anim.Paused = false
e.SetComponent(anim)
```

Resetting is especially important when the previous clip's current frame is outside the new clip's frame list. A clip switch does not automatically reset playback state.

## Timing

Each clip has one fixed `FPS` value for all of its frames. The engine advances animations during the fixed phase, accumulating elapsed time until one or more frame durations have passed. A slow update can therefore advance several frames in one tick; the remaining partial-frame time is retained.

There are no per-frame durations, frame events, blending, reverse playback, or animation timelines in the current API. For those behaviors, coordinate the component from your own systems or wait for a future animation API rather than assuming the current clip model provides them.

## Troubleshooting

If an animation does not behave as expected, check the layer that owns the problem:

- **The entity fails to spawn:** `Animation.Clip` is empty or another component is invalid.
- **The game loop reports a missing clip:** the component names a clip that was not added to `g.Clips()`.
- **The sprite cannot resolve a frame:** the clip's atlas or region name does not exist.
- **The animation appears frozen:** check `Paused`, `FPS`, and `PlaybackMultiplier`.
- **A switched clip starts at a strange frame:** reset `Current` and `Elapsed` when changing `Clip`.
- **A spritesheet registration fails:** check that tile dimensions evenly divide the image and that the generated prefix matches the clip frame names.

## Where to go next

- [Assets](assets.md) - filesystem roots, atlas registration, and frame regions.
- [Rendering](rendering.md) - the `Sprite` and `AtlasSource` that animation updates.
- [ECS](ecs.md) - queries and component updates for gameplay control.
- [examples/animate](../../examples/animate) - a complete spritesheet animation example.
