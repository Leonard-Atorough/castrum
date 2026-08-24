package core

import (
	"reflect"
	"slices"
	"testing"

	"github.com/leonard-atorough/castrum/ecs"
)

type testIndexComponentA struct{}
type testIndexComponentB struct{}
type testIndexComponentC struct{}

func TestEntityIndex_ComponentIndex(t *testing.T) {
	idx := NewEntityIndex()
	atype := reflect.TypeFor[testIndexComponentA]()
	btype := reflect.TypeFor[testIndexComponentB]()
	ctype := reflect.TypeFor[testIndexComponentC]()

	idx.AddComponent(1, atype)
	idx.AddComponent(1, btype)
	idx.AddComponent(2, atype)
	idx.AddComponent(3, ctype)
	idx.AddComponent(1, atype) // duplicate should not create a second entry

	if got := idx.GetEntitiesWithComponent(atype); !idsMatchUnordered(got, []ecs.EntityID{1, 2}) {
		t.Fatalf("unexpected A entities: %#v", got)
	}
	if got := idx.GetEntitiesWithComponent(btype); !idsMatchUnordered(got, []ecs.EntityID{1}) {
		t.Fatalf("unexpected B entities: %#v", got)
	}
	if got := idx.GetEntitiesWithComponent(ctype); !idsMatchUnordered(got, []ecs.EntityID{3}) {
		t.Fatalf("unexpected C entities: %#v", got)
	}

	if got := idx.GetEntitiesWithComponents(atype, btype); !idsMatchUnordered(got, []ecs.EntityID{1}) {
		t.Fatalf("expected intersection to be {1}, got %#v", got)
	}
	if got := idx.GetEntitiesWithComponents(atype, ctype); len(got) != 0 {
		t.Fatalf("expected no entity to satisfy both A and C, got %#v", got)
	}

	idx.RemoveComponent(1, atype)
	if got := idx.GetEntitiesWithComponent(atype); !idsMatchUnordered(got, []ecs.EntityID{2}) {
		t.Fatalf("after removal, A should only contain entity 2, got %#v", got)
	}

	idx.UpdateComponent(NewEntity(5, "x"), atype, true)
	if got := idx.GetEntitiesWithComponent(atype); !idsMatchUnordered(got, []ecs.EntityID{2, 5}) {
		t.Fatalf("after update add, A should contain 2 and 5, got %#v", got)
	}
	idx.UpdateComponent(NewEntity(5, "x"), atype, false)
	if got := idx.GetEntitiesWithComponent(atype); !idsMatchUnordered(got, []ecs.EntityID{2}) {
		t.Fatalf("after update remove, A should contain only 2, got %#v", got)
	}
}

func TestEntityIndex_TagsAndTemplates(t *testing.T) {
	idx := NewEntityIndex()

	idx.AddTag(1, "player")
	idx.AddTag(1, "player")
	idx.AddTag(2, "player")
	idx.AddTag(3, "npc")

	if got := idx.GetEntitiesWithTag("player"); !idsMatchUnordered(got, []ecs.EntityID{1, 2}) {
		t.Fatalf("unexpected player entities: %#v", got)
	}
	if got := idx.GetEntitiesWithTag("npc"); !idsMatchUnordered(got, []ecs.EntityID{3}) {
		t.Fatalf("unexpected npc entities: %#v", got)
	}

	idx.UpdateTag(NewEntity(1, "player"), "player", false)
	if got := idx.GetEntitiesWithTag("player"); !idsMatchUnordered(got, []ecs.EntityID{2}) {
		t.Fatalf("after removing tag from entity 1, player set should be {2}, got %#v", got)
	}

	idx.AddTemplate(10, "enemy")
	idx.AddTemplate(10, "enemy")
	idx.AddTemplate(11, "enemy")
	idx.UpdateTemplate(NewEntity(10, "enemy"), "enemy", "boss")
	if got := idx.GetEntitiesWithTemplate("enemy"); !idsMatchUnordered(got, []ecs.EntityID{11}) {
		t.Fatalf("expected enemy template to only contain entity 11, got %#v", got)
	}
	if got := idx.GetEntitiesWithTemplate("boss"); !idsMatchUnordered(got, []ecs.EntityID{10}) {
		t.Fatalf("expected boss template to contain entity 10, got %#v", got)
	}
}

func TestEntityIndex_EmptyQueries(t *testing.T) {
	idx := NewEntityIndex()
	if got := idx.GetEntitiesWithComponent(reflect.TypeOf(testIndexComponentA{})); got != nil {
		t.Fatalf("empty component query should return nil, got %#v", got)
	}
	if got := idx.GetEntitiesWithComponents(reflect.TypeOf(testIndexComponentA{})); got != nil {
		t.Fatalf("empty multi-component query should return nil, got %#v", got)
	}
	if got := idx.GetEntitiesWithTag("missing"); got != nil {
		t.Fatalf("missing tag should return nil, got %#v", got)
	}
	if got := idx.GetEntitiesWithTemplate("missing"); got != nil {
		t.Fatalf("missing template should return nil, got %#v", got)
	}
}

func idsMatchUnordered(a, b []ecs.EntityID) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}

	left := append([]ecs.EntityID(nil), a...)
	right := append([]ecs.EntityID(nil), b...)
	slices.Sort(left)
	slices.Sort(right)

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
