# The ECS in depth

The [concepts chapter](../getting-started/concepts.md) introduces entities, components, systems, queries, and resources. This guide is the working reference for the `core` package: create world state during setup, validate it at the world boundary, then read and change it from systems.

## The world owns state

Create a world with `core.NewWorld`. The world owns entity storage and typed resources. A game normally creates it as part of `castrum.New`, then passes it to setup code and systems through `*core.Context`.

An entity is an identity, not a container. `World.NewEntity` stores the supplied components and returns a handle for convenient follow-up operations:

```go
player, err := world.NewEntity(
	core.Transform{Position: geom.Vector2{X: 100, Y: 80}},
	Health{Max: 100, Current: 100},
)
if err != nil {
	return fmt.Errorf("spawn player: %w", err)
}
```

The handle contains its `EntityID` and a local liveness flag; it does not contain the world or component storage. Component methods therefore receive the world explicitly:

```go
health, ok := player.Component[Health](world)
if ok {
	health.Current -= 10
	if err := player.SetComponent(world, health); err != nil {
		return err
	}
}
```

Use `Update` for the common get-mutate-set operation. Use `AddComponent` and `RemoveComponent` to change an entity's component set. `SetComponent` only replaces a component that is already present. `NewEntities` creates a batch with the same component set and follows the same validation rules as `NewEntity`.

`World.DestroyEntity` removes the entity from storage and kills the supplied handle. `Kill` only changes the handle's local state; it does not remove anything from the world. `EntityID` values are sequential and are not recycled during a world's lifetime. IDs collected from queries are safe to retain; create a handle with `core.NewEntity(id)` when an ID needs the entity helper methods again.

## Components and validation

Components are plain data. Attach several components to compose an entity instead of putting behavior or a world reference inside a component. Components with invariants can implement `core.Validatable`:

```go
type Health struct {
	Max     float64
	Current float64
}

func (h Health) Validate() error {
	if h.Max <= 0 {
		return fmt.Errorf("health max must be positive")
	}
	if h.Current < 0 || h.Current > h.Max {
		return fmt.Errorf("health current must be within [0, max]")
	}
	return nil
}
```

`NewEntity` validates components before allocating an entity ID. Component writes validate before mutation, so a rejected `SetComponent`, `Update`, or `AddComponent` leaves the previous stored value unchanged. A nil component is rejected. Decide deliberately whether a component's zero value is valid, useful as a default, or invalid; the [conventions guide](conventions.md) covers those choices.

`Transform` has one engine-managed dependency: spawning or adding a `Transform` also adds a `PrevTransform` snapshot. The fixed-loop capture system updates that snapshot before gameplay systems run. Game code reads `PrevTransform` for inspection but does not write or remove it; an entity without both components cannot be interpolated by the renderer.

## Systems and context

The `core` package defines the system contract; the game schedules systems in startup, frame, and fixed phases. `SystemFunc` adapts a function to that contract:

```go
func moveSystem(player *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		dt := ctx.DeltaTime.Seconds()
		return player.Update(ctx.World, func(t *core.Transform) {
			t.Position.X += 120 * dt
		})
	})
}
```

`Context` is read-only system input. It includes the world, frame and fixed-tick counters, the current `DeltaTime`, interpolation `Alpha`, logical render dimensions, and input snapshots. `DeltaTime` is the display-frame duration in `PhaseFrame` and the fixed tick interval in `PhaseFixed`. Register systems in dependency order within a phase: a reader must follow the system that writes the value it consumes. See the [scheduler reference](../reference/the-scheduler.md) for phase order and runner details.

Capture configuration, entity IDs, and reusable queries in the system closure. A query can be initialized on the first call, after setup has finished, and reused on subsequent calls.

## Resources

Resources are typed, world-owned singletons for configuration or shared services that should not be attached to an entity. `Provide` registers a lazy constructor. The constructor runs the first time `Resource` requests that type and receives the owning world:

```go
if err := world.Provide(func(*core.World) (*GameConfig, error) {
	return &GameConfig{Speed: 240}, nil
}); err != nil {
	return err
}

config, err := world.Resource[*GameConfig]()
if err != nil {
	return err
}
```

A resource is cached after successful resolution, so later requests return the same instance. A missing registration, constructor error, duplicate registration, or circular dependency returns an error. A failed lazy constructor is not cached and may be retried. Use `ProvideEager` with `ResolveEager` when setup should resolve a resource before the loop starts; eager resolution stops at the first error.

## Queries

Create a reusable query from a world. `With` requires component types, `Without` excludes component types, and `Where` applies a per-entity predicate:

```go
var wounded *core.Query

system := core.SystemFunc(func(ctx *core.Context) error {
	if wounded == nil {
		wounded = core.NewQuery(ctx.World).With(Health{})
	}
	for entry := range wounded.Execute() {
		entry.Update(func(h *Health) {
			h.Current -= 10
			if h.Current < 0 {
				h.Current = 0
			}
		})
	}
	return nil
})
```

Pass zero values as type markers. `With(nil)`, `With(&Health{})`, and their `Without` equivalents panic because query component types must be non-nil, non-pointer values. `Entry.Component`, `Entry.SetComponent`, and `Entry.Update` can access only types listed in `With`; attempting to write an unprefetched type panics. `First` returns the first matching entry and a boolean indicating whether one was found.

Entries are valid only during their iteration. Their component columns are shared scratch storage reused across archetypes, so retain `entry.ID()` rather than retaining an `Entry`. Query order is deterministic while the world is unchanged, and a query rematches its archetypes when the world's structure changes. One query supports one active iteration; use a separate query for nested iteration.

Value writes during `Execute` are supported. Structural changes are not: do not spawn, destroy, add components, or remove components during a pass. Collect IDs, finish the pass, then mint handles or apply structural changes afterward.

## Where to go next

- [Conventions](conventions.md) - zero values and validation rules.
- [Performance](performance.md) - query reuse and hot-path guidance.
- [Engine design](../reference/engine-design.md) - world storage and scheduling internals.
