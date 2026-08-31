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
	components := make([]core.Component, len(b.Components))
	for i, comp := range b.Components {
		instance, err := core.Resolve(comp.Type, comp.Properties)
		if err != nil {
			return nil, err
		}
		components[i] = instance
	}

	entity, err := world.CreateWithComponents(b.Name, components...)
	if err != nil {
		return nil, err
	}

	for _, tag := range b.Tags {
		world.AddTag(entity.ID, tag)
	}
	return entity, nil
}
