# Resources

A resource is a typed, singleton value stored on the world: configuration, an asset cache, a random source, the current input state. Systems read and mutate resources through `ctx.World`; the world handles construction, caching, and dependency wiring.

## Register, then request

Register a resource by type, with a constructor:

```go
type Config struct {
	StartLives int
}

world := game.World()

if err := world.Provide(func(w *core.World) (Config, error) {
	return Config{StartLives: 3}, nil
}); err != nil {
	panic(err)
}
```

Anywhere later - in a system, or in another resource's constructor - request it by type:

```go
game.AddSystem(core.PhaseStartup, "setup", core.SystemFunc(func(ctx *core.Context) error {
	cfg, err := ctx.World.Resource[Config]()
	if err != nil {
		return err
	}
	// use cfg.StartLives
	return nil
}))
```

Because resources are keyed by Go type, `Resource[Config]` always returns the same `Config` for the life of the world. No global variables needed.

## Lazy and eager resolution

`Provide` registers a lazily resolved resource: the constructor runs the first time it is requested, and the result is cached. Good for optional or expensive things - an asset cache constructed only if a level uses it.

`ProvideEager` registers a resource resolved when the game starts: `Game.Startup` resolves all eager resources before running `PhaseStartup` systems. Good for things the game needs unconditionally - a failed constructor surfaces as a startup error before a window opens.

A constructor receives the world, so one resource can depend on another:

```go
type Sprites struct{ /* ... */ }

if err := world.ProvideEager(func(w *core.World) (Sprites, error) {
	cfg, err := w.Resource[Config]() // pull in the dependency
	if err != nil {
		return Sprites{}, err
	}
	return loadSprites(cfg)
}); err != nil {
	panic(err)
}
```

If dependency wiring loops - `A` needs `B`, `B` needs `A` - the world detects the cycle and returns an error instead of recursing forever.

## Error summary

| Situation | Result |
|---|---|
| `Provide` with an already-registered type | error |
| `Resource[T]` for an unregistered type | error |
| constructor fails | error; the resource stays unconstructed and can be requested again |
| circular dependency | error |

See the [world reference](../reference/world.md) for the full method list.
