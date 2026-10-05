# Cheat sheet

Recall, not teaching. The patterns are explained in [key concepts](concepts.md) and assembled step by step in [your first game](your-first-game.md).

## Construct and run

```go
g, err := castrum.New(castrum.WithTitle("..."))     // game + world + camera
runner, err := ebitrun.New(g, ebitrun.WithWindowSize(w, h))
err = g.Run(runner)                                  // canonical entry; blocks until quit
```

## Options

Game options - the simulation's identity; all validation happens once, in `New`:

| Option | Sets |
|---|---|
| `castrum.WithTitle` | the game's title |
| `castrum.WithFilesystem` | the `fs.FS` asset paths resolve against ([publishing](../guides/publishing-your-game.md)) |
| `castrum.WithFixedTPS` | simulation ticks per second (default 60) |
| `castrum.WithMaxFrameTime` / `WithMaxTicksPerFrame` | the slow-frame guards ([concepts](concepts.md)) |
| `castrum.WithBindings` | the input actions map ([input](../guides/input.md)) |

Runner options - the platform; a different runner might have no window at all:

| Option | Sets |
|---|---|
| `ebitrun.WithWindowSize` | the OS window size |
| `ebitrun.WithLogicalSize` | the internal render resolution |
| `ebitrun.WithResizable` | whether the window can be resized |
| `ebitrun.WithoutVSync` | disables vsync |
| `ebitrun.WithAudioSampleRate` | the audio mixing rate (default 44100) |

## Spawning

```go
e, err := g.World().NewEntity(componentA, componentB, ...)  // literals, validated
err = e.Update(ctx.World, func(t *core.Transform) { ... })   // get-mutate-set
err = e.AddComponent(ctx.World, extraComponent)             // mid-game attach
```

## Systems and phases

| Phase | Runs | Use for |
|---|---|---|
| `core.PhaseStartup` | once, before the first frame | loading, world setup |
| `core.PhaseFrame` | once per display frame | reading input, UI |
| `core.PhaseFixed` | once per tick (default 60/s) | simulation, movement |

```go
g.AddSystem(core.PhaseFixed, "name", core.SystemFunc(func(ctx *core.Context) error {
	return nil
}))
```

## Queries

```go
q := core.NewQuery(world).With(A{}).Without(B{}).Where(func(e core.Entry) bool { return true })
for e := range q.Execute() { ... }   // build once; no spawning/despawning inside the pass
```

## Resources

```go
w.Provide(func(*core.World) (*T, error) { return &T{}, nil })  // once, before startup
v, err := w.Resource[*T]()                                     // by type
```

## Input

```go
ctx.Actions.Pressed("jump")            // via WithBindings
ctx.Input.KeyPressed(input.KeySpace)   // raw snapshot
ctx.Input.KeyHeld(input.KeyArrowUp)
```

## Audio

```go
shot := audio.NewSource(id)            // valid defaults; Volume must be in (0,1]
_, err := audio.OneShot(ctx.World, shot)
```

Eager audio preloads go after `ebitrun.New` - decoders are runner-registered.

## Quitting

```go
g.Quit()          // from any system; Run returns
g.Quitting()      // poll from other systems
```

## Zero-value rules

Zero values are valid and sensible unless the row says otherwise - the full rules live in [conventions](../guides/conventions.md).

| Value | Zero means |
|---|---|
| `Transform.Scale` | a zero axis reads as unscaled |
| `Sprite` | visible, filled, fully opaque, layer 0 |
| `animation.LoopMode` | `LoopNone` - play once, hold the last frame |
| `Animation.PlaybackMultiplier` | normal rate (0 is not paused) |
| `audio.Source.Volume` | invalid at zero - spawn via `audio.NewSource` |
| `AnimationClip.FPS` | invalid at zero - rejected at `Add` |
| `audio.LoopMode` / `audio.PauseMode` | `LoopNone` / `PauseHolds` |
| `audio.Group` / `audio.LoadMode` | `GroupSFX` / `LoadEager` |
