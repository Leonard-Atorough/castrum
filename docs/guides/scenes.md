# Scenes

A scene blueprint is a recipe for entities, not a separate ECS world. The `scene.Director` creates scene instances in the game's existing world and tracks which entities belong to each instance. Systems query scene-owned entities just like any other entities, and scene transitions do not change which systems are scheduled.

For a complete runnable example with scene transitions, see [Scenes](../../examples/scenes/main.go). For entities, components, systems, and resources, see [The ECS in depth](ecs.md).

## Enable scenes

Enable the scene systems when creating the game, then retrieve their director from the world:

```go
g, err := castrum.New(castrum.WithScenes())
if err != nil {
	return err
}
director := g.World().MustResource[*scene.Director]()
```

`WithDefaultSystems` also enables scenes. `WithScenes` registers a startup system that applies requests queued before startup and a frame system that applies requests made during gameplay. Pre-frame state transitions run before the frame scene system, so a scene request made by a state hook is applied before user frame systems in that same frame. A request queued by an ordinary frame system is applied on the next frame. `MustResource` is appropriate when scenes are enabled because the option provides the director. The director is single-threaded; do not access it concurrently.

## Define a blueprint

Give each blueprint a name and at least one root entity specification. An entity specification names an entity and provides its initial component values:

```go
type Velocity struct {
	X float64
}

level := &scene.Blueprint{
	Name: "level",
	EntityList: []scene.EntitySpec{{
		Name: "player",
		Components: []any{
			core.Transform{Position: geom.Vector2{X: 100, Y: 80}},
			Velocity{X: 60},
		},
	}},
}
```

The blueprint does not create entities until a request is processed. Before queueing, the director validates the blueprint: names must be non-empty, entity names must be unique among siblings, and every component must be a non-nil, non-pointer value. Components implementing `core.Validatable` must pass validation. Nested blueprints are checked as well, and asset IDs in `Blueprint.Assets` cannot be empty.

Register systems separately from the blueprint. For example, this reusable query moves any entity with both a `Transform` and `Velocity`, regardless of whether it was created by a scene:

```go
moving := core.NewQuery(g.World()).With(core.Transform{}, Velocity{})
if err := g.AddSystem(core.PhaseFixed, "move", core.SystemFunc(func(ctx *core.Context) error {
	for entity := range moving.Execute() {
		velocity, _ := entity.Component[Velocity]()
		entity.Update(func(t *core.Transform) {
			t.Position.X += velocity.X * ctx.DeltaTime.Seconds()
		})
	}
	return nil
})); err != nil {
	return err
}
```

Scene ownership manages entity lifetimes; it does not pause systems or add a runtime entity hierarchy. A child entity is still an ordinary ECS entity, and it does not inherit transforms from its parent specification. Entities created separately with `World.NewEntity` are not owned by a scene and remain when that scene is removed. To create an entity owned by a live scene instance from gameplay code, use `Director.Spawn`:

```go
if _, err := director.Spawn(ctx, levelID, core.Transform{}, Enemy{}); err != nil {
	return err
}
```

`Spawn` attaches the scene ownership record, so the entity is destroyed when that instance is removed or rebuilt. Pass the intended `InstanceID` explicitly when multiple instances may be live. `Spawn` requires a live instance; an ID returned by `Load` or `Switch` cannot be used until its request has been applied. Do not pass `scene.OwnershipRecord` as a component; the director manages it. See the [rendering guide](rendering.md) if scene entities need visual components.

## Queue scene requests

Use `Load` to add an instance without removing existing scenes, and `Switch` to replace live root instances with a new one. Both return an `InstanceID` reserved immediately:

```go
levelID, err := director.Load(level)
if err != nil {
	return err
}
```

The instance remains pending until the scene system processes the request. During startup, pre-startup systems run before the startup scene system; during each frame, pre-frame systems run before the frame scene system. `Load` and `Switch` validate the blueprint when called, but spawning entities and running hooks happen later and can still fail.

For example, queue a replacement for the current root scene with `Switch`:

```go
if _, err := director.Switch(field); err != nil {
	return err
}
```

Use `Remove` to tear down an instance and its nested scene instances. Use `Update` to rebuild a live instance from a replacement blueprint while preserving its ID:

```go
if err := director.Update(levelID, replacement); err != nil {
	return err
}
```

To remove a specific instance instead:

```go
if err := director.Remove(levelID); err != nil {
	return err
}
```

`Update` accepts only a live instance; it validates and snapshots the replacement when queued. The instance's `Name` remains the original name even when its replacement blueprint has a different name. Requests are processed in order. A failed request is consumed and returned as a system error; later requests remain queued for the next processing pass.

## Compose scenes

Use `EntitySpec.Children` for additional entities owned by the same scene instance, and `EntitySpec.SubScenes` for nested scenes with their own instance IDs and lifetimes. All parent-owned entities are created before the parent's `Enter` hook runs. Nested scenes enter after that hook. Removing or switching away from a parent also removes its nested instances.

`Switch` removes live root instances and their nested instances before entering its replacement. A `Load` instead leaves existing instances in place. Neither operation affects unrelated ECS entities or system registration.

## Inspect instances and hooks

Use `Get` to look up an instance by ID and `ByName` to find the first matching live or pending instance, ordered by ID:

```go
if current, ok := director.ByName("level"); ok {
	if err := director.Remove(current.ID); err != nil {
		return err
	}
}
```

Both methods return detached views. Mutating a returned `Instance` or its blueprint structure does not change the director; use `Update` to replace a live blueprint. Queued requests and returned views copy blueprint structure, asset and entity-spec slices, and component interface values. References held inside component values and state captured by hook closures remain shared.

`Hooks.Enter` runs after an instance's entities exist and before its nested scenes enter. If an `Enter` hook fails, entry is rolled back, including any nested scenes that already entered. `Hooks.Exit` runs while the instance's entities still exist. Exit hooks run children-first, and all hooks are attempted before teardown commits. If an exit hook returns an error, teardown still happens and the error is reported afterward. A `Switch` still attempts to enter its replacement after exit-hook errors, then reports those errors.

## Assets

`Blueprint.Assets` stores asset IDs as metadata; the scene system does not load them. Load assets through the normal asset APIs described in the [assets guide](assets.md).
