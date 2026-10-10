# Longer-term capability map

This page is a holding area for directions beyond the current release
proposals. None of these items is committed to a version. Promote an item
only when its user need, public contract, and maintenance cost are understood.

## Likely framework extensions

- **Entity hierarchy:** parent-child transforms, propagation order,
  interpolation, reparenting, and destruction semantics. Valuable, but a
  substantial design change to the current flat entity model.
- **UI:** text layout, hit testing, input focus, and a small widget set.
  Separate game HUD rendering from editor-style tooling.
- **Multiple cameras and viewports:** split-screen, render targets, camera
  ordering, and viewport input mapping.
- **Save/load:** explicit serialization boundaries and versioning for
  user-defined game state; do not assume arbitrary ECS reflection is safe.
- **Physics integration:** compare a maintained Go/Ebitengine-compatible
  library with an in-house implementation after the minimal collision API
  has real users and benchmark data.
- **Custom rendering:** shaders, render targets, and post-processing should
  build on stable runner/rendering seams and remain optional.
- **Asynchronous and batch assets:** add only with clear cancellation,
  failure, ownership, and synchronization semantics.
- **Tilemaps and asset hot reload:** first choose the authoring/import format
  and establish development-versus-release and cache ownership contracts.
- **Particles:** define bounded allocation, lifetime, and rendering costs
  before adding a general-purpose effect system.

## Backend flexibility: from convention to contract

The engine core is already backend-free in code: no package outside `ebitrun`
imports Ebitengine, `castrum.Game.Run` accepts any `Runner`, and the neutral
layers describe *what* to draw, play, and bind — `render` collects a
backend-neutral `DrawList`, `audio` defines the `Controller` contract, `input`
uses castrum-owned key enums that a runner maps, and `asset` decodes textures
through the standard library while runners register their own audio codecs.
The remaining work is turning this convention into a contract other backends
can implement, and eventually into a module graph that does not mention
Ebitengine at all.

Items in dependency order:

- **Runner contract reference + conformance harness:** one page listing what a
  runner must provide — lifecycle (`Startup`/`Advance`), the resources it binds
  (asset providers, audio controller), what it publishes on `core.Context`
  (input snapshot, logical dimensions), draw-list consumption, error
  surfacing — plus a conformance test suite a third-party runner can run
  against a game. User need is clear (alternate backends); cost is mostly
  documentation and tests. First promotion candidate.
- **Reference headless runner:** a windowless runner that drives the game on a
  virtual clock with no-op presentation and a null audio controller. It proves
  the contract page, enables full-game smoke tests in CI without a GPU, and is
  the natural host for simulation-only uses (servers, deterministic replay
  front-ends). Promote together with the contract page, not before.
- **Adapter module split:** move `ebitrun` into its own Go module so `go get`
  of the core never records Ebitengine in a consumer's module graph. Go's
  module graph pruning already keeps unused Ebitengine code out of consumer
  builds, and the split costs nested-module tagging and release choreography.
  Do it only when a second backend actually exists.
- **Portable overlays:** `Runner.AddDraw` is backend-typed by design. If users
  need overlays that survive a backend swap, route them through the same
  sprite/draw-list path rather than adding a new abstraction.

Accepted couplings, recorded so they are not relitigated:

- `go-text/typesetting` stays in the neutral asset layer: it is itself
  backend-free and gives every runner identical text measurement.
- Audio codec registration stays runner-owned; supported formats are a backend
  concern.
- `ebitrun` remains the default, recommended runner. Backend flexibility is
  for alternate runners, not a de-emphasis of Ebitengine.

## Larger bets

- **Visual editor or inspector:** first establish reliable runtime
  introspection and a proven content workflow; an editor is a separate
  product-sized investment.
- **Networking and deterministic replay:** require explicit determinism,
  serialization, and transport boundaries; do not infer them from a fixed
  timestep alone.
- **3D rendering:** outside the current 2D-first product direction.
- **Hot code reload:** not the same as asset hot reload; requires a toolchain
  and lifecycle design with platform-specific limits.

## Ongoing quality work

These are continuous responsibilities rather than feature releases:

- Keep examples buildable and runnable, including one representative game
  that uses the recommended architecture.
- Test engine contracts with unit, integration, race, and supported-platform
  checks appropriate to each package.
- Publish useful Go documentation, focused guides, troubleshooting help, and
  migration notes for breaking pre-1.0 changes.
- Track performance with repeatable benchmarks and evidence-based budgets.
- Maintain clear Ebitengine and Go version support, build instructions, and
  release compatibility notes.
