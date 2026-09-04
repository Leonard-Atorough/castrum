package animation

import (
	"fmt"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/internal/core"
)

type AnimationEventType int

const (
	FrameEventType AnimationEventType = iota
	CompleteEventType
)

type AnimationEvent struct {
	EntityID   core.EntityID
	Type       AnimationEventType
	FrameIndex int
}

type Manager struct {
	events []AnimationEvent
}

func NewManager() *Manager {
	return &Manager{
		events: make([]AnimationEvent, 0, 64),
	}
}

func (am *Manager) Events() []AnimationEvent {
	return am.events
}

func (am *Manager) Update(world *core.World, delta float64) error {
	am.events = am.events[:0] // clear previous frame events

	// Use the new query builder to iterate over Animatable entities
	for entry := range world.NewQuery().WithRequiredComponents(components.Animatable{}).Execute() {
		animComp := entry.Get[components.Animatable]()
		entityID := entry.EntityID

		if _, exists := animComp.Animations[animComp.CurrentAnimation]; !exists {
			continue
		}

		if !animComp.Animations[animComp.CurrentAnimation].Playing {
			continue
		}

		current := animComp.Animations[animComp.CurrentAnimation]
		current.FrameTime += delta

		if current.FrameTime >= current.FrameSpeed {
			current.FrameTime = 0
			current.FrameIndex++

			// Emit frame event
			if current.FrameIndex < len(current.Frames) {
				am.events = append(am.events, AnimationEvent{
					EntityID:   entityID,
					Type:       FrameEventType,
					FrameIndex: current.FrameIndex,
				})

				// Trigger frame callback if defined
				if current.FrameEvents != nil {
					if callback, exists := current.FrameEvents[current.FrameIndex]; exists {
						callback()
					}
				}
			}

			if current.FrameIndex >= len(current.Frames) {
				if current.Loop {
					current.FrameIndex = 0
				} else {
					current.FrameIndex = len(current.Frames) - 1
					current.Playing = false

					// Emit completion event
					am.events = append(am.events, AnimationEvent{
						EntityID: entityID,
						Type:     CompleteEventType,
					})

					// Trigger completion callback if defined
					if current.Callback != nil {
						current.Callback()
					}
				}
			}
		}

		animComp.Animations[animComp.CurrentAnimation] = current
		_ = core.SetComponent(world, entityID, animComp)
	}
	return nil
}

func (am *Manager) Play(world *core.World, entityID core.EntityID) error {
	animComp, err := core.GetComponent[components.Animatable](world, entityID)
	if err != nil {
		return err
	}
	current := animComp.Animations[animComp.CurrentAnimation]
	current.Playing = true
	current.FrameTime = 0
	current.FrameIndex = 0
	animComp.Animations[animComp.CurrentAnimation] = current
	return core.SetComponent(world, entityID, animComp)
}

func (am *Manager) Pause(world *core.World, entityID core.EntityID) error {
	animComp, err := core.GetComponent[components.Animatable](world, entityID)
	if err != nil {
		return err
	}
	current := animComp.Animations[animComp.CurrentAnimation]
	current.Playing = false
	animComp.Animations[animComp.CurrentAnimation] = current
	return core.SetComponent(world, entityID, animComp)
}

func (am *Manager) Stop(world *core.World, entityID core.EntityID) error {
	animComp, err := core.GetComponent[components.Animatable](world, entityID)
	if err != nil {
		return err
	}
	current := animComp.Animations[animComp.CurrentAnimation]
	current.Playing = false
	current.FrameTime = 0
	current.FrameIndex = 0
	animComp.Animations[animComp.CurrentAnimation] = current
	return core.SetComponent(world, entityID, animComp)
}

func (am *Manager) Reset(world *core.World, entityID core.EntityID) error {
	animComp, err := core.GetComponent[components.Animatable](world, entityID)
	if err != nil {
		return err
	}
	current := animComp.Animations[animComp.CurrentAnimation]
	current.Playing = false
	current.FrameTime = 0
	current.FrameIndex = 0
	animComp.Animations[animComp.CurrentAnimation] = current
	return core.SetComponent(world, entityID, animComp)
}

func (am *Manager) SwitchAnimation(world *core.World, entityID core.EntityID, animationName string) error {
	animComp, err := core.GetComponent[components.Animatable](world, entityID)
	if err != nil {
		return err
	}
	if _, exists := animComp.Animations[animationName]; !exists {
		return fmt.Errorf("animation %s does not exist", animationName)
	}
	animComp.CurrentAnimation = animationName
	return core.SetComponent(world, entityID, animComp)
}

func (am *Manager) IsPlaying(world *core.World, entityID core.EntityID) (bool, error) {
	animComp, err := core.GetComponent[components.Animatable](world, entityID)
	if err != nil {
		return false, err
	}
	return animComp.Animations[animComp.CurrentAnimation].Playing, nil
}

func (am *Manager) GetFrameIndex(world *core.World, entityID core.EntityID) (int, error) {
	animComp, err := core.GetComponent[components.Animatable](world, entityID)
	if err != nil {
		return 0, err
	}
	return animComp.Animations[animComp.CurrentAnimation].FrameIndex, nil
}

func (am *Manager) GetFrameTime(world *core.World, entityID core.EntityID) (float64, error) {
	animComp, err := core.GetComponent[components.Animatable](world, entityID)
	if err != nil {
		return 0, err
	}
	return animComp.Animations[animComp.CurrentAnimation].FrameTime, nil
}

func (am *Manager) GetFrameSpeed(world *core.World, entityID core.EntityID) (float64, error) {
	animComp, err := core.GetComponent[components.Animatable](world, entityID)
	if err != nil {
		return 0, err
	}
	return animComp.Animations[animComp.CurrentAnimation].FrameSpeed, nil
}

func (am *Manager) GetFrames(world *core.World, entityID core.EntityID, animationNames ...string) (map[string][]string, error) {
	animComp, err := core.GetComponent[components.Animatable](world, entityID)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for _, name := range animationNames {
		result[name] = animComp.Animations[name].Frames
	}
	return result, nil
}
