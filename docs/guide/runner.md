# Runners

Castrum splits the engine in two: the `Game` owns configuration, schedules, and the fixed loop; a **runner** owns the platform - the window, the draw surface, and input - and drives the game.

```go
type Runner interface {
	Run() error
}
```

`game.Run(runner)` hands control over and blocks until the game quits. The runner calls the game's loop hooks (`Startup`, `Advance`) at the right times, so the simulation logic never touches platform code. If you ever swap renderers or run headless in a test, the `Game` is unchanged - only the runner is replaced.

## The Ebitengine runner

`castrum/ebitrun` provides the default runner, backed by [Ebitengine](https://ebitengine.org). The import path matches the package name, so no import alias is needed:

```go
import "github.com/Leonard-Atorough/castrum/ebitrun"
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

The engine renders the world itself each frame: it collects the primary camera and every visible sprite entity, interpolates their positions between the last two ticks, culls against the camera's view, sorts by `Layer` → `SortOrder` → world Y, and blits the result with positions snapped to whole pixels. There is no world-drawing code to write - declare `Camera` and `Sprite` entities and they appear.

What you register with `AddDraw` are **overlays**, on top of the engine-rendered world. Overlays are backend-typed by design - the runner owns the canvas type (`*ebiten.Image` here), so they do not transfer across runners - and run in registration order, once per display frame:

```go
runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
	ebitenutil.DebugPrint(screen, "score: 12")
	return nil
})
```

A draw error is stored and surfaces from `Run` on the next frame, because some platforms cannot report draw errors at draw time. The stored error names the failing layer: `"engine draw"` for the engine's own rendering, `"draw"` for an overlay.

## Interpolated drawing

`ctx.Alpha` in an overlay is the fixed-loop remainder in `[0, 1)`: how far the simulation has progressed toward the next tick. The engine applies it to sprites automatically; overlays that draw their own geometry apply it by hand - interpolate positions between the last two ticks:

```go
runner.AddDraw(func(ctx *core.Context, screen *ebiten.Image) error {
	x := lastPos + (currPos-lastPos)*ctx.Alpha
	// draw at x - smooth at any display refresh rate
	return nil
})
```

See [the loop](the-loop.md#alpha-and-interpolation) for how alpha is produced.

## Quitting

From any system, call `game.Quit()`. The runner observes it and stops its platform loop; `game.Run` returns. Closing the window also terminates the runner. `game.Quitting()` reports whether a quit was requested, if other systems want to react.
