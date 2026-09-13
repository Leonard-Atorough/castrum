---
name: engine-architecture-migration
description: >
  Reference architecture for migrating Castrum toward its v1 game-engine
  layout. Consult when moving packages, deciding public versus internal APIs,
  changing Game or World ownership, adding engine services, or migrating
  subsystem backends such as input, rendering, audio, UI, assets, and physics.
---

# Castrum Engine Architecture Migration

This skill is the standing reference for Castrum's v1 package layout and
ownership model. Use it before moving code or exposing a new engine service.
Migrate in small, tested slices; do not perform a broad package rewrite merely
to make the tree match this document.

## Core architecture

Castrum has three roots of responsibility:

```text
Game
  owns runtime and platform services
  owns one World and long-lived service lifetimes

World
  owns entities, components, hierarchy, systems, and typed resources
  is the simulation boundary passed to gameplay systems

Scene
  organizes world entities and controls lifecycle/transitions
  does not own the World
```

The central rule is:

> `Game` is the application root; `World` is the simulation root; systems
> receive `World`, not `Game`.

Do not introduce package-level global singletons. Multiple `Game` or `World`
instances must remain possible for tests, headless simulation, replay, and
future multiplayer use.

## Target repository layout

The target v1 tree is:

```text
castrum.go                 # Game root and Ebiten lifecycle
config.go                  # Public configuration
errors.go                  # High-level engine errors

ecs/                       # Stable gameplay-facing ECS API
components/                # Built-in data components
geom/                      # Stable math and geometry
scene/                     # Public scene definitions and transitions
events/                    # Public typed event API
assets/                    # Public asset loading and caching API
animation/                 # Public animation definitions and playback
physics/                   # Public collision types and queries
input/                     # Public physical-input abstractions and actions
atlas/                     # Public runtime atlas representation

internal/
  runtime/                 # Clock, fixed timestep, phases, loop orchestration
  ecs/                     # Archetypes, registries, storage implementation
  input/                   # Ebiten polling and backend translation
  render/                  # Renderer orchestration and GPU-facing details
  physics/                 # Spatial index and physics implementation details
  assets/                  # Worker lifecycle, cache internals, hot reload
  platform/                # Window, audio device, and OS integration

cmd/
  castrum/                 # Future project/tool CLI
  game/                    # Reference integration game

examples/                  # Small public-API examples
benchmark/                 # Performance and allocation benchmarks
```

The current repository may temporarily differ from this layout. Package
movement is complete only when imports, tests, documentation, and ownership
agree.

## Public package policy

Public packages are stable gameplay-facing contracts:

```text
castrum, ecs, components, geom, scene, events, assets,
animation, physics, input, atlas
```

A public package may expose types that users need to create gameplay, scenes,
components, bindings, assets, collision shapes, or events. It must not expose
storage structures or platform implementation details merely because they are
currently convenient.

Keep these internal:

```text
Ebiten polling and input translation
fixed-timestep scheduler internals
archetype and registry storage
spatial-index implementation
render extraction, batching, and GPU orchestration
asset worker and watcher implementation
window, audio-device, and OS integration
```

If a user can implement normal gameplay without importing a package, prefer
keeping that package internal. If users need to implement a stable extension,
expose a narrow interface in a public package rather than exposing the
implementation package.

Do not create `pkg/castrum` solely to enforce a single import. Castrum may have
several stable public packages; the boundary is API ownership and documented
stability, not import count.

## Game ownership

`Game` owns services whose lifetime is tied to the running application, window,
platform, or frame loop:

```text
World lifetime
Scene manager lifetime
Asset manager lifetime
Input polling service
Audio service/device
Renderer
Fixed-timestep loop and clock
Window/platform integration
Debug, profiling, and developer tooling
```

Expose deliberate accessors, not public fields:

```go
func (g *Game) World() *ecs.World
func (g *Game) Scenes() *scene.Manager
func (g *Game) Assets() *assets.Manager
func (g *Game) Input() input.Reader
func (g *Game) Events() *events.Bus
func (g *Game) Audio() *audio.Service
```

Only add an accessor when the service is a supported user-facing capability.
The accessor and its return type must be documented and tested.

Avoid two authoritative copies of a service. If a system needs the same event
bus, asset registry, animation store, or scene manager, `Game` may register the
same pointer as a typed World resource; it must not create a second instance.

## World resource policy

Keep `World` typed resources. They are useful for per-world dependencies and
simulation state, especially for systems that must work in tests or headless
runs.

Good World resources:

```text
event bus used by simulation systems
animation clip store
deterministic random source
input snapshot or simulation-visible input reader
scene-local registries
save/load context
game-specific simulation state
navigation or economy state
```

Do not use World resources for every engine service. Keep these on `Game` or
inside internal runtime packages:

```text
Ebiten window state
GPU renderer and frame buffers
OS audio device
wall-clock and fixed-step scheduler
filesystem watchers
platform handles
profiler and developer console
```

A resource belongs in `World` when it is scoped to one simulation and systems
need it during updates. A service belongs on `Game` when it owns a platform,
window, goroutine, GPU, or application lifetime.

World resources must be registered under the public contract when systems need
them. Do not register an internal concrete type that gameplay code cannot name;
register the supported public interface instead, for example:

```go
world.SetResource[input.Reader](inputService)
```

## Input as the model boundary

Input is the reference implementation of backend isolation:

```text
Ebiten polling
  -> internal/input backend adapter
  -> public physical state
  -> configured ActionMap
  -> input.Reader
  -> gameplay systems
```

Gameplay must query actions, not Ebiten keys:

```go
if input.ActionPressed(input.Action("jump")) {
    player.Jump()
}
```

The public `input` package owns action names, bindings, modifiers, normalized
physical state, snapshots, and replay buffers. `internal/input` owns Ebiten
polling and translation. Add a backend seam and fake-backend tests before
making platform polling more complex.

Bindings are configuration data, and action names are arbitrary game-level
identifiers. Do not bake gameplay meaning into physical key names:

```yaml
input:
  bindings:
    Jump_Up:
      - key: space
    move_up:
      - key: w
      - key: arrow_up
```

## Future subsystem placement

Use these ownership rules for v1 additions:

```text
audio/                 public playback types and service contract
internal/platform/     Ebiten audio device and backend details

particles/             public emitter/configuration types if users create them
internal/render/       particle simulation/render extraction implementation

ui/                     public retained widgets and layout contracts
internal/render/ui/     drawing and platform integration

save/                   public save/load format and APIs
internal/serialization/ reflection, codecs, and storage details

tilemap/                public map data and runtime types
internal/render/        tile batching and GPU details

profiler/               public opt-in profiling hooks
internal/profiling/    implementation and exporters
```

Do not make UI ECS-driven by default. Keep the retained widget tree separate
from simulation ECS state unless a deliberate integration contract is added.
Do not make camera visibility control physics simulation. Do not make audio or
rendering implementation details required by headless systems.

## Migration sequence

Use this order for package and ownership migrations:

### Phase 0: write the contract

Before moving files, identify:

- the public types users must construct or consume;
- the implementation types that can become internal;
- the owner of each service lifetime;
- the World resources systems require;
- one focused behavior test that proves the boundary.

### Phase 1: isolate one backend

Move one platform implementation, such as Ebiten input polling, behind a
public interface. Keep compatibility shims only when they reduce migration risk
and mark them for removal. Do not duplicate authoritative state.

### Phase 2: move implementation details

Move storage, scheduling, rendering orchestration, spatial indexing, or worker
code to `internal/` only after its public contract is tested. Update imports in
one slice and preserve behavior.

### Phase 3: wire ownership

Make `Game` construct and own the service. Register the same service as a World
resource only when systems need it. Ensure scene transitions and Game shutdown
release owned resources and goroutines.

### Phase 4: migrate examples and documentation

The reference game and examples must use only the intended public API. Update
Godoc examples and remove references to methods or packages that do not exist.

### Phase 5: remove compatibility surface

Delete old public implementations only after repository-wide callers, tests,
examples, and documentation have migrated. Use a separate commit when removal
is substantial.

## Completion checks

Before declaring a migration complete, run:

```text
gofmt -w <changed Go files>
go vet ./...
go test ./...
git diff --check
```

Also verify:

- no public package imports a platform backend unnecessarily;
- no gameplay system requires `*Game` when `*ecs.World` is sufficient;
- no service has two authoritative instances;
- exported APIs have Godoc comments;
- internal packages are not referenced by examples or user-facing docs;
- focused tests prove the moved boundary and one end-to-end path works;
- goroutine-owning services have an explicit shutdown path.

Stop the migration when the ownership contract is clear and the feature works.
Do not continue moving files solely for visual symmetry.
