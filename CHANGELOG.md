# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> v0.0.x releases are omitted — they represent the experimental
> prototyping phase. The first rendered release is v0.1.0, the first
> public beta. See git history for v0.0.x details.

## [Unreleased]

### Added
- Implement the sprite rendering pipeline (#11)
- Implement basic pure mathematics package in the engine with vector2d and rectangle2d support (ported from castrum-old) (#9)
- Add unified asset server with atlas registry and runner texture provider (#8)
- Implement asset loading with caching and codec registration (#7)
- Add archetype query layer with cached matches and error-based construction (#6)
- Integrate entity into archetype and world (#5)
- Implement archetype storage strategy (#4)
- Add core engine skeleton with scheduler and world management (#3)
- Write out the basic contract for systems, schedules and the Ebiten runner. (#2)
- Intial commit scaffolding repo and laying down basic files

### Documentation
- Port over useful files, scripts and templates from deprecated castrum project (#1)

### Fixed
- Accept marker values in query With and Without instead of reflect.Type (#10)

### Internal
- Reserve v0.1.0 for the first public beta (#12)


