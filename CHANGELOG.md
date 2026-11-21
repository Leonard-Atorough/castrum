# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> v0.0.x releases are omitted - they represent the experimental
> prototyping phase. The first rendered release is v0.1.0, the first
> public beta. See git history for v0.0.x details.

## [Unreleased]

### Added
- **breaking:** Update transform struct with tags and scene example (#247)
- Register asset types and preload file assets (#243)
- Register the sprite component under sprite.Sprite (#242)
- Implement the games data format "file" to unlock data driven workflow  (#238)
- Implement tick-stamped system-level scheduling (#234)
- Add Added/Changed query filters over per-row change ticks (#229)
- Implement resource and component registry (#225)
- Implement gamepad disconnect handling and release button actions (#221)
- Add animated sprite collection and conflict resolution (#216)
- Implement AnimatedSprite and SpriteClip (#215)
- Add loop count to Completed signal and update tests (#214)
- Implement new documentation website and migrate old v0.3.0 documentation (#205) (#206)
- Support display preference persistence (#194)
- **breaking:** Persist bindings and global deadzone (#193)
- Support preference persistence (#192)
- Add scripts for example validation and module pinning; upda… (#189)
- **breaking:** Replace atlas registration with serializable spritesheets (#188)
- Implement directory-backed storage and associated tests (#187)
- Enhance asset loading with cache policies (#186)
- **breaking:** Add strict paired JSON codecs (#184)
- Save typed values through stores or writers (#183)
- **breaking:** Scope decoder configuration per loader (#182)
- Enhance scheduling system with new phase management and entry methods (#180)
- Implement Reader.Consume for processing all unread signals (#179)
- Add audio, input, and display preferences (#162)
- Simulation time resource and the unified gametime package (#158)
- Add settings package for managing game preferences with validation and cloning (#148)
- Implement raycast functionality and improve collision detection tests (#147)
- **breaking:** Implement and enhance spatial audio system with listener support (#146)
- **breaking:** Add playlists and separate source controls (#143)
- Version documentation by release (#142)
- Implement state machine and state system, allowing for world-wide state management (#134)
- Implement pre-startup and pre-frame phases for improved system execution order (#133)
- Spawn entities owned by scene instances (#132)

### Changed
- **breaking:** Compose visual components, replace the Drawable union (#239)
- **breaking:** Update codec registration to allow shared access and immediate effect on loaders and savers (#236)
- Rename package blueprint to serialization and update references (#227)
- Redesign scene example to implement new scene workflow in a clearer way (#226)
- Implement scene redesign (#224)
- **breaking:** Simplify entity struct to a comparable handle wih world-owned lifetime. (#222)
- **breaking:** Replace animation with new AnimatedSprite component and system (#218)
- **breaking:** Move sprite from render package to standalone package (#213)
- Streamline sprite styling by introducing SpriteStyle (#212)
- **breaking:** Implement simplified audio API with unified player (#210)
- **breaking:** Protect engine system ordering (#181)
- Replace receiver name 's' with 'l' for consistency (#178)
- **breaking:** Registration strategy - declaration-driven systems, instance-driven features (#159)

### Documentation
- Remove old v0.3.0 archive (#208)
- Provide unified, correct documentation on time-control and "time" in castrum (#161)
- Document the unreleased playlist API (#144)

### Fixed
- **breaking:** Improve engine initialization by switching from errors to panics and reducing ceremony between game and runner (#261)
- Clean up some godoc to read better and less like placeholders (#259)
- Migrate fixtures off the removed sprite shape fields (#256)
- Define Style.Color as straight alpha and restore its tints (#246)
- Fix correctness bugs in serialization package around resources and type validation. (#235)
- Document the pause/scale same-advance boundary (#232)
- Release the texture cache write lock on duplicate misses (#220)
- Correct normals and contact lifecycles (#172)
- Prevent scaled clock and timer overflow (#171)
- Stabilize large-coordinate geometry queries (#169)
- Preserve rotated and stroked drawable visibility (#168)
- Honor playlist edits and playback validation (#166)
- Harden storage cleanup and resource resolution (#167)
- Bound grid cell expansion (#165)
- Reject unsafe frame indices (#164)
- Reserve director ownership records (#163)
- Improve query update performance and fix invalid query cache (#138)
- Batch archetype insertion and streamline entity creation (#137)
- Prevent out-of-bounds overflow in region validation (#136)
- Improve NaN and infinite value validation (#135)

### Internal
- Pin the File format's JSON shape (#258)
- Pin entity-vs-query write validation matrix (#231)
- Restore supported development commands (#170)

## [0.3.0] - 2026-10-30

### Added
- Implement built-in assets package with Go Regular font (#129)
- Add starter project template with basic game structure and movement system (#125)
- Add blueprint-based scene management (#124)
- Add system group management with AddSystemToGroup and SetGroupEnabled (#123)
- Integrate and enhance debug panel with tests and documentation (#117)
- Emit collision entered and exit signals, and audio finished signal. (#116)
- Add typed signals with tick and frame delivery windows (#115)

### Changed
- Replace Drain with Next for signal reading (#130)
- Strengthen and simplify package tests (#127)
- Refactor tests and remove deprecated time control and tween integration tests (#126)
- Update roadmap documentation and remove outdated files (#121)

### Fixed
- Release tutorial asset packaging (#131)
- **breaking:** Reject pointer components (#122)

## [0.2.0] - 2026-10-19

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


