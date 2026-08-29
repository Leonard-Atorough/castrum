package blueprint

import "github.com/leonard-atorough/castrum/internal/core"

type Manager struct {
	blueprints map[string]*Blueprint
	registry   *Registry
	loader     *Loader
}

func NewManager() *Manager {
	reg := NewRegistry()
	loader := NewLoader(reg)
	return &Manager{
		blueprints: make(map[string]*Blueprint),
		registry:   reg,
		loader:     loader,
	}
}

func (m *Manager) RegisterComponent(typeName string, constructor func(properties map[string]any) (core.Component, error)) {
	m.registry.Register(typeName, constructor)
}

func (m *Manager) LoadBlueprintFromPath(path string) (*Blueprint, error) {
	bp, err := m.loader.LoadFromPath(path)
	return handleBlueprintLoad(err, m, bp)
}

func (m *Manager) LoadBlueprintFromString(data string) (*Blueprint, error) {
	bp, err := m.loader.LoadFromString(data)
	return handleBlueprintLoad(err, m, bp)
}

func (m *Manager) LoadBlueprintFromBytes(data []byte) (*Blueprint, error) {
	bp, err := m.loader.LoadFromBytes(data)
	return handleBlueprintLoad(err, m, bp)
}

func (m *Manager) GetBlueprint(name string) (*Blueprint, bool) {
	bp, exists := m.blueprints[name]
	return bp, exists
}

func (m *Manager) SpawnBlueprint(world *core.World, name string) (*core.EntityID, error) {
	bp, exists := m.blueprints[name]
	if !exists {
		return nil, ErrBlueprintNotFound
	}
	return m.loader.Spawn(world, bp)
}

func (m *Manager) ListBlueprints() []string {
	names := make([]string, 0, len(m.blueprints))
	for name := range m.blueprints {
		names = append(names, name)
	}
	return names
}

func (m *Manager) ListRegisteredComponents() []string {
	return m.registry.List()
}

func handleBlueprintLoad(err error, m *Manager, bp *Blueprint) (*Blueprint, error) {
	if err != nil {
		return nil, err
	}
	m.blueprints[bp.Name] = bp
	return bp, nil
}
