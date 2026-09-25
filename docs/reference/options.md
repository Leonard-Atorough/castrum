# Options Reference

Both the `Game` and the Ebitengine runner use the options pattern: constructors return an option that is applied over defaults. Invalid values panic at construction - a setup error, not a runtime one.

## Game options

`castrum` package. Window and graphics settings do not live here; they belong to the runner.

| Option | Effect | Default | Panics when |
|---|---|---|---|
| `castrum.WithTitle(title)` | game title, displayed by the runner | `"castrum"` | - |
| `castrum.WithFixedTPS(tps)` | fixed simulation rate, ticks per second | `60` | not positive |
| `castrum.WithMaxFrameTime(d)` | per-frame accumulator clamp (spiral-of-death guard) | `250ms` | not positive |
| `castrum.WithMaxTicksPerFrame(n)` | max fixed ticks per display frame; excess time is dropped | `5` | not positive |

The converged values are exposed on `game.Options()` as the `castrum.Options` struct. The derived tick interval `1/FixedTPS` is available as `Options.FixedDT()`.

Guidance: leave `MaxFrameTime` and `MaxTicksPerFrame` at the defaults unless profiling says otherwise. See [the loop](../guide/the-loop.md).

## Runner options

`castrum/ebitrun` package.

| Option | Effect | Default | Panics when |
|---|---|---|---|
| `ebitrun.WithWindowSize(w, h)` | window size in pixels | `1280x720` | not positive |
| `ebitrun.WithLogicalSize(w, h)` | internal render resolution | `1280x720` | not positive |
| `ebitrun.WithResizable()` | enable window resizing | fixed-size window | - |
| `ebitrun.WithoutVSync()` | disable vsync | vsync on | - |

See [runners](../guide/runner.md) for what window vs logical size means.
