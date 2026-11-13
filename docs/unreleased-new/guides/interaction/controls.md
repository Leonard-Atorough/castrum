# Give the player controls

Players might use a keyboard, a mouse, or a controller to do the same thing. Rather than making your game rules depend on a particular device, give each control an **action** name and bind one or more inputs to it. Your systems can then ask whether the player is moving or firing.

This guide uses a player entity that already has a `core.Transform`. If you haven't made one yet, the existing [ECS guide](../../../unreleased/guides/ecs.md) shows how to create entities and add systems. The complete [input example](../../../../examples/input) combines movement, mouse aiming, and camera controls.

## Bind actions to devices

Create the game with bindings for horizontal movement and firing:

```go
func newControlledGame() (*castrum.Game, error) {
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
	g, err := castrum.New(castrum.WithBindings(bindings))
	if err != nil {
		return nil, fmt.Errorf("create game with controls: %w", err)
	}
	return g, nil
}
```

This function uses `fmt` and the `castrum` and `input` packages. A/D form a signed axis, and the left stick offers another way to move. Space or the controller's south button activates `"fire"`. Multiple bindings for an action are alternatives; for axes, the input with the greatest absolute value wins rather than being added to the others.

`castrum.New` validates and copies the bindings. A bad binding fails game creation; changing the original map afterward doesn't change the live controls. Without bindings or input preferences, there is no action map, and its nil-safe reads return zero. If an action never responds, check its name and that the game was created with the intended bindings.

## Read actions in a system

The following fixed-tick system moves a player horizontally and calls your game's firing function on a press. `fire` is a function you supply, for example to create a projectile; it returns any error to the game loop.

```go
func controlPlayer(player *core.Entity, fire func(*core.Context) error) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		distance := ctx.Actions.Axis("move_x") * 180 * ctx.DeltaTime.Seconds()
		if err := player.Update(ctx.World, func(t *core.Transform) {
			t.Position.X += distance
		}); err != nil {
			return fmt.Errorf("move player: %w", err)
		}
		if ctx.Actions.Pressed("fire") {
			if err := fire(ctx); err != nil {
				return fmt.Errorf("fire: %w", err)
			}
		}
		return nil
	})
}
```

Register it after creating the player, with a non-nil `fire` function:

```go
if err := g.AddSystem(core.PhaseFixed, "player.controls", controlPlayer(player, fire)); err != nil {
	return fmt.Errorf("register player controls: %w", err)
}
```

These excerpts use `fmt` and the `core` package. With a visible player and a firing function in place, A/D or the stick should move the player, while each press of Space or the south button calls `fire` once.

`Axis` and `Held` read the latest sampled level. For one-off simulation actions, use `Pressed` and `Released` in fixed phases: the engine collects edges between ticks, so even a short tap can reach the next tick. Both edges can arrive together if a tap begins and ends between ticks. Use `JustPressed` and `JustReleased` in frame-phase systems instead; those describe the current frame. The engine updates and delivers the action map for you.

## Aim at the world with the mouse

The cursor from the default runner is in logical-screen coordinates. To aim at something in the world, convert it using the most recently rendered camera view:

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

This helper uses `errors`, `fmt`, and the `core`, `geom`, and `render` packages. Call it from a system, handle the error, and aim only when the boolean is true. Before the first view is rendered, the resource may not exist yet; skipping that update is normal. The view reflects the last rendered camera, so it matches what the player saw rather than a camera transform changed since then. See [2D space and the camera](../../concepts/2d-space.md) for the coordinate systems.

`ctx.Input` is the runner's read-only physical snapshot. It's useful for cursor position and wheel movement; read the wheel in a frame system, not once per fixed tick, because catch-up ticks share the same frame snapshot.

## Change controls later

For a controls menu in a game configured with bindings, get a copy with `ctx.Actions.Bindings()`, edit it, then apply the complete replacement with `ctx.Actions.SetBindings(...)` and handle its error. The replacement takes effect on the next input update; invalid bindings leave the existing map unchanged. Keep another way to operate the menu while a player is rebinding controls.

The existing [input guide](../../../unreleased/guides/input.md) covers gamepad deadzones, chords, and runtime rebinding in more depth. For saving the player's choices, see [input preferences](../../../unreleased/guides/input.md#store-desired-input-preferences) and the [settings guide](../../../unreleased/guides/settings.md).

Next, make movement matter by [detecting and responding to collisions](collisions.md).
