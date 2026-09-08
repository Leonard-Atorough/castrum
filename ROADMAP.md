# Castrum Engine Development Roadmap

> **Version**: 1.2  
> **Last Updated**: 2026-09-08  
> **Status**: Phases 1–3 complete; Core engine ready for 0.1.0 release. Phases 4–5 deferred.

This roadmap outlines the development path for Castrum, a 2D game engine built in Go using the Ebiten library.

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

- [x] Phases 1–3 complete (ECS, input, rendering, scenes all stable)
- [ ] Playable prototype game demonstrating all core systems
- [ ] Comprehensive test coverage (>80% core systems)
- [ ] Documentation for developers (README, examples, API guide)
- [ ] Public API stable (`pkg/castrum` only; no breaking changes promised after 0.1.0)

### Future Versions

**0.2.0 & Beyond:** Advanced features (persistence, editor, audio, networking, etc) as community needs drive.

---

## Next Milestone: Playable Prototype

Build a simple game (turn-based strategy ideal at 60 TPS) that combines phases 1–3. This validates all core systems end-to-end before moving to persistence (phase 4).

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

## Phase 0: Foundation ✅ COMPLETE

Project structure, build pipeline, and basic window rendering.

- [x] Repository structure (Go conventions)
- [x] Makefile with build, test, lint targets
- [x] Go module initialized
- [x] Ebiten window and 60fps rendering
- [x] License and documentation

---

## Phase 1: Core Data Structures ✅ COMPLETE

Entity-Component-System foundation with efficient data structures.

### Completed

- [x] Entity ID generation and lifecycle
- [x] Component storage with type-based indexing
- [x] Entity creation/destruction and component add/remove
- [x] Tag-based querying system
- [x] Entity hierarchy (parent/child relationships)
- [x] System interface and priority-based scheduling
- [x] Timer system for delayed/scheduled actions
- [x] Scene interface and implementation
- [x] Archetype-based storage (performance optimization)
- [x] Comprehensive unit + integration tests (>80% coverage)
- [x] Benchmark suite for performance validation

---

## Phase 2: Simulation Loop ✅ COMPLETE

Deterministic game loop with fixed timestep and input handling.

### Completed

- [x] Fixed timestep accumulator (clamped at 0.25s to prevent spiral-of-death)
- [x] Configurable fixed delta time (`fixedDelta = 1.0 / TicksPerSecond`)
- [x] Input system with frame normalization (Ebiten polling, `InputState` per tick)
- [x] Input buffer infrastructure (ring buffer ready for deterministic replay)
- [x] Pause/resume simulation support (`Config.Engine.Paused`)
- [x] Time scaling (slow motion, fast forward; `Config.Engine.TimeScale`)
- [x] Simulation step debugging hooks (FPS/TPS overlay, gated by `Config.Engine.EnableDebug`)

### Not Implemented (Low Priority)

- [ ] Frame interpolation (deferred: not visible at 60 TPS for turn-based/RTS games)

---

## Phase 3: Rendering & Scenes ✅ COMPLETE

Rendering pipeline and scene management for game state transitions.

### Completed

- [x] Sprite rendering with texture loading (via `ebitenutil.NewImageFromFileSystem`)
- [x] Primitive shape rendering (rectangle, circle, line)
- [x] Camera system (single camera: position, zoom, rotation, bounds)
- [x] Static render layers (Layer0–Layer10, LayerDebug) with draw-order bucketing
- [x] Render layer y-sorting (depth-based visual ordering for isometric/top-down)
- [x] Scene interface and implementation with lifecycle hooks (OnLoad, OnUnload)
- [x] Scene manager with transition support
- [x] Entity cleanup on scene unload (entities untagged, data preserved for scene swaps)
- [x] Debug overlay (FPS/TPS/camera position, gated by `Config.Engine.EnableDebug`)
- [x] Scene stack for nested/overlay scenes (pause menu on top of game) — straightforward once base stabilizes

### Not Implemented (Deferred)

- [ ] Multi-camera / viewport support (minimap, split-screen) — design saved; defer until needed
- [ ] UI rendering layer — separate axis-aligned pass; defer until UI components exist
- [ ] ECS Particle system — lower priority feature

### Performance Notes

Layer bucketing and component lookups have negligible impact at current entity counts. Profile-driven optimization deferred.

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
