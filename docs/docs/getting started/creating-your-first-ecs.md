---
sidebar_position: 2
---

# Creating your first entities, components and systems

ECS stands for Entity-Component-System. It is a design pattern commonly used in game development to manage game objects and their behavior. In an ECS architecture:

- **Entities** are unique identifiers that represent objects in the game world.
- **Components** are data containers that hold the properties of entities.
- **Systems** contain the logic that operates on entities with specific components.

Castrum uses this pattern to efficiently manage game objects and their interactions. Traditional game engines often couple data and behaviour together. This also a good architecture. ECS trades some of this coupling for flexibility and performance by separating data from logic.

In simple terms, ECS separates the "what" (data) from the "how" (logic). Entities and components define the "what", while systems define the "how".

## Entities and Components

Entities are a simple handle struct containing a `uint64` identifier that uniquely represents an object in the game world. They do not hold any data themselves; instead, they are associated with components that store the actual data.

```go
type EntityID uint64
type Entity struct {id EntityID}
```

Lets define an entity called box. In the previous guide we setup a project with an empty Castrum project. In the `main.go` file add the following code:

```go
// add this in the import block:
// github.com/Leonard-Atorough/castrum/core

// add this before the runner execution
box, err := g.World().NewEntity()
```

Great... but this on its own won't do much. An entity without any components is like a key to an empty chest. To make it useful, we need to add components that define its properties and behavior.

Components are, as mentioned, data containers. In Castrum a component is a Go struct that can optionally implement a `Validatable` interface. It doesn't have to but every `Validatable` component is checked before it is stored.

Let's define and add a component called `spin` to out entity, with a single field called speed:

```go
type spin struct { speed float64 }
```

We can now update our earlier call to create an entity and add the component directly in the call. This is one of the main ways you'll create an entity with components in Castrum.

```go
box, err := g.World().NewEntity(spin{ speed: 10.0 })
```

Now we have an entity handle with a component that holds some data. We just need something that can act. That's where systems come in.

## Systems

In Castrum a system is a function that matches the system function signature. Systems are executed in phases called `Schedules`. A schedule is essentially a collection of systems that are run in a specific order during the game loop.

Define a system by writing a function that conforms to the system function signature. Add the following system to your project:

```go
func Rotate(ctx \*core.Context) error {
// logic implemented here
return nil
}

```

For now we won't add any logic to our system, we'll do so in the next section. In the meantime, you can register this system with a schedule to ensure it gets executed during the game loop.

Register the system in the game like this:

```go
if err := g.AddSystem(core.PhaseFrame, "box-rotation", Rotate); err != nil {
    panic(err)
}
```

So now we have a system registered, and a component added to our entity, we are ready to make it spin. But to do this we'll need to learn how to query for entities with specific components and update their data within a system.

## Querying and Updating Components

A `Query` in Castrum is used to find entities that have specific components. To create a query, you specify the component types you are interested in.

To find all entities with a `spin` component, you would create a query like this:

```go
spinQuery := core.NewQuery(g.World()).With(spin{})
```
