package core

import (
	"errors"
	"strings"
	"testing"
)

type atlas struct{ name string }
type depA struct{}
type depB struct{}
type eagerFail struct{}
type eagerSecond struct{}

func TestLazyResourceResolvesOnce(t *testing.T) {
	w := NewWorld()
	ctors := 0
	if err := w.Provide(func(w *World) (*atlas, error) {
		ctors++
		return &atlas{name: "sprites"}, nil
	}); err != nil {
		t.Fatalf("Provide: %v", err)
	}
	if ctors != 0 {
		t.Fatalf("lazy resource constructed at registration: %d ctor calls", ctors)
	}

	a, err := w.Resource[*atlas]()
	if err != nil {
		t.Fatalf("first Resource: %v", err)
	}
	if a.name != "sprites" {
		t.Errorf("instance = %+v, want name %q", a, "sprites")
	}
	if ctors != 1 {
		t.Errorf("ctor calls after first fetch = %d, want 1", ctors)
	}

	a2, err := w.Resource[*atlas]()
	if err != nil {
		t.Fatalf("second Resource: %v", err)
	}
	if ctors != 1 {
		t.Errorf("ctor calls after second fetch = %d, want 1 (cached singleton)", ctors)
	}
	if a2 != a {
		t.Error("second fetch returned a different instance")
	}
}

func TestResourceNotRegistered(t *testing.T) {
	w := NewWorld()
	_, err := w.Resource[*atlas]()
	if err == nil {
		t.Fatal("fetching an unregistered resource should error")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("error = %q, want it to name the missing type", err)
	}
}

func TestProvideDuplicateRejected(t *testing.T) {
	w := NewWorld()
	if err := w.Provide(func(w *World) (*atlas, error) {
		return &atlas{name: "first"}, nil
	}); err != nil {
		t.Fatalf("first Provide: %v", err)
	}
	err := w.Provide(func(w *World) (*atlas, error) {
		return &atlas{name: "second"}, nil
	})
	if err == nil {
		t.Fatal("duplicate Provide should return an error")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Errorf("error = %q, want it to say already registered", err)
	}

	a, err := w.Resource[*atlas]()
	if err != nil {
		t.Fatalf("Resource after rejected duplicate: %v", err)
	}
	if a.name != "first" {
		t.Errorf("instance = %q, want the first registration kept", a.name)
	}
}

func TestCtorFailureNotCached(t *testing.T) {
	w := NewWorld()
	ctors := 0
	failing := true
	if err := w.Provide(func(w *World) (*atlas, error) {
		ctors++
		if failing {
			return nil, errors.New("boom")
		}
		return &atlas{name: "recovered"}, nil
	}); err != nil {
		t.Fatalf("Provide: %v", err)
	}

	if _, err := w.Resource[*atlas](); err == nil {
		t.Fatal("failing ctor should surface its error")
	}
	failing = false
	a, err := w.Resource[*atlas]()
	if err != nil {
		t.Fatalf("retry after transient failure: %v", err)
	}
	if a.name != "recovered" {
		t.Errorf("instance = %q, want recovered construction", a.name)
	}
	if ctors != 2 {
		t.Errorf("ctor calls = %d, want 2 (failure not cached)", ctors)
	}
}

func TestCircularDependencyDetected(t *testing.T) {
	w := NewWorld()
	if err := w.Provide(func(w *World) (*depA, error) {
		_, err := w.Resource[*depB]()
		if err != nil {
			return nil, err
		}
		return &depA{}, nil
	}); err != nil {
		t.Fatalf("Provide depA: %v", err)
	}
	if err := w.Provide(func(w *World) (*depB, error) {
		_, err := w.Resource[*depA]()
		if err != nil {
			return nil, err
		}
		return &depB{}, nil
	}); err != nil {
		t.Fatalf("Provide depB: %v", err)
	}

	_, err := w.Resource[*depA]()
	if err == nil {
		t.Fatal("circular dependency should error")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error = %q, want it to name the cycle", err)
	}
}

func TestLazyDependencyChain(t *testing.T) {
	w := NewWorld()
	bCtors := 0
	if err := w.Provide(func(w *World) (*depB, error) {
		bCtors++
		return &depB{}, nil
	}); err != nil {
		t.Fatalf("Provide depB: %v", err)
	}
	if err := w.Provide(func(w *World) (*depA, error) {
		if _, err := w.Resource[*depB](); err != nil {
			return nil, err
		}
		return &depA{}, nil
	}); err != nil {
		t.Fatalf("Provide depA: %v", err)
	}

	if bCtors != 0 {
		t.Fatalf("dependency constructed before dependent: %d ctor calls", bCtors)
	}
	if _, err := w.Resource[*depA](); err != nil {
		t.Fatalf("Resource depA: %v", err)
	}
	if bCtors != 1 {
		t.Errorf("dependency ctor calls = %d, want 1", bCtors)
	}
}

func TestEagerResolvesAtResolveEager(t *testing.T) {
	w := NewWorld()
	ctors := 0
	if err := w.ProvideEager(func(w *World) (*atlas, error) {
		ctors++
		return &atlas{name: "eager"}, nil
	}); err != nil {
		t.Fatalf("ProvideEager: %v", err)
	}
	if ctors != 0 {
		t.Fatalf("eager resource constructed at registration: %d ctor calls", ctors)
	}

	if err := w.ResolveEager(); err != nil {
		t.Fatalf("ResolveEager: %v", err)
	}
	if ctors != 1 {
		t.Errorf("ctor calls after ResolveEager = %d, want 1", ctors)
	}

	if _, err := w.Resource[*atlas](); err != nil {
		t.Fatalf("Resource after ResolveEager: %v", err)
	}
	if ctors != 1 {
		t.Errorf("ctor calls after fetch = %d, want 1 (already resolved)", ctors)
	}
}

func TestResolveEagerStopsAtFirstFailure(t *testing.T) {
	w := NewWorld()
	if err := w.ProvideEager(func(w *World) (*eagerFail, error) {
		return nil, errors.New("eager boom")
	}); err != nil {
		t.Fatalf("ProvideEager eagerFail: %v", err)
	}
	secondCtors := 0
	if err := w.ProvideEager(func(w *World) (*eagerSecond, error) {
		secondCtors++
		return &eagerSecond{}, nil
	}); err != nil {
		t.Fatalf("ProvideEager eagerSecond: %v", err)
	}

	if err := w.ResolveEager(); err == nil {
		t.Fatal("ResolveEager should fail on the first failing resource")
	}
	if secondCtors != 0 {
		t.Errorf("second ctor calls = %d, want 0 (resolution stops at first failure)", secondCtors)
	}
}

func TestCtorReceivesWorld(t *testing.T) {
	w := NewWorld()
	var received *World
	if err := w.Provide(func(world *World) (*atlas, error) {
		received = world
		return &atlas{}, nil
	}); err != nil {
		t.Fatalf("Provide: %v", err)
	}
	if _, err := w.Resource[*atlas](); err != nil {
		t.Fatalf("Resource: %v", err)
	}
	if received != w {
		t.Error("ctor did not receive the owning world")
	}
}

func TestNewEntityAssignsSequentialIDs(t *testing.T) {
	w := NewWorld()
	e1 := w.NewEntity(position{})
	e2 := w.NewEntity(position{})
	if e1.ID() == e2.ID() {
		t.Errorf("entities got the same ID %d", e1.ID())
	}
	if e2.ID() != e1.ID()+1 {
		t.Errorf("IDs not sequential: %d then %d", e1.ID(), e2.ID())
	}
	for _, e := range []*Entity{e1, e2} {
		if !e.IsAlive() {
			t.Errorf("entity %d should be alive after creation", e.ID())
		}
	}
}

func TestNewEntitiesCreatesBatch(t *testing.T) {
	w := NewWorld()
	entities := w.NewEntities(3, position{x: 1})
	if len(entities) != 3 {
		t.Fatalf("NewEntities(3) returned %d entities, want 3", len(entities))
	}
	seen := map[EntityID]bool{}
	for _, e := range entities {
		if seen[e.ID()] {
			t.Errorf("duplicate ID %d in batch", e.ID())
		}
		seen[e.ID()] = true
		if !e.HasComponent[position](w) {
			t.Errorf("entity %d missing its components", e.ID())
		}
	}
}

func TestDestroyEntityKillsAndRemovesComponents(t *testing.T) {
	w := NewWorld()
	e := w.NewEntity(position{})
	if err := w.DestroyEntity(e); err != nil {
		t.Fatalf("DestroyEntity: %v", err)
	}
	if e.IsAlive() {
		t.Error("destroyed entity should be killed")
	}
	if _, ok := e.Component[position](w); ok {
		t.Error("Component on a destroyed entity should report false")
	}
}

func TestDestroyEntityTwiceErrors(t *testing.T) {
	w := NewWorld()
	e := w.NewEntity(position{})
	if err := w.DestroyEntity(e); err != nil {
		t.Fatalf("first DestroyEntity: %v", err)
	}
	if err := w.DestroyEntity(e); err == nil {
		t.Error("destroying an already-destroyed entity should error")
	}
}

func TestDestroyEntitiesAll(t *testing.T) {
	w := NewWorld()
	entities := w.NewEntities(2, position{})
	if err := w.DestroyEntities(entities); err != nil {
		t.Fatalf("DestroyEntities: %v", err)
	}
	for _, e := range entities {
		if e.IsAlive() {
			t.Errorf("entity %d should be dead", e.ID())
		}
		if e.HasComponent[position](w) {
			t.Errorf("entity %d components should be gone", e.ID())
		}
	}
}

func TestDestroyEntitiesStopsAtFirstError(t *testing.T) {
	w := NewWorld()
	e1 := w.NewEntity(position{})
	e2 := w.NewEntity(position{})
	if err := w.DestroyEntity(e1); err != nil {
		t.Fatalf("DestroyEntity: %v", err)
	}
	if err := w.DestroyEntities([]*Entity{e1, e2}); err == nil {
		t.Error("DestroyEntities should fail when any entity fails to destroy")
	}
	if !e2.IsAlive() {
		t.Error("DestroyEntities should stop at the first failure, leaving later entities untouched")
	}
}
