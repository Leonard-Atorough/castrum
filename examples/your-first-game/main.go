// Command your-first-game is the companion program to the getting-started
// tutorial. Drive the tank with the keyboard, aim with the mouse, and survive
// waves of Panzers that speed up as the score rises. Destroy 100 to win; a
// collision with an enemy ends the run. The game also demonstrates audio,
// timers, collision, and camera-following text.
//
// Run from the repository root:
//
//	go run ./examples/your-first-game
package main

import (
	"embed"
	"errors"
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/builtin"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/render"
	"github.com/Leonard-Atorough/castrum/timer"
)

// This example embeds its assets; the tutorial loads them from a downloaded
// bundle in the working directory. See docs/guides/assets.md.
//
//go:embed sprites audio
var files embed.FS

const (
	AssetMusic  = "audio/arpmedia-retro-arcade-game-music-577821.mp3"
	AssetSFX    = "audio/GUNArtl_Rocket Launcher Fire_02.wav"
	AssetHull   = "sprites/T-34/ww2_top_view_hull4.png"
	AssetTurret = "sprites/T-34/ww2_top_view_turret4.png"
	AssetEnemy  = "sprites/Panzer 4/ww2_top_view_hull2.png"
	// AssetFont is the font path used by the game's text sprites.
	AssetFont = builtin.Font

	Speed     = 100 // world units per second
	TurnSpeed = 3   // radians per second

	// The tile gives the tank a fixed landmark as the camera follows.
	TileSize = 160 // world units per side

	// The camera follows the tank at 1.5 times the default zoom.
	CameraZoom = 1.5

	// The gun fires once per cooldown, from the barrel tip.
	FireCooldown = 750 * time.Millisecond
	BarrelLength = 64  // turret center to muzzle, world units
	BulletSpeed  = 800 // world units per second
	BulletRadius = 3   // world units
	CullMargin   = 100 // world units past the view a bullet may fly

	// Collision masks let bullets hit enemies and enemies hit bullets
	// or the player, while excluding other pairs.
	bulletLayer = 1
	enemyLayer  = 2
	playerLayer = 3

	// The hit circle matches the visible hull, not its larger texture.
	TankRadius = 16 // world units, the hull's hit circle

	// Enemies move slower than the tank, so retreating opens distance.
	EnemySpeed  = 60 // world units per second
	EnemyRadius = 16 // world units
	SpawnEvery  = 1500 * time.Millisecond

	// Each kill shortens the spawn interval, down to MinSpawnEvery.
	SpawnShave    = 12 * time.Millisecond
	MinSpawnEvery = 300 * time.Millisecond

	// Destroy WinScore enemies to win.
	WinScore = 100
)

var SpawnPos = geom.Vector2{X: 640, Y: 360}

// bullet marks projectile entities owned by bulletsSystem.
type bullet struct{}

// enemy marks tanks that spawn outside the view and chase the player.
type enemy struct{}

// game stores the run's score and outcome on one entity.
type game struct {
	// Score is the number of enemies destroyed by bullets.
	Score int
	// Won and Lost record the outcome and stop gameplay.
	Won  bool
	Lost bool
}

// bulletColor is the fill color for bullet circles.
var bulletColor = color.NRGBA{R: 245, G: 220, B: 130, A: 255}

// tileColor is the fill color for the start tile.
var tileColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}

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
			input.MouseButtonInput{Button: input.MouseButtonLeft},
		},
	}

	g, err := castrum.New(
		castrum.WithTitle("your first game"),
		castrum.WithBindings(bindings),
		castrum.WithFilesystem(files),
		castrum.WithTimer(),
		castrum.WithCollision(),
	)
	if err != nil {
		return err
	}

	// Add the tutorial's looping background music as a streamed source.
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

	// A fixed landmark makes the camera's movement visible as the tank drives.
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

	// The hull listens only for enemy contacts; dangerSystem uses them to
	// detect when an enemy reaches the player.
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

	// The turret is the gun, so its cooldown timer lives on the turret.
	// Start it paused: a stopped timer means the gun is ready to fire.
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

	// Each completion of this running repeating timer spawns a wave.
	spawner, err := g.World().NewEntity(
		timer.NewTimer(SpawnEvery, true),
	)
	if err != nil {
		return err
	}

	// Keep the score and run outcome together on one entity.
	stats, err := g.World().NewEntity(game{})
	if err != nil {
		return err
	}

	// Show the remaining cooldown below the turret; hide the readout when
	// the gun is ready.
	cooldown, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos.Add(geom.Vector2{X: 0, Y: 72}), Scale: geom.Vector2{X: 1, Y: 1}},
		render.Sprite{Drawable: render.TextSource{Font: AssetFont, Text: "", Size: 18}},
	)
	if err != nil {
		return err
	}

	// Keep the result banner hidden until the run ends, then draw it above
	// the game world.
	banner, err := g.World().NewEntity(
		core.Transform{Position: SpawnPos},
		render.Sprite{Hidden: true, Layer: 10},
	)
	if err != nil {
		return err
	}

	// Load assets during setup so missing or invalid files fail before play.
	// Textures and fonts use different asset types.
	if _, err := g.World().MustResource[*asset.Loader]().Load[asset.TextureData](AssetHull); err != nil {
		return err
	}
	if _, err := g.World().MustResource[*asset.Loader]().Load[asset.TextureData](AssetTurret); err != nil {
		return err
	}
	if _, err := g.World().MustResource[*asset.Loader]().Load[asset.TextureData](AssetEnemy); err != nil {
		return err
	}
	if err := builtin.Register(g.World().MustResource[*asset.Loader]()); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}
	if _, err := g.World().MustResource[*asset.Loader]().Load[asset.AudioData](AssetSFX); err != nil {
		return err
	}

	// Set the camera zoom before the follow system runs.
	if err := g.MainCamera().Update(g.World(), func(c *core.Transform) {
		c.Scale = geom.Vector2{X: CameraZoom, Y: CameraZoom}
	}); err != nil {
		return err
	}

	// Use a world-space text sprite, repositioned each tick to stay at the
	// camera's top-left as it follows the tank.
	scoreHUD, err := g.World().NewEntity(
		core.Transform{},
		render.Sprite{Drawable: render.TextSource{Font: AssetFont, Text: "", Size: 20}},
	)
	if err != nil {
		return err
	}

	if err := g.AddSystem(core.PhaseFixed, "player.hull.move", moveSystem(hull, stats)); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "player.turret", turretSystem(turret, hull)); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "camera.follow", cameraSystem(g.MainCamera(), hull)); err != nil {
		return err
	}
	// Run after camera.follow so the HUD uses the updated camera position.
	if err := g.AddSystem(core.PhaseFixed, "hud.score", scoreSystem(scoreHUD, g.MainCamera(), stats)); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "player.fire", fireSystem(turret, stats)); err != nil {
		return err
	}
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

	return g.Run(runner)
}

// over reports whether the run has reached a win or loss state.
func over(stats *core.Entity, ctx *core.Context) bool {
	run, _ := stats.Component[game](ctx.World)
	return run.Won || run.Lost
}

func moveSystem(hull, stats *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if over(stats, ctx) {
			return nil
		}
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

func turretSystem(turret, hull *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		view, err := ctx.World.Resource[*render.CameraView]()
		if errors.Is(err, core.ErrResourceNotRegistered) {
			// The first draw has not published a view yet, so skip aiming.
			return nil
		}
		if err != nil {
			return err
		}

		cursor := view.ScreenToWorld(ctx.Input.Cursor(), ctx.LogicalWidth, ctx.LogicalHeight)
		aim := cursor.Sub(hullTransform.Position)

		return turret.Update(ctx.World, func(t *core.Transform) {
			t.Position = hullTransform.Position
			t.Rotation = math.Atan2(aim.X, -aim.Y)
		})
	})
}

func scoreSystem(hud, camera, stats *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		run, _ := stats.Component[game](ctx.World)
		cameraTransform, _ := camera.Component[core.Transform](ctx.World)
		halfW := float64(ctx.LogicalWidth) / (2 * cameraTransform.Scale.X)
		halfH := float64(ctx.LogicalHeight) / (2 * cameraTransform.Scale.Y)
		if err := hud.Update(ctx.World, func(t *core.Transform) {
			t.Position = cameraTransform.Position.
				Sub(geom.Vector2{X: halfW, Y: halfH}).
				Add(geom.Vector2{X: 80, Y: 24})
		}); err != nil {
			return err
		}
		return hud.Update(ctx.World, func(s *render.Sprite) {
			s.Drawable = render.TextSource{
				Font: AssetFont,
				Text: fmt.Sprintf("score %d / %d", run.Score, WinScore),
				Size: 20,
			}
		})
	})
}

// cameraSystem centers the camera on the hull. The renderer interpolates
// its transform to smooth the view between fixed ticks.
func cameraSystem(camera, hull *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		hullTransform, _ := hull.Component[core.Transform](ctx.World)
		return camera.Update(ctx.World, func(t *core.Transform) {
			t.Position = hullTransform.Position
		})
	})
}

// fireSystem fires on a tick-latched press when the cooldown is idle. It
// spawns a bullet at the muzzle and restarts the cooldown. Pressed preserves
// taps that occur between fixed ticks.
func fireSystem(turret, stats *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if over(stats, ctx) {
			return nil
		}
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

// spawnBullet creates a projectile at at with the given rotation. Its
// collision mask includes enemies only, so it cannot hit the player.
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

// bulletsSystem moves projectiles and destroys them on impact or beyond the
// camera view. Bullets contact enemies only, so each hit can be scored as a
// kill. Process hits from the bullet side: an enemy's contacts also include
// the player, and scoring those would reward a ramming enemy.
func bulletsSystem(camera, stats *core.Entity) core.System {
	var live *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if live == nil {
			live = core.NewQuery(ctx.World).With(core.Transform{}, bullet{})
		}
		if over(stats, ctx) {
			return nil
		}
		dt := ctx.DeltaTime.Seconds()

		cameraTransform, _ := camera.Component[core.Transform](ctx.World)
		halfW := float64(ctx.LogicalWidth) / (2 * cameraTransform.Scale.X)
		halfH := float64(ctx.LogicalHeight) / (2 * cameraTransform.Scale.Y)

		for e := range live.Execute() {
			// The query only prefetches Transform and bullet; fetch Contacts
			// from the world through the entity ID.
			contacts, hasContacts := core.NewEntity(e.ID()).Component[collision.Contacts](ctx.World)
			if hasContacts && len(contacts.Current) > 0 {
				for _, hit := range contacts.Current {
					other := core.NewEntity(hit.Other)
					if _, ok := other.Component[enemy](ctx.World); !ok {
						continue // Another bullet may have destroyed it this tick.
					}
					if err := ctx.World.DestroyEntity(other); err != nil {
						return err
					}
					if err := stats.Update(ctx.World, func(s *game) {
						s.Score++
						if s.Score >= WinScore {
							s.Won = true
						}
					}); err != nil {
						return err
					}
				}
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

// spawnSystem shortens the interval as the score rises, down to
// MinSpawnEvery. Each timer completion spawns one enemy at a random point
// just outside the camera view.
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
		cameraTransform, _ := camera.Component[core.Transform](ctx.World)
		halfW := float64(ctx.LogicalWidth) / (2 * cameraTransform.Scale.X)
		halfH := float64(ctx.LogicalHeight) / (2 * cameraTransform.Scale.Y)

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

// spawnEnemy creates one enemy tank on the enemy layer, with collisions
// enabled for bullets and the player. Each enemy is a single hull entity;
// adding a separate turret would require entity hierarchies.
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

// enemiesSystem moves each enemy toward the tank.
func enemiesSystem(hull, stats *core.Entity) core.System {
	var targets *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if targets == nil {
			targets = core.NewQuery(ctx.World).With(core.Transform{}, enemy{})
		}
		if over(stats, ctx) {
			return nil
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

// dangerSystem marks the run lost when the hull contacts an enemy.
func dangerSystem(hull, stats *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		contacts, has := hull.Component[collision.Contacts](ctx.World)
		if !has || len(contacts.Current) == 0 {
			return nil
		}
		return stats.Update(ctx.World, func(s *game) { s.Lost = true })
	})
}

// resultSystem hides the banner during play and displays the result over
// the tank when the run ends.
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

// reloadReadoutSystem shows the remaining time while the cooldown runs and
// hides the readout when the gun is ready.
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
