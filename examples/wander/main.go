// Command wander demonstrates the v0.1.0 renderer: an atlas sprite
// wanders to random screen points at fixed ticks, the engine renders it
// interpolated between ticks, and a user DrawFunc draws an overlay
// above the engine-rendered world.
//
// Run from the repository root:
//
//	go run ./examples/wander
package main

import (
	"embed"
	"fmt"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/Leonard-Atorough/castrum/geom"
)

//go:embed Dungeon_Character_2.png
var files embed.FS

const (
	// The sheet is 112x32 pixels of 16x16 tiles: 7 across, 2 down, so
	// the grid atlas registers char_0 through char_13. Any region works.
	region = "char_0"
	speed  = 120 // world units per second

	// The camera sits at the center of the 1280x720 logical resolution
	// at zoom 1, so world coordinates are screen coordinates and every
	// random point lands on screen.
	screenW, screenH = 1280, 720
)

func main() {
	if err := run(); err != nil {
		fmt.Println("wander:", err)
	}
}

func run() error {
	g, err := castrum.New(castrum.WithTitle("castrum — wander"))
	if err != nil {
		return err
	}

	// Register the atlas at startup, before the window opens: a bad
	// path or uneven tile division fails the launch, not the first
	// frame.
	if err := g.AddSystem(core.PhaseStartup, "register-atlas", core.SystemFunc(func(ctx *core.Context) error {
		server, err := ctx.World.Resource[*asset.Server]()
		if err != nil {
			return err
		}
		return server.RegisterGridAtlas("characters", "Dungeon_Character_2.png", 16, 16, "char")
	})); err != nil {
		return err
	}

	if _, err := g.World().NewEntity(
		core.Camera{Zoom: 1, Primary: true},
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: screenH / 2},
			Scale:    geom.Vector2{X: 1, Y: 1},
		},
		core.PrevTransform{Position: geom.Vector2{X: screenW / 2, Y: screenH / 2}},
	); err != nil {
		return err
	}

	sprite, err := g.World().NewEntity(
		core.AtlasSprite{Atlas: "characters", Region: region},
		// Sprite's zero value is the shown, opaque sprite — nothing
		// to set for the common case.
		core.Sprite{},
		core.Transform{
			Position: geom.Vector2{X: screenW / 2, Y: screenH / 2},
			Scale:    geom.Vector2{X: 4, Y: 4}, // 16px tiles read better at 64px
		},
		core.PrevTransform{Position: geom.Vector2{X: screenW / 2, Y: screenH / 2}},
	)
	if err != nil {
		return err
	}

	// Wander: each fixed tick moves the sprite toward its target at a
	// constant speed; reaching it picks a new random one. The engine's
	// prev-transform capture and the renderer's interpolation make the
	// motion smooth without any per-frame work here.
	target := randomPoint()
	if err := g.AddSystem(core.PhaseFixed, "wander", core.SystemFunc(func(ctx *core.Context) error {
		transform, ok := sprite.Component[core.Transform](ctx.World)
		if !ok {
			return nil
		}
		delta := target.Sub(transform.Position)
		distance := delta.Length()
		step := speed * ctx.DeltaTime.Seconds()
		if distance <= step {
			transform.Position = target
			target = randomPoint()
		} else {
			transform.Position = transform.Position.Add(delta.Mul(step / distance))
		}
		return sprite.SetComponent(ctx.World, transform)
	})); err != nil {
		return err
	}

	runner, err := ebitrun.New(g, ebitrun.WithFilesystem(files))
	if err != nil {
		return err
	}

	// User draws run after the engine-rendered world, so this lands on
	// the world.
	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		ebitenutil.DebugPrint(screen, "castrum wander — engine world below, this overlay above")
		return nil
	})

	return runner.Run()
}

// randomPoint picks a world point; with the camera framing the screen,
// every point is visible.
func randomPoint() geom.Vector2 {
	return geom.Vector2{
		X: rand.Float64() * screenW,
		Y: rand.Float64() * screenH,
	}
}
