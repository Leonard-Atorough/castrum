# Engine design

This page maps the packages that make up Castrum and the responsibility of each layer. It is an architecture reference, not an API index; exported names and signatures belong in the [Go API reference](https://pkg.go.dev/github.com/Leonard-Atorough/castrum).

## Package ownership

| Package            | Responsibility                                                                                            |
| ------------------ | --------------------------------------------------------------------------------------------------------- |
| `castrum`          | Game construction, options, schedules, fixed-loop control, and engine-system registration.                |
| `core`             | The public world model: entities, components, queries, resources, phases, systems, and context.           |
| `asset`            | Filesystem-backed asset loading, typed decoding, caching, and atlas registration.                         |
| `animation`        | Atlas-backed clips and the animation component/system.                                                    |
| `audio`            | Source state, one-shots, mixer state, and the backend-free audio controller contract.                     |
| `input`            | Raw snapshots and named action maps.                                                                      |
| `ebitrun`          | The default Ebitengine runner: window, draw surface, input polling, texture conversion, and audio codecs. |
| `internal/ecs`     | The private component storage implementation.                                                             |
| `internal/runtime` | The private schedule implementation.                                                                      |

The public packages describe game state and engine contracts. `ebitrun` binds those contracts to a platform backend. The `internal/` packages are implementation details and are not extension points for game code.

## Dependency direction

The center of the engine is backend-free. `core` does not import Ebitengine, and `castrum` owns the game loop without knowing how a window or sound device works. The default runner imports the backend and provides the platform behavior over the engine's public seams.

This direction is tested rather than left as a convention: the root package's dependency test rejects backend dependencies in the core engine. A backend change should therefore be localized to the runner unless it changes a public contract.

## Options at the edge

`castrum.New` accepts functional options such as `WithTitle`, `WithFilesystem`, and `WithFixedTPS`. It applies those options, validates the resulting values, and creates the game from plain configuration. Invalid options fail at construction instead of being rediscovered by each downstream subsystem.

Runner options are separate because they describe the platform: `ebitrun.New` owns window size, logical resolution, resizing, vsync, and audio sample rate. A different runner can expose a different set without changing the game configuration.

## Systems all the way down

The engine's behavior uses the same system mechanism available to games. Transform snapshotting, input publication, animation advancement, and audio reconciliation are registered into schedules at construction time. User systems are then registered after the engine systems and run in phase order.

There is no separate privileged update path for ordinary engine behavior. The [scheduler](the-scheduler.md) explains the resulting ordering and timing guarantees.

## Shared state

Cross-cutting services live as typed world resources. `Game.AssetServer`, `Game.Clips`, and `Game.Mixer` are setup conveniences for resources that systems can resolve through `World.Resource`. This gives each world its own service instances and avoids package globals.

The [world and storage](world-and-storage.md) page covers resource resolution and ownership. The [runner separation](runner-separation.md) explains which services are supplied only after a runner is constructed.

## Stable versus private seams

Game code should extend the public vocabulary: components for per-entity state, resources for world-owned state, systems for behavior, and runner callbacks for backend-specific drawing. It should not depend on archetype storage, schedule slices, or runner internals.

Those private implementations can be reorganized while preserving the public contracts documented by the guides and this reference tier.
