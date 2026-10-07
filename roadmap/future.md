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
