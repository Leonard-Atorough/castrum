# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> v0.0.x releases are omitted - they represent the experimental
> prototyping phase. The first rendered release is v0.1.0, the first
> public beta. See git history for v0.0.x details.

## [Unreleased]

### Added
- Implement chord input binding for simultaneous key actions and update related documentation (#114)
- Implement MovePosition method for constrained camera movement and add tests (#104)
- Enhance mouse wheel behavior and documentation for frame-scoped displacement (#102)
- Add per-source stereo balance across playback modes (#101)
- Implement tweening functionality with easing curves and tests (#100)
- Add support for named timer groups with pause and scale functionality (#99)
- Implement camera rotation, anchor, and bounds; zoom unified on Transform.Scale (#98)
- Implement game-level time control with pause and time scaling features (#95)

### Changed
- **breaking:** Rename Server to Loader and New to NewLoader (#29) (#113)
- Move configuration options and input wiring code out of main file into files within main package and update stale documentation. (#103)

### Documentation
- Enhance documentation for World and input packages (#96)

### Fixed
- Streamline component validation and archetype cleanup for efficiency (#112)
- Update component removal to snapshot surviving values in type order (#110)
- Release completed playback resources and reject NaN volume (#105)

### Internal
- Verify transform interpolation across angle and scale boundaries  (#107)

### Miscellaneous
- Enhance README with contributing section

### Performance
- Reduce per-tick bookkeeping (#109)

## [0.1.0] - 2026-10-10

### Added
- Add benchmarks for ECS, asset loading, collision, and rendering (#71)
- Add optional debug overlay for frame and tick rates (#70)
- Ship the playable your-first-game tutorial and tank example (#61)
- Replace AssetServer and related accessors with MustResource for resource retrieval (#60)
- Add text rendering support with TextSource drawable (#58)
- Refactor Animation and Clip types, update related logic and documentation
- Implement timer component with one-shot and repeating functionality, including system integration and comprehensive documentation (#56)
- Add the collision stack — shape predicates, spatial grid, and Contacts lifecycle (#55)
- Add a simple roadmap and feature set documentation (#54)
- Enhance PrevTransform to include rotation and scale, update related logic (#47)
- Remove obsolete guide rewrite skill documentation (#46)
- Add comprehensive guides for Castrum engine features (#45)
- Add getting started docs and example game (#44)

### Changed
- Enhance documentation clarity in camera, collector, and sprite components (#69)
- Enhance documentation clarity in ecs, runtime, and spatial packages (#68)
- Improve documentation clarity in input package (#67)
- Improve documentation clarity in ebitrun package (#66)
- Enhance documentation clarity across core files (#65)
- Enhance documentation clarity and improve comments across collider, contacts, and system files (#64)
- Enhance documentation clarity and improve asset handling in various files (#63)
- Improve documentation clarity across multiple packages (#62)
- Tighten the public api ahead of 0.1.0 render split, unexports, opt-in subsystems (#59)
- Consolidate transform logic and auto-register previous transform when transform is added (#43)

### Fixed
- Refactor animation to provide completion state accessors and remove unneeded methods (#57)

### Internal
- Add docs for contributors, update readme and fix deployments to release environment (#73)


