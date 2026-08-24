package engine

import (
	"github.com/leonard-atorough/castrum/internal/ecs"
	"github.com/leonard-atorough/castrum/internal/system"
)

type SystemAPI struct {
	mgr  *system.Manager
	wrld *ecs.World
}

func (s *SystemAPI) Register(name string, sys System) error {
	return s.mgr.Register(system.Player, name, sys, s.wrld)
}

func (s *SystemAPI) Unregister(name string) error {
	return s.mgr.Unregister(name, s.wrld)
}

func (s *SystemAPI) Update(world *ecs.World, deltaTime float64) {
	s.mgr.Update(world, deltaTime)
}
