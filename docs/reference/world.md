# World Reference

`core.World` holds the game's shared state. It embeds `core.Resources`, so resource methods are called directly on the world: `world.Provide(...)`, `world.Resource[T]()`. Obtain it from `game.World()` or `ctx.World`.

```go
world := game.World()
```

Constructing one directly (`core.NewWorld()`) is possible but normally unnecessary - `castrum.New` creates the game's world for you.

## Methods

| Method | Effect |
|---|---|
| `world.Provide[T](ctor)` | register `T` with constructor `ctor`, resolved lazily on first request |
| `world.ProvideEager[T](ctor)` | register `T`, resolved at `Game.Startup` before `PhaseStartup` systems |
| `world.Resource[T]()` | return `T`, running its constructor if this is the first request; cached after |
| `world.ResolveEager()` | resolve all eager resources; called by `Game.Startup` |

All constructors have the shape:

```go
func(w *core.World) (T, error)
```

Receiving the world lets one resource depend on another. Cycles are detected and reported as errors.

## Errors

| Situation | Error |
|---|---|
| registering an already-registered type | `resource of type T is already registered` |
| requesting an unregistered type | `resource of type T is not registered` |
| constructor fails | the constructor's error; the resource stays unconstructed and a later request retries |
| cycle during resolution | `circular dependency detected for resource of type T` |

See [resources](../guide/resources.md) for the guide-level treatment.
