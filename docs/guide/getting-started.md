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
	ebitrun "github.com/Leonard-Atorough/castrum/ebiten"
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

	// typically not required, the runner registers a default draw function automatically.
	// you can omit this if you don't need a custom draw function.
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
- `ebitrun.New` builds the default runner, backed by [Ebitengine](https://ebitengine.org). It owns the window, the draw surface, and input.
- `runner.AddDraw` registers a draw function, which receives the context and the render target. The runner holds a primary draw func for the main thread so typically a custom draw function is not required.
- `game.Run(runner)` hands control over and blocks until the game quits.

## Where to go next

- [The loop](the-loop.md) - how fixed ticks, frame systems, and interpolation fit together.
- [Systems](systems.md) - writing and organizing game logic.
- [Resources](resources.md) - typed, shared state such as configuration.
- [Runners](runner.md) - windows, drawing, and the runner contract.
