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
- `ChordInput` requires every member binding to be active together; construct it with `Chord` (experimental).

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

## Chords (experimental)

Use `input.Chord` for a combination that should press when its last requirement becomes active and release when any requirement becomes inactive:

```go
bindings := input.Bindings{
	"dash": {
		input.Chord(
			input.KeyInput{Key: input.KeyA},
			input.KeyInput{Key: input.KeyB},
		),
		input.PadButtonInput{Button: input.PadEast},
	},
}
```

Either key may be held first. In this example, A and B together or the gamepad's east button activate dash: members inside a chord combine with AND, while the action's binding list still combines with OR. Each binding contributes its own press and release edges, so releasing one binding may report a release while another keeps the action held.

Chords may mix keyboard, mouse, gamepad button, and axis bindings. An axis member counts as active after its deadzone and direction filtering; a key pair counts as active when its keys do not cancel. The chord itself is digital and contributes no `Axis` value. Members using `Pad: 0` must all match the same connected controller. Explicit pad addresses may intentionally select different controllers.

Chords require at least two members and cannot contain other chords or refer to named actions. Use value-form bindings, not pointers. `input.New` and `SetBindings` reject malformed chords and invalid deadzones; a failed replacement leaves the existing bindings and state unchanged.

## Axes and gamepads

Gamepad buttons use the standard position names `PadSouth`, `PadEast`, `PadWest`, and `PadNorth`, so bindings are not tied to vendor glyphs. `Pad: 0` means any connected gamepad; `Pad: 1` addresses the first connected pad, `Pad: 2` the second, and so on. The current implementation supports four pads.

Axis bindings use the normalized backend value. Stick axes range from `-1` to `1`; stick Y is positive downward. Triggers range from `0` to `1`. If an action has multiple axis bindings, the value with the greatest absolute magnitude wins rather than the values being added.

The default axis deadzone is `0.15`. Values below the deadzone read as zero; crossing into or out of the active range produces the action's press or release edge. The game-level `WithBindings` option uses this default. Code that owns an `input.ActionMap` directly can pass a deadzone to `input.New`; zero selects the default and values outside `[0, 1]` are rejected.

The experimental `PadAxisInput.Deadzone` field overrides the map's deadzone for that binding:

```go
"throttle": []input.Input{
	input.PadAxisInput{Axis: input.PadRightTrigger, Deadzone: 0.05},
},
```

Zero inherits the map's deadzone; there is no separate zero-deadzone override. Deadzones must be finite and in `[0, 1]`. A value exactly at the threshold is active, and values outside the deadzone retain their original magnitude rather than being rescaled.

`Axis` is a latest-sampled level, not a latched event or accumulated displacement. Fixed catch-up ticks share the latest frame's value; ticking does not clear it. If an axis returns to zero between ticks, its press and release edges can both reach the next tick while `Axis` already reads zero.

## Rebinding

The action map copies the bindings when it is created, so changing the original map or slices later does not silently change the game. For runtime remapping, replace the map through `ctx.Actions.SetBindings`:

```go
err := ctx.Actions.SetBindings(input.Bindings{
	"jump": []input.Input{
		input.KeyInput{Key: input.KeyJ},
	},
})
```

The replacement takes effect on the next input update. Chords restart inactive, so a newly satisfied chord can produce a fresh press. `ctx.Actions.Bindings()` returns a copy suitable for displaying or editing in a settings screen; mutating that returned value, including chord member slices, does not mutate the action map.

## Raw snapshots

Use `ctx.Input` when you need a physical device directly, such as cursor position or the current update's scroll offset:

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

## Mouse wheel: frame-scoped displacement

The default runner polls the wheel once per runner update, before the frame phase. `ctx.Input.Wheel()` returns that update's signed X/Y scroll offsets, not a held state, velocity, or accumulated position. Offsets may be fractional or larger than one; do not assume one unit per wheel notch. Positive Y is scroll up and negative Y is scroll down. An update with no scrolling reports zero.

Read `ctx.Input.Wheel()` in `PhaseFrame` for UI scrolling and other presentation controls. Reads do not consume the snapshot: every reader sees the same value. A runner update is not necessarily a rendered frame, so drawing the latest snapshot is useful for diagnostics but is not a lossless input handler.

Ebitengine accumulates wheel offsets over its input sampling interval before the runner reads them. The snapshot therefore contains the net displacement, not just one wheel event. Opposite movements within that interval can cancel; event order is not preserved. Castrum copies this value rather than accumulating it again across updates.

```go
// Inside a PhaseFrame system:
scroll := ctx.Input.Wheel()
```

Handle the displacement once per frame-phase update, without multiplying it by `DeltaTime` or time scale. The [input example](../../examples/input) does this for wheel zoom alongside rate-based Q/E controls. Frame systems continue running during `Game.Pause`, so mouse-based presentation controls need no special pause buffer or replay policy.

The snapshot is frame-scoped even when read by fixed systems: catch-up ticks share it, while an update without a tick may be replaced before any fixed system reads it. Applying its wheel offset once per fixed tick would repeat movement during catch-up. That is not duplicate delivery by the poller. Castrum does not currently provide tick-scoped mouse input, wheel action bindings, or an ordered input-event queue.

## Pick world positions

The default runner reports the cursor in logical-screen coordinates, not window/device pixels, including when the window is scaled. Convert it with the render-published view rather than reconstructing a `CameraView` from component state:

```go
func aimSystem() core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		view, err := ctx.World.Resource[*render.CameraView]()
		if errors.Is(err, core.ErrResourceNotRegistered) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read camera view: %w", err)
		}

		aim := view.ScreenToWorld(
			ctx.Input.Cursor(),
			ctx.LogicalWidth,
			ctx.LogicalHeight,
		)
		aimAt(aim)
		return nil
	})
}
```

The resource contains the most recently drawn, interpolated view, including its zoom, rotation, anchor, and bounds clamp. It is one frame stale for frame and fixed readers, and is not registered until the renderer first collects a view. Skipping `core.ErrResourceNotRegistered` on that first tick is intentional; do not use `MustResource` here. Picking through this view is more accurate than reading current camera components because the aim matches what is on screen.

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
