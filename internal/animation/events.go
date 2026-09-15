package animation

import "github.com/leonard-atorough/castrum/ecs"

// AnimationEventType identifies the kind of animation event emitted
// by the animation [System].
type AnimationEventType int

const (
	// AnimationEventFinished is emitted when a non-looping clip reaches
	// the last frame. The entity's Playing flag is set to false.
	AnimationEventFinished AnimationEventType = iota
	// AnimationEventLooped is emitted when a looping clip wraps from the
	// last frame back to the first.
	AnimationEventLooped
	// AnimationEventPingPong is reserved for future bidirectional playback.
	AnimationEventPingPong
	// AnimationEventFrame is emitted when playback crosses a frame
	// tagged via [ClipBuilder.OnFrame]. The EventName field carries
	// the name given to OnFrame.
	AnimationEventFrame
	// AnimationEventStart is reserved for future use.
	AnimationEventStart
)

// AnimationEvent is emitted by the animation [System] on the
// [events.EventBus]. Subscribe with [events.EventBus.On].
type AnimationEvent struct {
	Type       AnimationEventType
	ClipID     string
	EntityID   ecs.EntityID
	FrameIndex int    // valid for [AnimationEventFrame]
	EventName  string // valid for [AnimationEventFrame]
}
