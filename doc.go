// Package castrum provides a component-based game engine built on Ebiten.
//
// # Architecture
//
// Castrum uses an Entity Component System (ECS) architecture with the following core concepts:
//   - [Entity]: A unique actor in the game world, identified by [EntityID].
//   - [Component]: Pure data structures attached to entities (Transform, Collider, Renderable, etc.).
//   - [System]: Stateful logic that processes components each frame. Developers extend behavior here.
//   - [World]: The container managing all entities, components, and queries.
//   - [Scene]: A logical grouping of entities (levels, menus, etc.). Scenes control structure; Systems control logic.
//
// # Quick Start
//
// Create a game and register a custom system:
//
//	game, err := castrum.NewGame(config, filesystem)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer game.World.Cleanup()
//
//	// Register a custom system for game logic
//	err = game.Systems.Register("mySystem", 0, &MyGameSystem{}, game.World)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Load a scene
//	sceneManager := game.Scenes()
//	sceneManager.LoadScene(game.World, "mainLevel")
//
//	// Run game loop via ebiten
//	if err := ebiten.RunGameWithOptions(game, &ebiten.RunGameOptions{}); err != nil {
//		log.Fatal(err)
//	}
//
// # Extension Points
//
// ## Custom Systems
//
// Implement the [System] interface to add game logic:
//
//	type MyGameSystem struct{}
//
//	func (s *MyGameSystem) Init(world *castrum.World) error {
//		// Called once when the system is registered
//		return nil
//	}
//
//	func (s *MyGameSystem) Update(world *castrum.World, delta float64) error {
//		// Called each frame with delta time in seconds
//		q := world.NewQuery().WithRequiredComponents(reflect.TypeOf((*Transform)(nil)).Elem()).Build()
//		for _, entity := range q.Entities() {
//			// Process entities with Transform component
//		}
//		return nil
//	}
//
//	func (s *MyGameSystem) Shutdown(world *castrum.World) error {
//		// Called when the system is unregistered
//		return nil
//	}
//
// ## Custom Scenes
//
// Implement the [Scene] interface for custom scene logic:
//
//	type MenuScene struct{}
//
//	func (s *MenuScene) OnLoad(world *castrum.World, data map[string]any) error {
//		// Called when scene is loaded; data contains scene-specific state
//		return nil
//	}
//
//	func (s *MenuScene) OnUnload(world *castrum.World) error {
//		// Called when scene is unloaded
//		return nil
//	}
//
// ## Custom Components
//
// Create any struct as a component—no interfaces required:
//
//	type Health struct {
//		Current int
//		Max     int
//	}
//
//	type PlayerController struct {
//		Speed float64
//	}
//
// Add optional hooks if you need lifecycle events:
//
//	func (h *Health) OnCreate(id castrum.EntityID) {
//		// Called when component is first added to an entity
//	}
//
//	func (h *Health) OnDestroy(id castrum.EntityID) {
//		// Called when component is removed from an entity
//	}
//
// # Public API
//
// The following packages are intended for developers:
//   - castrum (this package): Game lifecycle, World, Entity management, Scenes, Systems
//   - [github.com/leonard-atorough/castrum/components]: Built-in components (Transform, Renderable, Collider, etc.)
//   - [github.com/leonard-atorough/castrum/geom]: Geometry primitives (Vector2, Rect, Circle, Polygon)
//
// The internal/ package is for engine internals and should not be imported directly.
//
// # Error Handling
//
// Use [errors.Is] to check for specific error conditions:
//
//	entity, err := game.World.Create("player")
//	if err != nil {
//		if errors.Is(err, castrum.ErrInvalidEntity) {
//			// Handle invalid entity state
//		}
//		return err
//	}
//
// # Scene Management
//
// Scenes are loaded/unloaded via the scene manager:
//
//	scenes := game.Scenes()
//	if err := scenes.LoadScene(game.World, "level1"); err != nil {
//		log.Fatalf("failed to load scene: %v", err)
//	}
//
// # Component Queries
//
// Query entities with specific components:
//
//	// Find all entities with Transform and Collider
//	q := game.World.NewQuery().
//		WithRequiredComponents(reflect.TypeOf((*components.Transform)(nil)).Elem(),
//			reflect.TypeOf((*components.Collider)(nil)).Elem()).
//		Build()
//	for _, entity := range q.Entities() {
//		// Process entity
//	}
//
// # Engine Subsystems (Built-in)
//
// The engine manages these systems automatically:
//   - Input: Keyboard, mouse, gamepad input state
//   - Physics: Collision detection
//   - Animation: Sprite animation playback
//   - Render: Drawing entities and debug info
//   - Camera: Viewport transformation
//   - Spatial: Spatial indexing for efficient queries
//   - Timers: Timed callbacks
//
// Developers typically don't interact with these directly—they're configured via [Config] and accessed via [Game.Systems] if needed.
//
// # What NOT to Use
//
// The following are internal implementation details and should not be used:
//   - ComponentRegistry (internal component type tracking)
//   - Archetype, ArchetypeKey (internal entity storage)
//   - Hierarchy (internal parent-child tracking)
//   - Error wrapper types (WorldError, EntityError, etc.)—use sentinel errors instead
//
// These implementation details may change without notice.
package castrum
