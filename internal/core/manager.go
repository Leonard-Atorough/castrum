package core

import (
	"errors"
	"slices"
)

type systemEntry struct {
	name     string
	priority int
	system   System
}

// Manager schedules and runs Systems in ascending priority order — lower
// priority values run first (and shut down last). Systems registered with
// the same priority run in registration order.
type Manager struct {
	systems     []systemEntry
	nameToIndex map[string]int
}

func NewManager() *Manager {
	return &Manager{
		nameToIndex: make(map[string]int),
	}
}

// Register adds sys under name, scheduled at the given priority (lower runs
// earlier). Init is called immediately; if it returns an error the system is
// not added.
func (sm *Manager) Register(name string, priority int, sys System, world *World) error {
	if _, exists := sm.nameToIndex[name]; exists {
		return &SystemError{Name: name, Op: "Register", Err: ErrSystemAlreadyRegistered}
	}

	if err := sys.Init(world); err != nil {
		return &SystemError{Name: name, Op: "Register", Err: err}
	}

	idx := len(sm.systems)
	for i, e := range sm.systems {
		if e.priority > priority {
			idx = i
			break
		}
	}
	sm.systems = slices.Insert(sm.systems, idx, systemEntry{name: name, priority: priority, system: sys})
	sm.reindex()

	return nil
}

// Unregister removes and shuts down the named system.
func (sm *Manager) Unregister(name string, world *World) error {
	idx, exists := sm.nameToIndex[name]
	if !exists {
		return &SystemError{Name: name, Op: "Unregister", Err: ErrSystemNotFound}
	}

	sys := sm.systems[idx].system
	sm.systems = slices.Delete(sm.systems, idx, idx+1)
	sm.reindex()

	if err := sys.Shutdown(world); err != nil {
		return &SystemError{Name: name, Op: "Unregister", Err: err}
	}
	return nil
}

// Update runs all systems in priority order. It stops and returns an error
// at the first system that fails.
func (sm *Manager) Update(world *World, deltaTime float64) error {
	for _, entry := range sm.systems {
		if err := entry.system.Update(world, deltaTime); err != nil {
			return &SystemError{Name: entry.name, Op: "Update", Err: err}
		}
	}
	return nil
}

// Shutdown shuts down all systems in reverse priority order, continuing on
// error and joining any failures into the returned error.
func (sm *Manager) Shutdown(world *World) error {
	var errs []error
	for i := len(sm.systems) - 1; i >= 0; i-- {
		entry := sm.systems[i]
		if err := entry.system.Shutdown(world); err != nil {
			errs = append(errs, &SystemError{Name: entry.name, Op: "Shutdown", Err: err})
		}
	}

	sm.systems = nil
	sm.nameToIndex = make(map[string]int)

	return errors.Join(errs...)
}

// GetSystem returns the named system.
func (sm *Manager) GetSystem(name string) (System, error) {
	idx, exists := sm.nameToIndex[name]
	if !exists {
		return nil, &SystemError{Name: name, Op: "GetSystem", Err: ErrSystemNotFound}
	}
	return sm.systems[idx].system, nil
}

// Systems returns all registered systems in scheduled (priority) order.
func (sm *Manager) Systems() []System {
	systems := make([]System, len(sm.systems))
	for i, e := range sm.systems {
		systems[i] = e.system
	}
	return systems
}

// Count returns the total number of registered systems.
func (sm *Manager) Count() int {
	return len(sm.systems)
}

// Has checks if a system with the given name is registered.
func (sm *Manager) Has(name string) bool {
	_, exists := sm.nameToIndex[name]
	return exists
}

func (sm *Manager) reindex() {
	clear(sm.nameToIndex)
	for i, e := range sm.systems {
		sm.nameToIndex[e.name] = i
	}
}
