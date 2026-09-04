# Castrum Engine Development Roadmap

> **Version**: 1.1  
> **Last Updated**: 2026-09-04  
> **Status**: Phase 1 complete; Phase 2 & 3 ~85% done—core loop, input, rendering, scenes all working

This roadmap outlines the development path for Castrum, a 2D game engine built in Go using the Ebiten library. The roadmap aligns with the architectural principles defined in `.github/agents/AGENT.md` and reflects the current state of the codebase.

---

## Versioning Strategy

### Version 0.1.0: Core Engine Ready (Current Target)

**Release Goal:** A functional, production-capable 2D game engine with core systems working end-to-end.

**What's Included:**

- ✅ ECS foundation (entities, components, systems, queries)
- ✅ Fixed-timestep game loop
- ✅ Scene management with transitions
- ✅ Rendering pipeline (sprites, primitives, layering, single camera)
- ✅ Input handling (keyboard, mouse, gamepad normalization)
- ✅ Configuration management
- ✅ Timer and animation systems
- ✅ Collision detection
- ✅ Asset loading (textures, images)

**What's NOT Required:**

- Editor / Visual tooling
- Persistence (save/load)
- Advanced physics (gravity, momentum, constraints)
- Audio system
- Networking
- Multi-camera / viewport compositing

**Exit Criteria:**

- [ ] Phases 2 & 3 complete (input, rendering, scenes all stable)
- [ ] Playable prototype game demonstrating all core systems
- [ ] Comprehensive test coverage (>80% core systems)
- [ ] Documentation for developers (README, examples, API guide)
- [ ] Public API stable (`pkg/castrum` only; no breaking changes promised after 0.1.0)

### Future Versions

**0.2.0 & Beyond:** Advanced features (persistence, editor, audio, networking, etc) as community needs drive.

---

## Immediate Next Steps (Priority Order)

1. **Input buffer for replays** (Phase 2.4) - Ring buffer to snapshot `InputState` each tick. Enables deterministic playback.
2. **Pause/resume gates** (Phase 2.6) - Add `Config.Engine.Paused bool`. Cost: one branch per tick.
3. **Time scaling** (Phase 2.7) - Add `Config.Engine.TimeScale float64`. Cost: one multiply per tick.
4. **Playable prototype** - Combine phases 2+3 into a simple game (RTS/strategy ideal at 60 TPS). Validates all systems.
5. **Phase 4 (Persistence)** - Save/load comes next once you need it (new game milestone).

---

## Current State Assessment

### Completed

- [x] Project structure established (`ecs/`, `engine/`, `internal/`, `config/`)
- [x] Core ECS data structures (`EntityID`, `Component`, `World` interface)
- [x] Internal World implementation (`internal/core/world.go`)
- [x] Component storage system with type-based indexing
- [x] Entity hierarchy and parent-child relationships
- [x] Tag-based querying system
- [x] System manager with priority-based scheduling (replaced the old fixed Core/Player layer split)
- [x] Timer management system
- [x] Scene management foundation (load/unload/transition, lifecycle hooks)
- [x] Game runtime with fixed-timestep loop, wired to `Config.Engine.TicksPerSecond`
- [x] Configuration management
- [x] Comprehensive unit + integration test coverage for `internal/core`
- [x] Build pipeline (Makefile) with a real runnable entry point (`cmd/game`)
- [x] Scene system
- [x] Rendering integration - `Game.Draw` wired to `Renderer.DrawScene`
- [x] Camera system (single camera: position/zoom/rotation/bounds, `NewCamera` constructor)
- [x] Static render layers (`Layer0..Layer10` + `LayerDebug`) with draw-order bucketing
- [x] Primitive shape rendering (rectangle/circle/line) - no texture asset required
- [x] Debug overlay (FPS/TPS/camera position) gated by `Config.Engine.EnableDebug`
- [x] Sprite/texture rendering - `texture.Store.Load` loads images from disk via `ebitenutil.NewImageFromFileSystem`
- [x] Input system with frame normalization - `Input.Update()` polls Ebiten, normalizes to `InputState`

### Partially Implemented

- [ ] Camera system - solid for a single camera; multi-camera/viewport support (minimap, split-screen) not designed in code yet (see Phase 3 notes)
- [ ] Scene entity cleanup - `OnUnload` untags entities but doesn't `DestroyEntity`/`Cleanup` them (intentional: preserves entity data across scene swaps)
- [ ] Input buffer for deterministic replay - infrastructure exists, buffer itself not yet implemented
- [ ] Pause/resume and time scaling - infrastructure ready, feature gates not yet added

### Not Started

- [ ] Input handling
- [ ] Asset pipeline (texture loading, audio, fonts)
- [ ] Serialization/persistence
- [ ] CLI tooling
- [ ] Visual editor
- [ ] Performance profiling
- [ ] Hot-reloading

---

## Phase Overview

| Phase | Name                 | Goal                                        | Status          | Estimated Duration |
| ----- | -------------------- | ------------------------------------------- | --------------- | ------------------ |
| 0     | Foundation           | Bootstrap, window rendering, build pipeline | **Complete**    | 1-2 weeks          |
| 1     | Core Data Structures | ECS implementation, entity lifecycle        | **Complete**    | 3-4 weeks          |
| 2     | Simulation Loop      | Fixed timestep, input handling, determinism | **In Progress** | 1-2 weeks (85%)    |
| 3     | Rendering & Scenes   | ECS rendering integration, scene management | **In Progress** | 1 week (90%)       |
| 4     | Persistence & Assets | Save/load, async asset loading              | **Not Started** | 3-4 weeks          |
| 5     | Tooling & Polish     | CLI, editor, profiling, documentation       | **Not Started** | 4-6 weeks          |

---

## Phase 0: Foundation (The Bootstrap) ✅ COMPLETE

### Objective

Establish project structure, build pipeline, and basic window rendering.

### Acceptance Criteria

- [x] Repository structure follows Go conventions
- [x] Makefile with build, test, lint, clean targets
- [x] Go module initialized with proper dependencies
- [x] CI/CD pipeline configured (GitHub Actions)
- [x] Ebiten window opens and maintains 60fps
- [x] Project compiles without errors
- [x] Basic license and documentation in place

### Current Status

**Complete**. The project has:

- Proper Go module structure (`go.mod`, `go.sum`)
- Working Makefile with build and test targets
- License in place
- Clean package organization

### Next Steps

- Set up GitHub Actions for CI/CD
- Add code coverage reporting
- Configure linting in CI

---

## Phase 1: Core Data Structures (The Skeleton) 🟡 IN PROGRESS

### Objective

Implement the Entity-Component-System foundation with efficient data structures.

### Acceptance Criteria

| ID   | Task                                          | Status  | Effort | Priority |
| ---- | --------------------------------------------- | ------- | ------ | -------- |
| 1.1  | Entity ID generation and management           | ✅ Done | 2h     | High     |
| 1.2  | Component storage with type-based indexing    | ✅ Done | 4h     | High     |
| 1.3  | Entity creation and destruction               | ✅ Done | 3h     | High     |
| 1.4  | Component add/remove/get operations           | ✅ Done | 4h     | High     |
| 1.5  | Entity querying by component type             | ✅ Done | 3h     | High     |
| 1.6  | Tag system for entity categorization          | ✅ Done | 3h     | High     |
| 1.7  | Entity hierarchy (parent/child relationships) | ✅ Done | 4h     | High     |
| 1.8  | World interface with all operations           | ✅ Done | 2h     | High     |
| 1.9  | Internal World implementation                 | ✅ Done | 6h     | High     |
| 1.10 | System interface and base implementation      | ✅ Done | 4h     | High     |
| 1.11 | System manager with layer support             | ✅ Done | 4h     | High     |
| 1.12 | Timer system for delayed/scheduled actions    | ✅ Done | 4h     | Medium   |
| 1.13 | Scene interface and basic implementation      | ✅ Done | 4h     | High     |
| 1.14 | Comprehensive unit tests for core packages    | ✅ Done | 8h     | High     |
| 1.15 | Benchmark tests for performance validation    | ✅ Done | 4h     | Medium   |

### Technical Details

#### ECS Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        ecs Package (Public API)             │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────────┐ │
│  │  World      │  │  EntityID   │  │      Component       │ │
│  │  Interface  │  │  Type Alias │  │      Interface       │ │
│  └─────────────┘  └─────────────┘  └──────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                    internal/core Package                         │
│  ┌─────────────────────────────────────────────────────────┐     │
│  │                    World Struct                         │     │
│  │  - entities: map[EntityID]*entity                       │     │
│  │  - store: *componentStore                               │     │
│  │  - index: entityIndex                                   │     │
│  │  - hierarchy: *Hierarchy                                │     │
│  └─────────────────────────────────────────────────────────┘     │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐ │
│  │ componentStore   │  │   entityIndex    │  │    Hierarchy    │ │
│  │ - type-based     │  │ - tag-based      │  │ - parent/child  │ │
│  │   storage        │  │   queries        │  │   relationships │ │
│  └──────────────────┘  └──────────────────┘  └─────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

#### System Architecture

- **Layer Support**: Core (always runs) vs Player (conditional) systems
- **Lifecycle**: Init -> Update -> Shutdown
- **Query-based**: Systems query for entities with specific components

### Technical Risks

| Risk                                            | Impact | Mitigation                             |
| ----------------------------------------------- | ------ | -------------------------------------- |
| Memory overhead from type reflection in queries | High   | Implement caching for frequent queries |
| Component storage fragmentation                 | Medium | Use generational indices for EntityID  |
| Query performance with many entity types        | High   | Implement archetype-based storage      |

> Note: archetype-based storage is now implemented (`internal/core/archetype.go`), so this risk is largely mitigated - the remaining cost is `World.Query`'s linear scan over archetypes, which is fine until the number of distinct archetypes grows large.

### Phase 1 Completion Checklist

- [x] All acceptance criteria tasks completed
- [x] Unit tests for all core packages (>80% coverage)
- [x] Benchmark tests showing <1ms for 1000 entity queries
- [ ] Documentation for all public APIs (partial: code-level docs exist, need markdown guide)
- [ ] Performance profiling integrated (benchmarks exist, CLI profiling not yet tooled)

---

## Phase 2: Simulation Loop (The Heartbeat) 🟡 IN PROGRESS

### Objective

Implement the deterministic game loop with fixed timestep, input handling, and proper separation of simulation from rendering.

### Acceptance Criteria

| ID  | Task                                      | Status         | Effort | Priority |
| --- | ----------------------------------------- | -------------- | ------ | -------- |
| 2.1 | Fixed timestep accumulator implementation | ✅ Done        | 4h     | High     |
| 2.2 | Configurable fixed delta time             | ✅ Done        | 2h     | Medium   |
| 2.3 | Input system with frame normalization     | ✅ Done        | 6h     | High     |
| 2.4 | Input buffer for deterministic replay     | 🟡 Partial     | 4h     | Medium   |
| 2.5 | Frame interpolation for smooth rendering  | ❌ Not Started | 4h     | Medium   |
| 2.6 | Pause/resume simulation support           | 🟡 Partial     | 3h     | Low      |
| 2.7 | Time scaling (slow motion, fast forward)  | 🟡 Partial     | 3h     | Low      |
| 2.8 | Simulation step debugging hooks           | ✅ Done        | 4h     | Medium   |

### Technical Details

#### Game Loop Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        GameRuntime                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  Fixed Timestep Loop                                        │ │
│  │                                                             │ │
│  │  accumulator += deltaTime                                   │ │
│  │  while accumulator >= fixedDelta:                         │ │
│  │      accumulator -= fixedDelta                            │ │
│  │      UpdateSystems(fixedDelta)  ───► Deterministic       │ │
│  │      UpdateTimers(fixedDelta)    ──► Simulation           │ │
│  │      UpdateWorldState()         ──► Logic                │ │
│  │                                                             │ │
│  │  Render() ───────────────────────────► Visual Output     │ │
│  │         (with interpolation if needed)                   │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

#### Input System Design

```
┌─────────────────────────────────────────────────────────────┐
│                      Input System                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────┐   │
│  │ Raw Input   │    │ Normalized  │    │ Input Buffer    │   │
│  │ (per frame) │───►│ Input       │───►│ (for replay)    │   │
│  └─────────────┘    │ (per step)  │    └─────────────────┘   │
│                     └─────────────┘                          │
└─────────────────────────────────────────────────────────────┘
```

### Technical Risks

| Risk                                        | Impact | Mitigation                                     |
| ------------------------------------------- | ------ | ---------------------------------------------- |
| Accumulator overflow with large delta times | Medium | Clamp maximum delta time                       |
| Input buffer memory usage                   | Low    | Implement circular buffer with size limit      |
| Determinism across platforms                | High   | Use fixed-point math for critical calculations |

### Dependencies

- Phase 1 (Core Data Structures) must be complete

### Current Status

**In Progress**. Core loop complete:

- Fixed timestep accumulator: working, **clamped at `MaxDelta = 0.25s`** (prevents spiral-of-death on stalls)
- Configurable delta time: working (`fixedDelta = 1.0 / TicksPerSecond`)
- Input system: working (`Input.Update()` polls Ebiten each tick, normalizes into frame-stable `InputState`, provides key/mouse queries)
- Simulation step debugging: working (FPS/TPS overlay via `DrawDebugInfo`, gated by `Config.Engine.EnableDebug`)

Remaining items on this phase: input buffer for replays, frame interpolation (lower priority for non-fast-paced games), pause/resume gates, time scaling gates.

### Next Steps

- Implement input buffer: on each tick, snapshot the `InputState` into a ring buffer (fixed size, ~120 frames at 60 TPS). Enables deterministic replay by replaying input snapshots instead of polling Ebiten.
- Add pause/resume gates: `if g.Config.Engine.Paused { skip accumulator step }` - already clamped, so cost-free.
- Add time scaling: `delta *= g.Config.Engine.TimeScale` before accumulator - local decision, no loop changes needed.
- Frame interpolation (3x priority): defer until fast-moving entities on screen show tick popping. At 60 TPS for turn-based/RTS, not noticeable.
- Multi-rate systems (optional future): if some systems need 30 TPS while others run 60, use priority buckets in manager and run subset per phase.

---

## Phase 3: Rendering & Scenes (The Visuals) 🟡 IN PROGRESS

### Objective

Integrate rendering with the ECS and implement scene management for logical game state transitions.

### Acceptance Criteria

| ID   | Task                                     | Status          | Effort | Priority |
| ---- | ---------------------------------------- | --------------- | ------ | -------- |
| 3.1  | Renderer as standalone system            | ⚫ Not Required | 4h     | High     |
| 3.2  | Sprite rendering component               | ✅ Done         | 4h     | High     |
| 3.3  | Camera system with viewport management   | 🟡 Partial      | 6h     | High     |
| 3.4  | Render layers and z-ordering             | ✅ Done         | 4h     | High     |
| 3.5  | Renderer y-sorting                       | ✅ Done         | 4h     | High     |
| 3.6  | Scene interface refinement               | ✅ Done         | 4h     | High     |
| 3.7  | Scene manager with transition support    | ✅ Done         | 6h     | High     |
| 3.8  | Scene lifecycle hooks (OnLoad, OnUnload) | ✅ Done         | 4h     | High     |
| 3.9  | Entity cleanup on scene unload           | 🟡 Partial      | 3h     | Medium   |
| 3.10 | Scene stack for nested scenes            | ❌ Not Started  | 4h     | Medium   |
| 3.11 | UI rendering layer                       | ❌ Not Started  | 8h     | Medium   |
| 3.12 | Particle system                          | ❌ Not Started  | 6h     | Low      |


### Technical Details

#### Rendering Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Render Pipeline                           │
│                                                             │
│  World State                                               │
│      │                                                    │
│      ▼                                                    │
│  ┌─────────────┐                                          │
│  │  Render     │←── Systems update render components     │
│  │  System     │    (Position, Sprite, Visibility, etc.)    │
│  └──────┬──────┘                                          │
│         │                                                  │
│         ▼                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐      │
│  │  Camera     │    │  Layers     │    │  Ebiten     │      │
│  │  System     │───►│  System     │───►│  Renderer   │      │
│  └─────────────┘    └─────────────┘    └─────────────┘      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### Scene Management Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Scene Manager                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  Scenes: map[string]Scene                                  │ │
│  │  Current: string                                           │ │
│  │  Stack: []string  (for nested scenes)                     │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  LoadScene(name, scene) ──► Add to scenes map               │
│  TransitionTo(name) ───────► Call OnUnload, then OnLoad       │
│  PushScene(name) ──────────► Add to stack, call OnLoad        │
│  PopScene() ───────────────► Call OnUnload, remove from stack │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Technical Risks

| Risk                                  | Impact | Mitigation                         |
| ------------------------------------- | ------ | ---------------------------------- |
| Render performance with many entities | High   | Implement sprite batching          |
| Z-ordering conflicts                  | Medium | Document clear layering guidelines |
| Scene transition memory leaks         | Medium | Implement proper cleanup hooks     |

### Dependencies

- Phase 1 (Core Data Structures) must be complete
- Phase 2 (Simulation Loop) should be complete

### Current Status

**In Progress**. Core rendering complete:

- `Renderer.DrawScene` draws every `Renderable`+`Transform` entity, dispatching to textured-sprite or primitive-shape path based on `Renderable.TexturePath`
- Static render layers (`Layer0..Layer10`, plus `LayerDebug` drawn last) implemented, bucketed per frame
- Debug overlay (FPS/TPS/camera position) wired behind `Config.Engine.EnableDebug`
- Scene load/unload/transition and lifecycle hooks all working
- **Sprite/texture rendering: ✅ DONE** - `texture.Store.Load` now loads images via `ebitenutil.NewImageFromFileSystem`, caches by path
- **Render layers and z-ordering: ✅ DONE** - Sorted by Layer primary, then Y position within same layer (isometric/top-down support)
- **Renderer y-sorting: ✅ DONE** - Secondary sort key after Layer, handles depth-based visual ordering
- **Entity cleanup on unload: 🟡 Partial** - `OnUnload` untags entities (intentional design: preserves data across scene swaps). Destroy semantics not needed yet.
- Single camera working; multi-camera design saved in repo memory.

⚠️ **Efficiency notes** (low priority, profile-driven):

- Layer bucketing allocates fresh map every frame - could use fixed array indexed by layer. No impact until many entities.
- `GetComponent[Renderable]` called twice per entity per frame - could cache in bucket struct. Same.
- Neither worth touching until profiling shows impact.

### Next Steps

- Multi-camera/viewport design (minimap, split-screen) - design note saved in repo memory; defer until feature needed.
- Scene stack (3.9): stack-based overlay scenes (pause menu on top of game). Straightforward once base manager stabilizes.
- UI rendering layer (3.10): separate axis-aligned UI pass after world, no camera transform. Defer until UI components exist.
- Particle system (3.11): ECS-based emitters + particles with lifetime. Lower priority than core features.
- **All current blocking items resolved.** Sprite art now loadable. Next milestone: playable game prototype.

---

## Phase 4: Persistence & Assets (The Memory) ❌ NOT STARTED

### Objective

Implement save/load functionality for world state and async asset management.

### Acceptance Criteria

| ID   | Task                                                | Status         | Effort | Priority |
| ---- | --------------------------------------------------- | -------------- | ------ | -------- |
| 4.1  | Serialization format selection (JSON, binary, etc.) | ❌ Not Started | 2h     | High     |
| 4.2  | Entity serialization/deserialization                | ❌ Not Started | 6h     | High     |
| 4.3  | Component serialization registry                    | ❌ Not Started | 4h     | High     |
| 4.4  | World state save/load                               | ❌ Not Started | 4h     | High     |
| 4.5  | Save game versioning and migration                  | ❌ Not Started | 4h     | Medium   |
| 4.6  | Asset interface definition                          | ❌ Not Started | 2h     | High     |
| 4.7  | Async asset loading system                          | ❌ Not Started | 6h     | High     |
| 4.8  | Asset caching and reference counting                | ❌ Not Started | 4h     | Medium   |
| 4.9  | Texture atlas support                               | ❌ Not Started | 4h     | Low      |
| 4.10 | Audio asset management                              | ❌ Not Started | 4h     | Medium   |
| 4.11 | Font asset management                               | ❌ Not Started | 3h     | Medium   |

### Technical Details

#### Serialization Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Serialization System                        │
│                                                             │
│  ┌─────────────┐    ┌─────────────────────────────────────┐  │
│  │  World      │    │             Serializers               │  │
│  │  State      │───►│  ┌───────────┐  ┌───────────┐       │  │
│  └─────────────┘    │  │  Entity   │  │ Component │       │  │
│                     │  │ Serializer│  │Serializers│       │  │
│                     │  └───────────┘  └───────────┘       │  │
│                     └─────────────────────────────────────┘  │
│                                  │                              │
│                                  ▼                              │
│                     ┌───────────────────────┐                  │
│                     │    Storage Format     │                  │
│                     │  (JSON, Binary, etc.)  │                  │
│                     └───────────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
```

### Technical Risks

| Risk                                | Impact | Mitigation                                       |
| ----------------------------------- | ------ | ------------------------------------------------ |
| Serialization version compatibility | High   | Implement schema versioning with migration paths |
| Asset loading deadlocks             | High   | Implement timeout mechanisms                     |
| Memory bloat from cached assets     | Medium | Implement LRU cache with size limits             |

### Dependencies

- Phase 1 (Core Data Structures) must be complete
- Phase 3 (Rendering) should be complete for asset types

---

## Phase 5: Tooling & Polish (The UX) ❌ NOT STARTED

### Objective

Provide developer tools to accelerate content creation and improve the development experience.

### Acceptance Criteria

| ID  | Task                               | Status         | Effort | Priority |
| --- | ---------------------------------- | -------------- | ------ | -------- |
| 5.1 | CLI tool for project scaffolding   | ❌ Not Started | 4h     | Medium   |
| 5.2 | CLI tool for asset validation      | ❌ Not Started | 3h     | Medium   |
| 5.3 | CLI tool for performance profiling | ❌ Not Started | 4h     | Medium   |
| 5.4 | Visual editor foundation           | ❌ Not Started | 12h    | High     |
| 5.5 | Scene composition in editor        | ❌ Not Started | 8h     | High     |
| 5.6 | Hot-reloading for code and assets  | ❌ Not Started | 8h     | Medium   |
| 5.7 | Comprehensive API documentation    | ❌ Not Started | 8h     | High     |
| 5.8 | Usage examples and tutorials       | ❌ Not Started | 8h     | High     |
| 5.9 | Performance benchmark suite        | ❌ Not Started | 4h     | Medium   |

### Dependencies

- All previous phases should be complete

---

## Cross-Cutting Concerns

### Performance Targets

| Metric             | Target                     | Measurement Method |
| ------------------ | -------------------------- | ------------------ |
| Entity create rate | 10,000+ entities/sec       | Benchmark test     |
| Query performance  | <1ms for 10,000 entities   | Benchmark test     |
| System update rate | 1,000+ systems/frame       | Benchmark test     |
| Memory per entity  | <100 bytes                 | Profiling          |
| FPS stability      | 60fps with 5,000+ entities | Stress test        |

### Quality Gates

- [ ] All public APIs documented with godoc
- [ ] Unit test coverage >80% for core packages
- [ ] No critical linting errors
- [ ] No goroutine leaks detected
- [ ] No memory leaks in 1-hour stress test
- [ ] Deterministic simulation verified

### Documentation Requirements

- [ ] API reference documentation
- [ ] Getting started guide
- [ ] Architecture overview
- [ ] Example projects (Hello World, Simple Game)
- [ ] Contribution guidelines

---

## Milestone Summary

| Milestone | Description          | Target Date | Status         |
| --------- | -------------------- | ----------- | -------------- |
| **M0**    | Project Bootstrap    | 2026-08-23  | ✅ Complete    |
| **M1**    | Core ECS Complete    | 2026-09-15  | 🟡 In Progress |
| **M2**    | Simulation Loop      | 2026-10-15  | ❌ Not Started |
| **M3**    | Rendering & Scenes   | 2026-11-15  | ❌ Not Started |
| **M4**    | Persistence & Assets | 2026-12-15  | ❌ Not Started |
| **M5**    | Tooling & Polish     | 2027-01-31  | ❌ Not Started |
| **1.0**   | Stable Release       | 2027-02-28  | ❌ Not Started |

---

## Release Criteria

### Alpha Release (After Phase 1)

- Core ECS functional
- Basic game loop working
- Can create entities and add components
- Unit tests passing

### Beta Release (After Phase 3)

- Rendering pipeline functional
- Scene management working
- Input handling implemented
- Basic examples provided

### Release Candidate (After Phase 4)

- Save/load functionality
- Asset management
- Performance targets met
- Documentation complete

### Stable 1.0 Release (After Phase 5)

- All acceptance criteria met
- All quality gates passed
- Tooling complete
- Tutorials and examples provided

---

## Contributing

This roadmap is a living document. To contribute:

1. **Review the current state** - Check off completed items as they're finished
2. **Add details** - Break down large tasks into smaller, actionable items
3. **Update estimates** - Adjust effort estimates based on actual experience
4. **Identify risks** - Add new risks as they're discovered
5. **Propose changes** - Open issues or PRs to discuss roadmap modifications

---

## Appendices

### Appendix A: Package Responsibilities

| Package           | Responsibility                             | Public/Internal |
| ----------------- | ------------------------------------------ | --------------- |
| `castrum`         | Root package, exports main engine API      | Public          |
| `castrum/config`  | Configuration management                   | Public          |
| `castrum/ecs`     | ECS type definitions and interfaces        | Public          |
| `castrum/engine`  | Main engine API (Game, World access, etc.) | Public          |
| `internal/core`   | Core ECS implementation                    | Internal        |
| `internal/engine` | Engine runtime implementation              | Internal        |
| `internal/scene`  | Scene management implementation            | Internal        |
| `internal/system` | System manager implementation              | Internal        |
| `internal/timers` | Timer management implementation            | Internal        |

### Appendix B: Key Design Decisions

| Decision                    | Rationale                                                 | Impact            |
| --------------------------- | --------------------------------------------------------- | ----------------- |
| ECS Architecture            | Separation of data and behavior, performance, flexibility | Core architecture |
| Interface-based API         | Encapsulation, testability, extensibility                 | Public API design |
| Fixed timestep              | Determinism, simulation stability                         | Simulation loop   |
| Internal package separation | Compiler-enforced encapsulation                           | Project structure |

### Appendix C: Glossary

| Term                    | Definition                                             |
| ----------------------- | ------------------------------------------------------ |
| **ECS**                 | Entity-Component-System architectural pattern          |
| **Entity**              | Unique identifier for a game object                    |
| **Component**           | Pure data attached to an entity                        |
| **System**              | Logic that processes entities with specific components |
| **Scene**               | Logical grouping of entities with lifecycle management |
| **World**               | Container for all entities, components, and systems    |
| **Fixed Timestep**      | Constant time interval for simulation updates          |
| **Frame Interpolation** | Smooth rendering between simulation steps              |

---
