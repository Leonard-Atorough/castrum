---
sidebar_position: 2
---

# Entities, Components and Systems

We've already briefly talked about `ECS` or Entity Component System architecture but if you're not familiar with it you've probably been scratching your head in a bit of confusion so lets break it down.
Entity Component System is a software paradigm that aims to separate the **logic** or **how** of a software system from the **data** or **what**.

In plainer terms, `Entities` are unique objects, consist of `Components` and are modified by `Systems`.

## Entities - Unique Objects in the Game

In Castrum entities represent unique objects within the game. An entity is a simple handle or identifier. It holds no data other than the unique ID that distinguishes it from other entities and carries methods to interact with its components.

```go
entity, err := game.World().NewEntity()
if err != nil {
	// handle the error
}

if err := entity.AddComponent(game.World(), core.Transform{Position: geom.Vector2{X: 0, Y: 0}}); err != nil {
	// handle the error
}
```

## Components - Data Containers

A component is a data container, specified using a pure struct. Components can be as granular or as broad as needed, and often the size and shape of your components will depend on the game you're building. To define a component you simply define a struct:

```go
type Pet struct {
    Name string
    Age  int
    Breed string
}
```

You can then use this struct to add a component to an entity when creating it:

```go
entity, err := game.World().NewEntity(Pet{Name: "Fido", Age: 3, Breed: "Labrador"})
if err != nil {
	// handle the error
}
```

Or you can add the component to an existing entity using the `AddComponent` method on the entity handle:

```go
if err := entity.AddComponent(game.World(), Pet{Name: "Fido", Age: 3, Breed: "Labrador"}); err != nil {
	// handle the error
}
```

## Systems - Logic Executors

Systems in castrum are functions that run on a schedule. These functions contain the game logic and operate on entities and their components. A system must implement the `System` interface:

```go
type system struct {
}

func (s *system) Update(ctx *Context) error {
    // implement system logic here
    return nil
}

func newSystem() core.System {
    return &system{}
}
```

Alternatively, a system must conform to the function signature expected by the engine for systems:

```go
func newSystem() core.System {
    return core.SystemFunc(func(ctx *Context) error {
        // implement system logic here
        return nil
    })
}
```

Either option is valid; the choice depends on your preference and the complexity of the system logic you need to implement.
