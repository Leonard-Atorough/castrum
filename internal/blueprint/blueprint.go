package blueprint

import (
	"github.com/leonard-atorough/castrum/internal/core"
)

type Blueprint struct {
	Name       string          `yaml:"name"`
	Components []ComponentData `yaml:"components"`
	Tags       []string        `yaml:"tags"`
	Version    string          `yaml:"version"`
}

type ComponentData struct {
	Type       string         `yaml:"type"`
	Properties map[string]any `yaml:"properties"`
}

func (b *Blueprint) Spawn(world *core.World) (*core.Entity, error) {
	entity := world.CreateEntity(b.Name)

	comps := make([]core.Component, len(b.Components))
	for i, comp := range b.Components {
		component, err := core.Resolve(comp.Type, comp.Properties)
		if err != nil {
			return nil, err
		}
		comps[i] = component
	}

	// TODO: World batching. for now, iterate
	for _, comp := range comps {
		world.AddComponent(entity.ID, comp)
	}

	for _, tag := range b.Tags {
		world.AddTag(entity.ID, tag)
	}
	return entity, nil
}
