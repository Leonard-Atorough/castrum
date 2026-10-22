# Debugging

The debug panel is a small, in-memory diagnostics surface. Game code writes status and messages as data; the optional runner overlay draws them. The panel does not draw or change simulation state, and it is not a persistent logging system.

## The debug panel

`castrum.New` provides a `*debug.Panel` as a world resource. Retrieve it during setup and pass it to systems that report diagnostics:

```go
panel := g.World().MustResource[*debug.Panel]()
```

The panel is backend-independent: systems write data to it, and a viewer decides how to display that data. Use it on the game loop goroutine; `Panel` is not safe for concurrent access.

## Status lines and messages

Use a status line for something that is true now, and a message for something that happened:

```go
panel.Status("controls", "WASD: move")
panel.Message("level loaded")
```

`Status` creates a named slot on its first write. Later writes replace its text without changing its position. The text stays visible until it is updated or cleared; writing an empty string hides the line but keeps its slot, so it can reappear in the same place. Update changing values as the game runs, and write an empty string when a status no longer applies.

`Message` appends to a recent-message list that holds up to eight entries. When it fills, the oldest message is dropped. Write messages when events occur, rather than repeating the same information every frame.

`StatusLines` returns visible status text in first-write order; `Lines` returns retained messages from oldest to newest. Both return independent slices. `Clear` removes all messages and status slots without changing the built-in rate toggles.

## Built-in rates

The panel includes `debug.FPS` and `debug.TPS`, which the overlay shows by default. FPS measures display frames per second; TPS measures fixed simulation ticks per second. Toggle either readout with `Toggle`, and check its current state with `Enabled`:

```go
panel.Toggle(debug.FPS)
```

Use the `debug.FPS` and `debug.TPS` constants for these methods. Undefined `Builtin` values are ignored by `Toggle` and report disabled through `Enabled`.

## The debug overlay

`ebitrun.WithDebugOverlay()` enables the runner's development overlay; it is off by default:

```go
runner, err := ebitrun.New(g, ebitrun.WithDebugOverlay())
if err != nil {
	return err
}
```

The first line shows enabled rate readouts, adding `paused` or `scale` when either value differs from its default. Below it, the overlay draws the panel's status lines and recent messages. It draws after the world and does not change simulation timing or game state. For custom drawing on the backend screen, see [Overlays](rendering.md#overlays).

## Where to go next

- [Timer example](../../examples/timer) - report timer completions and control changes as messages.
- [Input example](../../examples/input) - update live cursor and control status.
- [Audio mixing example](../../examples/audio/mixing) - show live mixer values and pause events.
- [Performance](performance.md) and [Troubleshooting](troubleshooting.md) - investigate frame costs and runtime problems.
