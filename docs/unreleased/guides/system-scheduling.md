# Scheduling systems

The scheduler runs named systems in phases. Choose a phase for when the work belongs, then arrange systems in dependency order within that phase. A reader must run after the system that writes the value it reads.

## Choose a phase

Startup runs once after eager resources resolve:

| Phase | When it runs | Typical use |
| --- | --- | --- |
| `core.PhasePreStartup` | Before startup systems | Initial state hooks before startup scene work. |
| `core.PhaseStartup` | Once, before the first frame | One-time setup and loading. |
| `core.PhasePostStartup` | After startup systems | Work that depends on startup setup. |

Each display frame runs frame phases in order, followed by zero or more fixed ticks:

| Phase | When it runs | Typical use |
| --- | --- | --- |
| `core.PhasePreFrame` | Once per display frame, before frame systems | State transitions before frame scene work. |
| `core.PhaseFrame` | Once per display frame, before fixed ticks | UI, presentation, and frame-level input. |
| `core.PhasePostFrame` | Once per display frame, after frame systems and before fixed ticks | Frame-level observation or cleanup. |

Each fixed tick runs its phases in order:

| Phase | When it runs | Typical use |
| --- | --- | --- |
| `core.PhasePreFixed` | Once per tick, before fixed systems | Work that must precede all fixed systems. |
| `core.PhaseFixed` | Once per tick | Simulation, movement, and gameplay rules. |
| `core.PhasePostFixed` | Once per tick, after fixed systems | Tick-level observation or cleanup. |

A frame may have no fixed ticks or several catch-up ticks. Frame phases still run while the game is paused; fixed phases do not. `ctx.DeltaTime` is elapsed frame time in frame phases and the fixed tick interval in fixed phases. `PhasePreFixed` runs before fixed-phase engine systems such as transform capture and tick input, while `PhasePostFixed` runs after fixed systems. See [the scheduler reference](../reference/the-scheduler.md) for accumulator limits, interpolation, and signal windows.

## Register in dependency order

By default, `Game.AddSystem` appends a system to the selected phase, so registering a producer before its consumers puts it first. To change the order, insert a new system before or after an already registered system:

```go
func registerPlayerSystems(g *castrum.Game, player *core.Entity) error {
	prepare := core.SystemFunc(func(ctx *core.Context) error {
		return player.Update(ctx.World, func(t *core.Transform) {
			if t.Position.X < 0 {
				t.Position.X = 0
			}
		})
	})
	move := core.SystemFunc(func(ctx *core.Context) error {
		return player.Update(ctx.World, func(t *core.Transform) {
			t.Position.X += 100 * ctx.DeltaTime.Seconds()
		})
	})
	clamp := core.SystemFunc(func(ctx *core.Context) error {
		return player.Update(ctx.World, func(t *core.Transform) {
			if t.Position.X > 500 {
				t.Position.X = 500
			}
		})
	})
	if err := g.AddSystem(core.PhaseFixed, "player.move", move); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "player.prepare", prepare, castrum.Before("player.move")); err != nil {
		return err
	}
	return g.AddSystem(core.PhaseFixed, "player.clamp", clamp, castrum.After("player.move"))
}
```

The resulting order is `player.prepare`, `player.move`, then `player.clamp`. `castrum.Before` and `castrum.After` insert a new system relative to an existing regular system in the same phase. The target must already be registered; the internal first/last lifecycle hooks are not valid targets. These options insert new systems rather than moving existing ones or declaring a dependency graph, so cycles cannot be formed. Repeated `After` registrations for the same target place the newest system closest to that target.

Names appear in schedule errors and must be non-empty and unique within each phase. `AddSystem` returns an error for an invalid phase, an empty or duplicate name, a missing target, or invalid or conflicting options. Specify at most one position option and one non-empty group. A system error stops the current phase and propagates through `Startup` or `Advance`, and through the runner's `Run`. Engine features register their own systems where needed. Systems added normally after `castrum.New` run after its systems in the same phase, but relative options can insert before ordinary engine systems. Preserve dependencies such as input publication before input consumers and transform capture before gameplay movement.

## Add systems while the game runs

Systems may register other systems with `AddSystem`. Register all startup-phase systems before `Startup` begins; registrations in any startup phase are rejected once it starts, even from a startup system. If a system adds work to the frame or fixed phase that is currently running, that new system is eligible the next time that phase runs, not later in the active pass. If it adds work to a later frame or fixed phase, that phase can still run later in the same frame or tick. For example, a frame system can register a fixed system that runs if a fixed tick is due later in that `Advance` call. Startup systems may also register frame and fixed systems. Registration and group changes must not run concurrently with each other or with schedule execution.

## Group related systems

Use `castrum.InGroup` with `Game.AddSystem` when several systems should be enabled or disabled together, including systems in different phases. Groups start enabled, and a group's enabled state applies to systems registered in it later:

```go
func registerCombat(g *castrum.Game, enemy *core.Entity) error {
	prepare := core.SystemFunc(func(ctx *core.Context) error {
		return enemy.Update(ctx.World, func(t *core.Transform) {
			if t.Position.X < 0 {
				t.Position.X = 0
			}
		})
	})
	update := core.SystemFunc(func(ctx *core.Context) error {
		return enemy.Update(ctx.World, func(t *core.Transform) {
			t.Position.X += 20 * ctx.DeltaTime.Seconds()
		})
	})
	if err := g.AddSystem(core.PhaseFixed, "enemy.update", update, castrum.InGroup("combat")); err != nil {
		return err
	}
	if err := g.AddSystem(core.PhaseFixed, "enemy.prepare", prepare, castrum.Before("enemy.update"), castrum.InGroup("combat")); err != nil {
		return err
	}
	return g.SetGroupEnabled("combat", false)
}
```

Combine `InGroup` with `Before` or `After` to place grouped systems relative to an existing system. `SetGroupEnabled` returns an error for an empty or unregistered group. A change affects systems that have not yet run in the current phase. Scene transitions do not change group state; use explicit group control when systems should pause together.

For frame versus fixed input timing, see the [input guide](input.md). For state transitions before scene operations, see the [state guide](state.md).
