// Command collision demonstrates the collision stack: circle and
// rect colliders, layered masks, trigger pickups, a rotation-aware
// narrow phase, and the Contacts lifecycle read as component state -
// bar hits drain health and speed once per enter edge, pickups heal.
// The engine registers the collision system in the fixed phase; this
// program never registers it - it only spawns colliders and reads
// Contacts.
//
// Detection only: nothing here is pushed out of a shape, because a
// response is not the collision system's job. A game that wants the
// player stopped reads the same Contacts state and moves the player
// back itself.
//
// Run from the repository root:
//
//	go run ./examples/collision
package main

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
)

const (
	screenW, screenH = 1280, 720

	playerRadius = 20
	moveSpeed    = 400
	minMoveSpeed = 200
	maxHealth    = 100

	// A bar hit costs hitHealthLoss health and hitSpeedLoss speed,
	// once per enter edge; a pickup restores pickupHeal health,
	// clamped by maxHealth.
	hitHealthLoss = 5
	hitSpeedLoss  = 5
	pickupHeal    = 2

	// Layer numbers: collision.Layers and collision.Mask turn these
	// into the colliders' bitmasks. The hazard layer separates the
	// bars from the walls - both are solid to the touch, but only
	// the bars damage.
	playerLayer = 0
	pickupLayer = 1
	hazardLayer = 2

	wallThickness = 40

	barHalfLength = 160
	barHalfWidth  = 12
	barSpeed      = 0.02 // radians per fixed tick
	barSpacing    = screenW / 4

	pickupRadius = 14
	pickupCount  = 3

	// margin keeps respawning pickups clear of the walls.
	margin = 60
)

// rotation is an entity's spin speed in radians per fixed tick. The
// bars each carry one drawn at random, so they sweep at different
// rates; the spin system finds them by query, so nothing holds a
// collection of bar handles.
type rotation float64

// hazardLayers is the bars' layer bitmask, built from the layer
// number; the hazard system matches contacts against it.
var hazardLayers = collision.Layers(hazardLayer)

// playerStats is the player's mutable state: health and speed that
// bar hits drain and pickups restore. A value component, like every
// other component: systems write it through Entity.Update, which
// re-validates on the way in, and queries can match it by type.
type playerStats struct {
	maxHealth     int
	currentHealth int
	maxSpeed      int
	currentSpeed  int
}

func newPlayerStats(maxHealth, maxSpeed int) playerStats {
	return playerStats{
		maxHealth:     maxHealth,
		currentHealth: maxHealth,
		maxSpeed:      maxSpeed,
		currentSpeed:  maxSpeed,
	}
}

var (
	playerColor  = color.RGBA{R: 120, G: 220, B: 120, A: 255}
	wallColor    = color.RGBA{R: 90, G: 110, B: 140, A: 255}
	barColor     = color.RGBA{R: 170, G: 130, B: 200, A: 255}
	pickupColor  = color.RGBA{R: 240, G: 220, B: 90, A: 255}
	contactColor = color.RGBA{R: 230, G: 90, B: 90, A: 255}
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	bindings := input.Bindings{
		"up":    {input.KeyInput{Key: input.KeyW}, input.KeyInput{Key: input.KeyArrowUp}},
		"down":  {input.KeyInput{Key: input.KeyS}, input.KeyInput{Key: input.KeyArrowDown}},
		"left":  {input.KeyInput{Key: input.KeyA}, input.KeyInput{Key: input.KeyArrowLeft}},
		"right": {input.KeyInput{Key: input.KeyD}, input.KeyInput{Key: input.KeyArrowRight}},
	}

	g, err := castrum.New(
		castrum.WithTitle("castrum - collision"),
		castrum.WithBindings(bindings),
	)
	if err != nil {
		return err
	}

	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	// The player: a circle collider on layer 0 with its mask open to
	// every layer - collision.NewCollider's defaults. NewCollider
	// validates immediately, so the shape is known good here.
	playerCollider, err := collision.NewCollider(collision.CircleShape{Radius: playerRadius})
	if err != nil {
		return err
	}
	stats := newPlayerStats(maxHealth, moveSpeed)
	player, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: 0 + wallThickness*2}},
		core.Sprite{
			Drawable: core.CircleShape{Radii: geom.Vector2{X: playerRadius, Y: playerRadius}},
			Color:    playerColor,
		},
		playerCollider,
		stats,
	)
	if err != nil {
		return err
	}

	// The walls listen only to the player's layer, and so does the
	// bar below: wall-to-wall and wall-to-pickup pairs never even
	// reach the narrow phase.
	for _, wall := range []struct{ position, min, max geom.Vector2 }{
		{position: geom.Vector2{X: screenW / 2, Y: 0}, min: geom.Vector2{X: -screenW / 2, Y: 0}, max: geom.Vector2{X: screenW / 2, Y: wallThickness}},
		{position: geom.Vector2{X: screenW / 2, Y: screenH}, min: geom.Vector2{X: -screenW / 2, Y: -wallThickness}, max: geom.Vector2{X: screenW / 2, Y: 0}},
		{position: geom.Vector2{X: 0, Y: screenH / 2}, min: geom.Vector2{X: 0, Y: -screenH / 2}, max: geom.Vector2{X: wallThickness, Y: screenH / 2}},
		{position: geom.Vector2{X: screenW, Y: screenH / 2}, min: geom.Vector2{X: -wallThickness, Y: -screenH / 2}, max: geom.Vector2{X: 0, Y: screenH / 2}},
	} {
		if _, err := spawnRect(g.World(), wall.position, wall.min, wall.max, wallColor, collision.Layers(playerLayer)); err != nil {
			return err
		}
	}

	// Three spinning bars spaced equally across the middle, on the
	// hazard layers: solid to the touch like the walls, but a hit
	// costs health and speed. The collider is defined in local space,
	// so the narrow phase tests the rotated rectangle, not its fat
	// axis-aligned bounds. Each bar's rotation component carries its
	// own random speed.
	for i := range 3 {
		barX := screenW/2 + float64(i-1)*barSpacing
		if _, err := spawnRect(g.World(),
			geom.Vector2{X: barX, Y: screenH / 2},
			geom.Vector2{X: -barHalfLength, Y: -barHalfWidth},
			geom.Vector2{X: barHalfLength, Y: barHalfWidth},
			barColor,
			hazardLayers,
			rotation(randomSpin()),
		); err != nil {
			return err
		}
	}

	// Pickups live on their own layer, listening for the player, and
	// carry the trigger flag - detection without any response
	// implication.
	pickups := make([]*core.Entity, pickupCount)
	for i := range pickups {
		collider, err := collision.NewCollider(collision.CircleShape{Radius: pickupRadius})
		if err != nil {
			return err
		}
		collider.Layers = collision.Layers(pickupLayer)
		collider.Mask = collision.Mask(playerLayer)
		collider.Trigger = true
		pickups[i], err = g.World().NewEntity(
			core.Transform{Position: randomPosition()},
			core.Sprite{
				Drawable: core.CircleShape{Radii: geom.Vector2{X: pickupRadius, Y: pickupRadius}},
				Color:    pickupColor,
			},
			collider,
		)
		if err != nil {
			return err
		}
	}

	// score is written by the pickup system and read by the overlay;
	// the fixed loop runs both, so no synchronization is needed.
	score := 0

	// Move the player at its current speed and tint it by its
	// contacts: red while touching a solid, yellow while touching a
	// pickup. The contacts are already computed - the engine's
	// collision system runs earlier in the same fixed phase.
	if err := g.AddSystem(core.PhaseFixed, "player", core.SystemFunc(func(ctx *core.Context) error {
		stats, _ := player.Component[playerStats](ctx.World)
		step := float64(stats.currentSpeed) * ctx.DeltaTime.Seconds()
		delta := geom.Vector2{
			X: axisValue(ctx.Actions.Held("right")) - axisValue(ctx.Actions.Held("left")),
			Y: axisValue(ctx.Actions.Held("down")) - axisValue(ctx.Actions.Held("up")),
		}
		if err := player.Update(ctx.World, func(t *core.Transform) {
			t.Position = t.Position.Add(delta.Mul(step))
		}); err != nil {
			return err
		}

		tint := playerColor
		if contacts, ok := player.Component[collision.Contacts](ctx.World); ok && len(contacts.Current) > 0 {
			if contacts.Current[0].Trigger {
				tint = pickupColor
			} else {
				tint = contactColor
			}
		}
		return player.SetComponent(ctx.World, core.Sprite{
			Drawable: core.CircleShape{Radii: geom.Vector2{X: playerRadius, Y: playerRadius}},
			Color:    tint,
		})
	})); err != nil {
		return err
	}

	// Bar hits cost health and speed, applied once per enter edge -
	// leaning into a spinning bar drains nothing further until the
	// pair separates and re-enters, which a sweeping bar does on its
	// own. The hit side is told from the wall by the other
	// collider's layers: any overlap with the hazard mask counts.
	if err := g.AddSystem(core.PhaseFixed, "hazard", core.SystemFunc(func(ctx *core.Context) error {
		contacts, ok := player.Component[collision.Contacts](ctx.World)
		if !ok {
			return nil
		}
		hits := 0
		for _, hit := range entered(contacts) {
			if hit.Trigger {
				continue
			}
			other, ok := core.NewEntity(hit.Other).Component[collision.Collider](ctx.World)
			if ok && other.Layers&hazardLayers != 0 {
				hits++
			}
		}
		if hits == 0 {
			return nil
		}
		return player.Update(ctx.World, func(s *playerStats) {
			s.currentHealth = max(s.currentHealth-hits*hitHealthLoss, 0)
			s.currentSpeed = max(s.currentSpeed-hits*hitSpeedLoss, minMoveSpeed)
		})
	})); err != nil {
		return err
	}

	// Spin every entity that carries a rotation component, at its own
	// speed - the query finds the bars; no handle is held. The collider
	// follows the rotation for free: the collision system transforms
	// the shape by the transform every tick.
	var spin *core.Query
	if err := g.AddSystem(core.PhaseFixed, "bar", core.SystemFunc(func(ctx *core.Context) error {
		if spin == nil {
			spin = core.NewQuery(ctx.World).With(core.Transform{}, rotation(0))
		}
		for e := range spin.Execute() {
			speed, _ := e.Component[rotation]()
			e.Update(func(t *core.Transform) { t.Rotation += float64(speed) })
		}
		return nil
	})); err != nil {
		return err
	}

	// Collect pickups by deriving the enter edge from Contacts state:
	// a contact in Current that was not in Previous. The trigger flag
	// says which contacts are pickups; collecting one heals the
	// player, clamped by max health.
	if err := g.AddSystem(core.PhaseFixed, "pickups", core.SystemFunc(func(ctx *core.Context) error {
		for _, pickup := range pickups {
			contacts, ok := pickup.Component[collision.Contacts](ctx.World)
			if !ok {
				continue
			}
			for _, hit := range entered(contacts) {
				if !hit.Trigger {
					continue
				}
				score++
				if err := pickup.SetComponent(ctx.World, core.Transform{Position: randomPosition()}); err != nil {
					return err
				}
				if err := player.Update(ctx.World, func(s *playerStats) {
					s.currentHealth = min(s.currentHealth+pickupHeal, s.maxHealth)
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}

	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		// Read-only: everything the overlay prints is state the
		// systems already wrote this tick.
		stats, _ := player.Component[playerStats](ctx.World)
		text := fmt.Sprintf("score %d  health %d/%d  speed %d\n", score, stats.currentHealth, stats.maxHealth, stats.currentSpeed)
		text += "WASD or arrows to move; the bars hit, the pickups heal"
		if contacts, ok := player.Component[collision.Contacts](ctx.World); ok && len(contacts.Current) > 0 {
			hit := contacts.Current[0]
			text += fmt.Sprintf("\ntouching %d: penetration %.1f, normal (%.1f, %.1f)",
				hit.Other, hit.Penetration, hit.Normal.X, hit.Normal.Y)
		}
		ebitenutil.DebugPrint(screen, text)
		return nil
	})

	// g.Run is the canonical entry: the guard against a second run
	// lives on the game, not the runner.
	return g.Run(runner)
}

// spawnRect is a helper method to spawn a rectangular sprite and
// matching collider on the given layer bitmask, used for walls and
// spinning bars. The collider's mask admits only the player's layer.
// extra attaches more components - the bars add their rotation.
func spawnRect(world *core.World, position, min, max geom.Vector2, fill color.Color, layers uint32, extra ...any) (*core.Entity, error) {
	collider, err := collision.NewCollider(collision.RectShape{Min: min, Max: max})
	if err != nil {
		return nil, err
	}
	collider.Layers = layers
	collider.Mask = collision.Mask(playerLayer)
	size := max.Sub(min)
	return world.NewEntity(append([]any{
		core.Transform{Position: position},
		core.Sprite{Drawable: core.RectShape{Size: size}, Color: fill},
		collider,
	}, extra...)...)
}

// entered compares the current contacts with contacts in the previous
// frame and returns the new contacts that were not present before.
func entered(contacts collision.Contacts) []collision.Contact {
	var edges []collision.Contact
	for _, hit := range contacts.Current {
		known := false
		for _, previous := range contacts.Previous {
			if previous.Other == hit.Other {
				known = true
				break
			}
		}
		if !known {
			edges = append(edges, hit)
		}
	}
	return edges
}

func axisValue(held bool) float64 {
	if held {
		return 1
	}
	return 0
}

func randomPosition() geom.Vector2 {
	return geom.Vector2{
		X: margin + rand.Float64()*(screenW-2*margin),
		Y: margin + rand.Float64()*(screenH-2*margin),
	}
}

// randomSpin returns a random float64 value for the spin speed of a bar.
func randomSpin() float64 {
	speed := barSpeed * (0.5 + rand.Float64())
	if rand.Intn(2) == 0 {
		speed = -speed
	}
	return speed
}
