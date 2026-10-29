package runtime

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
)

func TestScheduleOrderAndGroups(t *testing.T) {
	s := NewSchedule(core.PhaseFixed, func(sys core.System, ctx *core.Context) error {
		return sys.Update(ctx)
	})
	var got []string
	add := func(name, group string) Entry[core.System] {
		return Entry[core.System]{Name: name, Group: group, System: core.SystemFunc(func(*core.Context) error {
			got = append(got, name)
			return nil
		})}
	}
	if err := s.Run(&core.Context{}); err != nil {
		t.Fatalf("empty schedule: %v", err)
	}
	s.Add(add("first", ""), add("grouped", "game"), add("last", ""))
	for i, tc := range []struct {
		enabled bool
		want    []string
	}{
		{true, []string{"first", "grouped", "last"}},
		{false, []string{"first", "last"}},
		{true, []string{"first", "grouped", "last"}},
	} {
		if i > 0 {
			s.SetGroupEnabled("game", tc.enabled)
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

func TestScheduleStopsAtError(t *testing.T) {
	s := NewSchedule(core.PhaseFixed, func(sys core.System, ctx *core.Context) error {
		return sys.Update(ctx)
	})
	wantErr := errors.New("system error")
	var got []string
	for _, name := range []string{"first", "failing", "last"} {
		s.Add(Entry[core.System]{Name: name, System: core.SystemFunc(func(*core.Context) error {
			got = append(got, name)
			if name == "failing" {
				return wantErr
			}
			return nil
		})})
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
