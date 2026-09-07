# Castrum Game Engine Copilot Instructions

These instructions guide the Copilot AI on how to assist with the Castrum Game Engine project. The goal is to provide accurate and context-aware code suggestions that align with the project's coding standards and architecture.

## Code Layout

- internal:
  - animation: Animation systems for managing and playing character and object animations.
  - assets: Asset management systems for loading and caching game resources.
  - audio: Audio systems for sound effects and music playback.
  - blueprint: Blueprint systems for defining reusable game object templates.
  - camera: Camera systems for managing viewports and rendering perspectives.
  - core: ECS core systems and entity management.
  - input: Input handling systems for keyboard, mouse, and game controllers.
  - physics: Physics systems for collision detection and rigid body dynamics.
  - render: Render systems for sprites and other visual elements.
  - scene: Scene management systems for organizing and transitioning between game levels.
  - spatial: Spatial index and system for efficient querying and management of entities in the game world.
  - timers: Timer systems for scheduling and managing timed events.
- geom: Geometry package providing primitive geometries as well as methods for working with them. Standard game dev offering.
- components: Core engine defined components

## Rules

- Prefer showing code over prose explanations; only explain if asked "why" or if test failures require explanation of related fixes.
- Error wrapping with %w. No panic outside main/init paths.
- Hot paths (RenderSystem, Transform math): no heap allocs per frame, preallocate slices with capacity, concrete types over interface dispatch.
- After changes: run `go vet ./... && go test ./...`, fix failures before responding.
- Don't refactor beyond the scope of my request, except to fix failures revealed by vet/test.
- Don't create documents when not asked to do so, if a document is deemed beneficial communicate this first. Aim to condense information into chat friendly messages.
- Avoid documenting information ovely verbosely.
