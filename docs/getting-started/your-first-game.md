# Your First Game

Let's get started. This page walks you from a new Go module to a small, playable tank game, adding one working feature at a time.

Castrum separates the game from the platform runner: the `Game` owns the world and its systems, while `ebitrun` supplies the window, rendering, and input. You will build the simulation from entities and components, then let fixed-tick systems update them.

Interested in a particular section? Jump to it using the links below.

## Table of Contents

- [Before you start](#before-you-start)
- [Setting up your environment](#setting-up-your-environment)
- [Setting up the engine and runner](#setting-up-the-engine-and-runner)
- [Creating the tank](#creating-the-tank)
- [Driving the tank](#driving-the-tank)
- [Aiming the turret](#aiming-the-turret)
- [Adding more features](#adding-more-features)
  - [Adding audio playback](#adding-audio-playback)
    - [Background music](#background-music)
    - [Sound effects](#sound-effects)
  - [A camera that follows the tank](#a-camera-that-follows-the-tank)
  - [Firing the gun](#firing-the-gun)
  - [A cooldown timer](#a-cooldown-timer)
  - [Adding enemies](#adding-enemies)
  - [Chasing enemies](#chasing-enemies)
  - [The reload readout](#the-reload-readout)
  - [Score, win, and loss](#score-win-and-loss)
- [Next steps](#next-steps)

## Before you start

Install Go 1.27 or later. On Windows, you do not need a C toolchain; the runner's dependencies include prebuilt binaries.

Familiarity with the [introduction](intro.md) and [key concepts](concepts.md) chapters is recommended, as this tutorial builds on those concepts without re-teaching them.

Each step leaves the game runnable.

Download `castrum-tutorial-assets-<version>.zip` from the [GitHub release matching your Castrum version](https://github.com/Leonard-Atorough/castrum/releases) and extract it into your project folder. The bundle supplies the `audio/`, `sprites/`, and `fonts/` files at the paths used below; you can substitute your own assets, but this bundle is the set the tutorial is verified with.

## Setting up your environment

Start with an empty directory and a fresh module. Keep your shell in this project directory as you work:

```sh
mkdir firstgame && cd firstgame
go mod init firstgame
go get github.com/Leonard-Atorough/castrum
```

Create `main.go` and extract the asset bundle in the project root. Asset paths resolve from the process working directory, so run the commands below from that directory.

## Setting up the engine and runner

Here is the full skeleton. Type it in or paste it.

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

	// tank entities: next section
	// systems: the section after that

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	return g.Run(runner)
}
```

`castrum.New` creates the game and its world. The bindings give names to the keys the game will read later. `ebitrun.New` supplies the window and input, and `g.Run(runner)` starts the game.

Timer, collision, and animation systems are opt-in; add their `With...` options when needed. `castrum.WithDefaultSystems()` enables all three.

Run it from the project root:

```sh
go run .
```

An empty window should open. Close it, then run `go run .` after each change.

Add imports as each section needs them.

## Creating the tank

The player consists of two co-located entities: a moving hull and an independently aiming turret.

Add `core`, `geom`, `asset`, `render`, and `image/color` to the import block. Then declare the asset paths and the spawn point, and create both entities below `castrum.New`:

```go
const (
	AssetHull   = "sprites/T-34/ww2_top_view_hull4.png"
	AssetTurret = "sprites/T-34/ww2_top_view_turret4.png"
)

var SpawnPos = geom.Vector2{X: 640, Y: 360}
```

```go
hull, err := g.World().NewEntity(
	core.Transform{Position: SpawnPos},
	render.Sprite{Drawable: render.TextureSource{Texture: AssetHull}},
)
if err != nil {
	return err
}
```

```go
turret, err := g.World().NewEntity(
	core.Transform{Position: SpawnPos},
	render.Sprite{Drawable: render.TextureSource{Texture: AssetTurret}, Layer: 1},
)
if err != nil {
	return err
}
```

Each entity pairs a `Transform` - where it is, in world units - with a `Sprite` - what it draws. Here the drawable is a `TextureSource`, which names a texture asset; the turret sits on the same position as the hull, and its system re-centers it every tick so the pair stays together. The turret's `Layer` is 1, one draw band above the hull: the pair shares a position, so every sort key within a layer ties, and the layer keeps the turret visibly on top.

One more static sprite: a white tile under the start position. An empty field gives the eye nothing to move against - once the camera follows, driving would read as standing still. Give the world a frame of reference:

```go
	TileSize = 160 // world units per side
```

```go
var tileColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
```

```go
	if _, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos},
		render.Sprite{
			Drawable:  render.RectShape{Size: geom.Vector2{X: TileSize, Y: TileSize}},
			Color:     tileColor,
			SortOrder: -1,
		},
	); err != nil {
		return err
	}
```

The tile's `SortOrder` is `-1`: sprites in the same layer draw in sort order, and the tank's zero sorts on top of the tile.

Textures load lazily by default, but preloading them makes a missing or invalid file fail during setup, before the game ever runs. Preload both next to the entities:

```go
if _, err := g.World().MustResource[*asset.Server]().Load[asset.TextureData](AssetHull); err != nil {
	return err
}
if _, err := g.World().MustResource[*asset.Server]().Load[asset.TextureData](AssetTurret); err != nil {
	return err
}
```

The tank should sit on its white tile at the center of the window, turret riding the hull. Nothing moves yet; that is the next system.

## Driving the tank

Now make the tank drive. Declare the movement speeds alongside the other constants:

```go
const (
	Speed     = 100 // world units per second
	TurnSpeed = 3   // radians per second
)
```

Add this function at the top level of the file, next to `run`:

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

Register it where the `// systems:` comment is waiting:

```go
if err := g.AddSystem(core.PhaseFixed, "player.hull.move", moveSystem(hull)); err != nil {
	return err
}
```

W or the up arrow should move the tank forward, S or the down arrow backward, and A/D or the left/right arrows should turn it. Fixed ticks and `dt` keep movement consistent across frame rates. Named actions let the system use `move_forward` without knowing which physical key produced it.

Try a few changes from the project directory:

- Change `Speed` and `TurnSpeed` and run again. Doubling them should feel twice as fast - that is `dt` doing its job.
- Drive forward while turning: the tank traces a circle, because the position advances along `facing` while the facing itself rotates.
- Hold forward and backward together. Each `if` runs every tick, so the two moves cancel out and the tank sits still - but turning still works, which is exactly how a tank pivots.

The tank should move smoothly. Fixed ticks update the simulation while the renderer interpolates its position between ticks.

## Aiming the turret

The turret needs the camera to aim at the mouse correctly. The cursor position is in screen coordinates, but the tank is in world coordinates. The camera's `Transform` tells us where the view is centered, and its `Camera` component tells us the zoom. `CameraView.ScreenToWorld` converts the cursor using both values.

Add `math` to the import block. Then add this turret system next to `moveSystem`. It keeps the turret mounted on the hull and rotates it toward the mouse:

```go
func turretSystem(turret, hull, camera *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		cameraData, _ := camera.Component[render.Camera](ctx.World)
		cameraTransform, _ := camera.Component[core.Transform](ctx.World)

		cursor := render.CameraView{
			Position: cameraTransform.Position,
			Zoom:     cameraData.Zoom,
		}.ScreenToWorld(ctx.Input.Cursor(), ctx.LogicalWidth, ctx.LogicalHeight)
		aim := cursor.Sub(hullTransform.Position)

		return turret.Update(ctx.World, func(t *core.Transform) {
			t.Position = hullTransform.Position
			t.Rotation = math.Atan2(aim.X, -aim.Y)
		})
	})
}
```

Register it after `player.hull.move`. The movement system writes the hull's position first, and the turret system reads that position in the same tick. `math.Atan2` produces the angle from the hull to the cursor; the negative Y argument keeps rotation zero pointed toward the top of the screen.

```go
if err := g.AddSystem(core.PhaseFixed, "player.turret", turretSystem(turret, hull, g.MainCamera())); err != nil {
	return err
}
```

The engine spawns a default camera behind the scenes, and `g.MainCamera()` returns it - the turret reads its position and zoom without the game creating one.

Move the mouse around the window; the turret should follow while the hull drives independently. Drive forward while aiming: the turret stays pointed at the cursor because its aim is recomputed from the hull's position each tick. Compare this with the [input example](../../examples/input), which uses the same camera helpers with QE zoom.

## Adding more features

The tank drives and aims. Now give it a world: music and sound first, then a camera that follows, the gun, and the enemies.

Add `audio` and `asset` to the import block before continuing. (`asset` is already imported from the tank section.)

### Adding audio playback

No game feels complete without some music and sound. In the examples folder, we've provided two audio tracks for you to use. The first is arpmedia-retro-arcade-game-music-577821.mp3 for background music. The second is GUNArtl_Rocket Launcher Fire_02.wav for a shooting sound effect.

#### Background music

Let's start with the background music. Declare the asset path and create an audio source entity before the runner, alongside the tank entities.

```go
const AssetMusic = "audio/arpmedia-retro-arcade-game-music-577821.mp3"

if _, err := g.World().NewEntity(audio.Source{
	Audio:  AssetMusic,
	Volume: 0.5,
	Group:  audio.GroupMusic,
	Loop:   audio.LoopForever,
	Load:   audio.LoadStream,
	Pause:  audio.PauseHolds,
}); err != nil {
	return err
}
```

The music starts on the first frame and loops continuously. `Volume: 0.5` plays it at half gain; `audio.LoadStream` avoids decoding a long track into memory, and `audio.PauseHolds` pauses it with the game.

The other audio settings are explained in the [audio guide](../guides/audio.md). Next, we'll add a sound effect that plays when you click the left mouse button.

#### Sound effects

First, preload the sound effect immediately after the existing `ebitrun.New(g)` block and before `g.Run(runner)`. This decodes the file during setup, so a missing or invalid asset fails before the first key press:

```go
const AssetSFX = "audio/GUNArtl_Rocket Launcher Fire_02.wav"

if _, err := g.World().MustResource[*asset.Server]().Load[asset.AudioData](AssetSFX); err != nil {
	return err
}
```

The preload must come after `ebitrun.New`, because the runner registers the audio decoders.

Now add a system next to `moveSystem`. It creates a one-shot play whenever the left mouse button is pressed:

```go
func sfxSystem() core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if !ctx.Input.MousePressed(input.MouseButtonLeft) {
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

Click the left mouse button. Each click plays the effect once; holding the button does not retrigger it until you release and click again. `audio.OneShot` creates the temporary audio entity and removes it when playback finishes.

The [audio guide](../guides/audio.md) covers volume controls, pause behavior, and the rest of the audio API. The [audio example](../../examples/audio) shows those controls in a complete program.

### A camera that follows the tank

The fixed view lets the tank drive off screen. Lets use the main camera to follow the hull. We'll zoom in so we can see the tank clearly while it moves around the field.

Set the camera's zoom once at startup, then recenter it every tick. Declare the zoom with the other constants:

```go
	CameraZoom = 1.5
```

Then set it, next to the system registrations. The zoom is a field on the camera's `Camera` component, so the write goes through the entity like any other component:

```go
	if err := g.MainCamera().Update(g.World(), func(c *render.Camera) { c.Zoom = CameraZoom }); err != nil {
		return err
	}
```

Add a system that copies the hull's position to the camera:

```go
func cameraSystem(camera, hull *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		return camera.Update(ctx.World, func(t *core.Transform) {
			t.Position = hullTransform.Position
		})
	})
}
```

```go
if err := g.AddSystem(core.PhaseFixed, "camera.follow", cameraSystem(g.MainCamera(), hull)); err != nil {
	return err
}
```

Register `camera.follow` after `player.turret` so it reads the hull's updated position. The renderer interpolates between ticks, smoothing the per-tick follow.

The turret's aim already works under the new view: `turretSystem` converts the cursor with the camera's own position and zoom, so the tank keeps shooting where you point no matter where it drives.

Drive away from the start. Try a few changes:

- Drive in any direction: the camera keeps the tank centered and the field scrolls around it.
- Set `CameraZoom` back to 1 and see the tank shrink to its old size - zoom scales the whole view, not one sprite.
- Zoom is a multiplier on world units: at 1.5, the tank's 100-pixel texture fills 150 pixels of the window.

### Firing the gun

The tank drives and aims, and a left-click plays a shot. Now make the left mouse button fire too: each click spawns a bullet at the barrel tip, and the bullet flies until it leaves view.

Add `collision` to the import block. Colliders only produce contacts when the engine's collision system is registered, and it is opt-in: add `castrum.WithCollision()` to the `castrum.New` call:

```go
	g, err := castrum.New(
		castrum.WithTitle("your first game"),
		castrum.WithBindings(bindings),
		castrum.WithCollision(),
	)
```

Then add the gun's constants - the bullet's speed and size, a cull margin, and the collision layers the bullets will live on:

```go
	BarrelLength = 64  // turret center to muzzle, world units
	BulletSpeed  = 800 // world units per second
	BulletRadius = 3   // world units
	CullMargin   = 100 // world units past the view a bullet may fly

	bulletLayer = 1
	enemyLayer  = 2
```

The `bullet` marker component tags the entities the bullet system owns, and a color paints them:

```go
type bullet struct{}
```

```go
var bulletColor = color.NRGBA{R: 245, G: 220, B: 130, A: 255}
```

Next, give the fire action a binding. Replace the raw mouse click check from the audio section with a named action, so the game reads its inputs through the bindings map. Add this entry to the bindings:

```go
		"fire": []input.Input{
			input.MouseButtonInput{Button: input.MouseButtonLeft},
		},
```

Now replace the `sfxSystem` function and its registration with the gun system. It plays the same shot and spawns a bullet at the barrel tip - the turret's position pushed out along its rotation by `BarrelLength`:

```go
func fireSystem(turret *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if !ctx.Actions.Pressed("fire") {
			return nil
		}
		transform, _ := turret.Component[core.Transform](ctx.World)
		tip := transform.Position.Add(geom.Vector2{X: 0, Y: -BarrelLength}.Rotate(transform.Rotation))
		if _, err := spawnBullet(ctx.World, tip, transform.Rotation); err != nil {
			return err
		}

		shot := audio.NewSource(AssetSFX)
		shot.Pause = audio.PauseContinues
		_, err := audio.OneShot(ctx.World, shot)
		return err
	})
}
```

One detail matters here: the fire system reads `Pressed`, not `JustPressed`. The system runs on fixed ticks, and `Pressed` is the tick-stable edge - the engine latches a press until the next tick consumes it, so a quick tap that lands between ticks still fires. `JustPressed` describes the current display frame instead, and a fixed system can miss it entirely. The [input guide](../guides/input.md) covers the distinction.

`spawnBullet` creates one live bullet: an entity with a position, a small circle to draw, and a collider. The collider sits on the bullet layer and listens to the enemy layer only - the targets arrive three sections from now, and it is this mask that will keep bullets from ever hitting the tank that fired them:

```go
func spawnBullet(world *core.World, at geom.Vector2, rotation float64) (*core.Entity, error) {
	collider, err := collision.NewCollider(collision.Circle{Radius: BulletRadius})
	if err != nil {
		return nil, err
	}
	collider.Layers = collision.Layers(bulletLayer)
	collider.Mask = collision.Mask(enemyLayer)
	return world.NewEntity(
		core.Transform{Position: at, Rotation: rotation, Scale: geom.Vector2{X: 1, Y: 1}},
		render.Sprite{
			Drawable: render.CircleShape{Radii: geom.Vector2{X: BulletRadius, Y: BulletRadius}},
			Color:    bulletColor,
		},
		collider,
		bullet{},
	)
}
```

The bullets themselves move in their own system. It queries the live bullets by their marker, moves each one along its facing, and destroys the ones that fly past what the camera can see. The camera arrives as a parameter, and the view's half extents are the logical size divided by the zoom:

```go
func bulletsSystem(camera *core.Entity) core.System {
	var live *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if live == nil {
			live = core.NewQuery(ctx.World).With(core.Transform{}, bullet{})
		}
		dt := ctx.DeltaTime.Seconds()

		cameraData, _ := camera.Component[render.Camera](ctx.World)
		cameraTransform, _ := camera.Component[core.Transform](ctx.World)
		halfW := float64(ctx.LogicalWidth) / (2 * cameraData.Zoom)
		halfH := float64(ctx.LogicalHeight) / (2 * cameraData.Zoom)

		for e := range live.Execute() {
			e.Update(func(t *core.Transform) {
				t.Position = t.Position.Add(geom.Vector2{X: 0, Y: -1}.Rotate(t.Rotation).Mul(BulletSpeed * dt))
			})

			p, _ := e.Component[core.Transform]()
			dx := math.Abs(p.Position.X - cameraTransform.Position.X)
			dy := math.Abs(p.Position.Y - cameraTransform.Position.Y)
			if dx > halfW+CullMargin || dy > halfH+CullMargin {
				if err := ctx.World.DestroyEntity(core.NewEntity(e.ID())); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
```

Register both systems where the `sfxSystem` registration used to be:

```go
if err := g.AddSystem(core.PhaseFixed, "player.fire", fireSystem(turret)); err != nil {
	return err
}
if err := g.AddSystem(core.PhaseFixed, "bullets", bulletsSystem(g.MainCamera())); err != nil {
	return err
}
```

Click the left mouse button. Bullets should stream from the barrel tip and vanish beyond the view. Try a few changes:

- Change `BulletSpeed` and `BarrelLength` and watch the stream tighten or spread.
- Fire while turning: each bullet keeps the rotation it was spawned with, because the transform is copied at spawn and never touches the turret again.

### A cooldown timer

A tank that fires as fast as you can press is a machine gun. Give the gun a cooldown: one shot, then a fixed reload before the next. The engine's timer component does the counting.

Add `timer` and `time` to the import block, then declare the cooldown alongside the gun constants:

```go
	FireCooldown = 750 * time.Millisecond
```

Timers advance only when the engine's timer system runs, and it is opt-in: add `castrum.WithTimer()` to the `castrum.New` call. Keep `castrum.WithCollision()`, which the bullet colliders already need. Without the timer option, `Timer` components sit frozen - the gun would never finish cooling down:

```go
	g, err := castrum.New(
		castrum.WithTitle("your first game"),
		castrum.WithBindings(bindings),
		castrum.WithCollision(),
		castrum.WithTimer(),
	)
```

The cooldown timer belongs to the turret - one timer per entity, and the turret is the gun. Replace the turret entity's spawn with this, which attaches the timer:

```go
	turret, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos},
		render.Sprite{Drawable: render.TextureSource{Texture: AssetTurret}, Layer: 1},
		timer.NewTimer(FireCooldown, false),
	)
	if err != nil {
		return err
	}
	if err := turret.Update(g.World(), func(t *timer.Timer) { t.Running = false }); err != nil {
		return err
	}
```

`NewTimer` starts timers running, so the spawn pauses it immediately. That one write sets up the whole state machine: a fresh paused timer and a completed one-shot both read `Running == false`, which means ready to fire. When the timer completes, it stops itself - `Running` is false again, and the gun is ready.

Now gate the gun on that state. Replace `fireSystem` with this version, which checks the timer before firing and restarts it after:

```go
func fireSystem(turret *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if !ctx.Actions.Pressed("fire") {
			return nil
		}
		gun, _ := turret.Component[timer.Timer](ctx.World)
		if gun.Running {
			return nil
		}

		transform, _ := turret.Component[core.Transform](ctx.World)
		tip := transform.Position.Add(geom.Vector2{X: 0, Y: -BarrelLength}.Rotate(transform.Rotation))
		if _, err := spawnBullet(ctx.World, tip, transform.Rotation); err != nil {
			return err
		}

		if err := turret.Update(ctx.World, func(t *timer.Timer) { t.Restart() }); err != nil {
			return err
		}

		shot := audio.NewSource(AssetSFX)
		shot.Pause = audio.PauseContinues
		_, err := audio.OneShot(ctx.World, shot)
		return err
	})
}
```

`Restart` resets the elapsed time, clears the completion, and runs the timer again - the same timer counts down every shot. The reload is 750 milliseconds of real time, and the timer advances on fixed ticks, so it pauses with the game for free.

Click the left mouse button after each reload to fire again. The [timers guide](../guides/timer.md) covers completion edges and repeating timers; the [timer example](../../examples/timer) shows them in a complete program.

### Adding enemies

Now give the bullets targets. Enemies will appear just outside the camera, then head for the tank. You will use the same timer as the reload, this time repeating to create a steady stream of enemies.

Add `math/rand/v2` to the import block. Declare the enemy's texture alongside the other asset constants, and the wave constants alongside the gun's:

```go
const AssetEnemy = "sprites/Panzer 4/ww2_top_view_hull2.png"
```

```go
	EnemySpeed  = 60 // world units per second
	EnemyRadius = 16 // world units
	SpawnEvery  = 1500 * time.Millisecond
```

Add an `enemy` marker and a helper that creates an enemy. Its collision mask accepts bullets, so shots can hit it:

```go
type enemy struct{}
```

```go
func spawnEnemy(world *core.World, at geom.Vector2) (*core.Entity, error) {
	collider, err := collision.NewCollider(collision.Circle{Radius: EnemyRadius})
	if err != nil {
		return nil, err
	}
	collider.Layers = collision.Layers(enemyLayer)
	collider.Mask = collision.Mask(bulletLayer)
	return world.NewEntity(
		core.Transform{Position: at, Scale: geom.Vector2{X: 1, Y: 1}},
		render.Sprite{Drawable: render.TextureSource{Texture: AssetEnemy}},
		collider,
		enemy{},
	)
}
```

Each enemy is a single hull sprite. That keeps this step focused on spawning and combat; the player turret is still the only independently moving turret.

Create the spawner next. Its timer starts immediately, so the first enemy appears after `SpawnEvery`:

```go
	spawner, err := g.World().NewEntity(
		timer.NewTimer(SpawnEvery, true),
	)
	if err != nil {
		return err
	}
```

Add the spawn system. When the timer completes, it chooses a point just beyond the visible screen and creates one enemy there:

```go
func spawnSystem(spawner, hull, camera *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		waves, _ := spawner.Component[timer.Timer](ctx.World)
		if !waves.JustCompleted(ctx.Tick) {
			return nil
		}

		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		cameraData, _ := camera.Component[render.Camera](ctx.World)
		halfW := float64(ctx.LogicalWidth) / (2 * cameraData.Zoom)
		halfH := float64(ctx.LogicalHeight) / (2 * cameraData.Zoom)

		angle := rand.Float64() * 2 * math.Pi
		radius := math.Hypot(halfW, halfH) + CullMargin
		at := hullTransform.Position.Add(geom.Vector2{
			X: math.Cos(angle) * radius,
			Y: math.Sin(angle) * radius,
		})
		_, err := spawnEnemy(ctx.World, at)
		return err
	})
}
```

Register the system and preload the texture with the other game assets:

```go
if err := g.AddSystem(core.PhaseFixed, "enemy.spawn", spawnSystem(spawner, hull, g.MainCamera())); err != nil {
	return err
}
```

```go
if _, err := g.World().MustResource[*asset.Server]().Load[asset.TextureData](AssetEnemy); err != nil {
	return err
}
```

Run the game. Every second and a half, an enemy should arrive from off-screen. Try a few changes:

- Change `SpawnEvery` and watch the pressure build or ease off.
- Drive in a circle: the spawn ring follows the tank, because the ring is placed around the hull's current position.

### Chasing enemies

The enemies are in the game, but they are not dangerous yet and your bullets fly straight through them, because the bullet system has no contact handling. Two additions make the fight real.

A bullet's mask admits enemies only, so its contacts name enemies by construction; an enemy's own contacts could never tell a bullet apart from the tank once both can touch it. Give the bullet loop a contact branch - add this at the top of the loop in `bulletsSystem`, before the movement update:

```go
			contacts, hasContacts := core.NewEntity(e.ID()).Component[collision.Contacts](ctx.World)
			if hasContacts && len(contacts.Current) > 0 {
				for _, hit := range contacts.Current {
					other := core.NewEntity(hit.Other)
					if _, ok := other.Component[enemy](ctx.World); !ok {
						continue // a simultaneous bullet already killed this enemy
					}
					if err := ctx.World.DestroyEntity(other); err != nil {
						return err
					}
				}
				if err := ctx.World.DestroyEntity(core.NewEntity(e.ID())); err != nil {
					return err
				}
				continue
			}
```

Now let's have the enemies chase the tank as it moves. Add the following system:

```go
func enemiesSystem(hull *core.Entity) core.System {
	var targets *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if targets == nil {
			targets = core.NewQuery(ctx.World).With(core.Transform{}, enemy{})
		}
		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		dt := ctx.DeltaTime.Seconds()

		for e := range targets.Execute() {
			e.Update(func(t *core.Transform) {
				aim := hullTransform.Position.Sub(t.Position)
				t.Rotation = math.Atan2(aim.X, -aim.Y)
				t.Position = t.Position.Add(aim.Normalize().Mul(EnemySpeed * dt))
			})
		}
		return nil
	})
}
```

```go
if err := g.AddSystem(core.PhaseFixed, "enemies", enemiesSystem(hull)); err != nil {
	return err
}
```

You do not need to call collision code anywhere. The collision system records hits for each entity, and the bullet branch reads them. The [collision guide](../guides/collision.md) explains the collision API when you are ready to explore it further.

Run it again. Enemies now converge from every side, and one bullet destroys each one. They are slower than the tank, so keep moving, turn, and shoot.

### The reload readout

Finish the combat feedback with a reload readout. While the gun reloads, the remaining time appears under the tank. It disappears when the gun is ready again.

Add `fmt` to the import block, and the font's path alongside the other asset constants - the bundle's `fonts/` directory holds a regular text font, and any ttf or otf file you drop in works the same:

```go
const AssetFont = "fonts/GoRegular.ttf"
```

The readout is one text sprite, spawned under the tank's start position. It begins empty and hidden:

```go
	cooldown, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos.Add(geom.Vector2{X: 0, Y: 72}), Scale: geom.Vector2{X: 1, Y: 1}},
		render.Sprite{Drawable: render.TextSource{Font: AssetFont, Text: "", Size: 18}},
	)
	if err != nil {
		return err
	}
```

Add this system. It follows the turret and updates the text while the reload timer is running:

```go
func reloadReadoutSystem(readout, turret *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		gun, _ := turret.Component[timer.Timer](ctx.World)
		turretTransform, _ := turret.Component[core.Transform](ctx.World)
		if err := readout.Update(ctx.World, func(t *core.Transform) {
			t.Position = turretTransform.Position.Add(geom.Vector2{X: 0, Y: 72})
		}); err != nil {
			return err
		}
		return readout.Update(ctx.World, func(s *render.Sprite) {
			if gun.Running {
				s.Hidden = false
				s.Drawable = render.TextSource{
					Font: AssetFont,
					Text: fmt.Sprintf("reloading %.1fs", (gun.Duration - gun.Elapsed).Seconds()),
					Size: 18,
				}
			} else {
				s.Hidden = true
			}
		})
	})
}
```

Register the system and preload the font next to the texture preloads - fonts and textures are different asset types, so each loads with its own:

```go
if err := g.AddSystem(core.PhaseFixed, "reload.readout", reloadReadoutSystem(cooldown, turret)); err != nil {
	return err
}
```

```go
if _, err := g.World().MustResource[*asset.Server]().Load[asset.FontData](AssetFont); err != nil {
	return err
}
```

Fire to see `reloading 0.7s` count down under the tank, then disappear. The [rendering guide](../guides/rendering.md) and [text example](../../examples/text) cover text styling and other ways to use it.

### Score, win, and loss

Give the fight an ending. Destroy `WinScore` enemies to win; if an enemy reaches the tank, the run ends. One `game` component will hold the score and the outcome.

First, replace the collision-layer constants from the gun section and add the scoring constants:

```go
	bulletLayer = 1
	enemyLayer  = 2
	playerLayer = 3

	TankRadius = 16 // world units, the hull's hit circle
```

```go
	SpawnShave    = 12 * time.Millisecond
	MinSpawnEvery = 300 * time.Millisecond

	WinScore = 100
```

Add the run state and create one entity to hold it:

```go
// game is the run's state: the kill count and the finished flags.
// One entity carries it, and systems read and write it like any
// other component.
type game struct {
	// Score counts enemy tanks destroyed by bullets.
	Score int
	// Won and Lost end the run: gameplay gates on them, and the
	// end banner reads them.
	Won  bool
	Lost bool
}
```

```go
	stats, err := g.World().NewEntity(game{})
	if err != nil {
		return err
	}
```

Replace the hull entity with this version so the tank can detect enemies that touch it:

```go
	playerCollider, err := collision.NewCollider(collision.Circle{Radius: TankRadius})
	if err != nil {
		return err
	}
	playerCollider.Layers = collision.Layers(playerLayer)
	playerCollider.Mask = collision.Mask(enemyLayer)
	hull, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos},
		render.Sprite{Drawable: render.TextureSource{Texture: AssetHull}},
		playerCollider,
	)
	if err != nil {
		return err
	}
```

Then replace `spawnEnemy` so enemies can collide with both bullets and the player:

```go
func spawnEnemy(world *core.World, at geom.Vector2) (*core.Entity, error) {
	collider, err := collision.NewCollider(collision.Circle{Radius: EnemyRadius})
	if err != nil {
		return nil, err
	}
	collider.Layers = collision.Layers(enemyLayer)
	collider.Mask = collision.Mask(bulletLayer, playerLayer)
	return world.NewEntity(
		core.Transform{Position: at, Scale: geom.Vector2{X: 1, Y: 1}},
		render.Sprite{Drawable: render.TextureSource{Texture: AssetEnemy}},
		collider,
		enemy{},
	)
}
```

Add this helper. Systems use it to stop gameplay once the run ends:

```go
func over(stats *core.Entity, ctx *core.Context) bool {
	run, _ := stats.Component[game](ctx.World)
	return run.Won || run.Lost
}
```

Update `moveSystem`, `fireSystem`, `bulletsSystem`, and `enemiesSystem` to accept `stats`. At the start of each `SystemFunc`, add this guard so the game stops responding after a win or loss:

```go
if over(stats, ctx) {
	return nil
}
```

Then use these registrations:

```go
if err := g.AddSystem(core.PhaseFixed, "player.hull.move", moveSystem(hull, stats)); err != nil {
	return err
}
```

```go
if err := g.AddSystem(core.PhaseFixed, "player.fire", fireSystem(turret, stats)); err != nil {
	return err
}
```

Each kill now scores. In `bulletsSystem`, inside the contact branch, immediately after the `DestroyEntity` that removes the enemy, add the score update - the score reaching `WinScore` wins the run:

```go
				if err := stats.Update(ctx.World, func(s *game) {
					s.Score++
					if s.Score >= WinScore {
						s.Won = true
					}
				}); err != nil {
					return err
				}
```

Make each score point increase the pace too. Replace `spawnSystem` with this version; it shortens the interval until it reaches `MinSpawnEvery`:

```go
func spawnSystem(spawner, hull, camera, stats *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if over(stats, ctx) {
			return nil
		}
		run, _ := stats.Component[game](ctx.World)
		interval := SpawnEvery - SpawnShave*time.Duration(run.Score)
		if interval < MinSpawnEvery {
			interval = MinSpawnEvery
		}
		if err := spawner.Update(ctx.World, func(t *timer.Timer) { t.Duration = interval }); err != nil {
			return err
		}

		waves, _ := spawner.Component[timer.Timer](ctx.World)
		if !waves.JustCompleted(ctx.Tick) {
			return nil
		}

		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		cameraData, _ := camera.Component[render.Camera](ctx.World)
		halfW := float64(ctx.LogicalWidth) / (2 * cameraData.Zoom)
		halfH := float64(ctx.LogicalHeight) / (2 * cameraData.Zoom)

		angle := rand.Float64() * 2 * math.Pi
		radius := math.Hypot(halfW, halfH) + CullMargin
		at := hullTransform.Position.Add(geom.Vector2{
			X: math.Cos(angle) * radius,
			Y: math.Sin(angle) * radius,
		})
		_, err := spawnEnemy(ctx.World, at)
		return err
	})
}
```

Add the loss check. It sets `Lost` as soon as an enemy touches the hull:

```go
func dangerSystem(hull, stats *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		contacts, has := hull.Component[collision.Contacts](ctx.World)
		if !has || len(contacts.Current) == 0 {
			return nil
		}
		return stats.Update(ctx.World, func(s *game) { s.Lost = true })
	})
}
```

Create a hidden text sprite for the result:

```go
	banner, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos},
		render.Sprite{Hidden: true, Layer: 10},
	)
	if err != nil {
		return err
	}
```

Add the result system. It shows `YOU WIN` or `GAME OVER` over the tank:

```go
func resultSystem(banner, hull, stats *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		run, _ := stats.Component[game](ctx.World)
		if !run.Won && !run.Lost {
			return banner.Update(ctx.World, func(s *render.Sprite) { s.Hidden = true })
		}
		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		if err := banner.Update(ctx.World, func(t *core.Transform) {
			t.Position = hullTransform.Position
		}); err != nil {
			return err
		}
		text := "GAME OVER"
		if run.Won {
			text = "YOU WIN"
		}
		return banner.Update(ctx.World, func(s *render.Sprite) {
			s.Hidden = false
			s.Drawable = render.TextSource{Font: AssetFont, Text: text, Size: 48}
		})
	})
}
```

Show the score in the top-left corner. Add the ebiten imports (`github.com/hajimehoshi/ebiten/v2` and its `ebitenutil`), then register this callback after `ebitrun.New`:

```go
	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		run, _ := stats.Component[game](ctx.World)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("score %d / %d", run.Score, WinScore), 16, 16)
		return nil
	})
```

Register the new systems. Replace the previous bullet, enemy, chase, and reload registrations with these:

```go
if err := g.AddSystem(core.PhaseFixed, "bullets", bulletsSystem(g.MainCamera(), stats)); err != nil {
	return err
}
if err := g.AddSystem(core.PhaseFixed, "enemy.spawn", spawnSystem(spawner, hull, g.MainCamera(), stats)); err != nil {
	return err
}
if err := g.AddSystem(core.PhaseFixed, "enemies", enemiesSystem(hull, stats)); err != nil {
	return err
}
if err := g.AddSystem(core.PhaseFixed, "game.danger", dangerSystem(hull, stats)); err != nil {
	return err
}
if err := g.AddSystem(core.PhaseFixed, "game.result", resultSystem(banner, hull, stats)); err != nil {
	return err
}
if err := g.AddSystem(core.PhaseFixed, "reload.readout", reloadReadoutSystem(cooldown, turret, stats)); err != nil {
	return err
}
```

Finally, update `reloadReadoutSystem` to accept `stats`. Start it with the same `over(stats, ctx)` check used by the movement and firing systems; when the run is over, hide the readout and return. The rest of the function stays the same.

```go
func reloadReadoutSystem(readout, turret, stats *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if over(stats, ctx) {
			return readout.Update(ctx.World, func(s *render.Sprite) { s.Hidden = true })
		}
		gun, _ := turret.Component[timer.Timer](ctx.World)
		turretTransform, _ := turret.Component[core.Transform](ctx.World)
		if err := readout.Update(ctx.World, func(t *core.Transform) {
			t.Position = turretTransform.Position.Add(geom.Vector2{X: 0, Y: 72})
		}); err != nil {
			return err
		}
		return readout.Update(ctx.World, func(s *render.Sprite) {
			if gun.Running {
				s.Hidden = false
				s.Drawable = render.TextSource{
					Font: AssetFont,
					Text: fmt.Sprintf("reloading %.1fs", (gun.Duration - gun.Elapsed).Seconds()),
					Size: 18,
				}
			} else {
				s.Hidden = true
			}
		})
	})
}
```

Run the game. The score climbs as enemies fall, waves arrive faster, and the run ends with a `YOU WIN` or `GAME OVER` banner. Try a few changes:

- Lower `WinScore` to reach a quick win.
- Adjust `SpawnShave` and `MinSpawnEvery` to make the final minutes easier or harder.
- Let an enemy reach the tank to check the loss screen too.

You now have a complete small game: the tank moves with the keyboard and aims at the mouse from a camera that follows it, the gun fires on a cooldown, enemies spawn out of sight, come faster as the score climbs, and end the run one way or the other - 100 kills for a win, one touch for a loss - while music and shot sounds play and the score, countdown, and result render as text.

## Next steps

Try changing the tank textures, the input bindings, or the wave pace first. When you are ready to add another feature, use the [guides](../guides) for rendering, input, collision, timers, animation, and audio, or browse the [examples](../../examples) for complete programs.

The [cheat sheet](cheat-sheet.md) is useful once you want to recall the patterns without walking through them again.
