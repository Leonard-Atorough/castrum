// Package animation provides sprite animation types for the castrum engine.
//
// Clips are defined via [castrum.Game.NewClip], which returns a
// [ClipBuilder]. Call builder methods to add frames, set the FPS and loop
// mode, then call Build to register the clip. Entities reference clips by
// ID via the [components.Animation] component.
//
// Animation events (loop, finish, frame-reached) are emitted on the
// [events.EventBus] as [Event] values. Subscribe with
// [events.EventBus.On].
package animation

import (
	internalanimation "github.com/leonard-atorough/castrum/internal/animation"
)

// Clip is an immutable animation definition: a sequence of atlas regions
// played back at a fixed frame rate. Clips are created via
// [castrum.Game.NewClip].
type Clip = internalanimation.AnimationClip

// ClipBuilder constructs a [Clip] using a fluent API. Obtained from
// [castrum.Game.NewClip].
type ClipBuilder = internalanimation.ClipBuilder

// Loop controls how a clip repeats after reaching the last frame.
type Loop = internalanimation.Loop

const (
	// LoopNone plays the clip once and holds on the last frame.
	LoopNone = internalanimation.LoopNone
	// LoopForever repeats the clip from the first frame after the last.
	LoopForever = internalanimation.LoopForever
)

// Event is emitted by the animation system when clips loop, finish, or
// reach a tagged frame.
type Event = internalanimation.AnimationEvent

// EventType identifies the kind of animation event.
type EventType = internalanimation.AnimationEventType

const (
	EventFinished = internalanimation.AnimationEventFinished
	EventLooped   = internalanimation.AnimationEventLooped
	EventFrame    = internalanimation.AnimationEventFrame
)
