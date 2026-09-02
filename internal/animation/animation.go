package animation

import (
	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/core"
)

type Manager struct {
}

func NewManager() *Manager {
	return &Manager{}
}

func (am *Manager) Update(world *core.World, delta float64) error {
	entities := world.Query(core.Types(components.Renderable{}, components.Transform{}, components.Animation{})...)

	for _, entityID := range entities {
		animComp, err := core.GetComponent[components.Animation](world, entityID)
		if err != nil {
			continue
		}
		renderComp, err := core.GetComponent[components.Renderable](world, entityID)
		if err != nil {
			continue
		}

		if !animComp.Playing {
			continue
		}

		animComp.FrameTime += delta

		if animComp.FrameTime >= animComp.FrameSpeed {
			animComp.FrameTime = 0
			animComp.FrameIndex++
			if animComp.FrameIndex >= len(animComp.Frames) {
				if animComp.Loop {
					animComp.FrameIndex = 0
				} else {
					animComp.FrameIndex = len(animComp.Frames) - 1
					animComp.Playing = false
				}
			}
		}

		renderComp.TexturePath = animComp.Frames[animComp.FrameIndex]
		_ = core.SetComponent(world, entityID, renderComp)
	}
	return nil
}

func (am *Manager) Play(world *core.World, entityID core.EntityID) error {
	animComp, err := core.GetComponent[components.Animation](world, entityID)
	if err != nil {
		return err
	}
	animComp.Playing = true
	animComp.FrameTime = 0
	animComp.FrameIndex = 0
	return core.SetComponent(world, entityID, animComp)
}

func (am *Manager) Stop(world *core.World, entityID core.EntityID) error {
	animComp, err := core.GetComponent[components.Animation](world, entityID)
	if err != nil {
		return err
	}
	animComp.Playing = false
	return core.SetComponent(world, entityID, animComp)
}

func (am *Manager) Reset(world *core.World, entityID core.EntityID) error {
	animComp, err := core.GetComponent[components.Animation](world, entityID)
	if err != nil {
		return err
	}
	animComp.Playing = false
	animComp.FrameTime = 0
	animComp.FrameIndex = 0
	return core.SetComponent(world, entityID, animComp)
}

func (am *Manager) IsPlaying(world *core.World, entityID core.EntityID) (bool, error) {
	animComp, err := core.GetComponent[components.Animation](world, entityID)
	if err != nil {
		return false, err
	}
	return animComp.Playing, nil
}

func (am *Manager) GetFrameIndex(world *core.World, entityID core.EntityID) (int, error) {
	animComp, err := core.GetComponent[components.Animation](world, entityID)
	if err != nil {
		return 0, err
	}
	return animComp.FrameIndex, nil
}

func (am *Manager) GetFrameTime(world *core.World, entityID core.EntityID) (float64, error) {
	animComp, err := core.GetComponent[components.Animation](world, entityID)
	if err != nil {
		return 0, err
	}
	return animComp.FrameTime, nil
}

func (am *Manager) GetFrameSpeed(world *core.World, entityID core.EntityID) (float64, error) {
	animComp, err := core.GetComponent[components.Animation](world, entityID)
	if err != nil {
		return 0, err
	}
	return animComp.FrameSpeed, nil
}

func (am *Manager) GetFrames(world *core.World, entityID core.EntityID) ([]string, error) {
	animComp, err := core.GetComponent[components.Animation](world, entityID)
	if err != nil {
		return nil, err
	}
	return animComp.Frames, nil
}
