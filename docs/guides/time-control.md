# Time control

Castrum separates display frames from fixed simulation ticks. Pause and time
scale control the fixed simulation only, so frame-phase systems, input, and
rendering can continue while the world is paused.

The [time-control example](../../examples/time-control/main.go) puts the pieces
together: Space pauses or resumes, the arrow keys change the scale, and the
debug overlay shows frame and tick rates.

## Pause and resume

Call `Pause` to stop fixed ticks and `Resume` to continue them:

```go
if game.Paused() {
	game.Resume()
} else {
	game.Pause()
}
```

Pause does not reset the time scale or discard the fixed-loop accumulator.
When the game resumes, the next fixed tick is due after the remainder that was
already accumulated plus enough new simulation time to reach one tick. Time
spent paused is not recovered.

Frame-phase systems still run while paused. Put pause-menu and resume controls
in the frame phase; a fixed-phase system cannot resume a paused game because
fixed updates are stopped. The frame phase receives unscaled elapsed time
(clamped by `MaxFrameTime`), and input polling and rendering continue. Timer
and animation systems advance on fixed ticks, so they stop with the simulation
when enabled. See the [timer](./timer.md) and [animation](./animation.md)
guides for their setup and behavior.

Pausing preserves the accumulator, so [`Game.Alpha`](../../castrum.go) stays
at its current interpolation fraction while paused. A renderer using Alpha
therefore keeps the last interpolated position until fixed updates resume.

## Scale simulation time

`SetTimeScale` changes how much elapsed time is added to the fixed-loop
accumulator. The default scale is `1`; a scale of `0.5` accumulates time at
half speed, and `2` accumulates it twice as fast:

```go
if err := game.SetTimeScale(0.5); err != nil {
	return err
}
```

The scale does not change frame-phase `DeltaTime`, nor the fixed tick interval
seen by fixed-phase systems. A value of zero or less returns an error; use
`Pause` to stop simulation time without discarding accumulated time. Changing
the scale while paused remembers it for when the game resumes.

`Advance` runs frame-phase systems before its fixed ticks. Changes made between
calls control the upcoming advance; changes made by a frame-phase system take
effect for the fixed work later in that same call. At high scales, the
`MaxTicksPerFrame` limit still applies: if the limit is reached, Castrum drops
the remaining backlog rather than running an unbounded number of ticks.

Game pause does not automatically pause audio. To pause audio too, use
`PauseAll` and `ResumeAll` on the game's audio `Mixer`; each play's pause mode
determines how it responds. See the [audio guide](./audio.md) for details.

## Inspect time in the debug overlay

Pass `ebitrun.WithDebugOverlay()` when creating the runner to draw `fps` and
`tps` in the window. It also shows `paused` when the game is paused and `scale`
when the time scale differs from `1`. The overlay is opt-in and only reports
state; it does not change simulation timing.

## Run the example

From the repository root:

```text
go run ./examples/time-control
```

Use Space or P to pause and resume. Use the up and down arrows (or W and S) to
speed up or slow down the simulation. The frame-phase controls remain
responsive while paused.