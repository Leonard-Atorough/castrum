# Set up the game

Before we can make a tank move, we need a window and a place for the game to live. We'll start with a small Go project and run an empty Castrum game.

## What you'll need

Install Go 1.27.2 or later and have a working Go development environment. The game uses Castrum's Ebitengine runner; on Linux, you may also need the windowing libraries listed in [Getting started for contributors](../../../../CONTRIBUTING.md#getting-started).

This tutorial follows the **development API**. `@main` below fetches the engine's current main branch, so it may change before the next release. For a stable released module, use the [tutorial for that release](../../../README.md) instead.

## Create a project

In a terminal, create an empty directory and initialize a Go module:

```sh
mkdir firstgame
cd firstgame
go mod init example.com/firstgame
go get github.com/Leonard-Atorough/castrum@main
```

Keep the terminal in `firstgame` for the rest of the tutorial. You can choose your own module path if you plan to publish the project.

We'll use the same art and audio as the completed example. Download `castrum-tutorial-assets-v0.3.0.zip` from the [v0.3.0 release](https://github.com/Leonard-Atorough/castrum/releases/tag/v0.3.0) and extract it **into the project directory**. You should have `sprites/` and `audio/` alongside `go.mod`, not inside an extra archive directory. These assets also work with the development API.

## Open a window

Create `main.go` in the project directory:

```go
package main

import (
	"fmt"

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
		"forward":  []input.Input{input.KeyInput{Key: input.KeyW}.Binding()},
		"backward": []input.Input{input.KeyInput{Key: input.KeyS}.Binding()},
		"left":     []input.Input{input.KeyInput{Key: input.KeyA}.Binding()},
		"right":    []input.Input{input.KeyInput{Key: input.KeyD}.Binding()},
		"fire":     []input.Input{input.MouseButtonInput{Button: input.MouseButtonLeft}.Binding()},
	}
	g, err := castrum.New(
		castrum.WithTitle("Tank game"),
		castrum.WithBindings(bindings),
	)
	if err != nil {
		return fmt.Errorf("create game: %w", err)
	}

	// Game setup goes here.

	runner, err := ebitrun.New(g)
	if err != nil {
		return fmt.Errorf("create runner: %w", err)
	}
	return g.Run(runner)
}
```

Run it:

```sh
go run .
```

An empty window should open. Close it to stop the program. If the command fails before opening a window, check the Go version and the platform prerequisites above.

`castrum.New` creates the game and its world. The game runs systems; the world holds the things those systems work on. The runner provides the window, drawing, and platform input. We named the controls up front so the movement and firing code can ask for actions such as `"forward"` instead of knowing which key produced them.

Next: [put a tank in the world and drive it](player.md).
