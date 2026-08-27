package core

import (
	"reflect"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

// TestWorld_ComponentLifecycle tests adding, removing, getting, and querying components.
func TestWorld_ComponentLifecycle(t *testing.T) {
	w := NewWorld()
	id := w.CreateEntity("Generic")

	// Test adding a component
	compA := testComponentA{value: 42}
	if err := w.AddComponent(id, compA); err != nil {
		t.Fatalf("AddComponent failed: %v", err)
	}

	// Test HasComponent
	if !w.HasComponent(id, reflect.TypeOf(compA)) {
		t.Fatal("HasComponent should return true after adding component")
	}

	// Test GetComponent
	got, err := w.GetComponent(id, reflect.TypeOf(compA))
	if err != nil {
		t.Fatalf("GetComponent failed: %v", err)
	}
	gotComp, ok := got.(testComponentA)
	if !ok || gotComp.value != 42 {
		t.Fatalf("GetComponent returned wrong component, got %#v", got)
	}

	// Test adding a second component
	compB := testComponentB{value: 99}
	if err := w.AddComponent(id, compB); err != nil {
		t.Fatalf("AddComponent (second) failed: %v", err)
	}

	// Test GetAllComponents
	all := w.Components(id)
	if len(all) != 2 {
		t.Fatalf("expected 2 components, got %d", len(all))
	}

	// Test RemoveComponent
	if err := w.RemoveComponent(id, reflect.TypeOf(compA)); err != nil {
		t.Fatalf("RemoveComponent failed: %v", err)
	}

	if w.HasComponent(id, reflect.TypeOf(compA)) {
		t.Fatal("HasComponent should return false after removing component")
	}

	// Component B should still exist
	if !w.HasComponent(id, reflect.TypeOf(compB)) {
		t.Fatal("HasComponent should return true for remaining component")
	}

	w.DestroyEntity(id, false)
	w.Cleanup()
	if w.Exists(id) {
		t.Fatalf("entity should be destroyed after cleanup")
	}
}

// TestWorld_ComponentQueries tests querying entities by component type.
func TestWorld_ComponentQueries(t *testing.T) {
	w := NewWorld()

	// Create entities with different component combinations
	id1 := w.CreateEntity("Generic")
	id2 := w.CreateEntity("Generic")

	compA := testComponentA{value: 1}
	compB := testComponentB{value: 2}

	// id1 has A and B
	w.AddComponent(id1, compA)
	w.AddComponent(id1, compB)

	// id2 has only A
	w.AddComponent(id2, compA)

	// Query for A
	queriedA := w.Query(reflect.TypeOf(compA))
	if len(queriedA) != 2 {
		t.Fatalf("expected 2 entities with A, got %d", len(queriedA))
	}

	// Query for B
	queriedB := w.Query(reflect.TypeOf(compB))
	if len(queriedB) != 1 {
		t.Fatalf("expected 1 entity with B, got %d", len(queriedB))
	}

	// Query for both A and B
	queriedAB := w.Query(reflect.TypeOf(compA), reflect.TypeOf(compB))
	if len(queriedAB) != 1 {
		t.Fatalf("expected 1 entity with both A and B, got %d", len(queriedAB))
	}
	if queriedAB[0] != id1 {
		t.Fatalf("expected id1 in query result, got %d", queriedAB[0])
	}
}

// TestWorld_TagManagement tests adding, removing, and querying by tags.
func TestWorld_TagManagement(t *testing.T) {
	w := NewWorld()
	id := w.CreateEntity("Generic")

	// Test adding a custom tag
	if err := w.AddTag(id, "soldier"); err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}

	has, err := w.HasTag(id, "soldier")
	if err != nil {
		t.Fatalf("HasTag failed: %v", err)
	}
	if !has {
		t.Fatal("HasTag should return true after adding tag")
	}

	// Test removing a tag
	if err := w.RemoveTag(id, "soldier"); err != nil {
		t.Fatalf("RemoveTag failed: %v", err)
	}

	has, err = w.HasTag(id, "soldier")
	if err != nil {
		t.Fatalf("HasTag failed: %v", err)
	}
	if has {
		t.Fatal("HasTag should return false after removing tag")
	}

	// Test QueryByTag
	id2 := w.CreateEntity("Generic")
	w.AddTag(id, "archer")
	w.AddTag(id2, "archer")

	archers := w.QueryByTag("archer")
	if len(archers) != 2 {
		t.Fatalf("expected 2 archers, got %d", len(archers))
	}

	w.DestroyEntity(id, false)
	w.DestroyEntity(id2, false)
	w.Cleanup()
}

// TestWorld_TemplateQueries tests querying entities by template.
func TestWorld_TemplateQueries(t *testing.T) {
	w := NewWorld()

	// The Create method applies template-based tags (Province, City, Generic)
	provinceID := w.CreateEntity("Province")
	cityID := w.CreateEntity("City")
	generic1ID := w.CreateEntity("Generic")
	generic2ID := w.CreateEntity("Generic")

	provinces := w.QueryByTemplate("Province")
	if len(provinces) != 1 || provinces[0] != provinceID {
		t.Fatalf("expected to find 1 Province, got %d", len(provinces))
	}

	cities := w.QueryByTemplate("City")
	if len(cities) != 1 || cities[0] != cityID {
		t.Fatalf("expected to find 1 City, got %d", len(cities))
	}

	generics := w.QueryByTemplate("Generic")
	if len(generics) != 2 {
		t.Fatalf("expected to find 2 Generic templates, got %d: %v", len(generics), generics)
	}

	w.DestroyEntity(provinceID, false)
	w.DestroyEntity(cityID, false)
	w.DestroyEntity(generic1ID, false)
	w.DestroyEntity(generic2ID, false)
	w.Cleanup()
}

// TestWorld_HierarchyNavigation tests parent-child relationships and navigation.
func TestWorld_HierarchyNavigation(t *testing.T) {
	w := NewWorld()

	root := w.CreateEntity("Generic")
	child1 := w.CreateEntity("Generic")
	child2 := w.CreateEntity("Generic")
	grandchild := w.CreateEntity("Generic")

	// Build hierarchy: root -> [child1, child2], child1 -> [grandchild]
	w.SetParent(child1, root)
	w.SetParent(child2, root)
	w.SetParent(grandchild, child1)

	// Test GetChildren
	children := w.ChildrenOf(root)
	if len(children) != 2 {
		t.Fatalf("expected 2 children of root, got %d", len(children))
	}

	// Test GetParent
	parent, hasParent := w.ParentOf(child1)
	if !hasParent || parent != root {
		t.Fatalf("expected child1 to have root as parent, got %d", parent)
	}

	// Test Detach
	w.Detach(child1)
	parent, hasParent = w.ParentOf(child1)
	if hasParent {
		t.Fatal("expected child1 to have no parent after detach")
	}

	// Grandchild should still have child1 as parent after child1 is detached from root
	parent, hasParent = w.ParentOf(grandchild)
	if !hasParent || parent != child1 {
		t.Fatalf("expected grandchild to still have child1 as parent")
	}

	w.DestroyEntity(root, false)
	w.DestroyEntity(child1, false)
	w.DestroyEntity(child2, false)
	w.DestroyEntity(grandchild, false)
	w.Cleanup()
}

// TestWorld_CompleteEntityLifecycle tests a complex scenario with components, tags, and hierarchy.
func TestWorld_CompleteEntityLifecycle(t *testing.T) {
	w := NewWorld()

	// Create a parent entity with components and tags
	parentID := w.CreateEntity("Generic")
	w.AddComponent(parentID, testComponentA{value: 100})
	w.AddTag(parentID, "warrior")
	w.AddTag(parentID, "leader")

	// Create child entities
	child1ID := w.CreateEntity("Generic")
	w.AddComponent(child1ID, testComponentA{value: 200})
	w.SetParent(child1ID, parentID)

	child2ID := w.CreateEntity("Generic")
	w.AddComponent(child2ID, testComponentB{value: 300})
	w.AddTag(child2ID, "warrior")
	w.SetParent(child2ID, parentID)

	// Verify initial state
	if w.Count() != 3 {
		t.Fatalf("expected 3 entities, got %d", w.Count())
	}

	// Query warriors
	warriors := w.QueryByTag("warrior")
	if len(warriors) != 2 {
		t.Fatalf("expected 2 warriors, got %d", len(warriors))
	}

	// Query by component A
	withA := w.Query(reflect.TypeOf(testComponentA{}))
	if len(withA) != 2 {
		t.Fatalf("expected 2 entities with component A, got %d", len(withA))
	}

	// Verify hierarchy
	parentChildren := w.ChildrenOf(parentID)
	if len(parentChildren) != 2 {
		t.Fatalf("expected 2 children, got %d", len(parentChildren))
	}

	// Destroy parent with cascade
	w.DestroyEntity(parentID, true)
	w.Cleanup()

	if w.Count() != 0 {
		t.Fatalf("expected 0 entities after cascade destroy, got %d", w.Count())
	}
}

// TestWorld_CountAndReset tests entity counting and world reset.
func TestWorld_CountAndReset(t *testing.T) {
	w := NewWorld()

	if w.Count() != 0 {
		t.Fatalf("expected 0 entities in new world, got %d", w.Count())
	}

	id1 := w.CreateEntity("Generic")
	w.AddComponent(id1, testComponentA{value: 1})
	w.AddTag(id1, "test")

	id2 := w.CreateEntity("Generic")
	w.CreateEntity("Generic") // third entity for testing Count

	if w.Count() != 3 {
		t.Fatalf("expected 3 entities, got %d", w.Count())
	}

	// Set up a hierarchy
	w.SetParent(id2, id1)

	// Verify query still works
	generics := w.QueryByTemplate("Generic")
	if len(generics) != 3 {
		t.Fatalf("expected 3 generic entities, got %d", len(generics))
	}

	// Reset the world
	w.Reset()

	if w.Count() != 0 {
		t.Fatalf("expected 0 entities after reset, got %d", w.Count())
	}

	generics = w.QueryByTemplate("Generic")
	if len(generics) != 0 {
		t.Fatalf("expected no entities after reset, got %d", len(generics))
	}

	// Should be able to create again
	newID := w.CreateEntity("Generic")
	if !w.Exists(newID) {
		t.Fatal("should be able to create new entity after reset")
	}
	if w.Count() != 1 {
		t.Fatalf("expected 1 entity after new create, got %d", w.Count())
	}
}

// TestWorld_ComponentErrors tests error handling for non-existent entities.
func TestWorld_ComponentErrors(t *testing.T) {
	w := NewWorld()
	nonExistent := ecs.EntityID(999)

	// Test AddComponent with non-existent entity
	if err := w.AddComponent(nonExistent, testComponentA{}); err == nil {
		t.Fatal("expected error adding component to non-existent entity")
	}

	// Test GetComponent with non-existent entity
	if _, err := w.GetComponent(nonExistent, reflect.TypeOf(testComponentA{})); err == nil {
		t.Fatal("expected error getting component from non-existent entity")
	}

	// Test RemoveComponent with non-existent entity
	if err := w.RemoveComponent(nonExistent, reflect.TypeOf(testComponentA{})); err == nil {
		t.Fatal("expected error removing component from non-existent entity")
	}

	// Test AddTag with non-existent entity
	if err := w.AddTag(nonExistent, "test"); err == nil {
		t.Fatal("expected error adding tag to non-existent entity")
	}

	// Test HasTag with non-existent entity
	if _, err := w.HasTag(nonExistent, "test"); err == nil {
		t.Fatal("expected error checking tag on non-existent entity")
	}

	// Test RemoveTag with non-existent entity
	if err := w.RemoveTag(nonExistent, "test"); err == nil {
		t.Fatal("expected error removing tag from non-existent entity")
	}
}
