package runtime

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
)

func TestScheduleOrderAndGroups(t *testing.T) {
	groups := make(map[string]bool)
	s := NewSchedule(core.PhaseFixed, groups)
	var got []string
	add := func(name, group string) Entry {
		return Entry{Name: name, Group: group, System: core.SystemFunc(func(*core.Context) error {
			got = append(got, name)
			return nil
		})}
	}
	if err := s.Run(&core.Context{}); err != nil {
		t.Fatalf("empty schedule: %v", err)
	}
	for _, entry := range []Entry{add("first", ""), add("grouped", "game"), add("last", "")} {
		if err := s.Add(entry); err != nil {
			t.Fatal(err)
		}
	}
	for i, tc := range []struct {
		enabled bool
		want    []string
	}{
		{true, []string{"first", "grouped", "last"}},
		{false, []string{"first", "last"}},
		{true, []string{"first", "grouped", "last"}},
	} {
		if i > 0 {
			groups["game"] = tc.enabled
		}
		got = nil
		if err := s.Run(&core.Context{}); err != nil {
			t.Fatalf("Run with group enabled=%t: %v", tc.enabled, err)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("group enabled=%t: ran %v, want %v", tc.enabled, got, tc.want)
		}
	}
}

func TestScheduleBoundaryAndRelativeOrder(t *testing.T) {
	s := NewSchedule(core.PhaseFixed, make(map[string]bool))
	var got []string
	add := func(name string) Entry {
		return Entry{Name: name, System: core.SystemFunc(func(*core.Context) error {
			got = append(got, name)
			return nil
		})}
	}

	if err := s.AddFirst(add("first")); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(add("anchor")); err != nil {
		t.Fatal(err)
	}
	if err := s.AddLast(add("last")); err != nil {
		t.Fatal(err)
	}
	if err := s.AddBefore("anchor", add("before")); err != nil {
		t.Fatal(err)
	}
	if err := s.AddAfter("anchor", add("after")); err != nil {
		t.Fatal(err)
	}
	if err := s.AddAfter("first", add("after-first")); err == nil {
		t.Fatal("AddAfter accepted an internal first-boundary entry")
	}
	if err := s.AddBefore("last", add("before-last")); err == nil {
		t.Fatal("AddBefore accepted an internal last-boundary entry")
	}

	if err := s.Run(&core.Context{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"first", "before", "anchor", "after", "last"}
	if !slices.Equal(got, want) {
		t.Fatalf("ran %v, want %v", got, want)
	}
}

func TestScheduleInsertionsDuringRunTakeEffectOnNextRun(t *testing.T) {
	s := NewSchedule(core.PhaseFixed, make(map[string]bool))
	var got []string
	inserted := false
	var add func(string) Entry
	add = func(name string) Entry {
		return Entry{Name: name, System: core.SystemFunc(func(*core.Context) error {
			got = append(got, name)
			if name == "inserter" && !inserted {
				inserted = true
				if err := s.AddFirst(add("first-added")); err != nil {
					return err
				}
				if err := s.Add(add("appended")); err != nil {
					return err
				}
				if err := s.AddLast(add("last-added")); err != nil {
					return err
				}
				if err := s.AddBefore("anchor", add("before")); err != nil {
					return err
				}
				if err := s.AddAfter("anchor", add("after")); err != nil {
					return err
				}
			}
			return nil
		})}
	}
	for _, entry := range []Entry{add("inserter"), add("anchor")} {
		if err := s.Add(entry); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.Run(&core.Context{}); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if want := []string{"inserter", "anchor"}; !slices.Equal(got, want) {
		t.Fatalf("first Run executed %v, want %v", got, want)
	}

	got = nil
	if err := s.Run(&core.Context{}); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	want := []string{"first-added", "inserter", "before", "anchor", "after", "appended", "last-added"}
	if !slices.Equal(got, want) {
		t.Fatalf("second Run executed %v, want %v", got, want)
	}
}

func TestScheduleFirstAndLastGroupsAreRegistered(t *testing.T) {
	groups := make(map[string]bool)
	s := NewSchedule(core.PhaseFixed, groups)
	var got []string
	add := func(name, group string) Entry {
		return Entry{Name: name, Group: group, System: core.SystemFunc(func(*core.Context) error {
			got = append(got, name)
			return nil
		})}
	}
	if err := s.AddFirst(add("first", "setup")); err != nil {
		t.Fatal(err)
	}
	if err := s.AddLast(add("last", "cleanup")); err != nil {
		t.Fatal(err)
	}

	if !groups["setup"] || !groups["cleanup"] {
		t.Fatal("groups in boundary entries were not registered")
	}
	groups["setup"] = false
	groups["cleanup"] = false
	if err := s.Run(&core.Context{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ran %v with both groups disabled, want no systems", got)
	}
}

func TestScheduleFailedInsertDoesNotReserveNames(t *testing.T) {
	s := NewSchedule(core.PhaseFixed, make(map[string]bool))
	add := func(name string) Entry {
		return Entry{Name: name, System: core.SystemFunc(func(*core.Context) error { return nil })}
	}
	if err := s.Add(add("existing")); err != nil {
		t.Fatal(err)
	}
	if err := s.AddBefore("missing", add("new")); err == nil {
		t.Fatal("AddBefore with a missing target returned no error")
	}
	if err := s.Add(add("new")); err != nil {
		t.Fatalf("failed insertion reserved name: %v", err)
	}
	if err := s.Add(add("existing")); err == nil {
		t.Fatal("Add with a duplicate name returned no error")
	}
}

func TestScheduleStopsAtError(t *testing.T) {
	s := NewSchedule(core.PhaseFixed, make(map[string]bool))
	wantErr := errors.New("system error")
	var got []string
	for _, name := range []string{"first", "failing", "last"} {
		if err := s.Add(Entry{Name: name, System: core.SystemFunc(func(*core.Context) error {
			got = append(got, name)
			if name == "failing" {
				return wantErr
			}
			return nil
		})}); err != nil {
			t.Fatal(err)
		}
	}
	err := s.Run(&core.Context{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run error = %v, want wrapped %v", err, wantErr)
	}
	if !strings.Contains(err.Error(), `"failing"`) || !strings.Contains(err.Error(), "fixed") {
		t.Errorf("error %q should name the system and phase", err)
	}
	if want := []string{"first", "failing"}; !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
}
