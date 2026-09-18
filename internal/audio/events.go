package audio

import "github.com/leonard-atorough/castrum/ecs"

type EventType int

const (
	EventTypeTrackStarted EventType = iota
	EventTypeTrackStopped
	EventTypeTrackPaused
	EventTypeTrackResumed
	EventTypeTrackLooped
)

type Event struct {
	Type     EventType
	TrackID  ID
	EntityID ecs.EntityID
}
