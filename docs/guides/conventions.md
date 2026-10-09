# Conventions and best practices

The engine follows a handful of rules everywhere. Learn them once, here, and you will recognize them in every guide. This page is the rulebook and the day-to-day checklist; the [reasoning behind these rules](../reference/core-principles.md) can be found in the core principles reference.

## Finish setup before the loop

`castrum.New` applies defaults, validates options, creates the world, and provides the engine's asset server, clip store, and audio mixer. Use that setup window to register assets and clips, spawn initial entities, configure the camera, and register systems. The runner should be the last setup step:

```go
func run() error {
	g, err := castrum.New(
		castrum.WithTitle("my game"),
		castrum.WithBindings(bindings),
	)
	if err != nil {
		return err
	}

	if err := g.World().MustResource[*asset.Server]().RegisterGridAtlas("characters", "characters.png", 16, 16, "char"); err != nil {
		return err
	}
	player, err := g.World().NewEntity(core.Transform{})
	if err != nil {
		return fmt.Errorf("spawn player: %w", err)
	}
	if err := g.AddSystem(core.PhaseFixed, "player.move", moveSystem(player)); err != nil {
		return err
	}

	runner, err := ebitrun.New(g)
	if err != nil {
		return err
	}
	return g.Run(runner)
}
```

The runner calls `Startup` before entering its platform loop. Startup systems run once, and eager resources are resolved before they run. `g.Run` is the canonical handoff and can be called only once. For headless tests or custom runners, call `Startup` and `Advance` directly instead of inventing a second game loop.

Capture setup state in system constructors. A system can retain an entity handle, configuration, or service pointer; it should not rediscover setup state on every tick.

## The zero value works

A component in its zero value is usable and means something sensible. `Sprite{}` is a sprite with its style defaults and an empty picture - legal, just invisible. `Transform{}` sits at the origin, unscaled.

So partial literals are idiomatic. Write the fields you care about and leave the rest at zero:

```go
render.Sprite{Color: color.White} // everything else zero, everything valid
```

Some fields have no sensible zero, and the engine says so at spawn. An `audio.Source` with an empty `Audio` is a spawn error, and its `Volume` must be in (0, 1] - a silent play is a bug in the spawn, and the engine treats it like one. These components meet you halfway: a constructor supplies the minimal valid form (`audio.NewSource`), or the component is meaningless without one named field (for example, `Animation.Clip`).

Between the two sit fields where zero means "the sensible default." A zero `Transform.Scale` axis reads as 1. A zero `PlaybackMultiplier` plays at the normal rate. The loop and pause enums name their zero value for the common case - `LoopNone`, `PauseHolds` - so the field reads truthfully at rest.

The test for which class a field falls into: read the field's name at zero, and see whether it tells the truth.

## Validation at the boundary

A component with value constraints implements `core.Validatable`, and the engine calls `Validate` whenever the component enters storage - at spawn, and again at every write: `SetComponent`, `Update`, and `AddComponent`.

So a component that would misbehave silently stays out of the world: an `Animation` with an empty clip, or a `Source` with a negative playback multiplier. Game-defined components can implement `Validatable` too - the engine checks for the interface, whatever the type.

The point of validating at the boundary is when the error surfaces. A spawn-time error fails in setup code, with a message naming the rule and the value that broke it. The same mistake found mid-frame is a stutter in a shipped game; found at the boundary, it is a line in your terminal while the stack still points at the mistake.

## Choose the phase deliberately

The game has three phases:

| Phase | Use it for |
| --- | --- |
| `core.PhaseStartup` | One-time setup and loading. |
| `core.PhaseFrame` | Display-frame work and raw per-frame input. |
| `core.PhaseFixed` | Simulation, movement, and gameplay rules. |

`PhaseFrame` runs once per display frame with elapsed frame time. `PhaseFixed` runs zero or more times after it, with a stable `DeltaTime` derived from `FixedTPS` (60 by default). Put simulation in the fixed phase so it does not depend on display refresh rate. Read named actions in the phase appropriate to the behavior; the [input guide](input.md) covers how action edges are carried into fixed ticks.

Systems run in registration order within a phase. Register a reader after the system that writes the state it needs. The engine's built-in systems are registered during `New`, before game systems are added; this includes transform snapshots, the optional subsystem systems their `With*` option enabled, and input systems when bindings are configured. Do not rely on ordering between systems that are not explicitly ordered by registration.

## Errors are values

Engine APIs return errors. Spawn an invalid component, load a missing asset, or register a duplicate clip - an error comes back, wrapped with the operation that produced it.

The template for game code:

```go
if _, err := g.World().NewEntity(/* components */); err != nil {
	return fmt.Errorf("spawning the player: %w", err)
}
```

The one accepted panic lives in `main`. A game that failed to construct has nothing left to recover into, so `main` panics the returned error and takes the stack trace with it. Everywhere else, errors travel up as values - errors from the loop surface from `g.Run`.

```go
func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
```

## State, not commands

The engine's systems make the world match what components declare. A `Sprite` with a picture renders. An `Animation` advances. An audio `Source` plays. Steering any of them is field writes: pause, resume, retarget, restart - all plain state on the component.

Instead of `play()`, `advance()`, or `redraw()` calls, the game writes state and the engine observes it on the next tick. Playback control is field writes, and the world catches up in the phase that owns the component.

Keep world and resource access on the game loop's goroutine. Setup happens before the runner takes control; systems, drawing callbacks, and resource access during the loop use the same ownership boundary. Do not read or mutate engine state from an unsynchronized background goroutine. If background work produces data, hand that data back explicitly and apply it from setup or a system.

## Keep durable entity IDs

An `Entity` handle is convenient immediately after spawning, but it is only a lightweight handle. Query entries are temporary because their component columns are reused during iteration. Keep `entry.ID()` when an entity must be remembered, then use that ID for later world operations or create a fresh handle with `core.NewEntity(id)` when an entity method is more convenient.

Do not retain a query `Entry` or use it after the query pass. Do not structurally change the world while iterating: spawning, destroying, adding, or removing components moves storage rows under the iterator. Finish the pass first, then apply those changes.

## Keep queries narrow and reusable

Create a query once and reuse it. `With` and `Without` express structural requirements, while `Where` handles a value-dependent condition:

```go
func damageSystem() core.System {
	var query *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if query == nil {
			query = core.NewQuery(ctx.World).With(Health{})
		}
		for entry := range query.Execute() {
			entry.Update(func(health *Health) {
				health.Current -= 10
			})
		}
		return nil
	})
}
```

Do not rebuild a query every tick. Do not structurally change the world during `Execute`: spawning, destroying, adding, or removing components moves storage rows under the iterator. Collect `EntityID` values, finish the pass, then mint handles or apply those changes. Value writes through `Entry.Update` and `Entry.SetComponent` are supported.

Keep query component reads and writes limited to the types listed in `With`.

## Make resource ownership clear

Use a component for per-entity state and a resource for one world-owned instance. Register game resources with `World.Provide` during setup, then resolve them by type. Use `ProvideEager` and `ResolveEager` when a constructor must succeed before startup; ordinary `Provide` is lazy and resolves on the first `Resource` call.

The engine exposes its shared services directly through `Game.AssetServer`, `Game.Clips`, and `Game.Mixer`, while the same instances are also world resources for systems. Prefer these accessors in setup code and the world resource API inside systems when that is the natural dependency boundary.

Use one filesystem root consistently. `WithFilesystem` accepts an `fs.FS`; the default resolves paths from the process working directory, while `embed.FS` makes the asset set part of a single binary. Register atlases and clips before entities or systems depend on them. The [assets](assets.md), [animation](animation.md), and [audio](audio.md) guides cover loading and streaming choices.

## Choose input explicitly

Use `WithBindings` when gameplay should read named actions from `ctx.Actions`. Use `ctx.Input` when a system needs raw device state such as cursor position or a key that does not belong in an action map. Without bindings, no `ActionMap` resource is created and `ctx.Actions` is nil; raw input still works when the runner publishes a snapshot.

Keep input sampling and simulation responsibilities separate: read frame-level information in `PhaseFrame`, and apply gameplay state changes in `PhaseFixed` when the behavior should advance at the fixed rate. The [input guide](input.md) shows both styles in one working example.

## What comes next

- [Performance](performance.md) - what the conventions cost and buy in the frame budget.
- [The examples](../../examples) - every rule above, in working programs.
- The [reasoning](../reference/core-principles.md) behind each rule - the reference tier.
- [ECS](ecs.md), [assets](assets.md), [animation](animation.md), and [audio](audio.md) for focused feature guidance.
