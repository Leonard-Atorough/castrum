---
sidebar_position: 2
---

# Entities, Components and Systems

We've already briefly talked about `ECS` or Entity Component System architecture but if you're not familiar with it you've probably been scratching your head in a bit of confusion so let's break it down.
Entity Component System is a software paradigm that aims to separate the **logic** or **how** of a software system from the **data** or **what**.

In plainer terms, `Entities` are unique objects that consist of `Components` and are modified by `Systems`.

## Entities: Unique Objects in the Game

In Castrum entities represent unique objects within the game. An entity is a simple handle or identifier. It holds no data other than the unique ID that distinguishes it from other entities and carries methods to interact with its components.

The snippets on this page assume `game` is the `*castrum.Game` returned by `castrum.New`. They leave out imports.

```go
entity, err := game.World().NewEntity()
if err != nil {
	// handle the error
}

if err := entity.AddComponent(game.World(), core.Transform{Position: geom.Vector2{X: 0, Y: 0}}); err != nil {
	// handle the error
}
```

## Components: Data Containers

A component is a data container, specified using a pure struct. Components can be as granular or as broad as needed, and often the size and shape of your components will depend on the game you're building. To define a component you simply define a struct:

```go
type Pet struct {
    Name string
    Age  int
    Breed string
}
```

Components must be plain struct values, not pointers. `AddComponent` returns an error if you pass a pointer.

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

## Systems: Logic Executors

Systems in Castrum contain the game logic. Each one runs in a phase and operates on entities and their components. To be a system, a value must implement the `core.System` interface:

```go
type system struct {
}

func (s *system) Update(ctx *core.Context) error {
    // implement system logic here
    return nil
}

func newSystem() core.System {
    return &system{}
}
```

Alternatively, wrap a function with `core.SystemFunc` if you don't need a struct:

```go
func newSystem() core.System {
    return core.SystemFunc(func(ctx *core.Context) error {
        // implement system logic here
        return nil
    })
}
```

Either option is valid; the choice depends on your preference and the complexity of the system logic you need to implement.

Register a system with `AddSystem`, passing the phase it should run in and a unique name. Put movement and physics in `core.PhaseFixed`, which runs at the simulation rate. Put input handling in `core.PhaseFrame`.

```go
if err := game.AddSystem(core.PhaseFixed, "pets.update", newSystem()); err != nil {
	// handle the error
}
```
