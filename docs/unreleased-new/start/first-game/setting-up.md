# Setting up

Let's make a small tank game. In this chapter we'll create a Go project and open an empty window. We'll fill that window together in the chapters that follow.

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

Keep the terminal in `firstgame` throughout the tutorial. You can choose your own module path if you plan to publish the project.

We'll use the art and audio from the completed tank example. Download `castrum-tutorial-assets-v0.3.0.zip` from the [v0.3.0 release](https://github.com/Leonard-Atorough/castrum/releases/tag/v0.3.0) and extract it **into the project directory**. You should have `sprites/` and `audio/` alongside `go.mod`, not inside an extra archive directory. These assets also work with the development API.

## Open a window

Create `main.go` in the project directory:

```go
package main

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/ebitrun"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	g, err := castrum.New(castrum.WithTitle("Tank game"))
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

## A quick look at Castrum

`castrum.New` creates a **game** and its **world**. The world holds the things in your game, and the game runs the rules that update them. The runner provides the window, drawing, and platform input.

You'll build those things using an entity-component system, often called an **ECS**. An entity is an identity; components describe it, and systems do work with those components. That's enough to get started. We'll introduce each piece when we use it, and you can return to the existing [key concepts](../../../unreleased/getting-started/concepts.md) for a deeper explanation.

Next: [define the player](defining-the-player.md).
