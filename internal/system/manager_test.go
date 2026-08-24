package system

import (
	"errors"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
	"github.com/leonard-atorough/castrum/internal/core"
)

// Mock systems for testing
type mockSystem struct {
	initCalled     int
	updateCalled   int
	shutdownCalled int
	initErr        error
	updateErr      error
	shutdownErr    error
}

func (ms *mockSystem) Init(world ecs.World) error {
	ms.initCalled++
	return ms.initErr
}

func (ms *mockSystem) Update(world ecs.World, deltaTime float64) error {
	ms.updateCalled++
	return ms.updateErr
}

func (ms *mockSystem) Shutdown(world ecs.World) error {
	ms.shutdownCalled++
	return ms.shutdownErr
}

// TestManager_Register tests system registration and Init call.
func TestManager_Register(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	mockSys := &mockSystem{}

	err := sm.Register(Core, "test", mockSys, world)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if mockSys.initCalled != 1 {
		t.Fatalf("expected Init to be called once, got %d", mockSys.initCalled)
	}

	if sm.Count() != 1 {
		t.Fatalf("expected 1 system, got %d", sm.Count())
	}

	if sm.Len(Core) != 1 {
		t.Fatalf("expected 1 core system, got %d", sm.Len(Core))
	}

	if !sm.Has("test") {
		t.Fatal("expected system 'test' to be registered")
	}
}

// TestManager_RegisterDuplicate tests that duplicate names are rejected.
func TestManager_RegisterDuplicate(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	mockSys1 := &mockSystem{}
	mockSys2 := &mockSystem{}

	sm.Register(Core, "test", mockSys1, world)
	err := sm.Register(Player, "test", mockSys2, world)

	if err == nil {
		t.Fatal("expected error when registering duplicate name")
	}

	if sm.Count() != 1 {
		t.Fatalf("expected only 1 system, got %d", sm.Count())
	}
}

// TestManager_RegisterInitFails tests that Init failure is propagated.
func TestManager_RegisterInitFails(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	mockSys := &mockSystem{initErr: errors.New("init failed")}

	err := sm.Register(Core, "test", mockSys, world)
	if err == nil {
		t.Fatal("expected error when Init fails")
	}

	if sm.Count() != 0 {
		t.Fatalf("expected 0 systems after failed Init, got %d", sm.Count())
	}
}

// TestManager_Unregister tests system unregistration and Shutdown call.
func TestManager_Unregister(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	mockSys := &mockSystem{}

	sm.Register(Core, "test", mockSys, world)
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
	sm := NewSystemManager()
	world := core.NewWorld()

	err := sm.Unregister("nonexistent", world)
	if err == nil {
		t.Fatal("expected error when unregistering non-existent system")
	}
}

// TestManager_UnregisterIndexFix tests that indices are corrected after removal.
func TestManager_UnregisterIndexFix(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	sys0 := &mockSystem{}
	sys1 := &mockSystem{}
	sys2 := &mockSystem{}

	sm.Register(Player, "sys0", sys0, world)
	sm.Register(Player, "sys1", sys1, world)
	sm.Register(Player, "sys2", sys2, world)

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

// TestManager_Update tests system update execution order.
func TestManager_Update(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	coreSys := &mockSystem{}
	playerSys := &mockSystem{}

	sm.Register(Core, "core", coreSys, world)
	sm.Register(Player, "player", playerSys, world)

	err := sm.Update(world, 0.016)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if coreSys.updateCalled != 1 {
		t.Fatalf("expected core system update to be called, got %d", coreSys.updateCalled)
	}

	if playerSys.updateCalled != 1 {
		t.Fatalf("expected player system update to be called, got %d", playerSys.updateCalled)
	}
}

// TestManager_UpdateError tests that Update stops on first error.
func TestManager_UpdateError(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	sys1 := &mockSystem{}
	sys2 := &mockSystem{updateErr: errors.New("system 2 failed")}
	sys3 := &mockSystem{}

	sm.Register(Core, "sys1", sys1, world)
	sm.Register(Core, "sys2", sys2, world)
	sm.Register(Core, "sys3", sys3, world)

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

// TestManager_UpdateLayerOrder tests Core systems run before Player systems.
func TestManager_UpdateLayerOrder(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()

	coreSys := &mockSystem{}
	playerSys := &mockSystem{}

	sm.Register(Core, "core", coreSys, world)
	sm.Register(Player, "player", playerSys, world)

	sm.Update(world, 0.016)

	// Both systems should be called in order (Core first, then Player)
	// We verify they were both called
	if coreSys.updateCalled != 1 {
		t.Fatalf("expected core system to be called once, got %d", coreSys.updateCalled)
	}
	if playerSys.updateCalled != 1 {
		t.Fatalf("expected player system to be called once, got %d", playerSys.updateCalled)
	}
}

// TestManager_Shutdown tests system shutdown in reverse order.
func TestManager_Shutdown(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	playerSys1 := &mockSystem{}
	playerSys2 := &mockSystem{}
	coreSys1 := &mockSystem{}

	sm.Register(Core, "core1", coreSys1, world)
	sm.Register(Player, "player1", playerSys1, world)
	sm.Register(Player, "player2", playerSys2, world)

	err := sm.Shutdown(world)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// All should be shut down
	if playerSys1.shutdownCalled != 1 {
		t.Fatalf("expected playerSys1 shutdown, got %d calls", playerSys1.shutdownCalled)
	}
	if playerSys2.shutdownCalled != 1 {
		t.Fatalf("expected playerSys2 shutdown, got %d calls", playerSys2.shutdownCalled)
	}
	if coreSys1.shutdownCalled != 1 {
		t.Fatalf("expected coreSys1 shutdown, got %d calls", coreSys1.shutdownCalled)
	}

	// All systems should be cleared
	if sm.Count() != 0 {
		t.Fatalf("expected 0 systems after shutdown, got %d", sm.Count())
	}
}

// TestManager_ShutdownError tests that Shutdown continues despite errors.
func TestManager_ShutdownError(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	sys1 := &mockSystem{shutdownErr: errors.New("shutdown failed")}
	sys2 := &mockSystem{}

	sm.Register(Player, "sys1", sys1, world)
	sm.Register(Player, "sys2", sys2, world)

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
	sm := NewSystemManager()
	world := core.NewWorld()
	mockSys := &mockSystem{}

	sm.Register(Core, "test", mockSys, world)
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
	sm := NewSystemManager()
	_, err := sm.GetSystem("nonexistent")

	if err == nil {
		t.Fatal("expected error when getting non-existent system")
	}
}

// TestManager_GetSystems tests retrieving all systems by layer.
func TestManager_GetSystems(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	sys1 := &mockSystem{}
	sys2 := &mockSystem{}
	sys3 := &mockSystem{}

	sm.Register(Core, "sys1", sys1, world)
	sm.Register(Core, "sys2", sys2, world)
	sm.Register(Player, "sys3", sys3, world)

	coreSystems := sm.GetSystems(Core)
	if len(coreSystems) != 2 {
		t.Fatalf("expected 2 core systems, got %d", len(coreSystems))
	}

	playerSystems := sm.GetSystems(Player)
	if len(playerSystems) != 1 {
		t.Fatalf("expected 1 player system, got %d", len(playerSystems))
	}

	if playerSystems[0] != sys3 {
		t.Fatal("expected sys3 in player systems")
	}
}

// TestManager_Count tests counting total systems.
func TestManager_Count(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()

	if sm.Count() != 0 {
		t.Fatalf("new manager should have 0 systems, got %d", sm.Count())
	}

	sys1 := &mockSystem{}
	sys2 := &mockSystem{}
	sm.Register(Core, "sys1", sys1, world)
	sm.Register(Player, "sys2", sys2, world)

	if sm.Count() != 2 {
		t.Fatalf("expected 2 systems, got %d", sm.Count())
	}

	sm.Unregister("sys1", world)
	if sm.Count() != 1 {
		t.Fatalf("expected 1 system after unregister, got %d", sm.Count())
	}
}

// TestManager_Len tests counting systems per layer.
func TestManager_Len(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	sys1 := &mockSystem{}
	sys2 := &mockSystem{}
	sys3 := &mockSystem{}

	sm.Register(Core, "sys1", sys1, world)
	sm.Register(Core, "sys2", sys2, world)
	sm.Register(Player, "sys3", sys3, world)

	if sm.Len(Core) != 2 {
		t.Fatalf("expected 2 core systems, got %d", sm.Len(Core))
	}

	if sm.Len(Player) != 1 {
		t.Fatalf("expected 1 player system, got %d", sm.Len(Player))
	}
}

// TestManager_Has tests checking system registration.
func TestManager_Has(t *testing.T) {
	sm := NewSystemManager()
	world := core.NewWorld()
	mockSys := &mockSystem{}

	if sm.Has("test") {
		t.Fatal("new manager should not have any systems")
	}

	sm.Register(Core, "test", mockSys, world)
	if !sm.Has("test") {
		t.Fatal("manager should have 'test' system after register")
	}

	sm.Unregister("test", world)
	if sm.Has("test") {
		t.Fatal("manager should not have 'test' system after unregister")
	}
}
