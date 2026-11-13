# Bind and read player actions

Use an action when your game cares about what the player wants to do, rather than which device they use. An action such as `"move_x"` can accept both a keyboard pair and a controller stick; the movement system reads one value.

This guide assumes you have a game setup function and a system that controls an existing entity. The [input example](../../../../examples/input) shows bindings, movement, a mouse-aimed turret, and camera controls together.

## Bind controls when creating the game

Pass your bindings to `castrum.New`. This setup function uses `castrum` and `input`:

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

The game copies the bindings, so later changes to the original map don't change the active controls. To change them while the game is running, see [Rebind an action](rebinding.md).

## Use the actions in a system

For continuous movement, read the axis in a fixed-phase system. Here `player` is an existing entity with a `core.Transform`:

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

Register the returned system in `core.PhaseFixed` and handle the `AddSystem` error. Holding D or moving the stick right moves the player right; A or a leftward stick moves left. The axis is a sampled level, so the system can use it every tick.

For a one-off gameplay action, use the fixed-tick press instead of repeatedly testing a held key:

```go
if ctx.Actions.Pressed("fire") {
	if err := fire(ctx); err != nil {
		return fmt.Errorf("fire: %w", err)
	}
}
```

This excerpt belongs inside a fixed-phase system; `fire` is your own error-returning game function. A short tap between ticks is retained for the next tick. If it begins and ends between ticks, both `Pressed` and `Released` can be true in that tick.

## Choose the right input view

Use `Held` for a button's current level, `Axis` for a signed or analog value, and `Pressed`/`Released` for edges in fixed phases. In frame-phase systems, use `JustPressed`/`JustReleased` for the current frame instead. The game updates the action map and delivers fixed edges; your systems do not call `Update` or `Tick`.

Without bindings or input preferences the action map is nil, and its read methods return zero. If an action doesn't respond, check that it was configured and that its name matches. For cursor position and wheel movement, read the physical snapshot as shown in [Use the mouse](pointer.md).

The existing [input guide](../../../unreleased/guides/input.md) covers controller deadzones, chords, and other binding choices in more depth.
