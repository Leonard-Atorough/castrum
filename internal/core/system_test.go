package core

import (
	"errors"
	"testing"
)

// Mock systems for testing
type mockSystem struct {
	initCalled     int
	updateCalled   int
	shutdownCalled int
	initErr        error
	updateErr      error
	shutdownErr    error

	name  string
	order *[]string // if set, records name on Update/Shutdown to verify ordering
}

func (ms *mockSystem) Init(world *World) error {
	ms.initCalled++
	return ms.initErr
}

func (ms *mockSystem) Update(world *World, deltaTime float64) error {
	ms.updateCalled++
	if ms.order != nil {
		*ms.order = append(*ms.order, ms.name)
	}
	return ms.updateErr
}

func (ms *mockSystem) Shutdown(world *World) error {
	ms.shutdownCalled++
	if ms.order != nil {
		*ms.order = append(*ms.order, ms.name)
	}
	return ms.shutdownErr
}

// TestManager_Register tests system registration and Init call.
func TestManager_Register(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	mockSys := &mockSystem{}

	err := sm.Register("test", 0, mockSys, world)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if mockSys.initCalled != 1 {
		t.Fatalf("expected Init to be called once, got %d", mockSys.initCalled)
	}

	if sm.Count() != 1 {
		t.Fatalf("expected 1 system, got %d", sm.Count())
	}

	if !sm.Has("test") {
		t.Fatal("expected system 'test' to be registered")
	}
}

// TestManager_RegisterDuplicate tests that duplicate names are rejected.
func TestManager_RegisterDuplicate(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	mockSys1 := &mockSystem{}
	mockSys2 := &mockSystem{}

	sm.Register("test", 0, mockSys1, world)
	err := sm.Register("test", 1, mockSys2, world)

	if !errors.Is(err, ErrSystemAlreadyRegistered) {
		t.Fatalf("expected ErrSystemAlreadyRegistered, got %v", err)
	}

	if sm.Count() != 1 {
		t.Fatalf("expected only 1 system, got %d", sm.Count())
	}
}

// TestManager_RegisterInitFails tests that Init failure is propagated.
func TestManager_RegisterInitFails(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	mockSys := &mockSystem{initErr: errors.New("init failed")}

	err := sm.Register("test", 0, mockSys, world)
	if err == nil {
		t.Fatal("expected error when Init fails")
	}

	if sm.Count() != 0 {
		t.Fatalf("expected 0 systems after failed Init, got %d", sm.Count())
	}
}

// TestManager_Unregister tests system unregistration and Shutdown call.
func TestManager_Unregister(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	mockSys := &mockSystem{}

	sm.Register("test", 0, mockSys, world)
	err := sm.Unregister("test", world)

	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	if mockSys.shutdownCalled != 1 {
		t.Fatalf("expected Shutdown to be called once, got %d", mockSys.shutdownCalled)
	}

	if sm.Count() != 0 {
		t.Fatalf("expected 0 systems after unregister, got %d", sm.Count())
	}

	if sm.Has("test") {
		t.Fatal("expected system 'test' to be unregistered")
	}
}

// TestManager_UnregisterNotFound tests unregistering non-existent system.
func TestManager_UnregisterNotFound(t *testing.T) {
	sm := NewManager()
	world := NewWorld()

	err := sm.Unregister("nonexistent", world)
	if !errors.Is(err, ErrSystemNotFound) {
		t.Fatalf("expected ErrSystemNotFound, got %v", err)
	}
}

// TestManager_UnregisterIndexFix tests that lookups stay correct after a middle system is removed.
func TestManager_UnregisterIndexFix(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	sys0 := &mockSystem{}
	sys1 := &mockSystem{}
	sys2 := &mockSystem{}

	sm.Register("sys0", 0, sys0, world)
	sm.Register("sys1", 1, sys1, world)
	sm.Register("sys2", 2, sys2, world)

	// Remove middle system
	sm.Unregister("sys1", world)

	// Verify remaining systems can still be found and unregistered
	retrieved, err := sm.GetSystem("sys0")
	if err != nil || retrieved != sys0 {
		t.Fatal("failed to retrieve sys0 after removal of sys1")
	}

	retrieved, err = sm.GetSystem("sys2")
	if err != nil || retrieved != sys2 {
		t.Fatal("failed to retrieve sys2 after removal of sys1")
	}

	// Unregister should still work
	err = sm.Unregister("sys0", world)
	if err != nil {
		t.Fatalf("Unregister sys0 failed: %v", err)
	}

	if sys0.shutdownCalled != 1 {
		t.Fatalf("expected sys0 Shutdown to be called, got %d", sys0.shutdownCalled)
	}
}

// TestManager_Update tests that Update runs every registered system.
func TestManager_Update(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	sys1 := &mockSystem{}
	sys2 := &mockSystem{}

	sm.Register("sys1", 0, sys1, world)
	sm.Register("sys2", 10, sys2, world)

	err := sm.Update(world, 0.016)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if sys1.updateCalled != 1 {
		t.Fatalf("expected sys1 update to be called, got %d", sys1.updateCalled)
	}

	if sys2.updateCalled != 1 {
		t.Fatalf("expected sys2 update to be called, got %d", sys2.updateCalled)
	}
}

// TestManager_UpdateError tests that Update stops on first error.
func TestManager_UpdateError(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	sys1 := &mockSystem{}
	sys2 := &mockSystem{updateErr: errors.New("system 2 failed")}
	sys3 := &mockSystem{}

	sm.Register("sys1", 0, sys1, world)
	sm.Register("sys2", 1, sys2, world)
	sm.Register("sys3", 2, sys3, world)

	err := sm.Update(world, 0.016)
	if err == nil {
		t.Fatal("expected error from failed system")
	}

	if sys1.updateCalled != 1 {
		t.Fatal("sys1 should have been updated")
	}

	if sys2.updateCalled != 1 {
		t.Fatal("sys2 should have been updated (where error occurred)")
	}

	if sys3.updateCalled != 0 {
		t.Fatal("sys3 should not have been updated (stopped at sys2 error)")
	}
}

// TestManager_UpdatePriorityOrder tests that lower-priority systems run first,
// and same-priority systems run in registration order.
func TestManager_UpdatePriorityOrder(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	var order []string

	low := &mockSystem{name: "low", order: &order}
	highA := &mockSystem{name: "highA", order: &order}
	highB := &mockSystem{name: "highB", order: &order}

	// Register out of priority order to prove sorting, not insertion order, wins.
	sm.Register("highA", 10, highA, world)
	sm.Register("low", 0, low, world)
	sm.Register("highB", 10, highB, world)

	if err := sm.Update(world, 0.016); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	want := []string{"low", "highA", "highB"}
	if len(order) != len(want) {
		t.Fatalf("expected order %v, got %v", want, order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("expected order %v, got %v", want, order)
		}
	}
}

// TestManager_Shutdown tests system shutdown in reverse priority order.
func TestManager_Shutdown(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	var order []string

	first := &mockSystem{name: "first", order: &order}
	second := &mockSystem{name: "second", order: &order}
	third := &mockSystem{name: "third", order: &order}

	sm.Register("first", 0, first, world)
	sm.Register("second", 1, second, world)
	sm.Register("third", 2, third, world)

	err := sm.Shutdown(world)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	want := []string{"third", "second", "first"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("expected shutdown order %v, got %v", want, order)
		}
	}

	if sm.Count() != 0 {
		t.Fatalf("expected 0 systems after shutdown, got %d", sm.Count())
	}
}

// TestManager_ShutdownError tests that Shutdown continues despite errors and joins them.
func TestManager_ShutdownError(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	sys1 := &mockSystem{shutdownErr: errors.New("shutdown failed")}
	sys2 := &mockSystem{}

	sm.Register("sys1", 0, sys1, world)
	sm.Register("sys2", 1, sys2, world)

	err := sm.Shutdown(world)
	// Shutdown should return error but complete all shutdowns
	if err == nil {
		t.Fatal("expected error from failed shutdown")
	}

	if sys1.shutdownCalled != 1 || sys2.shutdownCalled != 1 {
		t.Fatal("both systems should have been shut down despite error")
	}
}

// TestManager_GetSystem tests retrieving a system by name.
func TestManager_GetSystem(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	mockSys := &mockSystem{}

	sm.Register("test", 0, mockSys, world)
	retrieved, err := sm.GetSystem("test")

	if err != nil {
		t.Fatalf("GetSystem failed: %v", err)
	}

	if retrieved != mockSys {
		t.Fatal("retrieved system does not match registered system")
	}
}

// TestManager_GetSystemNotFound tests GetSystem with non-existent name.
func TestManager_GetSystemNotFound(t *testing.T) {
	sm := NewManager()
	_, err := sm.GetSystem("nonexistent")

	if !errors.Is(err, ErrSystemNotFound) {
		t.Fatalf("expected ErrSystemNotFound, got %v", err)
	}
}

// TestManager_Systems tests retrieving all systems in priority order.
func TestManager_Systems(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	sys1 := &mockSystem{}
	sys2 := &mockSystem{}
	sys3 := &mockSystem{}

	sm.Register("sys3", 2, sys3, world)
	sm.Register("sys1", 0, sys1, world)
	sm.Register("sys2", 1, sys2, world)

	systems := sm.Systems()
	if len(systems) != 3 {
		t.Fatalf("expected 3 systems, got %d", len(systems))
	}
	if systems[0] != sys1 || systems[1] != sys2 || systems[2] != sys3 {
		t.Fatal("Systems should be returned in priority order")
	}
}

// TestManager_Count tests counting total systems.
func TestManager_Count(t *testing.T) {
	sm := NewManager()
	world := NewWorld()

	if sm.Count() != 0 {
		t.Fatalf("new manager should have 0 systems, got %d", sm.Count())
	}

	sys1 := &mockSystem{}
	sys2 := &mockSystem{}
	sm.Register("sys1", 0, sys1, world)
	sm.Register("sys2", 1, sys2, world)

	if sm.Count() != 2 {
		t.Fatalf("expected 2 systems, got %d", sm.Count())
	}

	sm.Unregister("sys1", world)
	if sm.Count() != 1 {
		t.Fatalf("expected 1 system after unregister, got %d", sm.Count())
	}
}

// TestManager_Has tests checking system registration.
func TestManager_Has(t *testing.T) {
	sm := NewManager()
	world := NewWorld()
	mockSys := &mockSystem{}

	if sm.Has("test") {
		t.Fatal("new manager should not have any systems")
	}

	sm.Register("test", 0, mockSys, world)
	if !sm.Has("test") {
		t.Fatal("manager should have 'test' system after register")
	}

	sm.Unregister("test", world)
	if sm.Has("test") {
		t.Fatal("manager should not have 'test' system after unregister")
	}
}
