// Command your-first-game is the companion program to the getting
// started tutorial: a tank that drives with the keyboard and aims
// its turret at the mouse, from a camera that follows it. The gun
// fires on a cooldown; Panzers spawn just out of sight and chase
// the tank until a bullet finds them. Music and the shot sound
// effect play through the audio API, and the reload countdown
// renders as text.
//
// Run from the repository root:
//
//	go run ./examples/your-first-game
package main

import (
	"embed"
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/timer"
)

//go:embed sprites audio fonts
var files embed.FS

const (
	AssetMusic  = "audio/arpmedia-retro-arcade-game-music-577821.mp3"
	AssetSFX    = "audio/GUNArtl_Rocket Launcher Fire_02.wav"
	AssetHull   = "sprites/T-34/ww2_top_view_hull4.png"
	AssetTurret = "sprites/T-34/ww2_top_view_turret4.png"
	AssetEnemy  = "sprites/Panzer 4/ww2_top_view_hull2.png"
	AssetFont   = "fonts/GoRegular.ttf"

	Speed     = 100 // world units per second
	TurnSpeed = 3   // radians per second

	// The camera: 50% closer than the default zoom of 1, and
	// following the tank so the field scrolls as it drives.
	CameraZoom = 1.5

	// The gun: one shot per cooldown, spawned at the barrel tip.
	FireCooldown = 750 * time.Millisecond
	BarrelLength = 64  // turret center to muzzle, world units
	BulletSpeed  = 800 // world units per second
	BulletRadius = 3   // world units
	CullMargin   = 100 // world units past the view a bullet may fly

	// Collision layers: the masks below decide which pairs ever
	// reach the narrow phase - bullets admit enemies and nothing
	// else, so the tank cannot shoot itself.
	bulletLayer = 1
	enemyLayer  = 2

	// The enemies: slower than the tank, so a straight retreat
	// always opens the gap.
	EnemySpeed  = 60 // world units per second
	EnemyRadius = 32 // world units
	SpawnEvery  = 1500 * time.Millisecond
)

var SpawnPos = geom.Vector2{X: 640, Y: 360}

// bullet marks the entities the bullet system owns: fired by the
// tank, flying straight, destroyed on impact or on leaving view.
type bullet struct{}

// enemy marks the chasing tanks: spawned outside the view, they
// drive at the player until a bullet finds them.
type enemy struct{}

// bulletColor paints the bullets; the tanks keep their textures.
var bulletColor = color.NRGBA{R: 245, G: 220, B: 130, A: 255}

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
		"fire": []input.Input{
			input.KeyInput{Key: input.KeySpace},
		},
	}

	g, err := castrum.New(
		castrum.WithTitle("your first game"),
		castrum.WithBindings(bindings),
		castrum.WithFilesystem(files),
	)
	if err != nil {
		return err
	}

	// The music entity from the tutorial: a looping stream source.
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

	hull, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos},
		core.Sprite{Drawable: core.TextureSource{Texture: AssetHull}},
	)
	if err != nil {
		return err
	}

	// The turret carries the gun's cooldown timer - one timer per
	// entity, and the turret is the gun. It spawns paused: a fresh
	// timer and a completed one both read as ready to fire.
	turret, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos},
		core.Sprite{Drawable: core.TextureSource{Texture: AssetTurret}},
		timer.NewTimer(FireCooldown, false),
	)
	if err != nil {
		return err
	}
	if err := turret.Update(g.World(), func(t *timer.Timer) { t.Running = false }); err != nil {
		return err
	}

	// The spawner: a repeating timer whose completions are the
	// waves. NewTimer starts it running immediately.
	spawner, err := g.World().NewEntity(
		timer.NewTimer(SpawnEvery, true),
	)
	if err != nil {
		return err
	}

	// The reload readout: a text sprite under the tank, rewritten
	// every tick while the gun cools and hidden while it is ready.
	cooldown, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos.Add(geom.Vector2{X: 0, Y: 72}), Scale: geom.Vector2{X: 1, Y: 1}},
		core.Sprite{Drawable: core.TextSource{Font: AssetFont, Text: "", Size: 18}},
	)
	if err != nil {
		return err
	}

	// Preload every asset with a path, so a missing or invalid file
	// fails during setup rather than mid-game. Textures and fonts
	// are different asset types: load each with its own.
	if _, err := g.AssetServer().Load[asset.TextureData](AssetHull); err != nil {
		return err
	}
	if _, err := g.AssetServer().Load[asset.TextureData](AssetTurret); err != nil {
		return err
	}
	if _, err := g.AssetServer().Load[asset.TextureData](AssetEnemy); err != nil {
		return err
	}
	if _, err := g.AssetServer().Load[asset.FontData](AssetFont); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}
	if _, err := g.AssetServer().Load[asset.AudioData](AssetSFX); err != nil {
		return err
	}

	// The camera starts zoomed in; the follow system keeps it on
	// the tank from the first tick.
	if err := g.MainCamera().Update(g.World(), func(c *core.Camera) { c.Zoom = CameraZoom }); err != nil {
		return err
	}

	if err := g.AddSystem(core.PhaseFixed, "player.hull.move", moveSystem(hull)); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "player.turret", turretSystem(turret, hull, g.MainCamera())); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "camera.follow", cameraSystem(g.MainCamera(), hull)); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "player.fire", fireSystem(turret)); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "bullets", bulletsSystem(g.MainCamera())); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "enemy.spawn", spawnSystem(spawner, hull, g.MainCamera())); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "enemies", enemiesSystem(hull)); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "reload.readout", reloadReadoutSystem(cooldown, turret)); err != nil {
		return err
	}

	return g.Run(runner)
}

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

// spawnBullet creates one live bullet at a point, flying along the
// given rotation. A bullet is its own entity on the bullet layer,
// listening for enemies only, so the tank that fired it can never
// shoot itself.
func spawnBullet(world *core.World, at geom.Vector2, rotation float64) (*core.Entity, error) {
	collider, err := collision.NewCollider(collision.CircleShape{Radius: BulletRadius})
	if err != nil {
		return nil, err
	}
	collider.Layers = collision.Layers(bulletLayer)
	collider.Mask = collision.Mask(enemyLayer)
	return world.NewEntity(
		core.Transform{Position: at, Rotation: rotation, Scale: geom.Vector2{X: 1, Y: 1}},
		core.Sprite{
			Drawable: core.CircleShape{Radii: geom.Vector2{X: BulletRadius, Y: BulletRadius}},
			Color:    bulletColor,
		},
		collider,
		bullet{},
	)
}

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
	})
}

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
