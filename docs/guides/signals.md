# Signals

If you're coming from an engine like Godot, the idea is familiar: one part of the game reports that something happened, and other parts react. The difference is that Castrum doesn't call connected listeners. It stores typed values in a hub, and systems pull unread values from their readers when their update runs.

Readers don't compete for a shared queue. Each reader has its own cursor, so multiple systems can react to the same signal independently.

The runnable [signal example](../../examples/signals) demonstrates how to react to an engine signal, define and emit game signals, and carry a sequence across fixed and frame systems.

## The signal model

The hub is an engine-owned resource on the game's world. `castrum.New` provides it, so you can get it with `MustResource`:

```go
hub := g.World().MustResource[*signal.Hub]()
```

A signal is a Go value that represents something that happened. A small struct is a useful convention; the type identifies the stream, and no registration step is needed:

```go
type coreCharged struct{}

type launched struct {
	tick uint64
}
```

Each reader is for one signal type and keeps its own position in that stream. A new reader starts at the beginning of the current window, so it can read signals that were emitted before the reader was created.

## React to a signal

If a system reacts to one signal type, `Hub.Consumer` is usually the shortest path. It creates a reader for the system and calls your handler for each unread signal.

### The common case: a consumer system

In the example, the sequence system consumes `coreCharged` and starts the countdown:

```go
if err := g.AddSystem(core.PhaseFixed, "core.sequence", hub.Consumer(
	func(ctx *core.Context, _ coreCharged) error {
		state.phase = countdown
		return state.countdown.Update(ctx.World, func(t *timer.Timer) {
			t.Running = true
		})
	},
)); err != nil {
	return err
}
```

Each call to `Consumer` has an independent reader. Systems consuming the same signal type can each receive the signal while it is still in the window; they don't take turns consuming a shared queue.

### Reading several streams in one system

When one system needs more than one signal type, create a reader for each type and read each stream from the system's update:

```go
chargedFrame := hub.NewReader[coreCharged]()
launchedFrame := hub.NewReader[launched]()

if err := g.AddSystem(core.PhaseFrame, "core.announce", core.SystemFunc(
	func(ctx *core.Context) error {
		for _, ok := chargedFrame.Next(); ok; _, ok = chargedFrame.Next() {
			state.bannerText = "CORE CHARGED"
		}
		for event, ok := launchedFrame.Next(); ok; event, ok = launchedFrame.Next() {
			state.bannerText = fmt.Sprintf("LAUNCHED on tick %d", event.tick)
		}
		return nil
	},
)); err != nil {
	return err
}
```

## Emit a signal

You can also define and emit your own signals using the hub. For example, to emit the `coreCharged` signal when the core reaches full charge:

```go
if state.charge >= maxCharge {
	hub.Emit(coreCharged{})
}
```

`Emit` stores the value in the current window; it does not call readers immediately. `Reader.Next` reports whether a signal is available, so an empty read is expected. Whether a system gets a chance to read a signal depends on which window it is in and when that system runs. The next two sections explain those parts.

## Lifetime: the tick and frame windows

The game loop runs pre-frame and frame schedules followed by zero or more fixed ticks. The frame window opens before pre-frame systems and remains open through frame systems; frame signals can be read by later systems in either schedule and expire after the frame schedule ends.

Signals emitted during a fixed tick go into the tick window. At the end of that tick, the hub moves them into the frame window and clears the tick window. This means a tick signal can be read by a fixed system during that tick and by a pre-frame or frame system in the next display frame, using independent readers. If several fixed ticks run before the next frame, their signals collect in the frame window together.

The hub starts in frame mode. In normal game updates, the engine switches to tick mode after running pre-frame and frame systems, then flushes each tick into the frame window after fixed systems run.

## Ordering

Systems in a phase run in registration order. For same-tick delivery, register the emitter before the consumer: the consumer then reads after the signal has been emitted and can react in that tick.

If the consumer runs first, its read finds no signal. That is safe, but the system is not run again just because another system emits later. The signal still moves to the frame window at the end of the tick, where a frame reader can handle it in the next display frame; the earlier fixed-phase consumer will not see it on its next tick.

## Errors

`Consumer` returns the first handler error from its system update. The signal passed to the handler is already consumed; later unread signals remain available on the reader's next update. When reading manually, `Next` consumes a signal before returning it, so return handler errors from the system update rather than discarding them.

## Engine signals

Several engine features emit typed signals. Their guides explain each payload, when it is emitted, and how to consume it:

| Feature   | Signal                                  | Details                                                                     |
| --------- | --------------------------------------- | --------------------------------------------------------------------------- |
| Timers    | `timer.Completed`                       | [Timer signals](timer.md#signals)                                           |
| Animation | `animation.Completed`                   | [Animation signals](animation.md#signals)                                   |
| Collision | `collision.Entered`, `collision.Exited` | [Contact edge signals](collision.md#react-to-enters-and-exits-with-signals) |
| Audio     | `audio.Finished`                        | [Audio completion](audio.md#play-a-sound-effect)                            |

## Troubleshooting

- **The handler never runs:** check that the emitted and consumed types match exactly, that the system runs in a phase where the signal is available, and that it has not expired before the reader checks it. `Next` returning false is expected when no signal is waiting.
- **Another consumer seems to miss the signal:** create a separate `Consumer` or `Reader` for each consumer. Sharing one reader splits delivery because its cursor is shared.
- **A signal is emitted from another goroutine:** keep hub access on the game loop. The hub does not synchronize concurrent `Emit` and `Next` calls.

## Where to go next

For timer setup and behavior, see the [timer guide](timer.md); for how systems and phases are scheduled, see [The ECS in depth](ecs.md).
