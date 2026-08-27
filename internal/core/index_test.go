package core

import (
	"slices"
	"testing"
)

type testIndexComponentA struct{}
type testIndexComponentB struct{}
type testIndexComponentC struct{}

func TestEntityIndex_TagsAndTemplates(t *testing.T) {
	idx := NewEntityIndex()

	idx.AddTag(1, "player")
	idx.AddTag(1, "player")
	idx.AddTag(2, "player")
	idx.AddTag(3, "npc")

	if got := idx.GetEntitiesWithTag("player"); !idsMatchUnordered(got, []EntityID{1, 2}) {
		t.Fatalf("unexpected player entities: %#v", got)
	}
	if got := idx.GetEntitiesWithTag("npc"); !idsMatchUnordered(got, []EntityID{3}) {
		t.Fatalf("unexpected npc entities: %#v", got)
	}

	idx.UpdateTag(NewEntity(1, "player"), "player", false)
	if got := idx.GetEntitiesWithTag("player"); !idsMatchUnordered(got, []EntityID{2}) {
		t.Fatalf("after removing tag from entity 1, player set should be {2}, got %#v", got)
	}

	idx.AddTemplate(10, "enemy")
	idx.AddTemplate(10, "enemy")
	idx.AddTemplate(11, "enemy")
	idx.UpdateTemplate(NewEntity(10, "enemy"), "enemy", "boss")
	if got := idx.GetEntitiesWithTemplate("enemy"); !idsMatchUnordered(got, []EntityID{11}) {
		t.Fatalf("expected enemy template to only contain entity 11, got %#v", got)
	}
	if got := idx.GetEntitiesWithTemplate("boss"); !idsMatchUnordered(got, []EntityID{10}) {
		t.Fatalf("expected boss template to contain entity 10, got %#v", got)
	}
}

func idsMatchUnordered(a, b []EntityID) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}

	left := append([]EntityID(nil), a...)
	right := append([]EntityID(nil), b...)
	slices.Sort(left)
	slices.Sort(right)

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
