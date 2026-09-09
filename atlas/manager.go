package atlas

import (
	"sync"
)

// Manager orchestrates atlas creation and storage.
// It is lightweight: it stores atlases by ID and provides lookup.
type Manager struct {
	mu      sync.RWMutex
	atlases map[string]*TextureAtlas
}

// NewManager creates a new Manager.
func NewManager() *Manager {
	return &Manager{
		atlases: make(map[string]*TextureAtlas),
	}
}

// Add registers an atlas by ID.
func (m *Manager) Add(id string, atlas *TextureAtlas) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.atlases[id] = atlas
}

// Get retrieves an atlas by ID, or nil if not found.
func (m *Manager) Get(id string) *TextureAtlas {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.atlases[id]
}

// Has checks if an atlas with the given ID exists.
func (m *Manager) Has(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.atlases[id]
	return ok
}

// Remove unregisters an atlas by ID.
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.atlases, id)
}

// GetOrCreate returns an existing atlas or creates one via the provided builder function.
func (m *Manager) GetOrCreate(id string, fn func() (*TextureAtlas, error)) (*TextureAtlas, error) {
	m.mu.RLock()
	if atlas, ok := m.atlases[id]; ok {
		m.mu.RUnlock()
		return atlas, nil
	}
	m.mu.RUnlock()

	// Create outside the lock
	atlas, err := fn()
	if err != nil {
		return nil, err
	}

	m.Add(id, atlas)
	return atlas, nil
}
