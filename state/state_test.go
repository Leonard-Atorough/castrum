package state

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
)

func TestRequests(t *testing.T) {
	m := New("boot")
	if m.Current() != "boot" || !m.Is("boot") || m.Is("play") {
		t.Fatal("machine did not retain its initial state")
	}
	if _, ok := m.Pending(); ok {
		t.Fatal("new machine has a pending transition")
	}
	if err := m.Request("boot"); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Pending(); ok {
		t.Fatal("requesting the current state queued a transition")
	}
	if err := m.Request("play"); err != nil {
		t.Fatal(err)
	}
	for _, next := range []string{"play", "boot"} {
		if err := m.Request(next); err != nil {
			t.Fatalf("Request(%q): %v", next, err)
		}
	}
	if err := m.Request("done"); !errors.Is(err, ErrTransitionPending) {
		t.Fatalf("conflicting request error = %v, want ErrTransitionPending", err)
	}
	if next, ok := m.Pending(); !ok || next != "play" {
		t.Fatalf("pending = %q, %v, want play, true", next, ok)
	}
	if !m.Is("boot") {
		t.Fatal("request changed the current state synchronously")
	}
}

func TestHookOrderAndContext(t *testing.T) {
	m := New("boot")
	ctx := &core.Context{World: core.NewWorld(), Frame: 7}
	var got []string
	hook := func(label, current string) core.SystemFunc {
		return func(actual *core.Context) error {
			if actual != ctx || !m.Is(current) {
				t.Fatalf("%s: wrong context or current state %q", label, m.Current())
			}
			got = append(got, label)
			return nil
		}
	}
	m.OnEnter("boot", hook("boot:enter:1", "boot"))
	m.OnEnter("boot", hook("boot:enter:2", "boot"))
	m.OnExit("boot", hook("boot:exit:1", "boot"))
	m.OnExit("boot", hook("boot:exit:2", "boot"))
	m.OnEnter("play", hook("play:enter:1", "play"))
	m.OnEnter("play", hook("play:enter:2", "play"))
	system := NewSystem(m)
	if err := m.Request("play"); err != nil {
		t.Fatal(err)
	}
	if err := system.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if !m.Is("boot") {
		t.Fatal("first update processed a transition before initial entry")
	}
	for range 3 {
		if err := system.Update(ctx); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"boot:enter:1", "boot:enter:2", "boot:exit:1", "boot:exit:2", "play:enter:1", "play:enter:2"}
	if !slices.Equal(got, want) {
		t.Fatalf("hooks = %v, want %v", got, want)
	}
	if _, ok := m.Pending(); ok {
		t.Fatal("applied transition is still pending")
	}
}

func TestHookRequestsAreDeferred(t *testing.T) {
	for _, action := range []string{"initial enter", "exit", "enter"} {
		t.Run(action, func(t *testing.T) {
			m := New(0)
			request := func(*core.Context) error { return m.Request(2) }
			switch action {
			case "initial enter":
				m.OnEnter(0, request)
			case "exit":
				m.OnExit(0, request)
			case "enter":
				m.OnEnter(1, request)
			}
			system := NewSystem(m)
			ctx := &core.Context{}
			if err := system.Update(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if action != "initial enter" {
				if err := m.Request(1); err != nil {
					t.Fatal(err)
				}
				if err := system.Update(ctx); err != nil {
					t.Fatal(err)
				}
				want = 1
			}
			if m.Current() != want {
				t.Fatalf("state = %d, want %d before deferred request", m.Current(), want)
			}
			if next, ok := m.Pending(); !ok || next != 2 {
				t.Fatalf("pending = %d, %v, want 2, true", next, ok)
			}
			if err := system.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if !m.Is(2) {
				t.Fatalf("state = %d, want 2", m.Current())
			}
		})
	}
}

func TestHookErrors(t *testing.T) {
	for _, action := range []string{"initial enter", "exit", "enter"} {
		t.Run(action, func(t *testing.T) {
			m := New("boot")
			boom := errors.New("hook failed")
			calls := 0
			fail := func(*core.Context) error {
				calls++
				return boom
			}
			unreachable := func(*core.Context) error {
				t.Fatal("hook after failing hook ran")
				return nil
			}
			switch action {
			case "initial enter":
				m.OnEnter("boot", fail)
				m.OnEnter("boot", unreachable)
			case "exit":
				m.OnExit("boot", fail)
				m.OnExit("boot", unreachable)
			case "enter":
				m.OnEnter("play", fail)
				m.OnEnter("play", unreachable)
			}
			system := NewSystem(m)
			ctx := &core.Context{}
			err := system.Update(ctx)
			want := "boot"
			if action != "initial enter" {
				if err != nil {
					t.Fatal(err)
				}
				if err := m.Request("play"); err != nil {
					t.Fatal(err)
				}
				err = system.Update(ctx)
				if action == "enter" {
					want = "play"
				}
			}
			if !errors.Is(err, boom) || !strings.Contains(err.Error(), "hook for") {
				t.Fatalf("error = %v, want contextual wrapped hook error", err)
			}
			if !m.Is(want) {
				t.Fatalf("state = %q, want %q", m.Current(), want)
			}
			if _, ok := m.Pending(); ok {
				t.Fatal("failed transition was not consumed")
			}
			if err := system.Update(ctx); err != nil {
				t.Fatalf("next update replayed a failed hook: %v", err)
			}
			if calls != 1 {
				t.Fatalf("failing hook calls = %d, want 1", calls)
			}
		})
	}
}

func TestZeroValueMachine(t *testing.T) {
	var m Machine[int]
	var got []string
	m.OnEnter(0, func(*core.Context) error {
		got = append(got, "enter")
		return nil
	})
	m.OnExit(0, func(*core.Context) error {
		got = append(got, "exit")
		return nil
	})
	system := NewSystem(&m)
	if err := m.Request(1); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := system.Update(&core.Context{}); err != nil {
			t.Fatal(err)
		}
	}
	if !m.Is(1) || !slices.Equal(got, []string{"enter", "exit"}) {
		t.Fatalf("zero value state = %d, hooks = %v", m.Current(), got)
	}
}

func TestNilMachineReturnsError(t *testing.T) {
	if err := NewSystem[int](nil).Update(&core.Context{}); err == nil {
		t.Fatal("nil machine did not return an error")
	}
}

func TestNilHookReturnsError(t *testing.T) {
	m := New(0)
	m.OnEnter(0, nil)
	if err := NewSystem(m).Update(&core.Context{}); err == nil {
		t.Fatal("nil hook did not return an error")
	}
}

func TestExitHookCanRequestTheDestination(t *testing.T) {
	m := New(0)
	enters := 0
	m.OnExit(0, func(*core.Context) error { return m.Request(1) })
	m.OnEnter(1, func(*core.Context) error {
		enters++
		return nil
	})
	system := NewSystem(m)
	ctx := &core.Context{}
	if err := system.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Request(1); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := system.Update(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := m.Pending(); ok || !m.Is(1) || enters != 1 {
		t.Fatalf("state = %d, pending = %v, enter calls = %d, want 1, false, 1", m.Current(), ok, enters)
	}
}

func TestHookRequestSurvivesHookError(t *testing.T) {
	for _, action := range []string{"initial enter", "exit", "enter"} {
		t.Run(action, func(t *testing.T) {
			m := New(0)
			boom := errors.New("hook failed after request")
			requestAndFail := func(*core.Context) error {
				if err := m.Request(2); err != nil {
					return err
				}
				return boom
			}
			switch action {
			case "initial enter":
				m.OnEnter(0, requestAndFail)
			case "exit":
				m.OnExit(0, requestAndFail)
			case "enter":
				m.OnEnter(1, requestAndFail)
			}
			system := NewSystem(m)
			ctx := &core.Context{}
			err := system.Update(ctx)
			if action != "initial enter" {
				if err != nil {
					t.Fatal(err)
				}
				if err := m.Request(1); err != nil {
					t.Fatal(err)
				}
				err = system.Update(ctx)
			}
			if !errors.Is(err, boom) {
				t.Fatalf("error = %v, want hook error", err)
			}
			if next, ok := m.Pending(); !ok || next != 2 {
				t.Fatalf("pending = %d, %v, want 2, true", next, ok)
			}
			if action == "exit" {
				// Avoid replaying this deliberately failing exit hook.
				m.exitHooks[0] = nil
			}
			if err := system.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if !m.Is(2) {
				t.Fatalf("state = %d, want queued state 2", m.Current())
			}
		})
	}
}
