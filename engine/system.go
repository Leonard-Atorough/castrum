package engine

import (
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/internal/core"
	"github.com/leonard-atorough/castrum/internal/system"
)

type SystemAPI struct {
	mgr  *system.Manager
	wrld *core.World
}

func (s *SystemAPI) Register(name string, sys ecs.System) error {
	return s.mgr.Register(system.Player, name, sys, s.wrld)
}

func (s *SystemAPI) Unregister(name string) error {
	return s.mgr.Unregister(name, s.wrld)
}

func (s *SystemAPI) Update(deltaTime float64) {
	s.mgr.Update(s.wrld, deltaTime)
}
