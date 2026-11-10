# Examples

Each released runnable directory is a standalone Go module pinned to the released Castrum version in its `go.mod`. The released examples currently target v0.3.0 and intentionally use that release's API.

Their module paths use the `github.com/Leonard-Atorough/castrum-examples` namespace so they do not conflict with example packages already included in the Castrum v0.3.0 module.

Work-in-progress examples live under `unreleased/` and use a local `replace` to build against this checkout rather than pinning a released Castrum version. For example:

```sh
cd examples/unreleased/raycast
go run .
```

The [settings example](unreleased/settings/main.go) is a runnable command showing game-owned preferences, the [raycast example](unreleased/raycast) demonstrates collision raycasts, and the [timer](unreleased/timer), [signals](unreleased/signals), [signal-salvage](unreleased/signal-salvage), and [your-first-game](unreleased/your-first-game) examples moved here while they use the unreleased `gametime` package.

Run an example from its directory:

```sh
cd examples/audio/playback
go run .
```

Run its tests from the same directory with `go test ./...`. The `audio/` parent contains shared assets for the audio examples; it is not itself a runnable project.

The runnable projects are:

- `animate`
- `audio/balance`
- `audio/mixing`
- `audio/playback`
- `camera`
- `collision`
- `input`
- `primitive`
- `scenes`
- `text`
- `time-control`
- `tween`
- `wander`

The repository's root Go module tests the engine. CI tests these nested example modules separately.
