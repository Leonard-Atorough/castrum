# Examples

Each example runs from the repository root. They are kept current with
the engine's conventions - when the API changes, the examples change
with it.

```sh
go run ./examples/wander
```

## wander

The first-window example: sprites from a grid atlas, a camera that
follows a walking character, and interpolated motion. Start here if
you want to see the smallest game that draws.

```sh
go run ./examples/wander
```

## primitive

The shape drawables as a spectrum: one row per shape kind - rect,
circle/ellipse, line - and one column per variant - filled, outlined,
spinning, stretched, half-faded. Shows that shapes and sprites share
the same ordering and camera. No assets are loaded.

```sh
go run ./examples/primitive
```

## input

Bindings-driven input: WASD or arrows or a stick to drive a tank,
mouse aims the turret, QE zooms. Shows the ActionMap (bindings,
axis actions) and raw snapshot polling side by side.

```sh
go run ./examples/input
```

## animate

Frame animation: a torch flickering from a grid atlas, looping at 10
FPS. Shows the clip store, the Source-style component pattern, and
the renderer staying animation-blind.

```sh
go run ./examples/animate
```

## audio

Audio playback end to end: SPACE plays a sound effect (eager, decoded
once), a music track loops from the file (streamed, never fully in
memory), arrows adjust the master and music volumes, and ENTER
toggles the global pause - the music holds, the sound effect plays
through, because each play states its own pause behavior.

```sh
go run ./examples/audio
```
