// Command collision demonstrates layered colliders, rotating bars, trigger
// pickups, and contact enter events. Bar hits reduce health and speed;
// pickups restore health.
//
// The engine registers collision detection automatically. It reports overlaps
// but does not stop or separate objects.
//
// Run from this directory:
//
//	go run .
package main

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/collision"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/debug"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
	"github.com/Leonard-Atorough/castrum/input"
	"github.com/Leonard-Atorough/castrum/sprite"
)

const (
	screenW, screenH = 1280, 720

	playerRadius = 20
	moveSpeed    = 400
	minMoveSpeed = 200
	maxHealth    = 100

	hitHealthLoss = 5
	hitSpeedLoss  = 5
	pickupHeal    = 2

	// The hazard layer distinguishes damaging bars from walls.
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

	margin = 60
)

// rotation is measured in radians per fixed tick.
type rotation float64

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
	playerColor  = &color.RGBA{R: 120, G: 220, B: 120, A: 255}
	wallColor    = &color.RGBA{R: 90, G: 110, B: 140, A: 255}
	barColor     = &color.RGBA{R: 170, G: 130, B: 200, A: 255}
	pickupColor  = &color.RGBA{R: 240, G: 220, B: 90, A: 255}
	contactColor = &color.RGBA{R: 230, G: 90, B: 90, A: 255}
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	bindings := input.Bindings{
		"up":    {input.KeyInput{Key: input.KeyW}.Binding(), input.KeyInput{Key: input.KeyArrowUp}.Binding()},
		"down":  {input.KeyInput{Key: input.KeyS}.Binding(), input.KeyInput{Key: input.KeyArrowDown}.Binding()},
		"left":  {input.KeyInput{Key: input.KeyA}.Binding(), input.KeyInput{Key: input.KeyArrowLeft}.Binding()},
		"right": {input.KeyInput{Key: input.KeyD}.Binding(), input.KeyInput{Key: input.KeyArrowRight}.Binding()},
	}

	g := castrum.New(
		castrum.WithTitle("castrum - collision"),
		castrum.WithBindings(bindings),
	)

	hazardMask, err := collision.Layers(hazardLayer)
	if err != nil {
		return err
	}

	camera := g.MainCamera()
	if err := camera.Update(g.World(), func(t *core.Transform) {
		t.Position = geom.Vector2{X: screenW / 2, Y: screenH / 2}
	}); err != nil {
		return err
	}

	// NewCollider defaults to layer 0 and listens to every layer.
	playerCollider, err := collision.NewCollider(collision.Circle{Radius: playerRadius})
	if err != nil {
		return err
	}
	stats := newPlayerStats(maxHealth, moveSpeed)
	player, err := g.World().NewEntity(
		core.Transform{Position: geom.Vector2{X: screenW / 2, Y: 0 + wallThickness*2}},
		sprite.Primitive{Shape: sprite.ShapeCircle, Radii: geom.Vector2{X: playerRadius, Y: playerRadius},
			Style: sprite.Style{Color: playerColor}},
		playerCollider,
		stats,
	)
	if err != nil {
		return err
	}

	// Walls, bars, and pickups listen only to the player's layer.
	// Walls share that layer, so they can also contact each other.
	for _, wall := range []struct{ position, min, max geom.Vector2 }{
		{position: geom.Vector2{X: screenW / 2, Y: 0}, min: geom.Vector2{X: -screenW / 2, Y: 0}, max: geom.Vector2{X: screenW / 2, Y: wallThickness}},
		{position: geom.Vector2{X: screenW / 2, Y: screenH}, min: geom.Vector2{X: -screenW / 2, Y: -wallThickness}, max: geom.Vector2{X: screenW / 2, Y: 0}},
		{position: geom.Vector2{X: 0, Y: screenH / 2}, min: geom.Vector2{X: 0, Y: -screenH / 2}, max: geom.Vector2{X: wallThickness, Y: screenH / 2}},
		{position: geom.Vector2{X: screenW, Y: screenH / 2}, min: geom.Vector2{X: -wallThickness, Y: -screenH / 2}, max: geom.Vector2{X: 0, Y: screenH / 2}},
	} {
		if _, err := spawnRect(g.World(), wall.position, wall.min, wall.max, wallColor, playerLayer); err != nil {
			return err
		}
	}

	// Local-space box colliders rotate with the bars.
	for i := range 3 {
		barX := screenW/2 + float64(i-1)*barSpacing
		if _, err := spawnRect(g.World(),
			geom.Vector2{X: barX, Y: screenH / 2},
			geom.Vector2{X: -barHalfLength, Y: -barHalfWidth},
			geom.Vector2{X: barHalfLength, Y: barHalfWidth},
			barColor,
			hazardLayer,
			rotation(randomSpin()),
		); err != nil {
			return err
		}
	}

	// Pickups use triggers to distinguish their contacts from solid contacts.
	pickups := make([]core.Entity, pickupCount)
	for i := range pickups {
		collider, err := collision.NewCollider(collision.Circle{Radius: pickupRadius})
		if err != nil {
			return err
		}
		collider.Layers, err = collision.Layers(pickupLayer)
		if err != nil {
			return err
		}
		collider.Mask, err = collision.Layers(playerLayer)
		if err != nil {
			return err
		}
		collider.Trigger = true
		pickups[i], err = g.World().NewEntity(
			core.Transform{Position: randomPosition()},
			sprite.Primitive{Shape: sprite.ShapeCircle, Radii: geom.Vector2{X: pickupRadius, Y: pickupRadius},
				Style: sprite.Style{Color: pickupColor}},
			collider,
		)
		if err != nil {
			return err
		}
	}

	score := 0
	panel := g.World().MustResource[*debug.Panel]()
	panel.Message("WASD or arrows to move; the bars hit, the pickups heal")

	// Collision detection runs before user fixed systems, so these contacts
	// reflect positions before this tick's movement.
	if err := g.AddSystem(core.PhaseFixed, "player", func(ctx *core.Context) error {
		actions, _ := ctx.World.Resource[input.Actions]()
		stats, _ := player.Component[playerStats](ctx.World)
		step := float64(stats.currentSpeed) * ctx.DeltaTime.Seconds()
		delta := geom.Vector2{
			X: axisValue(actions.Held("right")) - axisValue(actions.Held("left")),
			Y: axisValue(actions.Held("down")) - axisValue(actions.Held("up")),
		}
		if err := player.Update(ctx.World, func(t *core.Transform) {
			t.Position = t.Position.Add(delta.Mul(step))
		}); err != nil {
			return err
		}

		tint := playerColor
		if contacts, ok := player.Component[collision.Contacts](ctx.World); ok && len(contacts.Current) > 0 {
			tint = pickupColor
			for _, hit := range contacts.Current {
				if !hit.Trigger {
					tint = contactColor
					break
				}
			}
		}
		return player.SetComponent(ctx.World, sprite.Primitive{Shape: sprite.ShapeCircle, Radii: geom.Vector2{X: playerRadius, Y: playerRadius},
			Style: sprite.Style{Color: tint}})
	}); err != nil {
		return err
	}

	// Apply damage on hazard contact entry, not every overlapping tick.
	if err := g.AddSystem(core.PhaseFixed, "hazard", hazardSystem(player, panel, hazardMask)); err != nil {
		return err
	}

	var spin *core.Query
	if err := g.AddSystem(core.PhaseFixed, "bar", func(ctx *core.Context) error {
		if spin == nil {
			spin = core.NewQuery(ctx.World).With(core.Transform{}, rotation(0))
		}
		for e := range spin.Execute() {
			speed, _ := e.Component[rotation]()
			e.Update(func(t *core.Transform) { t.Rotation += float64(speed) })
		}
		return nil
	}); err != nil {
		return err
	}

	// Collect and respawn pickups when a trigger appears in Current but not Previous.
	if err := g.AddSystem(core.PhaseFixed, "pickups", pickupSystem(player, pickups, &score, panel)); err != nil {
		return err
	}

	runner := ebitrun.New(ebitrun.WithDebugOverlay())

	runner.Run(g)
	return nil
}

func hazardSystem(player core.Entity, panel *debug.Panel, hazardMask uint32) core.SystemFunc {
	return func(ctx *core.Context) error {
		contacts, ok := player.Component[collision.Contacts](ctx.World)
		if !ok {
			return nil
		}
		hits := 0
		for _, hit := range entered(contacts) {
			if hit.Trigger {
				continue
			}
			other, ok := hit.Other.Component[collision.Collider](ctx.World)
			if ok && other.Layers&hazardMask != 0 {
				hits++
			}
		}
		if hits == 0 {
			return nil
		}
		if err := player.Update(ctx.World, func(s *playerStats) {
			s.currentHealth = max(s.currentHealth-hits*hitHealthLoss, 0)
			s.currentSpeed = max(s.currentSpeed-hits*hitSpeedLoss, minMoveSpeed)
			panel.Message(fmt.Sprintf("hazard hit: health %d/%d, speed %d",
				s.currentHealth, s.maxHealth, s.currentSpeed))
		}); err != nil {
			return err
		}
		return nil
	}
}

func pickupSystem(player core.Entity, pickups []core.Entity, score *int, panel *debug.Panel) core.SystemFunc {
	return func(ctx *core.Context) error {
		for _, pickup := range pickups {
			contacts, ok := pickup.Component[collision.Contacts](ctx.World)
			if !ok {
				continue
			}
			for _, hit := range entered(contacts) {
				if !hit.Trigger {
					continue
				}
				*score++
				if err := pickup.SetComponent(ctx.World, core.Transform{Position: randomPosition()}); err != nil {
					return err
				}
				if err := player.Update(ctx.World, func(s *playerStats) {
					s.currentHealth = min(s.currentHealth+pickupHeal, s.maxHealth)
					panel.Message(fmt.Sprintf("pickup: score %d, health %d/%d",
						*score, s.currentHealth, s.maxHealth))
				}); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func spawnRect(world *core.World, position, min, max geom.Vector2, fill *color.RGBA, layer int, extra ...any) (core.Entity, error) {
	collider, err := collision.NewCollider(collision.Box{Min: min, Max: max})
	if err != nil {
		return core.Entity{}, err
	}
	collider.Layers, err = collision.Layers(layer)
	if err != nil {
		return core.Entity{}, err
	}
	collider.Mask, err = collision.Layers(playerLayer)
	if err != nil {
		return core.Entity{}, err
	}
	size := max.Sub(min)
	return world.NewEntity(append([]any{
		core.Transform{Position: position},
		sprite.Primitive{Shape: sprite.ShapeRect, Rect: size, Style: sprite.Style{Color: fill}},
		collider,
	}, extra...)...)
}

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

func randomSpin() float64 {
	speed := barSpeed * (0.5 + rand.Float64())
	if rand.Intn(2) == 0 {
		speed = -speed
	}
	return speed
}
