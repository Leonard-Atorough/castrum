# Castrum Engine Development Roadmap

> **Version**: 1.0  
> **Last Updated**: 2026-09-08  
> **Target**: v1.0.0 with core features for indie games and pet projects

Castrum is a 2D game engine built in Go using Ebiten. This roadmap tracks features for v1.0.0, inspired by Bevy's v0.0 release. The goal is basic usability and suitability for creating indie games and small projects—not enterprise polish, but solid foundations.

---

## Done ✅

These features are implemented, tested, and integrated into the engine.

### Game & Worlds

- [x] Fixed-timestep game loop (60 TPS default, configurable)
- [x] Time scaling and pause/resume
- [x] FPS/TPS debugging overlay
- [x] Configuration management (YAML/TOML)

### ECS

- [x] Entity ID generation and lifecycle management
- [x] Component storage with type-based indexing
- [x] Entity creation/destruction and component add/remove
- [x] Tag-based and component queries
- [x] Entity hierarchy (parent/child relationships)
- [x] Archetype-based storage optimization
- [x] System interface and priority-based scheduling
- [x] Comprehensive unit and integration tests (>80% coverage)
- [x] Benchmark suite for performance validation

### 2D Rendering

- [x] Sprite rendering with texture loading
- [x] Primitive shape rendering (rectangle, circle, line)
- [x] Static render layers (Layer0–Layer10, LayerDebug) with draw-order bucketing
- [x] Depth-based layer y-sorting (isometric/top-down support)
- [x] Camera system (position, zoom, rotation, bounds)
- [x] Single camera support

### Scenes

- [x] Scene interface and lifecycle hooks (OnLoad, OnUnload)
- [x] Scene manager with transitions
- [x] Scene stack for nested/overlay scenes
- [x] Entity cleanup on scene unload

### Core Systems

- [x] Input handling (keyboard, mouse, gamepad normalization)
- [x] Input buffer infrastructure (deterministic replay ready)
- [x] Timer system for delayed/scheduled events with completion events
- [x] Animation system (sprite clip playback with loop/finish events)
- [x] Collision detection (AABB, circle, polygon; trigger support)
- [x] Spatial indexing for efficient collision queries

### Assets

- [x] Synchronous asset loading (blocking)
- [x] Asynchronous asset loading (worker pool, concurrent)
- [x] Batch asset loading with timeout support
- [x] Asset caching with reference tracking
- [x] Texture loading and management
- [x] Animation clip YAML definitions
- [x] Blueprint system (YAML-based entity templates with component properties)

### Events

- [x] Type-safe event bus
- [x] Publish-subscribe with handler registration
- [x] One-time handlers (Once)
- [x] Unsubscribe/cleanup
- [x] Event metadata (timestamp, source)
- [x] Built-in animation and timer completion events

---

## Accepted 🎯

Features committed for v1.0.0. Basic usability—not feature-complete, but suitable for indie games and pet projects.

### 2D Rendering: Texture Atlases

- [x] Texture atlas runtime support (load pre-baked atlases)
- [x] Sprite animation from atlas frames
- [x] Memory efficiency and batching optimization

### Sound

- [ ] Audio playback (WAV, MP3, OGG support via Ebiten)
- [ ] Sound effects and background music
- [ ] Volume control and mixing
- [ ] Spatial audio basics (pan)

### Particles

- [ ] Basic CPU particle system
- [ ] Particle emitters (burst and continuous)
- [ ] Simple particle physics (velocity, acceleration, lifetime)
- [ ] Particle rendering with layers

### Asset Management Polish

- [ ] Hot reload support for assets (development)
- [ ] Asset unloading and cleanup (manual LRU or implicit)
- [ ] Font asset loading and caching
- [ ] Generic asset pipeline for user-defined types

### Polish & Docs

- [ ] Public API documentation (godoc)
- [ ] Getting started guide
- [ ] Example projects (Hello World, simple platformer or puzzle game)
- [ ] Architecture overview document

---

## In Planning 🔄

Features under active discussion for v1.0.0 or v1.1.0. Not yet committed.

- [ ] Save/load system (world serialization)
- [ ] UI rendering layer and basic widgets
- [ ] Multi-camera and viewport support
- [ ] 2D physics engine (gravity, momentum, constraints)
- [ ] Tilemap rendering and editing
- [ ] Debugger/inspector tools
- [ ] Performance profiling tools
- [ ] Hot code reloading (development)

---

## Ideas 💡

Possible future features for v1.1.0+. Community feedback will guide prioritization.

- [ ] Atlas generation tooling (command-line atlas packer)
- [ ] Visual scene editor
- [ ] Networking and multiplayer support
- [ ] Advanced particle effects (GPU-based)
- [ ] Skeletal animation support
- [ ] 3D rendering (basic)
- [ ] Shader system and custom render passes
- [ ] Post-processing effects
- [ ] Hot code reloading
- [ ] Plugin system for extensions
- [ ] Asset pipeline and build system
- [ ] Profiler integration and optimization guides

---

## Quality Gates for v1.0.0

Before release, the following must be met:

- [x] ECS and core systems >80% test coverage
- [x] Core system tests: animation, input, collision, events, assets, scene, timers
- [ ] All public APIs documented with godoc
- [ ] No critical linting errors (`go vet ./...`)
- [ ] No goroutine leaks in stress tests
- [ ] Example game demonstrates all major systems end-to-end (should include at least: input, collision, animation, rendering, scenes, timers, assets, events)
- [ ] Performance benchmarks at or above baselines:
  - Entity creation: 10,000+ entities/sec
  - Query performance: <1ms for 10,000 entities
  - Sustained 60 FPS with 5,000+ entities on-screen
  - Memory per entity: <100 bytes

---

## Architecture Principles

**Castrum is an engine, not a library:**

- Single public import: `pkg/castrum`
- `internal/` is hidden and self-contained
- Developer-facing API hides ECS complexity
- Scenes are the primary user-facing structure

**Design for indie games:**

- Simple, predictable behavior
- Sensible defaults
- Clear error messages
- Minimal boilerplate
R
---

## Appendix: Version Numbering

- **v1.0.0**: Core engine stable, suitable for indie games and pet projects
- **v1.1.0+**: Advanced features (physics, UI, networking, editor) based on community feedback
