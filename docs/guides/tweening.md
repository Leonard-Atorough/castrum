# Tweening

A tween turns elapsed time into an interpolation weight. Use that weight to
move between captured positions, change scale or rotation, or fade a visual
property. Castrum provides the math and a small playhead; your system owns the
endpoints, advances time, and writes the resulting value.

The runnable [tween example](../../examples/tween/main.go) combines fixed-time
movement with a frame-time tint change. Pausing stops movement while the tint
continues. It uses shapes and needs no assets.

## The model

The pieces compose in this order:

```text
elapsed / duration -> normalized time -> easing weight -> interpolated value
```

`tween.Tween` stores duration, elapsed time, and an easing curve. `Update`
advances elapsed time and returns the eased weight; `Progress` reads the same
weight without advancing. Neither method knows what you are animating.

Playback state lives in [tween](../../tween/tween.go); pure easing and
interpolation stay in [geom](../../geom/interpolation.go).

`geom.Lerp` interpolates scalar values, and `geom.Vector2.Lerp` interpolates
positions or scales. Both extrapolate when the weight is outside `[0, 1]`.

The named curves are:

| Family | Functions | Motion |
| --- | --- | --- |
| Linear | `EaseLinear` | Constant speed. |
| Quadratic | `EaseInQuad`, `EaseOutQuad`, `EaseInOutQuad` | Accelerate, decelerate, or do both. |
| Cubic | `EaseInCubic`, `EaseOutCubic`, `EaseInOutCubic` | The same choices with a stronger change in speed. |

These curves map `[0, 1]` to `[0, 1]` and preserve its endpoints. You can call
them directly when another part of your game already owns elapsed time.
They do not clamp their input.

## Create and advance a motion

During setup, capture the start and target positions, create the playhead, and
register its advancing system. With an existing game `g`, this moves a circle
from left to right over four seconds of simulation time:

```go
start := geom.Vector2{X: 240, Y: 260}
end := geom.Vector2{X: 1040, Y: 260}
motion, err := tween.NewTween(4*time.Second, geom.EaseInOutCubic)
if err != nil {
	return err
}
body, err := g.World().NewEntity(
	core.Transform{Position: start},
	render.Sprite{
		Drawable: render.CircleShape{Radii: geom.Vector2{X: 24, Y: 24}},
		Color:    color.NRGBA{R: 100, G: 220, B: 140, A: 255},
	},
)
if err != nil {
	return err
}
if err := g.AddSystem(core.PhaseFixed, "tween.motion", core.SystemFunc(func(ctx *core.Context) error {
	weight, err := motion.Update(ctx.DeltaTime)
	if err != nil {
		return err
	}
	return body.Update(ctx.World, func(t *core.Transform) {
		t.Position = start.Lerp(end, weight)
	})
})); err != nil {
	return err
}
```

Keep `start` fixed throughout the transition. Interpolating from the body's
current position on every update changes the path and no longer describes a
four-second transition between those endpoints.

`NewTween` rejects non-positive durations. Passing a nil curve selects linear
easing. `Update` rejects negative deltas without changing playback state.
The zero value is uninitialized: it reads as progress zero and unfinished,
but attempting to advance it returns an error. `Reset` does not initialize it.

## Choose the clock

The helper does not read wall-clock time or register an engine system. Its
clock is the duration you pass to `Update`.

| Advancing phase | Use | Pause and time scale |
| --- | --- | --- |
| `core.PhaseFixed` | Simulation-affecting movement. | Stops during game pause and follows simulation scale. |
| `core.PhaseFrame` | Presentation-only changes such as tint or opacity. | Continues during game pause and receives unscaled frame time. |

Pass `ctx.DeltaTime` in either phase. Do not multiply the fixed delta by the
game's scale: scaling already changes how many ticks the scheduler runs.
Frame time is clamped by `MaxFrameTime`, and fixed work is limited by
`MaxTicksPerFrame`; a tween does not recover time those guards discard.
See [time control](time-control.md) and the
[scheduler reference](../reference/the-scheduler.md) for the clock contracts.

The example's frame system interpolates the swatch's RGB channels between
orange and blue, then writes `render.Sprite.Color`. Its transform stays fixed.
The selected curve is bounded, so the resulting channels stay within the
color's byte range.

## Keep tweening separate from render interpolation

Advance movement tweens in the fixed phase and write their results to
`core.Transform`, as above. The renderer already smooths position, rotation,
and scale between ticks; you do not need another tween in the draw step.

Use the easing curve for the whole transition, not for `ctx.Alpha`. Alpha
only blends the last two fixed states and should stay linear. Leave
`core.PrevTransform` to the engine, and avoid frame-phase transform writes:
they bypass the fixed-tick workflow that this smoothing relies on.

For a change that should continue while paused, tween a property such as tint
in the frame phase instead. See the
[scheduler reference](../reference/the-scheduler.md) for how snapshots and
alpha work.

## Preserve turns when tweening rotation

To spin an entity, interpolate its rotation in radians with `geom.Lerp`.
In the fixed-phase system above, replace the position update with this
two-turn spin:

```go
return body.Update(ctx.World, func(t *core.Transform) {
	t.Rotation = geom.Lerp(0, 4*math.Pi, weight)
})
```

Add the `math` import and use a non-square `render.RectShape` so the spin is
visible; the example's circle looks the same at every angle.

Angles preserve turns rather than choosing the shortest arc. Choose a target
angle that describes the direction and number of turns you want, and keep
the start angle fixed throughout the tween. Do not wrap each result into
`[0, 2*math.Pi)`: crossing that boundary can make render interpolation show
an unintended turn. The same advice applies to camera rotation.

## Observe completion and restart

`IsComplete` becomes true when elapsed time reaches duration. It remains true
until `Reset`, even if you keep calling `Update`. Excess time beyond completion
is discarded; there is no automatic loop or next transition.

For a one-time reaction, observe completion before and after an update and act
only when it changes from false to true. Alternatively, move your own gameplay
state out of the advancing state when the tween completes. Checking completion
alone on every tick repeats the reaction.

`Reset` sets elapsed time to zero without changing duration or curve. It does
not restore the animated property: your system must apply the new weight.
In the example, restarting while paused resets both playheads and updates the
tint, but the body's position holds until the next fixed tick after resume.

To retarget smoothly, capture the current value as a new start and choose a new
target before resetting. Construct another tween if the duration or curve
must change.

Completion depends on time, not the curve's output. Custom curves may overshoot
or have different endpoint weights; the helper evaluates the curve at normalized
time zero or one and does not clamp its result. Use a pure curve so repeated
observations agree.

## Own the playback state

Give each tween one advancing owner. Several consumers may call `Progress`,
but calling `Update` from two systems counts both supplied durations.
Do not advance the same tween in both frame and fixed phases.

Construction returns a value. Copying it creates independent elapsed-time
state, while copying a pointer shares the same playhead. Copies still share
the curve and any mutable state captured by its closure. Mutation methods use
pointer receivers; when a storage API returns a component copy, write the
updated value back through that API. The [ECS guide](ecs.md) explains component
updates and ownership.

There is no tween manager, target binding, callback, sequence, or looping mode.
Pause an individual tween by not advancing it. Sprite clip playback remains
the separate concern described in the [animation guide](animation.md).

## Run the example

From the repository root:

```text
go run ./examples/tween
```

- Space pauses or resumes simulation.
- 1, 2, and 3 select half, normal, and double simulation speed.
- R restarts both four-second transitions.
- Escape quits.

The moving circle and color swatch hold their endpoints after completion.
The on-screen weights are eased values, not raw elapsed-time percentages.
