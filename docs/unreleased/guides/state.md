# State

Castrum uses a finite state machine to manage and define the different modes or states of your game. 

You define the possible states as a `comparable` type, such as an `int` or `string`, create a machine with an initial state, and register it with the engine using `WithState`. Systems then interact with the machine to request transitions and respond to state changes. You can also register hooks against states to run code when entering or exiting them, such as switching scenes or initializing gameplay elements.

When a state transition is requested during gameplay it is queued for processing. The transition doesn't happen until the engine processes the request on the next display frame.

## Configure a machine

To begin, define a comparable type for your machine, such as a named `int` or `string`. `state.New[GameMode]()` starts in that type's zero value. Chain `WithInitial` to choose a different starting value, then register the machine with the engine using `WithState`:

```go
type GameMode int

const (
	ModeMenu GameMode = iota
	ModePlaying
	ModeGameOver
)

modes := state.New[GameMode]().WithInitial(ModeMenu)
g, err := castrum.New(
	castrum.WithState(modes),
	castrum.WithBindings(input.Bindings{
		"start": {input.KeyInput{Key: input.KeySpace}},
	}),
)
if err != nil {
	return err
}
```

The examples below use this `g` and `modes`. Register their systems and hooks during setup, before starting the runner. See [Input](input.md) for named actions and [the engine concepts](../getting-started/concepts.md) for running the game. For a working game that combines modes, scenes, and signals, see [Signal Salvage](../../../examples/unreleased/signal-salvage/main.go).

`WithState` registers the machine's system for startup and each display frame. Keep the `modes` pointer to use it from your systems, or retrieve the same machine from the world when you need it elsewhere:

```go
modes := g.World().MustResource[*state.Machine[GameMode]]()
```

That resource is guaranteed when you configure the game with `WithState`. The engine rejects a nil machine during construction. Without the option, it does not register or process your machine.

`WithInitial` is setup configuration, not a transition. It returns a new machine with copies of the registered hooks, leaving the original unchanged. Pending requests and initialization progress are not copied. Keep the returned pointer and pass that machine to the engine; calling `WithInitial` on an existing machine does not reset it.

## Request a transition

For a menu that starts the game when Space is pressed, register a frame system that checks the current mode and requests the next one:

```go
if err := g.AddSystem(core.PhaseFrame, "game.start", core.SystemFunc(
	func(ctx *core.Context) error {
		if modes.Is(ModeMenu) && ctx.Actions.JustPressed("start") {
			return modes.Request(ModePlaying)
		}
		return nil
	},
)); err != nil {
	return err
}
```

`Is` compares a value with the current state; `Current` returns that state. After this system requests `ModePlaying`, both still report `ModeMenu` until the engine processes the transition on the next display frame.

### If another request is already waiting

The machine holds one pending destination, not a queue of transitions. Requesting that destination again succeeds without adding work. Requesting a different destination returns an error wrapping `state.ErrTransitionPending` and leaves the earlier request intact.

Return request errors from systems, as above, or use `errors.Is(err, state.ErrTransitionPending)` if you need to handle a conflict explicitly. To inspect the waiting destination, call:

```go
next, pending := modes.Pending()
```

Only use `next` when `pending` is true. A successful `Request` means the request was accepted, not that the transition completed.

Requesting the current state is always a no-op, even when another destination is pending. It neither cancels that request nor reruns the current state's hooks. To restart the current state, call `Restart`:

```go
if err := modes.Restart(); err != nil {
	return err
}
```

The restart is queued for the next processing pass. Repeated restart requests coalesce. If another destination is already pending, `Restart` returns an error wrapping `state.ErrTransitionPending` and leaves that destination intact.

### Run gameplay in the appropriate mode

A mode change does not automatically pause, remove, or disable gameplay systems. Use `Is` inside a system to decide whether its work should run. For example, a fixed system that updates the player can begin with:

```go
if !modes.Is(ModePlaying) {
	return nil
}
```

The rest of that system then runs only while the machine is in `ModePlaying`. Read `Current` or `Is` in fixed systems too; they report the same machine state as frame systems. Replacing a scene changes its entities, not the systems registered with the engine.

## Use hooks to switch scenes

Register `OnEnter` and `OnExit` hooks for work that should happen once when a mode begins or ends. Hooks receive a `*core.Context` and return an error, just like a system.

For example, make entering the menu or playing mode switch to the appropriate scene. Add `castrum.WithScenes()` alongside `WithState` and `WithBindings` in the construction above, then retrieve the director and register the hooks:

```go
director := g.World().MustResource[*scene.Director]()

menu := &scene.Blueprint{
	Name:       "menu",
	EntityList: []scene.EntitySpec{{Name: "menu-root"}},
}
level := &scene.Blueprint{
	Name:       "level",
	EntityList: []scene.EntitySpec{{Name: "player"}},
}

modes.OnEnter(ModeMenu, func(*core.Context) error {
	_, err := director.Switch(menu)
	return err
})
modes.OnEnter(ModePlaying, func(*core.Context) error {
	_, err := director.Switch(level)
	return err
})
```

These minimal blueprints create named entities; add your menu and player components as described in [Scenes](scenes.md).

The menu's enter hook runs once at startup. When the start system later requests `ModePlaying`, the next frame's state pass runs the playing hook. Use `OnExit` for cleanup that belongs to leaving a mode.

For each transition, the engine runs:

1. The old state's exit hooks, in registration order.
2. The assignment to `Current` and the change signal. For a restart, the value stays the same and the signal's `From` and `To` are equal.
3. The new state's enter hooks, in registration order.

Exit hooks see the old mode; enter hooks see the new one. A hook can request another transition on the same machine, but that request waits until its next processing pass. Hooks do not recursively process requests, and the same pending-request rules apply.

### When hooks and scene operations run

At startup, the engine runs the initial state's enter hooks before queued scene operations and user startup systems. A request made during setup does not skip that initial entry; it waits until the first display frame.

With systems added normally, each display frame runs in this order:

```text
update bound input actions
process state transitions and their hooks
run user pre-frame systems
apply queued scene operations
run user frame systems
run any fixed ticks due
```

This is why a scene request from a state hook can be applied before normally appended gameplay systems run that frame. State hooks also see the current frame's named actions. Relative registration can insert a user system before an ordinary input, state, or scene system and change what that user system observes. Requests from ordinary frame or fixed systems wait until the next display frame's state pass.

State processing continues when the fixed simulation is paused and is not scaled by the game's time scale. To pause simulation rather than change game modes, see [Time control](time-control.md). For the full phase order, see [The scheduler](../reference/the-scheduler.md).

### A mode change is not a scene transaction

`Director.Switch` validates and queues a scene request. Entity creation and scene hooks run later, and can still fail after the mode has changed. A scene failure does not restore the previous mode, and side effects of state hooks are not rolled back.

Keep scene-switching work in state hooks when gameplay needs the replacement that frame. A scene request made by an ordinary frame signal consumer comes after the scene system and normally waits until the next frame. See [Queue scene requests](scenes.md#queue-scene-requests) for scene validation, instance IDs, and rollback rules.

## Observe lifecycle signals

Hooks coordinate the transition itself. Use signals when other systems should independently react to what happened, such as logging mode changes:

```go
hub := g.World().MustResource[*signal.Hub]()
if err := g.AddSystem(core.PhaseFrame, "game.mode-log", hub.Consumer(
	func(_ *core.Context, event state.Changed[GameMode]) error {
		log.Printf("game mode changed: %v -> %v", event.From, event.To)
		return nil
	},
)); err != nil {
	return err
}
```

This snippet uses the standard library's `log` package. Each `Consumer` has its own reader, so several systems can observe the same transition.

There are two state signals:

- `state.Initialized[GameMode]` carries `State`. It is emitted once, after the initial state's enter hooks succeed.
- `state.Changed[GameMode]` carries `From` and `To`. It is emitted after exit hooks succeed and `Current` is assigned, **before** the destination's enter hooks run. A restart also emits this signal, with equal `From` and `To` values.

Initial entry emits `Initialized`, not `Changed`: starting in the menu is not a transition from some other mode. Use a separate initialization consumer if you need that notification. Requests and no-ops emit neither signal. Signal types include the state type, so `Changed[GameMode]` and `Changed[string]` are separate streams.

In normal engine startup, initialization is available to later startup systems and first-frame consumers. Transition signals are available to later pre-frame, frame, and post-frame systems. Both expire at the frame's end, before fixed ticks: read `Current` or `Is` in fixed systems rather than relying on a frame notification.

A change signal reports the assignment, not successful enter hooks or scene replacement. An error can stop the game update before later consumers run, even though the signal was emitted. For reader lifetimes and delivery windows, see [Signals](signals.md).

## Handle transition failures

Return errors from hooks instead of discarding them. They propagate through the state system to `Startup` or `Advance`, and normally to `g.Run`, with the phase, system name, and failing hook identified.

What remains current depends on where the failure occurs:

| Failing hook | Result |
| --- | --- |
| Initial enter | The initial mode remains current; no `Initialized` is emitted. |
| Exit | The old mode remains current; no `Changed` is emitted. |
| Enter after a transition | The new mode remains current; `Changed` has already been emitted. |

In each case, remaining hooks stop. Failed transitions are consumed, and failed hooks are not retried automatically. Work done by earlier hooks is not undone. Requests made by hooks remain pending even if that hook or a later hook fails.

Avoid nil hooks: registration does not reject them, but reaching one during processing returns an error.

## Register multiple machines

Use separate machines when your game has independent state, such as its game mode and connection status. Give each a distinct state type and repeat `WithState`. This construction is an alternative to the single-machine setup above:

```go
type ConnectionState int

const (
	Offline ConnectionState = iota
	Online
)

connection := state.New[ConnectionState]()
g, err := castrum.New(
	castrum.WithState(modes),
	castrum.WithState(connection),
	castrum.WithBindings(input.Bindings{
		"start": {input.KeyInput{Key: input.KeySpace}},
	}),
)
if err != nil {
	return err
}
```

The connection starts in `Offline`, its type's zero value. You can retrieve each machine by its resource type: `*state.Machine[GameMode]` or `*state.Machine[ConnectionState]`. Construction rejects duplicate machine resource types, including registering the same pointer twice. Distinct named types identify independent machines; aliases do not create new resource types.

Machines run in option order, before scene operations. Each handles the request pending when its turn begins. A hook can request a transition on a later machine for processing that same frame. Requests to its own machine or an already-processed machine wait until the next pass. If a machine fails, later machines do not run and their requests remain pending; several machines' transitions are not one transaction.

## Know the machine's limits

A machine accepts any value of its comparable state type. Constants make your modes readable, but do not enforce an allowed-state list or a transition graph. Put game-specific transition rules in the systems that request changes. There is no built-in state stack or cancellation method.

Configure the machine during setup, then access it and its hooks only from the game loop's goroutine. The machine's zero value works too and starts in the state type's zero value.

For standalone scheduler use, `state.NewSystem(machine)` needs a world containing a non-nil `*signal.Hub`. It resolves and caches the hub before running initial hooks or consuming transitions. A resolution error leaves initialization and pending intent untouched. `castrum.New` supplies the hub, so no extra signal setup is needed for this guide's examples.
