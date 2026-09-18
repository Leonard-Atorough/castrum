package audio

import (
	"fmt"

	"github.com/leonard-atorough/castrum/components"
	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/events"
)

// System is the ECS system that drives audio playback. Each tick it queries
// entities with an [components.AudioPlayer] component and synchronizes the
// underlying player via [AudioService]. It detects new play requests, state
// changes (pause/resume), and natural completion of one-shot tracks.
type System struct {
	query  *ecs.Query
	svc    *AudioService
	events *events.EventBus
}

// NewSystem creates an uninitialized audio system. Resources are resolved
// during [System.Init].
func NewSystem(service *AudioService) *System {
	return &System{
		svc: service,
	}
}

// Init resolves the [AudioService] and [events.EventBus] world resources and
// builds the query for entities with an [components.AudioPlayer] component.
// It returns an error if either resource is missing.
func (s *System) Init(world *ecs.World) error {
	s.query = world.NewQuery().WithRequiredComponents(components.AudioPlayer{})
	bus, ok := world.GetResource[*events.EventBus]()
	if !ok {
		return fmt.Errorf("EventBus resource not found")
	}
	s.events = bus
	if s.svc == nil {
		return fmt.Errorf("AudioService not set")
	}
	return nil
}

// Update synchronizes per-entity audio players with their [components.AudioPlayer]
// component state. It handles three cases per entity:
//   - New play: the component requests playback but no player exists yet.
//   - State change: the Playing flag was flipped by external code (pause/resume).
//   - Completion: a non-looping track finished naturally.
//
// Entities with [components.PlaybackDespawn] mode are destroyed after the
// query loop completes; entities cannot be destroyed during iteration.
func (s *System) Update(world *ecs.World, delta float64) error {
	if s.query == nil || s.svc == nil || s.events == nil {
		return fmt.Errorf("audio system not properly initialized")
	}

	var despawnQueue []ecs.EntityID

	for entry := range s.query.Execute() {
		ap, _ := entry.Get[components.AudioPlayer]()

		// Skip idle entities: not playing and no player to clean up.
		if !ap.Playing && !s.svc.hasPlayer(entry.EntityID) {
			continue
		}

		// New play: component requests playback but no player exists yet.
		if ap.Playing && !s.svc.hasPlayer(entry.EntityID) {
			if err := s.svc.syncPlayer(entry.EntityID, ID(ap.TrackID), true, ap.Volume); err != nil {
				continue
			}
			s.events.Emit(Event{
				Type:     EventTypeTrackStarted,
				TrackID:  ID(ap.TrackID),
				EntityID: entry.EntityID,
			}, "audio.System")
			_ = world.SetComponent(entry.EntityID, ap)
			continue
		}

		// Completion: a non-looping track that was playing has finished.
		// Check this before the state-change branch so we don't restart
		// a finished one-shot.
		if ap.Playing && s.svc.hasPlayer(entry.EntityID) && !s.svc.isPlaying(entry.EntityID) && !s.svc.isLooping(entry.EntityID) {
			s.svc.removePlayer(entry.EntityID)
			ap.Playing = false
			s.events.Emit(Event{
				Type:     EventTypeTrackStopped,
				TrackID:  ID(ap.TrackID),
				EntityID: entry.EntityID,
			}, "audio.System")
			if ap.Mode == components.PlaybackDespawn {
				despawnQueue = append(despawnQueue, entry.EntityID)
			}
			_ = world.SetComponent(entry.EntityID, ap)
			continue
		}

		// State change: the Playing flag was flipped by external code.
		if ap.Playing != s.svc.isPlaying(entry.EntityID) {
			if err := s.svc.syncPlayer(entry.EntityID, ID(ap.TrackID), ap.Playing, ap.Volume); err != nil {
				continue
			}
		}

		_ = world.SetComponent(entry.EntityID, ap)
	}

	for _, id := range despawnQueue {
		_ = world.DestroyEntity(id, false)
	}

	return nil
}

// Shutdown stops and removes all audio players managed by this system.
func (s *System) Shutdown(world *ecs.World) error {
	if s.query == nil || s.svc == nil {
		return nil
	}
	for entry := range s.query.Execute() {
		if s.svc.hasPlayer(entry.EntityID) {
			s.svc.removePlayer(entry.EntityID)
		}
	}
	return nil
}
