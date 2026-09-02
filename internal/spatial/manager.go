package spatial

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/geom"
	"github.com/leonard-atorough/castrum/internal/core"
)

type Manager struct {
	index       *Index
	queryRadius float64
}

func NewManager(cellSize float64, radius float64) *Manager {
	return &Manager{
		index:       NewIndex(cellSize),
		queryRadius: radius,
	}
}

func (mgr *Manager) Update(world *core.World, deltaTime float64) {
	transforms := core.QueryFor[components.Transform](world)

	for _, entityID := range transforms {
		transform, _ := core.GetComponent[components.Transform](world, entityID)
		mgr.index.Update(entityID, transform.Position)
	}
}

func (mgr *Manager) Query(position geom.Vector2) []core.EntityID {
	return mgr.index.Query(position, mgr.queryRadius)
}
