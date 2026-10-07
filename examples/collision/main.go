// Command collision demonstrates the collision stack: circle and
// rect colliders, layered masks, trigger pickups, a rotation-aware
// narrow phase, and the Contacts lifecycle read as component state.
// The engine registers the collision system in the fixed phase; this
// program never registers it - it only spawns colliders and reads
// Contacts.
//
// Detection only: the player passes through walls, because a
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
	player, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: 0 + wallThickness*2}},
		core.Sprite{
			Drawable: core.CircleShape{Radii: geom.Vector2{X: playerRadius, Y: playerRadius}},
			Color:    playerColor,
		},
		playerCollider,
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
		if _, err := spawnRect(g.World(), wall.position, wall.min, wall.max, wallColor); err != nil {
			return err
		}
	}

	// Three spinning bars spaced equally across the middle: the
	// collider is defined in local space, so the narrow phase tests
	// the rotated rectangle, not its fat axis-aligned bounds. Each
	// bar's rotation component carries its own random speed.
	for i := range 3 {
		barX := screenW/2 + float64(i-1)*barSpacing
		if _, err := spawnRect(g.World(),
			geom.Vector2{X: barX, Y: screenH / 2},
			geom.Vector2{X: -barHalfLength, Y: -barHalfWidth},
			geom.Vector2{X: barHalfLength, Y: barHalfWidth},
			barColor,
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
		collider.Layer = 1
		collider.Mask = 1 << 0
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

	// Move the player and tint it by its contacts: red while touching
	// a solid, yellow while touching a pickup. The contacts are
	// already computed - the engine's collision system runs earlier
	// in the same fixed phase.
	if err := g.AddSystem(core.PhaseFixed, "player", core.SystemFunc(func(ctx *core.Context) error {
		step := moveSpeed * ctx.DeltaTime.Seconds()
		delta := geom.Vector2{
			X: axisValue(ctx.Actions.Held("right")) - axisValue(ctx.Actions.Held("left")),
			Y: axisValue(ctx.Actions.Held("down")) - axisValue(ctx.Actions.Held("up")),
		}
		return player.Update(ctx.World, func(t *core.Transform) {
			t.Position = t.Position.Add(delta.Mul(step))
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
	// says which contacts are pickups.
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
		// The tint mirrors the player's contact state: the same data
		// the pickup system reasoned about, shown as color.
		tint := playerColor
		if contacts, ok := player.Component[collision.Contacts](ctx.World); ok && len(contacts.Current) > 0 {
			if contacts.Current[0].Trigger {
				tint = pickupColor
			} else {
				tint = contactColor
			}
		}
		if err := player.SetComponent(ctx.World, core.Sprite{
			Drawable: core.CircleShape{Radii: geom.Vector2{X: playerRadius, Y: playerRadius}},
			Color:    tint,
		}); err != nil {
			return err
		}

		text := fmt.Sprintf("score %d - WASD or arrows to move\n", score)
		if contacts, ok := player.Component[collision.Contacts](ctx.World); ok && len(contacts.Current) > 0 {
			hit := contacts.Current[0]
			text += fmt.Sprintf("touching %d: penetration %.1f, normal (%.1f, %.1f)",
				hit.Other, hit.Penetration, hit.Normal.X, hit.Normal.Y)
		}
		ebitenutil.DebugPrint(screen, text)
		return nil
	})

	// g.Run is the canonical entry: the guard against a second run
	// lives on the game, not the runner.
	return g.Run(runner)
}

// spawnRect places one rect: a sprite to see and a collider to hit,
// sharing the transform. The collider's mask admits only the player's
// layer. extra attaches more components - the bars add their
// rotation.
func spawnRect(world *core.World, position, min, max geom.Vector2, fill color.Color, extra ...any) (*core.Entity, error) {
	collider, err := collision.NewCollider(collision.RectShape{Min: min, Max: max})
	if err != nil {
		return nil, err
	}
	collider.Mask = 1 << 0
	size := max.Sub(min)
	return world.NewEntity(append([]any{
		core.Transform{Position: position},
		core.Sprite{Drawable: core.RectShape{Size: size}, Color: fill},
		collider,
	}, extra...)...)
}

// entered returns the enter edges in contacts: pairs in Current that
// were not in Previous. Stay is the intersection and exit the
// difference the other way - the same comparison scales to whatever
// a game needs to react to.
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

// randomSpin returns a bar's spin speed: the base rate scaled by a
// random modifier, with a random direction.
func randomSpin() float64 {
	speed := barSpeed * (0.5 + rand.Float64())
	if rand.Intn(2) == 0 {
		speed = -speed
	}
	return speed
}
