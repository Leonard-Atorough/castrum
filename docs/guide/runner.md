# Runners

Castrum splits the engine in two: the `Game` owns configuration, schedules, and the fixed loop; a **runner** owns the platform - the window, the draw surface, and input - and drives the game.

```go
type Runner interface {
	Run() error
}
```

`game.Run(runner)` hands control over and blocks until the game quits. The runner calls the game's loop hooks (`Startup`, `Advance`) at the right times, so the simulation logic never touches platform code. If you ever swap renderers or run headless in a test, the `Game` is unchanged - only the runner is replaced.

## The Ebitengine runner

`castrum/ebiten` provides the default runner, backed by [Ebitengine](https://ebitengine.org). Because it shares its package name with Ebitengine, import it with an alias:

```go
import ebitrun "github.com/Leonard-Atorough/castrum/ebiten"
```

```go
game := castrum.New(castrum.WithTitle("My Game"))
runner := ebitrun.New(game,
	ebitrun.WithWindowSize(1920, 1080),
	ebitrun.WithLogicalSize(640, 360),
	ebitrun.WithResizable(),
)
```

Window options are runner concerns, not game options - a different runner (say, a terminal or a server) would have none. The full list is in the [options reference](../reference/options.md#runner-options).

### Window size vs logical size

The window is what the player sees; the **logical size** is the internal resolution your draw functions render to, scaled up by the platform. Render at `320x180` and your pixel-art game is the same game on every screen - the window scales, the logical canvas does not.

## Draw functions

Drawing is registered on the runner, not the game - the runner owns the canvas type (`*ebiten.Image` here), so draw functions do not transfer across runners. They run in registration order, after the fixed loop settles, once per display frame:

```go
runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
	screen.Clear()
	// draw the world state
	return nil
})
```

A draw error is stored and surfaces from `Run` on the next frame, because some platforms cannot report draw errors at draw time.

## Interpolated drawing

`ctx.Alpha` in a draw function is the fixed-loop remainder in `[0, 1)`: how far the simulation has progressed toward the next tick. Multiply movement by it to render *between* the last two ticks:

```go
runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
	x := lastPos + (currPos-lastPos)*ctx.Alpha
	// draw at x - smooth at any display refresh rate
	return nil
})
```

Skip this while prototyping; add it when fast-moving objects start to look choppy. See [the loop](the-loop.md#alpha-and-interpolation) for how alpha is produced.

## Quitting

From any system, call `game.Quit()`. The runner observes it and stops its platform loop; `game.Run` returns. Closing the window also terminates the runner. `game.Quitting()` reports whether a quit was requested, if other systems want to react.
