# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> Pre-v0.2.0 releases (v0.0.0, v0.1.0) are omitted — they represent the
> experimental prototyping phase. See git history for details.

## [Unreleased]

### Changed
- Move bump.sh into .github/scripts, rewrite benchstat gate in shell

### Internal
- Simplify release workflow, conventional-commits changelog, pre-release support

### Miscellaneous
- Add tag and scene tag tests
- Add collider tests and fix type casting
- Add camera tests and fix data type casting for json
- Add sprite component tests and fix casting issue for json
- Add tests for timers
- Add animation test and update animation to cast to int type properly
- Add transform tests, update transform component
- Add new animation test file
- Add scene tag and tag components
- Update timer to have serialization and validation, add doc comments
- Migrate camera to satisfy component core desire
- Migrate collider to dedicated file and introduce new collider offset. Make physics system offset aware
- Move animation to standalone file and implement serialization
- Add missing godoc
- Update examples to set opacity and update rendering to handle sprite flipping and opacity
- Migrate sprite component to dedicated file and update it to serialize
- Update component registration to adhere to new serializable signature
- Migrate asset tests components to use new serializeable signature
- Add serialization to transform component

### Performance
- Replace github-action-benchmark with benchstat + allocs/op gating
- Stabilize CI benchmarks (1s benchtime, skip sub-ns, advisory ns/op)

## [0.2.0] - 2026-09-16

### Added
- Redesign animation system with internal/public split, Game.NewClip facade, and bug fixes
- Add path normalization for asset loading and saving, enhance tests for path handling
- Enhance Loader with with singleflight support to prevent duplicate requests causing loader or cache errors.
- Enhance Saver and Loader with invalidation support, add tests for TextureProvider
- Enhance Saver with SavePath and Save methods, add test cases for asset saving
- Implement asset loader and saver with caching and encoding support
- Implement archetype, component registry and storage management with tests in internal/ecs package
- Enhance system registration with priority constants and improve collider shape validation
- Expand Polygon functionality with validation and introduce Segment type
- Enhance collisionProxy with layer and mask, add tests for filter changes
- Add configurable action mappings
- Enhance spatial indexing with QueryInto method and seen tracking
- Add comprehensive benchmark results for various game loop scenarios and entity management
- Fix undiscovered archetype boxing issue caused by indirecting components to []Component. Use reflection internally to handle type coercion before storing in the archetype, and then type coerce out of the archetype.
- Refactor phsysic system part 1
- Add scheduler for later
- Refactor structure to be more idiomatic. The goal of this is to make the engine easier to use as is and prevent hiding internal unecessarily.
- Enhance Emit method to support one-time subscriptions and prevent duplicate calls
- Improve test coverage by refining test commands and formatting
- Add comprehensive tests for Polygon methods and functionality
- Refactor input snapshot tests for clarity and coverage
- Add shutdown test to ensure no panic or error occurs
- Remove GetTypeInfo tests after function deprecation
- Add result methods tests for query functionality
- Enhance camera and timer tests, add new functionality for NewCamera and NewTimer
- Add tests for layer cap and default primitive type in NewSprite
- Add unit tests for NewSprite and update layer types in components
- Add unit tests for NewTransform and NewTransformWithDefault functions
- Refactor atlas management and add test coverage for Builder and AtlasStore
- Update AnimationClip and AnimationClipBuilder with tests; replace Manager with AnimationClipStore

### Changed
- Optimize snapshot cloning using maps.Copy
- Isolate backend polling
- Refactor: enhance PhysicsSystem state management with seen, candidates, and tested maps
- Update benchmark documentation and add utility benchmarks for component type lookups and entity reuse
- Update game loop benchmark descriptions for clarity and accuracy
- Update benchmark tests for improved memory tracking and entity creation
- Move event bus into resource container as managed singleton. This ensures that the event bus being accessed by one system is the same as the event bus being accessed by another.
- Refactor tests
- Streamline benchmark workflow by removing unused steps and improving output handling

### Documentation
- Add pull request template for improved contribution guidelines

### Fixed
- Fix fluent builders for atlas and clip
- Sprite doc comment now explains that RegionName is the initial/fallback region for static sprites, and that when an Animation component is present the renderer uses the animation's current frame instead. No code change needed — the renderer already handles this.
- Fix builder assetID reference
- Fix(animation):
- Removed the composite (atlasID, assetID) key. The store is now keyed by atlasID alone.
- Document the finalised beta launch assets package.
- Scale is now always a mutiplier. sprite gains a new size field for when we render primitives
- Fix a leftover issue where developers could create Atlases without a service. All atlas creation now goes through game.
- Fix test that broke on linux
- Fix some minor issues to improve performance and clean up code.
- Add/update tests to better validate the design of the renderer and atlas handling
- Fix: pointer-vs-value type mismatch in the atlas store
- Collision bug where the collider physics system was scaling the collider by the transform component scale. Collider should be independent
- Fix clear cache not notifying its listeners
- Load default physics settings when omitted
- Update benchmark alert thresholds and disable fail-on-alert
- Fix missing event bus issue in tests
- Fix: fix migrateEntityToNewArchetype errors:
- Use FNV-1a hash with full type string to prevent collisions
- Remove re-exports now that we've moved packages externally
- Fix blueprint so it uses error type
- Clean up animation doc comments, adding a package level doc as well.
- Update pull request trigger and adjust alert/fail thresholds
- Simplify cache key to avoid cross-PR pollution
- Isolate benchmark cache per pull request to prevent cross-PR pollution
- Update alert and fail thresholds for benchmark results
- Add step to create cache directory before downloading benchmark history
- Fix(query): optimize slice allocation in All and EntityIDs methods
- Correct output file path for benchmark results
- Fix(renderer): optimize render item handling by reusing buffer to reduce allocations
- Implement minor fixes to asset loading logic and improve error handling
- Update event subscription syntax for consistency
- Format fixes
- Update changelog preview command to include all unreleased changes
- Ensure explicit mode override is honored in version bumping

### Internal
- Add better test coverage to public asset package
- Remove engine architecture migration documentation
- Add comprehensive tests for cache and registry functionalities
- Add unit tests for physics system initialization and updates
- Add refactor option to pull request template
- Test fixes

### Miscellaneous
- Add origin to transform component to allow for offsetting transforms
- Add test coverage to animation package
- Add sample atlas autosliced and rendered
- Use a name prefix instead of a name func
- Document atlas package
- Move color to sprite, making transform entirely focused on spatial positioning and orientation.
- Mover example, demoing default input config, primitive rendering and system reg
- Gitignore executables in the example folder
- Implement high severity fixes with atlas validation and err handling
- Move debug rendering to internal render config, unalias-atlas in renderer
- To support new atlas operations and lay the groundwork for future improvements in animation handling, implement better renderer pathway. Now easier to follow and more readable.
- Add updated texture tests covering more non-file I/O paths
- Add coverage for atlas service
- Add test coverage for atlas
- Implement atlas store invalidation and update internal atlas file to service
- Integrate sptiresheet based textures to render path
- Migrate from old atlas pipeline to new atlas pipeline, with builder first implementation and texture integration
- Migrate from old atlas pipeline to new atlas pipeline, with builder first implementation and texture integration
- Update tests
- Add atlasid and region name to sprite
- Rename sprite.layer to sprite.renderLayer
- Move renderer back into internal, rename textureprovider to texture to fix clash
- Implement builder pattern for atlas
- Remove unecessary type arguments and tidy go mod file
- Refactor asset management: Split resource.go into saver.go and texture.go, implement texture provider
- Refactor ECS World Management and Component Handling
- Refactor geometry structures and collision detection
- Refactor collision point calculation with axes
- Enhance NewGrid validation for cellSize
- Refactor physics collision system: enhance collision detection and remove spatial index
- Refactor game structure and remove obsolete camera system
- Benchmark refinements
- Reduce struct allocation for small archetypes, prefer linear scan
- Minor test improvements
- Add iteration methods and thread safe R-W access
- Finalise the scheduler basic design, movng game loop code into protected space
- Fully replace fatal failure with error failure, allowing tests to run on after one fails


