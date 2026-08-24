package system

import (
	"fmt"

	"github.com/leonard-atorough/castrum/internal/ecs"
)

type systemEntry struct {
	name   string
	system System
}
type Manager struct {
	coreSystems   []systemEntry
	playerSystems []systemEntry
	nameToIndex   map[string]struct {
		layer Layer
		index int
	}
}

func NewSystemManager() *Manager {
	return &Manager{
		coreSystems:   []systemEntry{},
		playerSystems: []systemEntry{},
		nameToIndex: make(map[string]struct {
			layer Layer
			index int
		}),
	}
}

func (sm *Manager) Register(layer Layer, name string, sys System, world *ecs.World) error {
	if _, exists := sm.nameToIndex[name]; exists {
		return fmt.Errorf("system with name %s already registered", name)
	}

	// Call Init before adding to collection to avoid orphaned entries on failure
	if err := sys.Init(world); err != nil {
		return fmt.Errorf("failed to initialize system %s: %v", name, err)
	}

	entry := systemEntry{
		name:   name,
		system: sys,
	}

	switch layer {
	case Core:
		sm.coreSystems = append(sm.coreSystems, entry)
		sm.nameToIndex[name] = struct {
			layer Layer
			index int
		}{
			layer: Core,
			index: len(sm.coreSystems) - 1,
		}
	case Player:
		sm.playerSystems = append(sm.playerSystems, entry)
		sm.nameToIndex[name] = struct {
			layer Layer
			index int
		}{
			layer: Player,
			index: len(sm.playerSystems) - 1,
		}
	default:
		return fmt.Errorf("unknown layer %v", layer)
	}

	return nil
}

func (sm *Manager) Unregister(name string, world *ecs.World) error {
	info, exists := sm.nameToIndex[name]
	if !exists {
		return fmt.Errorf("system with name %s not found", name)
	}

	var sys System
	var systems *[]systemEntry

	switch info.layer {
	case Core:
		sys = sm.coreSystems[info.index].system
		sm.coreSystems = append(sm.coreSystems[:info.index], sm.coreSystems[info.index+1:]...)
		systems = &sm.coreSystems
	case Player:
		sys = sm.playerSystems[info.index].system
		sm.playerSystems = append(sm.playerSystems[:info.index], sm.playerSystems[info.index+1:]...)
		systems = &sm.playerSystems
	default:
		return fmt.Errorf("unknown layer %v", info.layer)
	}

	// Rebuild indices for all systems in this layer (since indices shift after removal)
	for i, entry := range *systems {
		sm.nameToIndex[entry.name] = struct {
			layer Layer
			index int
		}{
			layer: info.layer,
			index: i,
		}
	}

	// Shutdown the system
	if err := sys.Shutdown(world); err != nil {
		return fmt.Errorf("failed to shutdown system %s: %v", name, err)
	}

	delete(sm.nameToIndex, name)
	return nil
}

// Update runs all systems in layer priority order: Core systems first, then Player systems.
// Returns error if any system fails (stops execution at first error).
func (sm *Manager) Update(world *ecs.World, deltaTime float64) error {
	for _, entry := range sm.coreSystems {
		if err := entry.system.Update(world, deltaTime); err != nil {
			return fmt.Errorf("core system %s failed: %w", entry.name, err)
		}
	}
	for _, entry := range sm.playerSystems {
		if err := entry.system.Update(world, deltaTime); err != nil {
			return fmt.Errorf("player system %s failed: %w", entry.name, err)
		}
	}
	return nil
}

func (sm *Manager) GetSystem(name string) (System, error) {
	info, exists := sm.nameToIndex[name]
	if !exists {
		return nil, fmt.Errorf("system with name %s not found", name)
	}

	switch info.layer {
	case Core:
		return sm.coreSystems[info.index].system, nil
	case Player:
		return sm.playerSystems[info.index].system, nil
	default:
		return nil, fmt.Errorf("unknown layer %v", info.layer)
	}
}

func (sm *Manager) GetSystems(layer Layer) []System {
	switch layer {
	case Core:
		systems := make([]System, len(sm.coreSystems))
		for i, entry := range sm.coreSystems {
			systems[i] = entry.system
		}
		return systems
	case Player:
		systems := make([]System, len(sm.playerSystems))
		for i, entry := range sm.playerSystems {
			systems[i] = entry.system
		}
		return systems
	default:
		return nil
	}
}

// Shutdown shuts down all systems in reverse order (Player first, then Core).
// Returns error if any system fails (continues shutdown of remaining systems).
func (sm *Manager) Shutdown(world *ecs.World) error {
	var lastErr error

	// Shutdown player systems in reverse order
	for i := len(sm.playerSystems) - 1; i >= 0; i-- {
		entry := sm.playerSystems[i]
		if err := entry.system.Shutdown(world); err != nil {
			lastErr = fmt.Errorf("failed to shutdown player system %s: %v", entry.name, err)
		}
	}

	// Shutdown core systems in reverse order
	for i := len(sm.coreSystems) - 1; i >= 0; i-- {
		entry := sm.coreSystems[i]
		if err := entry.system.Shutdown(world); err != nil {
			lastErr = fmt.Errorf("failed to shutdown core system %s: %v", entry.name, err)
		}
	}

	// Clear all systems and nameToIndex map
	sm.coreSystems = nil
	sm.playerSystems = nil
	sm.nameToIndex = make(map[string]struct {
		layer Layer
		index int
	})

	return lastErr
}

// Count returns the total number of registered systems.
func (sm *Manager) Count() int {
	return len(sm.coreSystems) + len(sm.playerSystems)
}

// Len returns the number of systems in the specified layer.
func (sm *Manager) Len(layer Layer) int {
	switch layer {
	case Core:
		return len(sm.coreSystems)
	case Player:
		return len(sm.playerSystems)
	default:
		return 0
	}
}

// Has checks if a system with the given name is registered.
func (sm *Manager) Has(name string) bool {
	_, exists := sm.nameToIndex[name]
	return exists
}
