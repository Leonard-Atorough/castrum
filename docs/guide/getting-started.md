# Getting Started with Castrum

## Installation

To install Castrum, use `go get`:

```sh
go get github.com/Leonard-Atorough/castrum
```

Note: Castrum requires Go 1.27 or later.

## A minimal game

Create a new module and a `main.go`:

```sh
mkdir hello && cd hello
go mod init hello
go get github.com/Leonard-Atorough/castrum
```

```go
package main

import (
	"image/color"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/ebitrun"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	game, err := castrum.New(castrum.WithTitle("Hello, Castrum"))
	if err != nil {
		panic(err)
	}

	if err := game.AddSystem(core.PhaseFixed, "hello", core.SystemFunc(func(ctx *core.Context) error {
		// Game logic runs here, at a fixed rate.
		return nil
	})); err != nil {
		panic(err)
	}

	runner, err := ebitrun.New(game)
	if err != nil {
		panic(err)
	}

	// Overlays run after the engine-rendered world. With no sprites in
	// the world yet, this Fill paints the whole window.
	runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
		screen.Fill(color.RGBA{R: 20, G: 24, B: 33, A: 255})
		return nil
	})

	if err := game.Run(runner); err != nil {
		panic(err)
	}
}
```

Run it:

```sh
go run .
```

A window opens and fills with a dark blue color. Close the window, or call `game.Quit()` from a system, to stop.

## What each piece does

- `castrum.New` builds a `Game` from options. The `With*` options cover the simulation rate and the title; invalid values panic at setup, not mid-game.
- `game.AddSystem` registers a system into a phase. `"hello"` names the registration - it appears in error messages, so pick something meaningful. Systems in the same phase run in registration order.
- `ebitrun.New` builds the default runner, backed by [Ebitengine](https://ebitengine.org). It owns the window, the draw surface, and input, and it also provides the asset server your sprites load through.
- `runner.AddDraw` registers an overlay. The engine renders the world itself - camera and sprite entities, interpolated and sorted - before any overlay runs; overlays are for text, HUDs, and effects on top.
- `game.Run(runner)` hands control over and blocks until the game quits.

## Put a sprite on screen

The engine renders sprites on its own - there is no drawing code to write. Declare a camera, register your art, spawn an entity, and move it in a fixed system.

A primary camera frames the world. Position it at the center of the logical resolution with zoom 1 and world coordinates match screen coordinates:

```go
if _, err := game.World().NewEntity(
	core.Camera{Zoom: 1, Primary: true},
	core.Transform{Position: geom.Vector2{X: 640, Y: 360}, Scale: geom.Vector2{X: 1, Y: 1}},
	core.PrevTransform{Position: geom.Vector2{X: 640, Y: 360}},
); err != nil {
	panic(err)
}
```

Register a sprite sheet as a grid atlas, at startup - a bad path or uneven tile division fails the launch before a window opens. A 112x32 sheet of 16x16 tiles becomes `char_0` through `char_13`:

```go
if err := game.AddSystem(core.PhaseStartup, "register-atlas", core.SystemFunc(func(ctx *core.Context) error {
	server, err := ctx.World.Resource[*asset.Server]()
	if err != nil {
		return err
	}
	return server.RegisterGridAtlas("characters", "sheet.png", 16, 16, "char")
})); err != nil {
	panic(err)
}
```

A sprite is an entity with one component: the style plus what to draw. The `Drawable` sum carries the picture — an atlas region or a standalone texture — or a shape. `Sprite`'s zero-value style is shown and opaque: set `Hidden` to stop showing it, `Transparency` to fade it, `Layer`/`SortOrder` to order it against other sprites.

```go
if _, err := game.World().NewEntity(
	core.Sprite{Drawable: core.AtlasSource{Atlas: "characters", Region: "char_0"}},
	core.Transform{Position: geom.Vector2{X: 640, Y: 360}, Scale: geom.Vector2{X: 4, Y: 4}},
	core.PrevTransform{Position: geom.Vector2{X: 640, Y: 360}},
); err != nil {
	panic(err)
}
```

`core.TextureSource{Texture: "sky.png"}` draws a standalone image whole, and the shapes — `core.RectShape{Size: ...}`, `core.CircleShape{Radius: ...}`, `core.LineShape{To: ...}` — draw geometry with the same component, styled by `Tint` and outlined with `Outline` + `StrokeWidth`. One `Drawable` per sprite, enforced by the sum: a sprite cannot declare two pictures.

Now move it in a fixed system, by writing `Transform` each tick:

```go
sprite.SetComponent(ctx.World, core.Transform{Position: next, Scale: spriteScale})
```

That is the whole loop. The engine snapshots each entity's previous position, and between ticks it renders every sprite interpolated between the last two positions - smooth motion at any display rate, with no per-frame code. Sprites off the camera's view are culled; positions snap to whole pixels so pixel art stays crisp while moving. A sprite whose atlas, region, or texture fails to resolve errors the frame and names the handle, so a typo surfaces on the first rendered frame instead of silently drawing nothing.

A complete, runnable version - a sprite wandering between random points, with an overlay - lives in the repository at `examples/wander`:

```sh
go run ./examples/wander
```

## Where to go next

- [The loop](the-loop.md) - how fixed ticks, frame systems, and interpolation fit together.
- [Systems](systems.md) - writing and organizing game logic.
- [Resources](resources.md) - typed, shared state such as configuration.
- [Runners](runner.md) - windows, drawing, and the runner contract.
