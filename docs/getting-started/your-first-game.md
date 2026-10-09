# Your First Game

<!-- TUTORIAL, COMPLETE - a hand-held, step-by-step walk from empty
     directory to a small tank game. All sections are prose.

     VERIFICATION: both the engine/runner skeleton exactly as printed
     (bindings with placeholder comments, no entity yet) and the
     accumulated end-state program (fire bindings, hull and turret
     texture entities, drive system with the facing vector, aim
     system with camera math, music entity, camera zoom and follow,
     gun system with cooldown timer and bullet spawning, bullet
     system culling by view, spawner system over a repeating timer
     with ring placement, enemy chase system with destroy-on-hit,
     reload text readout, texture/font/audio preloads) build and
     vet against the current module - both verified in scratch
     modules, the end-state with the exact assets in
     examples/your-first-game. Keep every block verbatim-synced with
     the verified programs when editing.

     GROWTH (planned sections - add beats, not prose, until then):
     - UI
     - Enemy turrets, when entity hierarchies land
     The layout grows by inserting sections under "Adding more features". -->

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
- [Next steps](#next-steps)

## Before you start

Before you start, make sure you have Go 1.27 or later installed. On Windows, no C toolchain is required as the runner's dependencies ship pre-built.

Familiarity with the [introduction](intro.md) and [key concepts](concepts.md) chapters is recommended, as this tutorial builds on those concepts without re-teaching them.

In this tutorial, you'll build a tank game: drive the tank with the arrow keys or WASD, aim the turret at the mouse, and fire bullets on a cooldown - while Panzers spawn just out of sight and chase you down. Music and a shot sound effect play through the audio API, and the reload countdown renders as text.

Each section adds one feature, and every intermediate program runs successfully.

You also need the tutorial asset bundle: an `audio/` directory with a sound effect and a music track, a `sprites/` directory with textures for your tank and the enemy tanks, and a `fonts/` directory with a text font. From the [GitHub releases page](https://github.com/Leonard-Atorough/castrum/releases), download `castrum-tutorial-assets-<version>.zip` from the release matching the Castrum version you installed, then extract it into your project folder.

The code refers to them as `audio/...`, `sprites/...`, and `fonts/...`, so the paths line up with nothing extra to configure. The tank textures arrive in the next section; the audio files come into play in the audio section; the font arrives last, when the reload countdown becomes text.

You can use your own files instead, but the bundle is the set this tutorial can promise works.

## Setting up your environment

Start with an empty directory and a fresh module. Keep your shell in this project directory as you work:

```sh
mkdir firstgame && cd firstgame
go mod init firstgame
go get github.com/Leonard-Atorough/castrum
```

Create `main.go` in the project root. Extract the tutorial asset bundle there too, so the project contains `main.go`, `audio/`, `sprites/`, and `fonts/`.

By default, asset paths resolve from the process working directory. Run each command in this tutorial from the project root so paths such as `sprites/T-34/ww2_top_view_hull4.png` resolve to the extracted files.

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

The engine's optional subsystem systems are opt-in: `castrum.WithTimer()`, `castrum.WithCollision()`, and `castrum.WithAnimation()` register them, and `castrum.WithDefaultSystems()` registers all three. This game needs none of them yet - timers, collision, and animation enter through those options when a later section adds them.

Run it from the project root:

```sh
go run .
```

An empty window should open. Close it, then continue to create the tank. Use the same command after each later change to run the game again.

Add new imports when later sections use `core`, `geom`, `color`, `audio`, `asset`, `math`, `collision`, `timer`, `time`, `fmt`, or `math/rand/v2`.

## Creating the tank

The player is a tank: two entities sharing one position - a hull that handles movement, and a turret that rides on top of it and aims.

Add `core`, `geom`, and `asset` to the import block. Then declare the asset paths and the spawn point, and create both entities below `castrum.New`:

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
	render.Sprite{Drawable: render.TextureSource{Texture: AssetTurret}},
)
if err != nil {
	return err
}
```

Each entity pairs a `Transform` - where it is, in world units - with a `Sprite` - what it draws. Here the drawable is a `TextureSource`, which names a texture asset; the turret sits on the same position as the hull, and its system re-centers it every tick so the pair stays together.

Textures load lazily by default, but preloading them makes a missing or invalid file fail during setup, before the game ever runs. Preload both next to the entities:

```go
if _, err := g.AssetServer().Load[asset.TextureData](AssetHull); err != nil {
	return err
}
if _, err := g.AssetServer().Load[asset.TextureData](AssetTurret); err != nil {
	return err
}
```

Run the game again. The tank sits at the center of the window, turret riding the hull. Nothing moves yet - that is the next system.

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

Run the game again. W or the up arrow should move the tank forward, S or the down arrow should move it backward, and A/D or the left/right arrows should turn it. The system runs on fixed ticks, and `dt` keeps the speed consistent across different frame rates. The named actions let the system use `move_forward` without knowing which physical key produced them.

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
		cameraData, _ := camera.Component[core.Camera](ctx.World)
		cameraTransform, _ := camera.Component[core.Transform](ctx.World)

		cursor := core.CameraView{
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

Run the game and move the mouse around the window; the turret should follow it while the hull drives independently. Drive forward while aiming: the turret keeps pointing at the cursor no matter where the hull goes, because the aim is recomputed from the hull's position every tick. Compare this system with the [input example](../../examples/input), which drives the same camera helpers with QE zoom.

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

Run the game again. The music starts on the first frame and loops continuously. The volume is a plain multiplier: `0.5` plays the track at half gain, which suits a music track behind gameplay. `audio.LoadStream` is a good choice for a long music track because it does not decode the entire file into memory, and `audio.PauseHolds` makes the track pause with the game.

The other audio settings are explained in the [audio guide](../guides/audio.md). Next, we'll add a sound effect that plays when you press the spacebar.

#### Sound effects

First, preload the sound effect immediately after the existing `ebitrun.New(g)` block and before `g.Run(runner)`. This decodes the file during setup, so a missing or invalid asset fails before the first key press:

```go
const AssetSFX = "audio/GUNArtl_Rocket Launcher Fire_02.wav"

if _, err := g.World().MustResource[*asset.Server]().Load[asset.AudioData](AssetSFX); err != nil {
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

### A camera that follows the tank

The tank drives, but the fixed view lets it drive off the screen. Give the game a camera: zoomed in a little, and following the tank wherever it goes.

The engine spawns a default camera - zoom 1 at the world origin - and `g.MainCamera` returns it. Two writes put it to work: a zoom set once at startup, and a system that re-centers it every tick. Declare the zoom with the other constants:

```go
	// The camera: 50% closer than the default zoom of 1, and
	// following the tank so the field scrolls as it drives.
	CameraZoom = 1.5
```

Then set it, next to the system registrations. The zoom is a field on the camera's `Camera` component, so the write goes through the entity like any other component:

```go
	// The camera starts zoomed in; the follow system keeps it on
	// the tank from the first tick.
	if err := g.MainCamera().Update(g.World(), func(c *core.Camera) { c.Zoom = CameraZoom }); err != nil {
		return err
	}
```

The follow system is the same shape as the turret's mount: read the hull, write the camera. Add it next to `turretSystem`:

```go
// cameraSystem keeps the camera centered on the hull. The view
// follows the tank wherever it drives; the renderer interpolates
// the camera like any other transform, so the scrolling is smooth.
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

Register `camera.follow` after `player.turret`. The camera reads the hull's position after the systems that write it, and the renderer interpolates the camera between ticks like any other transform - the scrolling is smooth even though the follow snaps every tick.

The turret's aim already works under the new view: `turretSystem` converts the cursor with the camera's own position and zoom, so the tank keeps shooting where you point no matter where it drives.

Run the game and drive away from the start. Try a few changes:

- Drive in any direction: the camera keeps the tank centered and the field scrolls around it.
- Set `CameraZoom` back to 1 and see the tank shrink to its old size - zoom scales the whole view, not one sprite.
- Zoom is a multiplier on world units: at 1.5, the tank's 100-pixel texture fills 150 pixels of the window.

### Firing the gun

The tank drives and aims, and the spacebar plays a shot. Now make the spacebar actually fire: each press spawns a bullet at the barrel tip, and the bullet flies until it leaves view.

Add `collision` and `color` to the import block. Then add the gun's constants - the bullet's speed and size, a cull margin, and the collision layers the bullets will live on:

```go
	// The gun: one shot per cooldown, spawned at the barrel tip.
	BarrelLength = 64  // turret center to muzzle, world units
	BulletSpeed  = 800 // world units per second
	BulletRadius = 3   // world units
	CullMargin   = 100 // world units past the view a bullet may fly

	// Collision layers: the masks below decide which pairs ever
	// reach the narrow phase - bullets admit enemies and nothing
	// else, so the tank cannot shoot itself.
	bulletLayer = 1
	enemyLayer  = 2
```

The `bullet` marker component tags the entities the bullet system owns, and a color paints them:

```go
// bullet marks the entities the bullet system owns: fired by the
// tank, flying straight, destroyed on impact or on leaving view.
type bullet struct{}
```

```go
// bulletColor paints the bullets; the tanks keep their textures.
var bulletColor = color.NRGBA{R: 245, G: 220, B: 130, A: 255}
```

Next, give the fire key a binding - the raw spacebar check from the audio section becomes a named action, so every key the game reads lives in the bindings map. Add this entry to the bindings:

```go
		"fire": []input.Input{
			input.KeyInput{Key: input.KeySpace},
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
// spawnBullet creates one live bullet at a point, flying along the
// given rotation. A bullet is its own entity on the bullet layer,
// listening for enemies only, so the tank that fired it can never
// shoot itself.
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
// bulletsSystem moves every live bullet along its facing and
// destroys it on impact - its Contacts hold an enemy - or when it
// flies past what the camera can see.
func bulletsSystem(camera *core.Entity) core.System {
	var live *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if live == nil {
			live = core.NewQuery(ctx.World).With(core.Transform{}, bullet{})
		}
		dt := ctx.DeltaTime.Seconds()

		cameraData, _ := camera.Component[core.Camera](ctx.World)
		cameraTransform, _ := camera.Component[core.Transform](ctx.World)
		halfW := float64(ctx.LogicalWidth) / (2 * cameraData.Zoom)
		halfH := float64(ctx.LogicalHeight) / (2 * cameraData.Zoom)

		for e := range live.Execute() {
			contacts, hasContacts := e.Component[collision.Contacts]()
			if hasContacts && len(contacts.Current) > 0 {
				if err := ctx.World.DestroyEntity(core.NewEntity(e.ID())); err != nil {
					return err
				}
				continue
			}

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

Run the game and press the spacebar. Bullets stream out of the barrel tip and vanish once they leave the view. Try a few changes:

- Change `BulletSpeed` and `BarrelLength` and watch the stream tighten or spread.
- Fire while turning: each bullet keeps the rotation it was spawned with, because the transform is copied at spawn and never touches the turret again.

### A cooldown timer

A tank that fires as fast as you can press is a machine gun. Give the gun a cooldown: one shot, then a fixed reload before the next. The engine's timer component does the counting.

Add `timer` and `time` to the import block, then declare the cooldown alongside the gun constants:

```go
	FireCooldown = 750 * time.Millisecond
```

The cooldown timer belongs to the turret - one timer per entity, and the turret is the gun. Replace the turret entity's spawn with this, which attaches the timer:

```go
	// The turret carries the gun's cooldown timer - one timer per
	// entity, and the turret is the gun. It spawns paused: a fresh
	// timer and a completed one both read as ready to fire.
	turret, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos},
		render.Sprite{Drawable: render.TextureSource{Texture: AssetTurret}},
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
// fireSystem fires the gun: when the fire key is pressed and the
// cooldown is not running, a bullet spawns at the barrel tip - the
// turret center pushed out along the turret's rotation - and the
// cooldown timer restarts, clearing its completion so it can count
// down again. Fixed systems read Pressed, the tick-stable edge:
// a tap that lands between ticks still fires.
func fireSystem(turret *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if !ctx.Actions.Pressed("fire") {
			return nil
		}
		gun, _ := turret.Component[timer.Timer](ctx.World)
		if gun.Running {
			return nil // still reloading
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

Run the game and hold the spacebar: one shot per reload, on rhythm. The [timers guide](../guides/timer.md) covers completion edges, repeating timers, and the rest of the timer API; the [timer example](../../examples/timer) shows the contracts in a complete program.

### Adding enemies

Bullets that vanish off the field are practice. Add the other half of the fight: Panzers that appear just beyond what the camera can see and hunt the tank. They arrive on a schedule - and a schedule is a repeating timer, the same component the cooldown uses in its one-shot form.

Add `math/rand/v2` to the import block. Declare the enemy's texture alongside the other asset constants, and the wave constants alongside the gun's:

```go
const AssetEnemy = "sprites/Panzer 4/ww2_top_view_hull2.png"
```

```go
	// The enemies: slower than the tank, so a straight retreat
	// always opens the gap.
	EnemySpeed  = 60 // world units per second
	EnemyRadius = 32 // world units
	SpawnEvery  = 1500 * time.Millisecond
```

The `enemy` marker component tags them for their system, and `spawnEnemy` creates one, with the same collider pattern as the bullets, mirrored - the enemy sits on the enemy layer and listens to the bullet layer. Both sides must carry the other's layer for the pair to ever meet:

```go
// enemy marks the chasing tanks: spawned outside the view, they
// drive at the player until a bullet finds them.
type enemy struct{}
```

```go
// spawnEnemy creates one enemy tank at a point, on the enemy
// layer, listening for bullets only. An enemy is a single hull
// entity: giving it a turret of its own would need an entity
// hierarchy, which the engine does not have yet.
func spawnEnemy(world *core.World, at geom.Vector2) (*core.Entity, error) {
	collider, err := collision.NewCollider(collision.CircleShape{Radius: EnemyRadius})
	if err != nil {
		return nil, err
	}
	collider.Layers = collision.Layers(enemyLayer)
	collider.Mask = collision.Mask(bulletLayer)
	return world.NewEntity(
		core.Transform{Position: at, Scale: geom.Vector2{X: 1, Y: 1}},
		core.Sprite{Drawable: core.TextureSource{Texture: AssetEnemy}},
		collider,
		enemy{},
	)
}
```

An enemy is a single hull entity, with no turret sprite of its own: attaching one entity to another needs an entity hierarchy, which the engine does not have yet. When hierarchies land, a turret can ride each enemy hull the same way the player's turret system rides the player's hull. For now the hull texture alone reads as the tank.

The spawner is one entity carrying a repeating timer. `NewTimer` starts it running, so the first wave is one interval away:

```go
	// The spawner: a repeating timer whose completions are the
	// waves. NewTimer starts it running immediately.
	spawner, err := g.World().NewEntity(
		timer.NewTimer(SpawnEvery, true),
	)
	if err != nil {
		return err
	}
```

The spawn system reads the timer's completion edge and places one enemy just outside the view. The ring math is the view's half extents again: the half-diagonal plus the cull margin is guaranteed off-screen in every direction, and a random angle picks the point on it:

```go
// spawnSystem keeps the waves coming: every time the spawner's
// repeating timer completes, one enemy appears just outside what
// the camera can see, at a random angle around the tank.
func spawnSystem(spawner, hull, camera *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		waves, _ := spawner.Component[timer.Timer](ctx.World)
		if !waves.JustCompleted(ctx.Tick) {
			return nil
		}

		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		cameraData, _ := camera.Component[core.Camera](ctx.World)
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

`JustCompleted` reads true on exactly the tick the timer finishes - the completion edge - and the repeating timer starts counting again on its own, carrying its overshoot into the next interval so the waves never drift.

Register the system, and preload the enemy's texture next to the player's, so a missing or invalid file fails during setup:

```go
if err := g.AddSystem(core.PhaseFixed, "enemy.spawn", spawnSystem(spawner, hull, g.MainCamera())); err != nil {
	return err
}
```

```go
if _, err := g.AssetServer().Load[asset.TextureData](AssetEnemy); err != nil {
	return err
}
```

Run the game and drive: every second and a half a Panzer appears - always just outside your view, whatever direction you drive. Try a few changes:

- Change `SpawnEvery` and watch the pressure build or ease off.
- Drive in a circle: the spawn ring follows the tank, because the ring is placed around the hull's current position.

### Chasing enemies

Spawned enemies do nothing yet - the chase system gives them their behavior. It drives every enemy toward the tank, and removes the ones a bullet found. A current contact means the hit already happened - the bullet system is destroying the other half of the pair in the same tick:

```go
// enemiesSystem drives every enemy toward the tank and removes the
// ones a bullet found. A current contact means the hit already
// happened - the bullet system is destroying the other half of the
// pair in the same tick.
func enemiesSystem(hull *core.Entity) core.System {
	var targets *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if targets == nil {
			targets = core.NewQuery(ctx.World).With(core.Transform{}, collision.Contacts{}, enemy{})
		}
		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		dt := ctx.DeltaTime.Seconds()

		for e := range targets.Execute() {
			contacts, _ := e.Component[collision.Contacts]()
			if len(contacts.Current) > 0 {
				if err := ctx.World.DestroyEntity(core.NewEntity(e.ID())); err != nil {
					return err
				}
				continue
			}
			e.Update(func(t *core.Transform) {
				aim := hullTransform.Position.Sub(t.Position)
				t.Rotation = math.Atan2(aim.X, -aim.Y)
				t.Position = t.Position.Add(aim.Normalize().Mul(EnemySpeed * dt))
			})
		}
		return nil
		cameraData, _ := camera.Component[render.Camera](ctx.World)
		cameraTransform, _ := camera.Component[core.Transform](ctx.World)

		cursor := render.CameraView{
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
```

```go
if err := g.AddSystem(core.PhaseFixed, "enemies", enemiesSystem(hull)); err != nil {
	return err
}
```

The engine's collision system runs on its own every fixed tick and writes each collider's contacts onto the entity - nothing in this game calls any collision function. The chase system only reads that state. The masks are the whole targeting rule - a bullet that grazes your own hull passes right through it, because no bullet-hull pair ever reaches a collision test. The [collision guide](../guides/collision.md) covers shapes, layers, masks, and the contacts lifecycle.

Run the game: Panzers converge on the tank from every side, and one bullet each puts them down. The enemies are slower than the tank, so a straight retreat always opens the gap - kite, turn, and shoot.

### The reload readout

The last feature is a readout: while the gun reloads, the remaining time appears under the tank; when the gun is ready, the text is gone. Text in Castrum is a drawable like any other - it lives on a `Sprite`, so it culls, sorts, and moves with the entity.

Add `fmt` to the import block, and the font's path alongside the other asset constants - the bundle's `fonts/` directory holds a regular text font, and any ttf or otf file you drop in works the same:

```go
const AssetFont = "fonts/GoRegular.ttf"
```

The readout is one text sprite, spawned under the tank's start position. It begins empty and hidden:

```go
	// The reload readout: a text sprite under the tank, rewritten
	// every tick while the gun cools and hidden while it is ready.
	cooldown, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos.Add(geom.Vector2{X: 0, Y: 72}), Scale: geom.Vector2{X: 1, Y: 1}},
		core.Sprite{Drawable: core.TextSource{Font: AssetFont, Text: "", Size: 18}},
	)
	if err != nil {
		return err
	}
```

One system writes it, reading the same timer the gun gates on. The first write pins the text under the turret - the turret carries the gun, so the readout follows it wherever the tank drives. While the gun cools the text shows the remaining time; when the timer is idle - fresh or completed - the sprite hides:

```go
// reloadReadoutSystem writes the cooldown countdown as text: while
// the gun is reloading the text shows the remaining time under the
// turret, and when the timer is idle the text hides - the timer's
// Running field is the whole state machine.
func reloadReadoutSystem(readout, turret *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		gun, _ := turret.Component[timer.Timer](ctx.World)
		turretTransform, _ := turret.Component[core.Transform](ctx.World)
		if err := readout.Update(ctx.World, func(t *core.Transform) {
			t.Position = turretTransform.Position.Add(geom.Vector2{X: 0, Y: 72})
		}); err != nil {
			return err
		}
		return readout.Update(ctx.World, func(s *core.Sprite) {
			if gun.Running {
				s.Hidden = false
				s.Drawable = core.TextSource{
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

Run the game and fire: `reloading 0.7s` counts down under the tank and disappears the moment the gun is ready. A nil sprite color draws white text - the one drawable whose default is not black. The [rendering guide](../guides/rendering.md) covers text alongside the other drawables; the [text example](../../examples/text) shows fonts and text in a complete program.

You now have a small tank game: the tank moves with the keyboard and aims at the mouse from a camera that follows it, the gun fires on a cooldown, Panzers spawn out of sight and chase you down until a bullet finds them, music and shot sounds play through the audio API, and the reload countdown renders as text.

## Next steps

Try changing the tank textures, the input bindings, or the wave pace first. When you are ready to add another feature, use the [guides](../guides) for rendering, input, collision, timers, animation, and audio, or browse the [examples](../../examples) for complete programs.

The [cheat sheet](cheat-sheet.md) is useful once you want to recall the patterns without walking through them again.
