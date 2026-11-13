# Input

Players might move with a keyboard or a controller, aim with a mouse, and change their controls in a settings menu. Castrum gives you two ways to read that input: **actions** name what the player wants to do, while a **snapshot** describes the current physical devices.

Use actions for controls that can have multiple bindings or be remapped. Read the snapshot for device-specific data such as the cursor or wheel. This guide assumes an existing game and systems that act on its entities. The [input example](../../../../examples/input) brings movement, mouse aiming, and camera controls together; the [rebinding example](../../../../examples/rebinding) demonstrates a settings editor.

## Bind controls to actions

Give each action a name your game understands, then bind one or more physical inputs to it when creating the game. This setup function uses `castrum` and `input`:

```go
func newGame() (*castrum.Game, error) {
	bindings := input.Bindings{
		"move_x": {
			input.KeyPairInput{Negative: input.KeyA, Positive: input.KeyD}.Binding(),
			input.PadAxisInput{Axis: input.PadLeftStickX}.Binding(),
		},
		"fire": {
			input.KeyInput{Key: input.KeySpace}.Binding(),
			input.PadButtonInput{Button: input.PadSouth}.Binding(),
		},
	}
	return castrum.New(castrum.WithBindings(bindings))
}
```

Return the error to your caller: game creation rejects malformed binding structures. A/D and the left stick both control `"move_x"`; Space and the south controller button both activate `"fire"`. Inputs in the same list are alternatives. When more than one axis contributes, the value with the greatest absolute magnitude wins.

The game copies the bindings, so changing the original map later does not change the active controls. The engine creates the action map and updates it for you; systems read it through `ctx.Actions`.

## Use actions in a system

For continuous movement, read an axis in a fixed-phase system. Here `player` is an existing entity with a `core.Transform`:

```go
func movePlayer(player *core.Entity) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		distance := ctx.Actions.Axis("move_x") * 180 * ctx.DeltaTime.Seconds()
		return player.Update(ctx.World, func(t *core.Transform) {
			t.Position.X += distance
		})
	})
}
```

Register the returned system in `core.PhaseFixed` and handle the `AddSystem` error. Holding D or moving the stick right moves the player right; A or a leftward stick moves left. An axis is a sampled level, so the system can use it every tick.

For a one-off gameplay action, read a press rather than repeatedly testing whether the button is held:

```go
if ctx.Actions.Pressed("fire") {
	if err := fire(ctx); err != nil {
		return fmt.Errorf("fire: %w", err)
	}
}
```

This excerpt belongs inside a fixed-phase system; `fire` is your own error-returning game function. A short tap between ticks is retained for the next tick. If it begins and ends between ticks, both `Pressed` and `Released` can be true in that tick.

### Choose the right input view

Use `Held` for a button's current level and `Axis` for a signed or analog value. For edges in fixed phases, use `Pressed` and `Released`; frame-phase systems use `JustPressed` and `JustReleased` for the current frame. The game updates the action map and delivers fixed edges; your systems do not call `Update` or `Tick`.

Without bindings or input preferences the action map is nil, and its read methods return zero. If an action does not respond, check that it was configured and that its name matches. A scene transition flushes pending edges before tick delivery.

## Choose bindings for your controls

Bindings can use keys, mouse buttons, controller buttons, controller axes, or a pair of keys as an axis. `Pad: 0` accepts any connected controller; positive pad numbers address specific connected controllers. The standard button names describe positions rather than vendor-specific labels. Stick axes range from -1 to 1, with positive Y downward; triggers range from 0 to 1.

The default axis deadzone is `0.15`: values below it read as zero. Use input preferences to change the map-wide deadzone. An axis binding can also specify a direction or its own nonzero deadzone. The [input example](../../../../examples/input) demonstrates keyboard and gamepad controls together.

For a combination that should activate only when all its inputs are active, use an experimental chord:

```go
bindings := input.Bindings{
	"dash": {
		input.Chord(
			input.KeyInput{Key: input.KeyA}.Binding(),
			input.KeyInput{Key: input.KeyB}.Binding(),
		),
		input.PadButtonInput{Button: input.PadEast}.Binding(),
	},
}
```

Here A and B together **or** the controller button activate dash. A chord needs at least two valid physical bindings, cannot contain another chord, and contributes a digital state rather than an axis value. The action's binding list still represents alternatives. `KeyInput` also supports modifier keys when a key should require them.

## Aim at a world position

The runner reports the cursor in logical-screen coordinates. For aiming or selecting objects in the world, convert it through the last rendered camera view:

```go
func cursorInWorld(ctx *core.Context) (geom.Vector2, bool, error) {
	view, err := ctx.World.Resource[*render.CameraView]()
	if errors.Is(err, core.ErrResourceNotRegistered) {
		return geom.Vector2{}, false, nil
	}
	if err != nil {
		return geom.Vector2{}, false, fmt.Errorf("get camera view: %w", err)
	}
	return view.ScreenToWorld(ctx.Input.Cursor(), ctx.LogicalWidth, ctx.LogicalHeight), true, nil
}
```

The helper uses `errors`, `fmt`, and the `core`, `geom`, and `render` packages. Call it from a system, handle the error, and use the position only when the boolean is true. Before the renderer has produced its first view, the resource may not exist; skipping that update is normal.

The view represents the most recently rendered camera, including its framing, so the picked position matches what the player saw. It can differ from a camera transform changed since that draw. [2D space and the camera](../../concepts/2d-space.md) explains the coordinate conversion; the [collision guide](../collision/collision.md#find-something-along-a-line) shows how a world position can become a raycast target.

### Read the mouse wheel

In a frame-phase system, `ctx.Input.Wheel()` gives the current runner update's signed scroll offset. Use it as a displacement, not as a held state or a rate to multiply by `DeltaTime`; it can be fractional and resets when there is no scrolling. The [input example](../../../../examples/input) uses it to zoom.

Read wheel movement in a frame phase rather than applying it once per fixed tick. Catch-up ticks share the same frame snapshot, while an update with no fixed tick can be replaced before a tick reads it. `ctx.Input` is runner-owned and read-only; snapshot reads do not consume the value.

## Let players change their controls

To rebind an action, edit a copy of the active bindings and apply the replacement. This helper assumes the game was created with bindings and that `actions` is its non-nil `ctx.Actions`:

```go
func bindFireToKey(actions *input.ActionMap, key input.Key) error {
	if key <= input.KeyNone || key >= input.NumKeys {
		return fmt.Errorf("cannot bind fire to key %v", key)
	}
	bindings := actions.Bindings()
	bindings["fire"] = []input.Input{
		input.KeyInput{Key: key}.Binding(),
	}
	if err := actions.SetBindings(bindings); err != nil {
		return fmt.Errorf("rebind fire: %w", err)
	}
	return nil
}
```

The function uses `fmt` and `input`. Call it after the player confirms a key. It rejects `KeyNone` and out-of-range keys, which cannot activate the action. It replaces **all** bindings for `"fire"` with that key; preserve or add controller bindings if you want to keep them. Other actions remain in the copied map.

`Bindings` returns a copy. Editing it alone does not affect the game, and `SetBindings` validates the binding structure. An invalid replacement leaves the current controls unchanged, but `SetBindings` does not reject every nonfunctional key value; check captured input as above. A successful change takes effect on the next input update, not immediately in the current snapshot.

Keep controls for confirming or cancelling rebinding available independently of the action being edited. The [rebinding example](../../../../examples/rebinding) reserves raw keys for its editor and demonstrates confirm, cancel, and persistence.

### Keep controls between runs

Changing the live action map does not save it. For durable settings, maintain an input preference snapshot, validate and apply it, then save it using your project's storage. Restore the snapshot before creating the game or apply it to the action map later.

`input.NewPreferences` creates a validated, copied preference group from bindings; its `Value` supplies a snapshot for `castrum.WithInputPreferences` at startup. When the player confirms a change, update the group and apply its value with `ctx.Actions.SetPreferences`. Applying preferences and saving them are separate operations, so handle errors from both. The [rebinding example](../../../../examples/rebinding) shows the whole workflow; the [settings guide](../../../unreleased/guides/settings.md#save-and-restore-engine-preferences) covers storage and restoration.
