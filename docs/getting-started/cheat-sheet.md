# Cheat sheet

Recall, not teaching. The patterns are explained in [key concepts](concepts.md) and assembled step by step in [your first game](your-first-game.md).

## Construct and run

```go
g, err := castrum.New(castrum.WithTitle("..."))     // game + world + camera
runner, err := ebitrun.New(g, ebitrun.WithWindowSize(w, h))
err = g.Run(runner)                                  // canonical entry; blocks until quit
```

## Options

Common game options:

| Option                                              | Sets                                                                                      |
| --------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `castrum.WithTitle`                                 | the game's title                                                                          |
| `castrum.WithFilesystem`                            | the `fs.FS` used to resolve asset paths ([publishing](../guides/publishing-your-game.md)) |
| `castrum.WithFixedTPS`                              | simulation ticks per second (default 60)                                                  |
| `castrum.WithMaxFrameTime` / `WithMaxTicksPerFrame` | the slow-frame guards ([concepts](concepts.md))                                           |
| `castrum.WithBindings`                              | the input actions map ([input](../guides/input.md))                                       |

Common runner options:

| Option                        | Sets                                  |
| ----------------------------- | ------------------------------------- |
| `ebitrun.WithWindowSize`      | the OS window size                    |
| `ebitrun.WithLogicalSize`     | the internal render resolution        |
| `ebitrun.WithResizable`       | whether the window can be resized     |
| `ebitrun.WithoutVSync`        | disables vsync                        |
| `ebitrun.WithAudioSampleRate` | the audio mixing rate (default 44100) |

## Spawning

```go
e, err := g.World().NewEntity(componentA, componentB, ...)
err = e.Update(ctx.World, func(t *core.Transform) { ... })
err = e.AddComponent(ctx.World, extraComponent)
```

## Systems and phases

| Phase               | Runs                         | Use for              |
| ------------------- | ---------------------------- | -------------------- |
| `core.PhaseStartup` | once, before the first frame | loading, world setup |
| `core.PhaseFrame`   | once per display frame       | reading input, UI    |
| `core.PhaseFixed`   | once per tick (default 60/s) | simulation, movement |

```go
g.AddSystem(core.PhaseFixed, "name", core.SystemFunc(func(ctx *core.Context) error {
	return nil
}))
```

## Queries

```go
q := core.NewQuery(world).With(A{}).Without(B{}).Where(func(e core.Entry) bool { return true })
for e := range q.Execute() { ... }
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

| Binding                  | Use                                         |
| ------------------------ | ------------------------------------------- |
| `input.KeyInput`         | one keyboard key, optionally with modifiers |
| `input.KeyPairInput`     | two keys as a signed axis                   |
| `input.MouseButtonInput` | a mouse button                              |
| `input.PadButtonInput`   | a gamepad button                            |
| `input.PadAxisInput`     | a gamepad axis                              |

See the [input guide](../guides/input.md) for action state and binding examples.

## Audio

```go
shot := audio.NewSource(id)
_, err := audio.OneShot(ctx.World, shot)
```

See the [audio guide](../guides/audio.md) for persistent sources, loading, and mixer controls.

## Quitting

```go
g.Quit()          // from any system; Run returns
g.Quitting()      // poll from other systems
```
