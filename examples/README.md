# Examples

Each runnable directory is a standalone Go module pinned to the released Castrum version in its `go.mod`. The examples currently target v0.3.0 and intentionally use that release's API.

Their module paths use the `github.com/Leonard-Atorough/castrum-examples` namespace so they do not conflict with example packages already included in the Castrum v0.3.0 module.

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
- `signal-salvage`
- `signals`
- `text`
- `time-control`
- `timer`
- `tween`
- `wander`
- `your-first-game`

The repository's root Go module tests the engine. CI tests these nested example modules separately.
