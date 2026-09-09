package animation

import "github.com/leonard-atorough/castrum/internal/ecs"

type AnimationEventType int

const (
	EventClipFinished AnimationEventType = iota
	EventClipLooped
)

// AnimationEvent is emitted by the animation system when clips loop or finish.
type AnimationEvent struct {
	EntityID ecs.EntityID
	ClipID   string
	Type     AnimationEventType
}
