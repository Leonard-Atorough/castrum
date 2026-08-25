# Castrum Engine Development Roadmap

> **Version**: 1.0  
> **Last Updated**: 2026-08-25  
> **Status**: Phase 1 (Core Data Structures) - ~60% Complete

This roadmap outlines the development path for Castrum, a 2D game engine built in Go using the Ebiten library. The roadmap aligns with the architectural principles defined in `.github/agents/AGENT.md` and reflects the current state of the codebase.

---

## Current State Assessment

### Completed
- [x] Project structure established (`ecs/`, `engine/`, `internal/`, `config/`)
- [x] Core ECS data structures (`EntityID`, `Component`, `World` interface)
- [x] Internal World implementation (`internal/core/world.go`)
- [x] Component storage system with type-based indexing
- [x] Entity hierarchy and parent-child relationships
- [x] Tag-based querying system
- [x] System manager with layer support (Core/Player)
- [x] Timer management system
- [x] Scene management foundation
- [x] Game runtime with fixed-timestep loop
- [x] Configuration management
- [x] Basic test coverage for core packages
- [x] Build pipeline (Makefile)

### Partially Implemented
- [ ] Scene system (interface exists, needs refinement)
- [ ] Rendering integration (stubbed in `GameRuntime.Draw`)
- [ ] Input handling (not yet implemented)
- [ ] Asset management (not started)
- [ ] Serialization/persistence (not started)

### Not Started
- [ ] Camera system
- [ ] Render layers
- [ ] Asset pipeline
- [ ] CLI tooling
- [ ] Visual editor
- [ ] Performance profiling
- [ ] Hot-reloading

---

## Phase Overview

| Phase | Name | Goal | Status | Estimated Duration |
|-------|------|------|--------|-------------------|
| 0 | Foundation | Bootstrap, window rendering, build pipeline | **Complete** | 1-2 weeks |
| 1 | Core Data Structures | ECS implementation, entity lifecycle | **In Progress** | 3-4 weeks |
| 2 | Simulation Loop | Fixed timestep, input handling, determinism | **Not Started** | 2-3 weeks |
| 3 | Rendering & Scenes | ECS rendering integration, scene management | **Not Started** | 3-4 weeks |
| 4 | Persistence & Assets | Save/load, async asset loading | **Not Started** | 3-4 weeks |
| 5 | Tooling & Polish | CLI, editor, profiling, documentation | **Not Started** | 4-6 weeks |

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
| ID | Task | Status | Effort | Priority |
|----|------|--------|--------|----------|
| 1.1 | Entity ID generation and management | ✅ Done | 2h | High |
| 1.2 | Component storage with type-based indexing | ✅ Done | 4h | High |
| 1.3 | Entity creation and destruction | ✅ Done | 3h | High |
| 1.4 | Component add/remove/get operations | ✅ Done | 4h | High |
| 1.5 | Entity querying by component type | ✅ Done | 3h | High |
| 1.6 | Tag system for entity categorization | ✅ Done | 3h | High |
| 1.7 | Entity hierarchy (parent/child relationships) | ✅ Done | 4h | High |
| 1.8 | World interface with all operations | ✅ Done | 2h | High |
| 1.9 | Internal World implementation | ✅ Done | 6h | High |
| 1.10 | System interface and base implementation | ✅ Done | 4h | High |
| 1.11 | System manager with layer support | ✅ Done | 4h | High |
| 1.12 | Timer system for delayed/scheduled actions | ✅ Done | 4h | Medium |
| 1.13 | Scene interface and basic implementation | ✅ Done | 4h | High |
| 1.14 | Comprehensive unit tests for core packages | 🟡 Partial | 8h | High |
| 1.15 | Benchmark tests for performance validation | ❌ Not Started | 4h | Medium |

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
| Risk | Impact | Mitigation |
|------|--------|------------|
| Memory overhead from type reflection in queries | High | Implement caching for frequent queries |
| Component storage fragmentation | Medium | Use generational indices for EntityID |
| Query performance with many entity types | High | Implement archetype-based storage |

### Phase 1 Completion Checklist
- [ ] All acceptance criteria tasks completed
- [ ] Unit tests for all core packages (>80% coverage)
- [ ] Benchmark tests showing <1ms for 1000 entity queries
- [ ] Documentation for all public APIs
- [ ] Performance profiling integrated

---

## Phase 2: Simulation Loop (The Heartbeat) ❌ NOT STARTED

### Objective
Implement the deterministic game loop with fixed timestep, input handling, and proper separation of simulation from rendering.

### Acceptance Criteria
| ID | Task | Status | Effort | Priority |
|----|------|--------|--------|----------|
| 2.1 | Fixed timestep accumulator implementation | ❌ Not Started | 4h | High |
| 2.2 | Configurable fixed delta time | ❌ Not Started | 2h | Medium |
| 2.3 | Input system with frame normalization | ❌ Not Started | 6h | High |
| 2.4 | Input buffer for deterministic replay | ❌ Not Started | 4h | Medium |
| 2.5 | Frame interpolation for smooth rendering | ❌ Not Started | 4h | Medium |
| 2.6 | Pause/resume simulation support | ❌ Not Started | 3h | Low |
| 2.7 | Time scaling (slow motion, fast forward) | ❌ Not Started | 3h | Low |
| 2.8 | Simulation step debugging hooks | ❌ Not Started | 4h | Medium |

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
| Risk | Impact | Mitigation |
|------|--------|------------|
| Accumulator overflow with large delta times | Medium | Clamp maximum delta time |
| Input buffer memory usage | Low | Implement circular buffer with size limit |
| Determinism across platforms | High | Use fixed-point math for critical calculations |

### Dependencies
- Phase 1 (Core Data Structures) must be complete

---

## Phase 3: Rendering & Scenes (The Visuals) ❌ NOT STARTED

### Objective
Integrate rendering with the ECS and implement scene management for logical game state transitions.

### Acceptance Criteria
| ID | Task | Status | Effort | Priority |
|----|------|--------|--------|----------|
| 3.1 | Render system as ECS system | ❌ Not Started | 4h | High |
| 3.2 | Sprite rendering component | ❌ Not Started | 4h | High |
| 3.3 | Camera system with viewport management | ❌ Not Started | 6h | High |
| 3.4 | Render layers and z-ordering | ❌ Not Started | 4h | High |
| 3.5 | Scene interface refinement | 🟡 Partial | 4h | High |
| 3.6 | Scene manager with transition support | 🟡 Partial | 6h | High |
| 3.7 | Scene lifecycle hooks (OnLoad, OnUnload) | ❌ Not Started | 4h | High |
| 3.8 | Entity cleanup on scene unload | ❌ Not Started | 3h | Medium |
| 3.9 | Scene stack for nested scenes | ❌ Not Started | 4h | Medium |
| 3.10 | UI rendering layer | ❌ Not Started | 8h | Medium |
| 3.11 | Particle system | ❌ Not Started | 6h | Low |

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
| Risk | Impact | Mitigation |
|------|--------|------------|
| Render performance with many entities | High | Implement sprite batching |
| Z-ordering conflicts | Medium | Document clear layering guidelines |
| Scene transition memory leaks | Medium | Implement proper cleanup hooks |

### Dependencies
- Phase 1 (Core Data Structures) must be complete
- Phase 2 (Simulation Loop) should be complete

---

## Phase 4: Persistence & Assets (The Memory) ❌ NOT STARTED

### Objective
Implement save/load functionality for world state and async asset management.

### Acceptance Criteria
| ID | Task | Status | Effort | Priority |
|----|------|--------|--------|----------|
| 4.1 | Serialization format selection (JSON, binary, etc.) | ❌ Not Started | 2h | High |
| 4.2 | Entity serialization/deserialization | ❌ Not Started | 6h | High |
| 4.3 | Component serialization registry | ❌ Not Started | 4h | High |
| 4.4 | World state save/load | ❌ Not Started | 4h | High |
| 4.5 | Save game versioning and migration | ❌ Not Started | 4h | Medium |
| 4.6 | Asset interface definition | ❌ Not Started | 2h | High |
| 4.7 | Async asset loading system | ❌ Not Started | 6h | High |
| 4.8 | Asset caching and reference counting | ❌ Not Started | 4h | Medium |
| 4.9 | Texture atlas support | ❌ Not Started | 4h | Low |
| 4.10 | Audio asset management | ❌ Not Started | 4h | Medium |
| 4.11 | Font asset management | ❌ Not Started | 3h | Medium |

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
| Risk | Impact | Mitigation |
|------|--------|------------|
| Serialization version compatibility | High | Implement schema versioning with migration paths |
| Asset loading deadlocks | High | Implement timeout mechanisms |
| Memory bloat from cached assets | Medium | Implement LRU cache with size limits |

### Dependencies
- Phase 1 (Core Data Structures) must be complete
- Phase 3 (Rendering) should be complete for asset types

---

## Phase 5: Tooling & Polish (The UX) ❌ NOT STARTED

### Objective
Provide developer tools to accelerate content creation and improve the development experience.

### Acceptance Criteria
| ID | Task | Status | Effort | Priority |
|----|------|--------|--------|----------|
| 5.1 | CLI tool for project scaffolding | ❌ Not Started | 4h | Medium |
| 5.2 | CLI tool for asset validation | ❌ Not Started | 3h | Medium |
| 5.3 | CLI tool for performance profiling | ❌ Not Started | 4h | Medium |
| 5.4 | Visual editor foundation | ❌ Not Started | 12h | High |
| 5.5 | Scene composition in editor | ❌ Not Started | 8h | High |
| 5.6 | Hot-reloading for code and assets | ❌ Not Started | 8h | Medium |
| 5.7 | Comprehensive API documentation | ❌ Not Started | 8h | High |
| 5.8 | Usage examples and tutorials | ❌ Not Started | 8h | High |
| 5.9 | Performance benchmark suite | ❌ Not Started | 4h | Medium |

### Dependencies
- All previous phases should be complete

---

## Cross-Cutting Concerns

### Performance Targets
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Entity create rate | 10,000+ entities/sec | Benchmark test |
| Query performance | <1ms for 10,000 entities | Benchmark test |
| System update rate | 1,000+ systems/frame | Benchmark test |
| Memory per entity | <100 bytes | Profiling |
| FPS stability | 60fps with 5,000+ entities | Stress test |

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

| Milestone | Description | Target Date | Status |
|-----------|-------------|-------------|--------|
| **M0** | Project Bootstrap | 2026-08-23 | ✅ Complete |
| **M1** | Core ECS Complete | 2026-09-15 | 🟡 In Progress |
| **M2** | Simulation Loop | 2026-10-15 | ❌ Not Started |
| **M3** | Rendering & Scenes | 2026-11-15 | ❌ Not Started |
| **M4** | Persistence & Assets | 2026-12-15 | ❌ Not Started |
| **M5** | Tooling & Polish | 2027-01-31 | ❌ Not Started |
| **1.0** | Stable Release | 2027-02-28 | ❌ Not Started |

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

| Package | Responsibility | Public/Internal |
|---------|---------------|----------------|
| `castrum` | Root package, exports main engine API | Public |
| `castrum/config` | Configuration management | Public |
| `castrum/ecs` | ECS type definitions and interfaces | Public |
| `castrum/engine` | Main engine API (Game, World access, etc.) | Public |
| `internal/core` | Core ECS implementation | Internal |
| `internal/engine` | Engine runtime implementation | Internal |
| `internal/scene` | Scene management implementation | Internal |
| `internal/system` | System manager implementation | Internal |
| `internal/timers` | Timer management implementation | Internal |

### Appendix B: Key Design Decisions

| Decision | Rationale | Impact |
|----------|-----------|--------|
| ECS Architecture | Separation of data and behavior, performance, flexibility | Core architecture |
| Interface-based API | Encapsulation, testability, extensibility | Public API design |
| Fixed timestep | Determinism, simulation stability | Simulation loop |
| Internal package separation | Compiler-enforced encapsulation | Project structure |

### Appendix C: Glossary

| Term | Definition |
|------|------------|
| **ECS** | Entity-Component-System architectural pattern |
| **Entity** | Unique identifier for a game object |
| **Component** | Pure data attached to an entity |
| **System** | Logic that processes entities with specific components |
| **Scene** | Logical grouping of entities with lifecycle management |
| **World** | Container for all entities, components, and systems |
| **Fixed Timestep** | Constant time interval for simulation updates |
| **Frame Interpolation** | Smooth rendering between simulation steps |

---