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

func (b *Blueprint) Construct(world *core.World, registry *Registry) (*core.EntityID, error) {
	entity := world.CreateEntity(b.Name)

	comps := make([]core.Component, len(b.Components))
	for i, comp := range b.Components {
		component, err := registry.Resolve(comp.Type, comp.Properties)
		if err != nil {
			return nil, err
		}
		comps[i] = component
	}

	// TODO: World batching. for now, iterate
	for _, comp := range comps {
		world.AddComponent(entity, comp)
	}

	for _, tag := range b.Tags {
		world.AddTag(entity, tag)
	}
	return &entity, nil
}
