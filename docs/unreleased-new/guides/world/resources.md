# Share a score between systems

An enemy's health belongs on that enemy. A round's score is different: several systems may need the same value, but it doesn't describe one particular enemy. A **resource** gives the world one value associated with a type.

We'll keep a count of defeated enemies and make it available to the cleanup system from [Update a group of entities](systems.md). The snippets use the development API and assume the same game `g`, `Health`, and `removeDefeated` function.

## Provide the score

Define the data at package level:

```go
type RoundScore struct {
	Defeated int
}
```

During setup, before starting the runner, register its constructor:

```go
if err := g.World().Provide(func(*core.World) (*RoundScore, error) {
	return &RoundScore{}, nil
}); err != nil {
	return fmt.Errorf("provide round score: %w", err)
}
```

Import `fmt` and `github.com/Leonard-Atorough/castrum/core`. `Provide` registers a constructor; it doesn't call it yet. The first request for `*RoundScore` constructs the value, and successful later requests return that same pointer. A world can have only one registered resource of a given type.

## Count removals

In `removeDefeated`, retrieve the score at the start of the system, before the query and before removing anything:

```go
score, err := ctx.World.Resource[*RoundScore]()
if err != nil {
	return fmt.Errorf("get round score: %w", err)
}
```

Then increment `score.Defeated` inside the removal loop, immediately **after** a successful `DestroyEntity` call. Each removed enemy now counts once; a failed removal returns an error instead of being counted as a success.

Another system can request the same `*RoundScore` to display the number or decide when the round is won. Use `Resource` when resolution can fail or registration is not guaranteed. `MustResource` panics on a missing or failing resource and is intended for engine-owned wiring guaranteed by the engine, not as a substitute for handling errors from game-defined resources.

Register the resource before a system needs it. A missing registration or constructor failure is returned by `Resource`; a failed constructor can be tried again on a later request. If a resource must be resolved at startup instead of on first use, register it with `ProvideEager`: the game's startup resolves eager resources and returns an error if one fails.

## Choose who owns the data

Use a component for data belonging to a particular entity, a resource for one world-wide value or service, and an ordinary Go variable captured by a system when sharing through the world would add no value. A resource isn't automatically saved or reset between rounds; your game decides when to change or persist it.

World access belongs on the game-loop goroutine. If a background worker needs data, give it a copy and bring its result back to the game loop before changing the world. See the existing [ECS guide](../../../unreleased/guides/ecs.md#resources) for more resource options, and [settings](../../../unreleased/guides/settings.md) for a separate preferences workflow.
