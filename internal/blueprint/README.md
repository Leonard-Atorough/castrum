# Blueprint Package

## Overview

The blueprint package provides the code that manages blueprints in the engine. In castrum you can loosely think of a blueprint as similar to a PackedScene in Godot or a Prefab in Unity. A blueprint is a collection of components that can be instantiated into the world. Blueprints are used to define the structure of entities, including their components and properties.

## Anatomy of a Blueprint

Blueprints are yaml structured

Example blueprint structure:

```yaml
name: "MyBlueprint"
components:
  - type: "Transform"
    properties:
      position: [0, 0, 0]
      rotation: [0, 0, 0]
      scale: [1, 1, 1]
  - type: "MeshRenderer"
    properties:
      mesh: "MyMesh"
      material: "MyMaterial"
```

## In code the blueprint package provides the following main classes:

```go
package blueprint

// Blueprint represents a blueprint in the engine.
type Blueprint struct {
    Name       string
    Components []Component
}

// Component represents a component in a blueprint.
type Component struct {
    Type       string
    Properties map[string]interface{}
}
```

The key things I need to make this work:

- Loader for loading blueprints. Accepts an io.reader and returns an unmarshaled blueprint object. This is the main entry point for loading blueprints from files or other sources.
- Constructor for creating a blueprint from a reader. This function wraps the loader and provides a convenient way to directly obtain a blueprint instance.
- Accessor methods for retrieving blueprint information. These methods provide convenient ways to access the name, components, and version of a blueprint.
- Mutator methods for modifying blueprint information. These methods allow you to update the name, components, and version of a blueprint after it has been created.
- Validation methods for ensuring blueprint integrity. These methods check that the blueprint conforms to expected structures and contains all required components and properties.
- Serialization methods for saving blueprints. These methods allow you to convert a blueprint back into a YAML format for storage or transmission.

Example:

```go

type Loader struct{}

func (l *Loader) Load(r io.Reader) (*blueprint.Blueprint, error) {
    var bp blueprint.Blueprint
    decoder := yaml.NewDecoder(r)
    if err := decoder.Decode(&bp); err != nil {
        return nil, err
    }
    return &bp, nil
}

---
type Blueprint struct {
    Name       string
    Components []RawComponent
}

type RawComponent struct {
    Type       string
    Properties map[string]interface{}
}


func (b *Blueprint) Construct(reader io.Reader) error {
	decoder := yaml.NewDecoder(reader)
	return decoder.Decode(b)
}
// here we actually need to read the blueprint, construct the entity and add it to the world. Then we need to batch add the components so we don't have as many allocs.
```
