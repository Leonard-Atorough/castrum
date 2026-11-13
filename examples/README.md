# Examples

Each runnable directory is a standalone Go module. Examples on `main` follow the engine's development API; release tags preserve examples pinned to that exact Castrum release.

Their module paths use the `github.com/Leonard-Atorough/castrum-examples` namespace so they do not conflict with example packages already included in the Castrum v0.3.0 module.

For development, create a local Go workspace from the repository root:

```sh
go work init .
go work use -r ./examples
go work use ./templates/starter
```

If the workspace already exists, skip `init`. The ignored workspace resolves Castrum to the checkout without changing each module's dependency requirement. The current `unreleased/` directories participate in the same workspace and release pinning; their location does not select a different dependency policy.

For example:

```sh
cd examples/unreleased/raycast
go run .
```

The [settings example](unreleased/settings/main.go) shows game-owned preferences, the [raycast example](unreleased/raycast) demonstrates collision raycasts, and [timer](unreleased/timer), [signals](unreleased/signals), [signal-salvage](unreleased/signal-salvage), and [your-first-game](unreleased/your-first-game) cover simulation time, signals, and a complete game.

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

The repository's root Go module tests the engine. PR and release CI build, vet, and test all nested example modules separately against the checkout.

For standalone released examples, check out a release tag and disable workspaces with `GOWORK=off`. Running development source without the workspace may fail against its currently recorded release dependency. See [Contributing](../CONTRIBUTING.md#develop-examples-against-the-checkout) for the release and validation lifecycle.
