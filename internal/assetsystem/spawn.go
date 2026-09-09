// Package assetsystem instantiates entities in the ECS from blueprints loaded
// via assets.Assets. It is the ECS-coupled counterpart to the public assets package.
package assetsystem

import (
	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/internal/ecs"
)

// CreateFromBlueprint creates a new entity from a blueprint's component data.
func CreateFromBlueprint(world *ecs.World, bp *assets.Blueprint) (*ecs.Entity, error) {
	components := make([]ecs.Component, len(bp.Components))
	for i, comp := range bp.Components {
		instance, err := ecs.Resolve(comp.Type, comp.Properties)
		if err != nil {
			return nil, err
		}
		components[i] = instance
	}

	return world.CreateWithComponents(bp.Name, components...)
}
