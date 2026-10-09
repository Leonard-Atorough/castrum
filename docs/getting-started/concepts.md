# Key concepts

Every Castrum game is built from a small set of ideas: the **game** and **world**, **entities** and **components**, **systems**, **queries**, the **fixed timestep**, and **resources**. This chapter introduces each one and ends with a square moving under your control. A [glossary](#glossary) collects the terms at the end.

## The game and the world

Before diving into entities, components, and systems, it's important to understand the foundation. Two objects sit under everything: the **game** and the **world**.

The game - `castrum.New` returns it - owns the process: the options, the systems and their schedules, the loop that runs them, and the runner handoff. When you construct, configure, or register, you are talking to the game: `g.AddSystem(...)`, `g.Run(runner)`.

The world - `g.World()` returns it - owns the state: every entity, every component, every resource. When you create or look things up, you are talking to the world: `g.World().NewEntity(...)`, queries, resources.

The split is deliberate: the game is _what runs_, the world is _what exists_. The same world could be driven by a different game loop; systems receive both at once through the context, and never need the game handle at all.

Everything else in this chapter lives on one side of that line - mostly the world's.

## Entities and components

An entity is a thing in your game: the player, an enemy, a torch on the wall. An entity has no data of its own and no behavior - it is an identity that components attach to.

A component is a piece of plain data describing one aspect of a thing. `Transform` says where it is. `Sprite` says what it looks like. Neither has methods to call and neither inherits from anything. You compose a thing by attaching components to an entity:

```go
_, err := g.World().NewEntity(
	core.Transform{Position: geom.Vector2{X: 640, Y: 360}},
	render.Sprite{Drawable: render.RectShape{Size: geom.Vector2{X: 120, Y: 120}}},
)
```

The same components can be combined freely: a `Transform` with no `Sprite` is a thing that exists but is not drawn; a `Sprite` on a different entity is a second drawable that shares nothing with the first.

Validation happens at the world boundary, not on first access. `NewEntity` checks every component's constraints before the entity exists, and component writes validate again before changing stored data. Bad data is rejected at the earliest moment, not discovered misbehaving mid-game. When writing custom components, implementing the `core.Validatable` interface opts the component into those checks.

Components can have dependencies; dependencies are sometimes automatic: a `Transform` is always attached together with `PrevTransform`, the engine's snapshot the renderer interpolates from. You never write or remove it.

Components hold data; systems provide behavior. Systems read and modify components according to the game's rules, giving behavior to otherwise passive entities.

## Systems

A system is a function that runs on a schedule. It receives the _context_ - the world, the time step, the input - and changes components. All game logic in castrum is a system somewhere.

Register one with a phase and a name:

```go
if err := g.AddSystem(core.PhaseFixed, "move", moveSystem(square)); err != nil {
	return err
}
```

The name appears in error messages, which is reason enough to pick a good one. Systems run in registration order within their phase, and the engine registers its own first - the transform snapshot, the optional subsystem systems their `With*` option enabled, animation and audio among them - so your systems always run after the enabled ones: a `PhaseFixed` system of yours sees animation and audio already advanced for the tick.

There are three phases:

| Phase               | Runs                         | Use for              |
| ------------------- | ---------------------------- | -------------------- |
| `core.PhaseStartup` | once, before the first frame | loading, world setup |
| `core.PhaseFrame`   | once per display frame       | reading input, UI    |
| `core.PhaseFixed`   | once per simulation tick     | gameplay, movement   |

Use `PhaseFixed` for simulation and `PhaseFrame` for work that belongs to the display frame, such as reading raw input or updating UI.

Here is a complete system, the payoff of this chapter. It moves a square left and right:

```go
func moveSystem(square *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		dt := ctx.DeltaTime.Seconds()
		return square.Update(ctx.World, func(t *core.Transform) {
			if ctx.Actions.Held("move_right") {
				t.Position.X += Speed * dt
			}
			if ctx.Actions.Held("move_left") {
				t.Position.X -= Speed * dt
			}
		})
	})
}
```

Three pieces are worth naming:

- The closure captures the entity handle, so the system touches one square. Systems that touch many entities use a query - the [next section](#queries) covers them.
- `Update` is get-mutate-set sugar: it fetches the component, hands it to your function, and stores it back.
- `dt` makes the speed independent of the display rate. The [fixed timestep](#the-fixed-timestep) section explains why.

The `move_right` and `move_left` strings are _actions_ - named inputs the player triggered. They come from bindings registered with `WithBindings`; the [input guide](../guides/input.md) covers them in full.

## Queries

The move system captured one entity handle. Most systems should not: a game has many enemies, many pickups, many of everything, and holding a handle to each is bookkeeping the engine already does. A **query** is a filter over the world that matches entities by their components. Systems declare what they operate on; queries bring the entities.

```go
func tagSystem() core.System {
	var query *core.Query
	return core.SystemFunc(func(ctx *core.Context) error {
		if query == nil {
			query = core.NewQuery(ctx.World).
				With(core.Transform{}).
				Without(render.Sprite{})
		}
		for e := range query.Execute() {
			// e.Component, e.Update, e.SetComponent
		}
		return nil
	})
}
```

`With` requires components, `Without` excludes them, and `Where` filters on anything a component read can express. Inside the loop, the same entity vocabulary the move system used: `Component` reads, `Update` mutates.

Two rules of use:

- Build the query once, not per tick. The `if query == nil` in the closure is the idiom.
- Do not spawn or destroy entities inside the loop - a structural change during iteration is invalid. Collect the handles you need to act on, then act after the pass.

The [ECS guide](../guides/ecs.md) covers the rest of the query surface, including `First`, predicates, and the `Entry` vocabulary.

## The fixed timestep

Game simulation runs in **ticks**: fixed steps of time, 60 per second by default. This is the fixed timestep, and its benefit is determinism: the same inputs produce the same result whether the display runs at 144 Hz or stutters at 30. The engine accumulates real elapsed time each frame and runs however many ticks fit - zero on a fast frame, several on a slow one.

`ctx.DeltaTime` is the step size. Inside `PhaseFixed` it is always exactly one tick interval; inside `PhaseFrame` it is the real frame time. That is why the moving system can multiply by `dt` without caring about the display rate: whatever rate the ticks arrive at, each tick covers the same slice of time.

Drawing happens between ticks, and the renderer interpolates: it draws each moving thing between where it was at the previous tick and where it is now, so motion is smooth at any display rate. Games never do this by hand - every rendered entity already has it.

If a frame runs very long, two guards keep the game stable. `MaxFrameTime` (default 250ms) clamps how much time a single frame may bill, and `MaxTicksPerFrame` (default 5) caps catch-up ticks - the remainder is dropped rather than carried, so a slow frame slows the game down instead of snowballing into worse and worse lag. Both are options, and the defaults are right for almost every game.

## Resources

Components live on entities. Some state should not: the game's configuration, a shared store, a service - one instance for the whole game, not one per entity. A **resource** is typed shared state on the world.

Provide one, resolve it by type anywhere:

```go
if err := w.Provide(func(*core.World) (*GameConfig, error) {
	return &GameConfig{Speed: 240}, nil
}); err != nil {
	return err
}

cfg, err := w.Resource[*GameConfig]()
```

`Resource` returns an error for a type nobody provided. Provide before the startup phase so systems can rely on finding it.

The engine provides resources of its own - the asset server, the clip store, and the audio mixer. The guides reach them through game accessors such as `g.AssetServer()`.

## Glossary

- **Game** - the process: options, schedules, the loop, the runner handoff. Constructed by `castrum.New`.
- **World** - the container of all entities, components, and resources; the state the game drives.
- **Entity** - a thing in the game: an identity with no data or behavior of its own.
- **Component** - plain data attached to an entity, describing one aspect of it (`Transform`, `Sprite`).
- **System** - a function that runs on a schedule and changes components.
- **Query** - a filter over the world that matches entities by their components.
- **Phase** - one of the three schedules systems run on: startup, frame, or fixed.
- **Tick** - one fixed step of the simulation; 60 per second by default.
- **DeltaTime (`ctx.DeltaTime`)** - the step size of the current schedule: one tick interval in the fixed phase, real frame time in the frame phase.
- **Interpolation** - drawing between the previous tick's state and the current one, so motion looks smooth at any display rate.
- **Resource** - typed shared state on the world, one instance per game.
- **Action** - a named input the player triggered, produced from bindings.
- **Runner** - the platform layer: the window, the draw surface, and input. Covered in the [reference tier](../reference/runner-separation.md).

## Where to go next

- [Your first game](your-first-game.md) - every concept in this chapter, assembled into one program step by step.
- [The ECS in depth](../guides/ecs.md) - the full entity vocabulary, component validation, the Context contract, and query discipline.
- [Input](../guides/input.md) - the actions this chapter used, in full.
