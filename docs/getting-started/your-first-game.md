# Your First Game

<!-- TUTORIAL, COMPLETE - a hand-held, step-by-step walk from empty
     directory to a small tank-driving game. All sections are prose.

     VERIFICATION: both the engine/runner skeleton exactly as printed
     (three imports, no entity yet) and the accumulated end-state
     program (tank controls: move_forward/move_backward/turn_left/
     turn_right bindings, hull and turret texture entities, mount
     system, drive system with the facing vector, music entity, SFX
     system, texture preloads, post-runner audio preload) build and
     vet against the current module - both verified in scratch
     modules, the end-state with the exact assets in
     examples/your-first-game. Keep every block verbatim-synced with
     the verified programs when editing.

     GROWTH (planned sections - add beats, not prose, until then):
     - Timers (when timers land)
     - Text
     - UI
     The layout grows by inserting sections under "Adding more features". -->

Let's get started. This page will take you through the process of building your first game with Castrum, step by step.

Interested in a particular section? Jump to it using the links below.

## Table of Contents

- [Before you start](#before-you-start)
- [Setting up your environment](#setting-up-your-environment)
- [Setting up the engine and runner](#setting-up-the-engine-and-runner)
- [Creating the player entity](#creating-the-player-entity)
- [Writing your first system](#writing-your-first-system)
- [Try the game](#try-the-game)
- [Adding more features](#adding-more-features)
  - [Adding audio playback](#adding-audio-playback)
    - [Background music](#background-music)
    - [Sound effects](#sound-effects)
  - [Switching to sprite-drawn player](#switching-to-sprite-drawn-player)
- [Next steps](#next-steps)

## Before you start

Before you start, make sure you have Go 1.27 or later installed. On Windows, no C toolchain is required as the runner's dependencies ship pre-built.

Familiarity with the [introduction](intro.md) and [key concepts](concepts.md) chapters is recommended, as this tutorial builds on those concepts without re-teaching them.

In this tutorial, you'll build a game where a green square is controlled by the arrow keys or WASD, a looping music track plays in the background, and a sound effect is triggered by the spacebar. By the end, you'll upgrade from a moving square to a mouse-aiming tank.

Each section adds one feature, and every intermediate program runs successfully.

You also need the tutorial asset bundle: an `audio/` directory with a sound effect and a music track, and a `sprites/` directory with two tank textures. From the [GitHub releases page](https://github.com/Leonard-Atorough/castrum/releases), download `castrum-tutorial-assets-<version>.zip` from the release matching the Castrum version you installed, then extract it into your project folder.

The code refers to them as `audio/...` and `sprites/...`, so the paths line up with nothing extra to configure. The audio files come into play in the audio section; the textures wait until the end, when the square becomes a tank.

You can use your own files instead, but the bundle is the set this tutorial can promise works.

## Setting up your environment

Start with an empty directory and a fresh module:

```sh
mkdir firstgame && cd firstgame
go mod init firstgame
go get github.com/Leonard-Atorough/castrum
```

Create an empty `main.go` in the project root. Confirm that the project now contains `main.go`, `audio/`, and `sprites/`.

## Setting up the engine and runner

Here is the full skeleton. Type it in or paste it - every later section slots into one of the two comments:

```go
package main

import (
	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/input"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	bindings := input.Bindings{
		"move_forward": []input.Input{
			input.KeyInput{Key: input.KeyW},
			input.KeyInput{Key: input.KeyArrowUp},
		},
		"move_backward": []input.Input{
			input.KeyInput{Key: input.KeyS},
			input.KeyInput{Key: input.KeyArrowDown},
		},
		"turn_left": []input.Input{
			input.KeyInput{Key: input.KeyA},
			input.KeyInput{Key: input.KeyArrowLeft},
		},
		"turn_right": []input.Input{
			input.KeyInput{Key: input.KeyD},
			input.KeyInput{Key: input.KeyArrowRight},
		},
	}

	g, err := castrum.New(
		castrum.WithTitle("your first game"),
		castrum.WithBindings(bindings),
	)
	if err != nil {
		return err
	}

	// player entity: next section
	// systems: the section after that

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	return g.Run(runner)
}
```

`castrum.New` creates the game and its world. The bindings give names to the keys the game will read later. `ebitrun.New` supplies the window and input, and `g.Run(runner)` starts the game.

Run it now. An empty window should open. Close it, then continue to add the player.

Add new imports when later sections use `core`, `geom`, `color`, `audio`, or `asset`.

## Creating the player entity

Add this below `castrum.New`. The player needs a position and something to draw; movement comes in the next step.

```go
player, err := g.World().NewEntity(
	core.Transform{Position: geom.Vector2{X: 640, Y: 360}},
	core.Sprite{
		Drawable: core.RectShape{Size: geom.Vector2{X: 120, Y: 120}},
		Color:   color.RGBA{G: 200, A: 255},
	},
)
if err != nil {
	return err
}
```

This creates a green 120-by-120 square at the center of the window. Keep the `player` handle; the movement system will use it next.

## Writing your first system

Now make the square drive. Add this function at the top level of the file, next to `run`:

```go
func moveSystem(player *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		dt := ctx.DeltaTime.Seconds()
		return player.Update(ctx.World, func(t *core.Transform) {
			facing := geom.Vector2{X: 0, Y: -1}.Rotate(t.Rotation)
			if ctx.Actions.Held("move_forward") {
				t.Position = t.Position.Add(facing.Mul(Speed * dt))
			}
			if ctx.Actions.Held("move_backward") {
				t.Position = t.Position.Sub(facing.Mul(Speed * dt))
			}
			if ctx.Actions.Held("turn_left") {
				t.Rotation -= TurnSpeed * dt
			}
			if ctx.Actions.Held("turn_right") {
				t.Rotation += TurnSpeed * dt
			}
		})
	})
}
```

Declare the movement speeds and register the system where the `// systems:` comment is waiting:

```go
const (
	Speed     = 300 // world units per second
	TurnSpeed = 3   // radians per second
)

if err := g.AddSystem(core.PhaseFixed, "player.move", moveSystem(player)); err != nil {
	return err
}
```

Run the game again. W or the up arrow should move the square forward, S or the down arrow should move it backward, and A/D or the left/right arrows should turn it. The system runs on fixed ticks, and `dt` keeps the speed consistent across different frame rates. The named actions let the system use `move_forward` without knowing which physical key produced it.

## Try the game

Try a few changes from the project directory:

```sh
go run .
```

- Change `Speed` and `TurnSpeed` and run again. Doubling them should feel twice as fast - that is `dt` doing its job.
- Drive forward while turning: the square traces a circle, because the position advances along `facing` while the facing itself rotates.
- Hold forward and backward together. Each `if` runs every tick, so the two moves cancel out and the square sits still - but turning still works, which is exactly how a tank pivots.

The square should move smoothly. Fixed ticks update the simulation while the renderer interpolates its position between ticks.

## Adding more features

The square drives. Now add music, then a sound effect.

Add `audio` and `asset` to the import block before continuing.

### Adding audio playback

No game feels complete without some music and sound. In the examples folder, we've provided two audio tracks for you to use. The first is arpmedia-retro-arcade-game-music-577821.mp3 for background music. The second is GUNArtl_Rocket Launcher Fire_02.wav for a shooting sound effect.

#### Background music

Let's start with the background music. Declare the asset path and create an audio source entity before the runner, alongside the player entity.

```go
const AssetMusic = "audio/arpmedia-retro-arcade-game-music-577821.mp3"

if _, err := g.World().NewEntity(audio.Source{
	Audio:  AssetMusic,
	Volume: 1,
	Group:  audio.GroupMusic,
	Loop:   audio.LoopForever,
	Load:   audio.LoadStream,
	Pause:  audio.PauseHolds,
}); err != nil {
	return err
}
```

Run the game again. The music starts on the first frame and loops continuously. `audio.LoadStream` is a good choice for a long music track because it does not decode the entire file into memory, and `audio.PauseHolds` makes the track pause with the game.

The other audio settings are explained in the [audio guide](../guides/audio.md). Next, we'll add a sound effect that plays when you press the spacebar.

#### Sound effects

First, preload the sound effect immediately after the existing `ebitrun.New(g)` block and before `g.Run(runner)`. This decodes the file during setup, so a missing or invalid asset fails before the first key press:

```go
const AssetSFX = "audio/GUNArtl_Rocket Launcher Fire_02.wav"

if _, err := g.AssetServer().Load[asset.AudioData](AssetSFX); err != nil {
	return err
}
```

The preload must come after `ebitrun.New`, because the runner registers the audio decoders.

Now add a system next to `moveSystem`. It creates a one-shot play whenever the spacebar is pressed:

```go
func sfxSystem() core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if !ctx.Input.KeyPressed(input.KeySpace) {
			return nil
		}
		shot := audio.NewSource(AssetSFX)
		shot.Pause = audio.PauseContinues
		_, err := audio.OneShot(ctx.World, shot)
		return err
	})
}
```

Register the system where the `// systems:` comment is:

```go
if err := g.AddSystem(core.PhaseFrame, "player.sfx", sfxSystem()); err != nil {
	return err
}
```

Run the game and press the spacebar. Each press plays the effect once; holding the key does not retrigger it until you release and press again. `audio.OneShot` handles the temporary audio entity for you, including removing it when playback finishes.

The [audio guide](../guides/audio.md) covers volume controls, pause behavior, and the rest of the audio API. The [audio example](../../examples/audio) shows those controls in a complete program.

### Switching to a sprite-drawn player

The square has been our placeholder for now but let's replace it with a pair of sprites representing a hull and turret and turn the player into a tank; the hull will handle movement, and the turret will follow its position.

Add `math` to the import block, then replace the previous player entity creation with the following:

```go
const (
	AssetHull   = "sprites/ww2_top_view_hull4.png"
	AssetTurret = "sprites/ww2_top_view_turret4.png"
)

var (
	SpawnPos     = geom.Vector2{X: 640, Y: 360}
	TurretOffset = geom.Vector2{X: 0, Y: -16}
)

hull, err := g.World().NewEntity(
	core.Transform{Position: SpawnPos},
	core.Sprite{Drawable: core.TextureSource{Texture: AssetHull}},
)
if err != nil {
	return err
}

turret, err := g.World().NewEntity(
	core.Transform{Position: SpawnPos.Add(TurretOffset)},
	core.Sprite{Drawable: core.TextureSource{Texture: AssetTurret}},
)
if err != nil {
	return err
}
```

Both entities start at the center of the screen, with the turret slightly offset above the hull.

Next we need to update the movement system to control the hull entity instead of the previous player entity.

```go
func moveSystem(hull *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		dt := ctx.DeltaTime.Seconds()
		return hull.Update(ctx.World, func(t *core.Transform) {
			facing := geom.Vector2{X: 0, Y: -1}.Rotate(t.Rotation)
			if ctx.Actions.Held("move_forward") {
				t.Position = t.Position.Add(facing.Mul(Speed * dt))
			}
			if ctx.Actions.Held("move_backward") {
				t.Position = t.Position.Sub(facing.Mul(Speed * dt))
			}
			if ctx.Actions.Held("turn_left") {
				t.Rotation -= TurnSpeed * dt
			}
			if ctx.Actions.Held("turn_right") {
				t.Rotation += TurnSpeed * dt
			}
		})
	})
}
```

Also change the registration from `moveSystem(player)` to `moveSystem(hull)`:

```go
if err := g.AddSystem(core.PhaseFixed, "player.hull.move", moveSystem(hull)); err != nil {
	return err
}
```

The turret needs the camera to aim at the mouse correctly. The cursor position is in screen coordinates, but the tank is in world coordinates. The camera's `Transform` tells us where the view is centered, and its `Camera` component tells us the zoom. `CameraView.ScreenToWorld` converts the cursor using both values.

Add this turret system next to `moveSystem`. It keeps the turret mounted on the hull and rotates it toward the mouse:

```go
func turretSystem(turret, hull, camera *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		cameraData, _ := camera.Component[core.Camera](ctx.World)
		cameraTransform, _ := camera.Component[core.Transform](ctx.World)

		cursor := core.CameraView{
			Position: cameraTransform.Position,
			Zoom:     cameraData.Zoom,
		}.ScreenToWorld(ctx.Input.Cursor(), ctx.LogicalWidth, ctx.LogicalHeight)
		aim := cursor.Sub(hullTransform.Position)

		return turret.Update(ctx.World, func(t *core.Transform) {
			t.Position = hullTransform.Position.Add(TurretOffset.Rotate(hullTransform.Rotation))
			t.Rotation = math.Atan2(aim.X, -aim.Y)
		})
	})
}

if err := g.AddSystem(core.PhaseFixed, "player.turret", turretSystem(turret, hull, g.MainCamera())); err != nil {
	return err
}
```

Register `player.turret` after `player.hull.move`. The movement system writes the hull's position first, and the turret system reads that position in the same tick. `math.Atan2` produces the angle from the hull to the cursor; the negative Y argument keeps rotation zero pointed toward the top of the screen.

Finally, preload both textures before `g.Run(runner)`. This makes missing or invalid image files fail during setup:

```go
if _, err := g.AssetServer().Load[asset.TextureData](AssetHull); err != nil {
	return err
}
if _, err := g.AssetServer().Load[asset.TextureData](AssetTurret); err != nil {
	return err
}
```

Run the game again. The tank now moves and turns as one object. Try a few changes:

- Move the mouse around the window; the turret should follow it while the hull drives independently.
- Change `TurretOffset` and see the turret stay attached as the hull turns.
- Compare this system with the [input example](../../examples/input), which adds camera following and zoom.
- Add another entity to the tank and update it from a system in the same way.

You now have a small tank that moves with the keyboard, aims at the mouse, plays looping music, and fires a sound effect with the spacebar.

## Next steps

Try changing the tank textures or the input bindings first. When you are ready to add another feature, use the [guides](../guides) for rendering, input, animation, and audio, or browse the [examples](../../examples) for complete programs.

The [cheat sheet](cheat-sheet.md) is useful once you want to recall the patterns without walking through them again.
