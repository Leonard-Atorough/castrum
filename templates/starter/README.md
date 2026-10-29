# Castrum starter

A small, asset-free Go game project using Castrum. It demonstrates a standard
`cmd/` entry point, an `internal/game` package, input bindings, a fixed-tick
movement system, and the Ebitengine runner.

Requires Go 1.27.2 or later.

## Run

Copy this directory to your new project, then run these commands from the
copied directory:

```sh
go mod tidy
go run ./cmd/game
```

The blue square starts in the center of the window. Move it with **WASD**.

Before publishing your game, replace `example.com/mygame` in `go.mod` and
`cmd/game/main.go` with your own module path.

## Layout

- `cmd/game/main.go` constructs the game, creates the Ebitengine runner, and
  starts the game.
- `internal/game/game.go` defines the Castrum world, input bindings, player,
  and gameplay system.
- `go.mod` pins the template to the latest released Castrum version when this
  template was prepared.

Add assets under your project as needed and configure Castrum's asset loader
for their location. See the [asset guide](https://github.com/Leonard-Atorough/castrum/blob/main/docs/guides/assets.md)
and [first-game tutorial](https://github.com/Leonard-Atorough/castrum/blob/main/docs/getting-started/your-first-game.md)
for the next steps.
