# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Move bump.sh into .github/scripts, rewrite benchstat gate in shell

### Miscellaneous
- Merge pull request #124 from Leonard-Atorough/feat/solidify-component-contract
- Go fmt fixes
- Remove old component tests file
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
- Minor updates, remove old collider component
- Migrate collider to dedicated file and introduce new collider offset. Make physics system offset aware
- Fmt fix for animation
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
- Merge pull request #123 from Leonard-Atorough:feat/add-origin-to-transform
- Fmt fixes
- Add origin to transform component to allow for offsetting transforms
- Merge pull request #122 from Leonard-Atorough/fix/asset-builder-should-only-error-on-build
- Merge pull request #121 from Leonard-Atorough/feat/redesign-animation-system
- Add test coverage to animation package
- Merge pull request #120 from Leonard-Atorough:docs/add-godoc-to-atlas-package
- Add sample atlas autosliced and rendered
- Use a name prefix instead of a name func
- Document atlas package
- Merge pull request #119 from Leonard-Atorough:feat/first-example-and-godoc
- Fmt fixes
- Move color to sprite, making transform entirely focused on spatial positioning and orientation.
- Mover example, demoing default input config, primitive rendering and system reg
- Gitignore executables in the example folder
- Merge pull request #118 from Leonard-Atorough:refactor/atlas-builder-to-internal
- Merge pull request #117 from Leonard-Atorough:feat/fix-broken-atlas-pipeline
- Fmt fix
- Implement high severity fixes with atlas validation and err handling
- Move debug rendering to internal render config, unalias-atlas in renderer
- To support new atlas operations and lay the groundwork for future improvements in animation handling, implement better renderer pathway. Now easier to follow and more readable.
- Add updated texture tests covering more non-file I/O paths
- Clean up old comented out code
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
- Merge pull request #116 from Leonard-Atorough:fix/bug-where-collider-is-scaled-when-transform-component-is-scaled
- Merge pull request #115 from Leonard-Atorough/refactor/redesign-asset-package
- Format files
- Remove unecessary type arguments and tidy go mod file
- Potential fix for pull request finding, nil encoder being registered
- Potential fix for pull request finding
- Merge branch 'refactor/redesign-asset-package' of https://github.com/Leonard-Atorough/castrum into refactor/redesign-asset-package
- Potential fix for pull request finding, fix for potential decoder being nil and passed into wrapper.
- Potential fix for pull request finding, nil result not checked before cast
- Refactor asset management: Split resource.go into saver.go and texture.go, implement texture provider
- Merge pull request #114 from Leonard-Atorough/feat/public-private-ecs-separation
- Refactor ECS World Management and Component Handling
- Merge pull request #113 from Leonard-Atorough/feature/expand-the-geometry-package-to-cover-core-needs
- Fmt fixes
- Refactor geometry structures and collision detection
- Merge pull request #112 from Leonard-Atorough/feature/rewrite-the-spatial-index-and-collision-engine
- Refactor collision point calculation with axes
- Merge branch 'feature/rewrite-the-spatial-index-and-collision-engine' of https://github.com/Leonard-Atorough/castrum into feature/rewrite-the-spatial-index-and-collision-engine
- Enhance NewGrid validation for cellSize
- Refactor physics collision system: enhance collision detection and remove spatial index
- Merge pull request #111 from Leonard-Atorough/feat/input-action-map
- Merge pull request #110 from Leonard-Atorough:fix/spatial-index-query-allocs
- Merge pull request #109 from Leonard-Atorough/Leonard-Atorough/issue90
- Merge pull request #106 from Leonard-Atorough/fix/continue-migration-towards-production-state
- Fmt fixes
- Refactor game structure and remove obsolete camera system
- Merge pull request #105 from Leonard-Atorough/feature/refactor-physics-system
- Merge branch 'main' into feature/refactor-physics-system
- Merge pull request #104 from Leonard-Atorough:feat/continue-test-refinements
- Benchmark refinements
- Format fixes
- Clean up comments
- Reduce struct allocation for small archetypes, prefer linear scan
- Minor test improvements
- Add iteration methods and thread safe R-W access
- Merge pull request #103 from Leonard-Atorough/fix/move-ecs-out-of-internal
- Finalise the scheduler basic design, movng game loop code into protected space
- Fully replace fatal failure with error failure, allowing tests to run on after one fails
- Doc and format fixes
- Merge pull request #102 from Leonard-Atorough:feat/raise-test-coverage-to-above-threshold
- Merge pull request #101 from Leonard-Atorough/Leonard-Atorough/issue95
- Merge pull request #100 from Leonard-Atorough/Leonard-Atorough/issue89
- Merge pull request #88 from Leonard-Atorough:fix/bug-fixes-in-assets-package
- Merge pull request #87 from Leonard-Atorough:fix/clean-up-event-test
- Merge pull request #86 from Leonard-Atorough:feature/add-test-coverage-for-atlas-api
- Merge pull request #85 from Leonard-Atorough/feature/add-test-coverage-for-animation-api
- Merge pull request #84 from Leonard-Atorough/fix/attempt-another-workflow-fix
- Merge pull request #83 from Leonard-Atorough/fix/attempt-another-workflow-fix

## [0.0.0] - 2026-09-09

### Fixed
- Enhance release workflow with git-cliff installation and tag resolution

### Miscellaneous
- Merge pull request #82 from Leonard-Atorough:fix/attempt-another-workflow-fix

## [0.1.0] - 2026-09-09

### Added
- Enhance release workflow with version bump options and changelog preview
- Add step to fetch tags before generating changelog
- Refactor spatial index to use ecs.EntityID
- Feat(animation): add AnimationEvent and AnimationManager for clip handling
- Implement collision detection and resolution system with tests
- Introduce AnimationManager and AnimationClipBuilder
- Enhance asset management with animation and atlas support
- Implement camera system with ECS architecture and update rendering logic
- Feat:
- Finalise renderer layers and implement y-sorting for depth-based rendering
- Enhance input handling with modifiers and add unit tests for input snapshot
- Replace Manager with InputHandler and add input buffering
- Implement release preparation and publishing workflows with version bump options
- Migrate from semantic-release to custom release script with version bump options
- Add issue templates for bug reports, feature requests, documentation updates, general questions, and security vulnerabilities
- Enhance QueryCache with size, isEmpty, and contains methods; update world queries to utilize cache
- Implement LRU query cache for optimized query results management
- Implement lazy tag/template index maintenance for improved entity creation performance
- Add benchmarking capabilities for Castrum ECS engine with comprehensive tests and utilities
- Add unit and integration tests for scene and manager functionality
- Update entity management terminology and enhance scene lifecycle hooks
- Enhance SceneAPI and Manager with new scene management methods and interface adjustments
- Enhance scene management with new SceneAPI methods and lifecycle hooks
- Add configuration management, scene and scene manager implementation
- Add Scene and SceneManager interfaces for scene management
- Add configuration management with validation and default settings
- Implement error handling in TimerManager with custom TimerError type
- Enhance TimerManager to auto-cleanup one-shot timers after expiry
- Implement TimerManager and Timer functionality with unit tests
- Implement system manager with registration, unregistration, and update functionality
- Enhance ECS with component management, tag handling, and error handling
- Implement error handling and lifecycle methods in ECS world
- Add tag management to entity and implement tag handling in world

### Changed
- Simplify go vet and test commands to use go list
- Remove Timer and TimerSystem implementations and associated tests
- Rename Renderable to Sprite and update related components
- Rename CollisionSystem to CollisionProcessingSystem and add error handling for collision tests
- Implement asynchronous asset loading with concurrency control
- Update roadmap and component structures, remove unused rotator system
- Update Timer and TimerSystem to emit TimerCompletedEvent instead of callbacks
- Enhance game structure with scene management and component initialization
- Separate clips as assets from component state
- Separate clips as assets from component state
- Refactor: Update component retrieval to use new error-handling method
- Reimplement timer system using ECS design pattern
- Update spatial management to use SpatialIndexHandler and improve error handling
- Rename InputState to InputSnapshot and encapsulate modifier keys
- Remove unnecessary blank lines and improve code formatting
- Update SceneAPI and Manager to use SceneLifecycle for scene management
- Refactor: rename TimerManager to Manager for consistency and update related references
- Rename UpdateTimers method to Update for consistency
- Change World type in Game and SystemAPI to ecs.World for consistency
- Replace uint64 with EntityID for entity identification across ECS components

### Documentation
- Add skills md file to help ai assisted ui workflow
- Update copilot instructions to clarify document creation guidelines
- Update ROADMAP.md to reflect current development status and phases

### Fixed
- Update checkout action to v6 and standardize Go version specification
- Fix (assets): move assets and stores to public api space, keep spawn function in internal and expose on game.
- Standardize string quotes and improve release verification logic
- Enforce semantic versioning for tags and streamline release process
- Remove 'v' prefix from version output in workflow_dispatch
- Update changelog generation to remove 'v' prefix from tag
- Add comprehensive bounds validation to query result iteration
- Fix formatting
- Fix: Cleanly separate public and private code
- Optimise query performance and make some improvement to the benchmark package
- Fix go formatting
- Fix: Remove deprecated query paths from the API
- Fix: add error handling around animation panics
- Update release job conditions to include specific author checks
- Standardize quotes in version input description in release workflow
- Update release workflow to support version input and streamline changelog generation
- Update collision manager to define and use spatial index interface for improved idiomatic integration
- Add missing Linux dependencies installation step in PR checks workflow
- Consolidate timer into one file, improve thread safety and handle potential callback panics.
- Move camera code to its own package
- Include input and physics directories in linting and testing
- Move scene manager and update tests
- Run CI tests under xvfb with missing X11 dependency
- Add installation of Linux dependencies in CI workflow
- Remove CGO_ENABLED=0 from test command for compatibility
- Set CGO_ENABLED=0 for tests to ensure compatibility in non-linux environments
- Remove Linux dependency installation from CI workflow and delete obsolete texture tests
- Update CI workflow to install Linux dependencies and remove build constraints from texture tests
- Add build constraints for non-linux environments in texture_test.go
- Update references from collision to physics in PR checks
- Refactor collision system to use physics package and add collision tests
- Fix: Z fighting when y-sort cannot resolve rendering on same layer.
- Delete old changelog
- Encapsulate maxDelta and maxIterationsPerFrame constants and update timestep clamping logic
- Refactor .releaserc.json for improved semantic versioning rules and formatting
- Update go vet command to skip specific packages requiring X11
- Update Go version format and refine test command to include additional packages
- Update test command to skip specific tests during coverage run
- Fix usage of reflection for passing components.
- Fix broken timer test
- Correct spelling of "creating" in roadmap and update scene interface status to done

### Internal
- Remove unused indirect dependency on jezek/xgb
- Enhance camera tests with zoom and clamping scenarios
- Remove outdated release workflows and add new release configuration using git-cliff
- V0.1.2
- Remove unnecessary blank line in prepare-release workflow
- Update changelog and version to v0.1.1 with new features and bug fixes

### Miscellaneous
- Merge pull request #81 from Leonard-Atorough/feature/improve-asset-store
- Merge pull request #80 from Leonard-Atorough:yaml-fix
- Merge pull request #79 from Leonard-Atorough/continue-release-pipeline-fix
- Merge pull request #78 from Leonard-Atorough/continue-release-pipeline-fix
- Merge pull request #77 from Leonard-Atorough:update-dependencies
- Minor release fix
- Merge pull request #76 from Leonard-Atorough:fix/update-release-yaml
- Merge pull request #75 from Leonard-Atorough/feature/migrate-animation-to-atlas-and-manager
- Merge pull request #74 from Leonard-Atorough/implement-texture-atlas-and-updates-to-dependencies-and-tests
- Format fixes
- Merge pull request #73 from Leonard-Atorough:Leonard-Atorough/issue51
- Merge pull request #72 from Leonard-Atorough:Leonard-Atorough/issue53
- Merge pull request #71 from Leonard-Atorough/Leonard-Atorough/issue69
- Refactor subscription removal in event handlers
- Fmt fixes
- Implement EventBus for event-driven architecture
- Merge pull request #70 from Leonard-Atorough:create-skill-file-for-ui-work
- Merge pull request #68 from Leonard-Atorough/async-asset-loading
- Upgrade to ebiten 2.10.0
- Merge pull request #67 from Leonard-Atorough/minor-updates-to-components-tests-and-roadmap
- Merge pull request #66 from Leonard-Atorough/Leonard-Atorough/issue55
- Refactor camera system and components
- Merge pull request #65 from Leonard-Atorough/feature/scene-stack-and-query-filtering
- Implement resource management system and scene builder enhancements
- Merge pull request #64 from Leonard-Atorough/feat/animation-clip-asset-system
- Merge pull request #63 from Leonard-Atorough/benchmark-optimisation
- Merge pull request #62 from Leonard-Atorough/Leonard-Atorough/issue47
- Merge pull request #61 from Leonard-Atorough/Leonard-Atorough/issue59
- Refactor camera and collision systems
- Merge pull request #58 from Leonard-Atorough/Leonard-Atorough/issue46
- Merge pull request #57 from Leonard-Atorough/Leonard-Atorough/issue44
- Merge pull request #56 from Leonard-Atorough/Leonard-Atorough/issue43
- Fix: panic in hot paths
- Merge pull request #42 from Leonard-Atorough/redesign-ecs-component-accessors-to-use-generic-methods
- Refactor component access methods in core and related modules
- Merge pull request #41 from Leonard-Atorough/implement-camera-system
- Merge pull request #39 from Leonard-Atorough/reimplement-timer-with-ecs-design
- Merge pull request #36 from Leonard-Atorough/improved-version-and-release-management
- Merge pull request #35 from Leonard-Atorough/Leonard-Atorough/issue31
- Merge pull request #34 from Leonard-Atorough/Leonard-Atorough/issue24
- Merge pull request #30 from Leonard-Atorough/Leonard-Atorough/issue27
- Merge pull request #29 from Leonard-Atorough/Leonard-Atorough/issue25
- [FEATURE] Rename collision package to physics
- Merge pull request #28 from Leonard-Atorough/Leonard-Atorough/issue26
- [DOCS]  Add copilot-instructions file to the codebase
- Merge pull request #23 from Leonard-Atorough/Leonard-Atorough/issue13
- Merge pull request #22 from Leonard-Atorough/Leonard-Atorough/issue9
- Format fixes
- Merge branch 'main' into Leonard-Atorough/issue9
- Merge pull request #12 from Leonard-Atorough/finalise-renderer-layers-and-implement-y-sorting
- Merge pull request #10 from Leonard-Atorough/Leonard-Atorough/issue8
- Input Buffer for replays
- Implement engine level time scaling and management
- Merge pull request #7 from Leonard-Atorough/implement-simpler-versioning
- Apply batched suggestions from code review
- Merge pull request #6 from Leonard-Atorough/release/v0.1.2
- Clean up CHANGELOG by removing old entries
- Merge pull request #5 from Leonard-Atorough/chore/continue-working-on-release-tooling
- Merge pull request #4 from Leonard-Atorough/switch-from-semantic-release-to-release-script
- Merge pull request #3 from Leonard-Atorough/fix-sematic-versioning
- Merge pull request #2 from Leonard-Atorough/housekeeping-and-documentation-work
- Spacing fix
- Merge pull request #1 from Leonard-Atorough/improve-query-design-using-go-iterator
- Add query tests for superset and excluded semantics
- Merge branch 'improve-query-design-using-go-iterator' of https://github.com/Leonard-Atorough/castrum into improve-query-design-using-go-iterator
- Refactor component map creation in query.go
- Remove debug message from renderer
- Refactor query usage in stress test: replace reflection with concrete type for Position component
- Refactor query system: replace legacy query methods with new query builder for improved performance and composability
- Add camera package: create Manager.go file for camera management
- Integrate collision management: add Collision system, update Game structure, and modify configuration for collision handling
- Add animatable component and animation manager with event handling
- Refactor rendering logic: optimize entity processing with renderItem struct and improve draw order handling
- Implement collision manager with spatial indexing and add unit tests for collision detection
- Add bounding box methods for colliders and implement collision detection logic
- Enhance spatial management: update Manager to include query radius, modify NewManager signature, and integrate spatial updates in Game
- Rename types to geom
- Add spatial indexing and management: implement Index and Manager types for spatial entity tracking and querying
- Refactor Animation component: update Transform to use geom.Vector2; implement Play, Stop, Reset, IsPlaying, GetFrameIndex, GetFrameTime, GetFrameSpeed, and GetFrames methods in Animation Manager
- Refactor components to use geom package for vector types; update collision and player controller systems accordingly
- Add Collider interface and Box/CircleCollider implementations; create CollisionSystem for handling collisions
- Remove outdated comments from QueryAny method for clarity and maintainability
- Split input and input manager
- Merge branch 'main' of https://github.com/Leonard-Atorough/castrum
- Add game components and systems for example: Implement Velocity, Player, Pulse, CameraSystem, MovementSystem, and PlayerController; remove RotatorSystem
- Refactor asset management: Update NewGame to accept filesystem, enhance Assets and Store for improved texture and blueprint loading
- Add polygon support to rendering system: implement Polygon type and update PrimitiveRenderer for polygon drawing
- Add comprehensive benchmarks for Castrum ECS engine
- Refactor texture management: move Store struct with texture handling methods and add unit tests for texture retrieval
- Enhance CameraSystem: Add input handling for zoom functionality and update camera registration in main
- Add CameraSystem to manage camera position based on player entity
- Refactor rendering and input systems: Update roadmap, enhance component accessors, and implement pulse system for dynamic scaling
- Add SceneTag type alias and optimize Hierarchy.Add method for child attachment
- Enhance Scene management: Add SceneTag component for entity scene membership tracking
- Refactor ECS: Remove tag and template indexing, simplify entity management
- Refactor input and animation managers to unify naming conventions and update renderer initialization
- Implement input and movement systems, add animation manager, and define player and velocity components
- Add unit tests for core components and rendering
- Clamp delta time in Update method to prevent excessive accumulation
- Enhance rendering system with layered rendering and debug info display
- Add rendering system with primitive shapes and example game implementation
- Refactor component management in World and add integration tests
- Add typed component API for improved entity management and simplify component queries
- Add system-related error handling and refine System interface documentation
- Implement removeEntity method in Archetype for efficient entity removal
- Refactor system management: remove Layer type and update registration logic to prioritize systems
- Improve asset management
- Move system code into core
- Implementing the rendering system
- Add some notes to render system
- Lets get rendering
- Update benchmark to use new add entity methods
- Ecs improvements
- Bumbling about trying to build the rendering without losing my mind
- Continued refactoring towards solid core, now with go 1.27 support
- Simplify timers. Manager manages create, update and remove, timers manager state
- Simplify index functions
- More work on blueprint system
- Migrate from returning entityid to returning entity to external callers, simplifying design
- Remove redundant game.go
- Add basics for blueprint/schema functionality so devs can reuse yaml blueprints for entity creation
- Begin work on blueprint system
- Rearrange methods in world
- Update world tests a little to hopefully have better structure and be less verbose.
- Significantly simplify the contracts between packages, the public-private interface. This now slims down to a public interface (engine) and a private group of packages (internal). Developers interact with public which then communicates with private
- Renaming and organising code part 1
- Clean up world
- Merge branch 'main' of https://github.com/Leonard-Atorough/castrum
- Major phase 1 feature:
- Heavier benchmark on memory
- Continue to improve core performance by fine tuning caching and implementing in multi-component queries
- Update benchmark tests
- Restructure code to be a bit flatter, extract public world API to make world usability easier. Devs shouldn't have to worry about implementing world but this exposes the interface to all systems
- Add tests for component and entity management in ECS
- Implement ECS architecture with component, entity, and hierarchy management
- Move game.go to the public pkg
- Create basic game struct, the runtime manager of the engine
- Add Makefile with build, test, lint, install, clean, run, and dev commands
- Add LICENSE file with Apache License 2.0 terms and conditions
- Add CODEOWNERS and README.md files with initial content
- Initialize go.mod and go.sum files with dependencies.


