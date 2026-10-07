# The runner separation

Castrum separates the game from the platform that runs it. `castrum.Game` owns configuration, world state, schedules, and the fixed loop. A `Runner` owns the window, draw surface, input sampling, and backend services, then drives the game.

This boundary keeps the engine backend-free and makes headless or alternate runners possible without changing game state or systems.

## The contract

The public runner interface is intentionally small:

```go
type Runner interface {
	Run() error
}
```

The game hands control to a runner with `g.Run(runner)`. The game accepts that handoff once. The runner is responsible for calling startup and advancing the game according to its platform loop; the game remains responsible for system ordering and simulation time.

A custom runner can use `Startup` and `Advance` directly. That is useful for tests, deterministic harnesses, or a host that already owns the frame clock. It must still preserve the phase and error contracts described by the game API.

## The default runner

`ebitrun` is the shipped Ebitengine runner:

```go
g, err := castrum.New(castrum.WithTitle("My Game"))
if err != nil {
	return err
}

runner, err := ebitrun.New(g,
	ebitrun.WithWindowSize(1920, 1080),
	ebitrun.WithLogicalSize(640, 360),
)
if err != nil {
	return err
}

return g.Run(runner)
```

Window size is the platform-facing size. Logical size is the internal render resolution used for projection and culling; the runner scales that canvas to the window. `WithResizable`, `WithoutVSync`, and `WithAudioSampleRate` are runner concerns for the same reason.

Constructing `ebitrun` also binds runner-owned services to the game's resources. In particular, it registers the audio codecs and provides the texture and audio providers. Explicit audio preloads therefore belong after `ebitrun.New`; see [the asset pipeline](asset-pipeline.md) and the [audio guide](../guides/audio.md).

## Drawing across the boundary

The runner renders the engine's world first. `Runner.AddDraw` registers backend-typed draw callbacks that run afterward, in registration order, once per display frame. With `ebitrun`, a callback receives `*ebiten.Image`, so an overlay written for this runner is not portable to a runner with a different canvas type.

A draw callback can return an error, but the platform draw method cannot return one directly. `ebitrun` stores the first draw error and surfaces it from the next update. Game and setup errors otherwise travel through `g.Run` normally.

## What crosses the boundary

The runner publishes input and logical dimensions through `core.Context`, supplies backend providers for resolved assets, and reads the collected world draw state. It does not need to know the storage layout or schedule implementation. Game systems should use components, queries, resources, and context rather than reaching into runner internals.

The [engine design](engine-design.md) page describes the package direction; the [scheduler](the-scheduler.md) page describes the loop the runner drives.
