# Input

Castrum exposes input in two layers. The runner publishes a read-only `input.Snapshot` containing the current state of every device. Optional bindings resolve those physical inputs into named actions, so gameplay systems can depend on roles such as `"jump"` or `"move_x"` instead of on a particular key or controller.

Use bindings for controls that players may remap or that have more than one physical input. Read the raw snapshot for one-off device data such as the cursor position, mouse wheel, or a feature that does not need an action name.

## Configure bindings

Pass `WithBindings` when creating the game. Each action maps to one or more physical inputs; any bound input can activate the action.

```go
bindings := input.Bindings{
	"move_x": []input.Input{
		input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD},
		input.PadAxisInput{Axis: input.PadLeftStickX},
	},
	"jump": []input.Input{
		input.KeyInput{Key: input.KeySpace},
		input.PadButtonInput{Button: input.PadSouth},
	},
}

g, err := castrum.New(
	castrum.WithBindings(bindings),
)
```

The game creates the action map and publishes it as `ctx.Actions`. The engine owns its update order: it resolves the current snapshot before frame systems run, then delivers accumulated press and release edges before fixed systems run. Games do not need to call `Update` or `Tick` themselves.

The binding types are:

- `KeyInput` binds one key and can require up to three modifier keys. Empty modifier slots are ignored.
- `KeyPairInput` turns two keyboard keys into a signed axis: `Negative` contributes `-1` and `Positive` contributes `+1`.
- `MouseButtonInput` binds a mouse button.
- `PadButtonInput` binds a standard-layout gamepad button.
- `PadAxisInput` binds a gamepad axis, optionally limiting it to the negative or positive half with `Direction`.

See the [cheat sheet](../getting-started/cheat-sheet.md) for the complete key, mouse, button, and axis vocabulary.

## Read actions

Frame systems use the frame view:

```go
if ctx.Actions.JustPressed("jump") {
	startJump()
}
if ctx.Actions.JustReleased("jump") {
	stopCharging()
}
if ctx.Actions.Held("move_x") {
	move(ctx.Actions.Axis("move_x"), ctx.DeltaTime)
}
```

`JustPressed` and `JustReleased` describe edges in the current display frame. `Held` describes the current level. `Axis` returns a value in the input's natural range, normally `-1` to `1`, and `Duration` returns how long the action has been continuously held in seconds.

Fixed systems use `Pressed` and `Released` for tick-stable edges:

```go
func updatePlayer(ctx *core.Context) error {
	if ctx.Actions.Pressed("jump") {
		jump()
	}
	return nil
}
```

The engine latches edges between fixed ticks. A tap that begins and ends between two ticks still delivers both `Pressed` and `Released` in the next tick, and every fixed system sees the same delivered state. Within that tick, the state remains unchanged until the next tick consumes the latch. Use frame edges for presentation or frame-rate input, and fixed edges for simulation decisions.

Without `WithBindings`, `ctx.Actions` is nil, but its query methods are nil-safe and return zero values. No action map resource is created in that case.

## Keyboard and modifiers

Plain keys use the zero-value modifier array. A chord fires only while every listed modifier is held:

```go
"save": []input.Input{
	input.KeyInput{
		Key:       input.KeyS,
		Modifiers: [3]input.Key{input.KeyControlLeft},
	},
},
```

If a modifier is released while the trigger key remains held, the action stops being held without producing a release edge. Bind left and right modifier keys separately when both should be accepted.

## Axes and gamepads

Gamepad buttons use the standard position names `PadSouth`, `PadEast`, `PadWest`, and `PadNorth`, so bindings are not tied to vendor glyphs. `Pad: 0` means any connected gamepad; `Pad: 1` addresses the first connected pad, `Pad: 2` the second, and so on. The current implementation supports four pads.

Axis bindings use the normalized backend value. Stick axes range from `-1` to `1`; stick Y is positive downward. Triggers range from `0` to `1`. If an action has multiple axis bindings, the value with the greatest absolute magnitude wins rather than the values being added.

The default axis deadzone is `0.15`. Values below the deadzone read as zero; crossing into or out of the active range produces the action's press or release edge. The game-level `WithBindings` option uses this default. Code that owns an `input.ActionMap` directly can pass a deadzone to `input.New`; zero selects the default and values outside `[0, 1]` are rejected.

## Rebinding

The action map copies the bindings when it is created, so changing the original map or slices later does not silently change the game. For runtime remapping, replace the map through `ctx.Actions.SetBindings`:

```go
err := ctx.Actions.SetBindings(input.Bindings{
	"jump": []input.Input{
		input.KeyInput{Key: input.KeyJ},
	},
})
```

The replacement takes effect on the next input update. `ctx.Actions.Bindings()` returns a copy suitable for displaying or editing in a settings screen; mutating that returned value does not mutate the action map.

## Raw snapshots

Use `ctx.Input` when you need a physical device directly or need continuous data such as cursor and wheel movement:

```go
if ctx.Input.KeyPressed(input.KeySpace) {
	// fire
}
if ctx.Input.KeyHeld(input.KeyArrowUp) {
	// adjust
}

cursor := ctx.Input.Cursor()
wheel := ctx.Input.Wheel()
aim := ctx.Input.PadAxis(0, input.PadRightStickX)
```

Snapshot reads are nil-safe and bounds-safe. A missing runner snapshot, an invalid key or button, and an out-of-range pad all read as inactive or zero. The runner owns the snapshot; systems should treat it as read-only. Raw snapshot methods include keyboard, mouse buttons, cursor, wheel, gamepad connection, gamepad buttons, and gamepad axes.

## Direct action maps

Most games use the engine-owned map configured with `WithBindings`. The lower-level `input.ActionMap` is available for custom loops and tests:

```go
actions, err := input.New(bindings, 0)
if err != nil {
	return err
}

actions.Update(snapshot, deltaSeconds) // once per frame
actions.Tick()                         // once per fixed tick
```

`input.New` rejects nil bindings and deadzones outside `[0, 1]`. A nil snapshot is treated as every input being off. Call `Update` before reading frame queries and `Tick` before reading fixed-tick edge queries.

## Where to go next

The [input example](../../examples/input) shows keyboard, mouse, camera, and gamepad input together. The [first-game tutorial](../getting-started/your-first-game.md) shows the smallest named-action setup. The [audio guide](audio.md) uses the same fixed-versus-frame distinction for gameplay timing.
