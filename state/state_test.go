package state

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/signal"
)

func testContext(t *testing.T) *core.Context {
	t.Helper()
	world := core.NewWorld()
	if err := world.Provide(func(*core.World) (*signal.Hub, error) {
		return signal.NewHub(), nil
	}); err != nil {
		t.Fatal(err)
	}
	return &core.Context{World: world}
}

func TestMissingHubPreservesPendingState(t *testing.T) {
	ctx := &core.Context{World: core.NewWorld()}
	m := New[int]()
	if err := m.Request(1); err != nil {
		t.Fatal(err)
	}
	system := NewSystem(m)
	if err := system.Update(ctx); !errors.Is(err, core.ErrResourceNotRegistered) {
		t.Fatalf("missing hub error = %v", err)
	}
	if pending, ok := m.Pending(); m.initialized || !m.Is(0) || !ok || pending != 1 {
		t.Fatal("hub resolution failure changed initialization or pending intent")
	}
	if err := ctx.World.Provide(func(*core.World) (*signal.Hub, error) {
		return signal.NewHub(), nil
	}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := system.Update(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if !m.Is(1) {
		t.Fatal("pending transition did not survive hub resolution failure")
	}
}

func TestRequests(t *testing.T) {
	m := New[string]().WithInitial("boot")
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

func TestRestartReentersCurrentState(t *testing.T) {
	m := New[int]().WithInitial(1)
	var hooks []string
	m.OnExit(1, func(*core.Context) error {
		hooks = append(hooks, "exit")
		return nil
	})
	m.OnEnter(1, func(*core.Context) error {
		hooks = append(hooks, "enter")
		return nil
	})

	ctx := testContext(t)
	hub := ctx.World.MustResource[*signal.Hub]()
	changed := hub.NewReader[Changed[int]]()
	system := NewSystem(m)
	if err := system.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(hooks, []string{"enter"}) {
		t.Fatalf("initial hooks = %v, want [enter]", hooks)
	}
	if _, ok := changed.Next(); ok {
		t.Fatal("initial entry emitted Changed")
	}

	if err := m.Restart(); err != nil {
		t.Fatal(err)
	}
	if err := m.Restart(); err != nil {
		t.Fatalf("repeated Restart: %v", err)
	}
	if err := m.Request(1); err != nil {
		t.Fatalf("Request(current) with restart pending: %v", err)
	}
	if err := m.Request(2); !errors.Is(err, ErrTransitionPending) {
		t.Fatalf("conflicting request error = %v, want ErrTransitionPending", err)
	}
	if pending, ok := m.Pending(); !ok || pending != 1 {
		t.Fatalf("pending = %d, %v, want restart of 1", pending, ok)
	}
	if !slices.Equal(hooks, []string{"enter"}) {
		t.Fatalf("Restart ran hooks before processing: %v", hooks)
	}

	if err := system.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(hooks, []string{"enter", "exit", "enter"}) {
		t.Fatalf("restart hooks = %v, want [enter exit enter]", hooks)
	}
	if event, ok := changed.Next(); !ok || event != (Changed[int]{From: 1, To: 1}) {
		t.Fatalf("restart signal = %+v, %t, want Changed{1, 1}", event, ok)
	}
	if _, ok := m.Pending(); ok {
		t.Fatal("restart remained pending after processing")
	}

	if err := m.Request(2); err != nil {
		t.Fatal(err)
	}
	if err := m.Restart(); !errors.Is(err, ErrTransitionPending) {
		t.Fatalf("Restart with another destination pending = %v, want ErrTransitionPending", err)
	}
	if pending, ok := m.Pending(); !ok || pending != 2 {
		t.Fatalf("pending = %d, %v, want 2 after rejected restart", pending, ok)
	}
}

func TestWithInitialCopiesConfiguration(t *testing.T) {
	original := New[int]()
	calls := 0
	original.OnEnter(1, func(*core.Context) error { calls++; return nil })
	if err := NewSystem(original).Update(testContext(t)); err != nil {
		t.Fatal(err)
	}
	if err := original.Request(2); err != nil {
		t.Fatal(err)
	}
	configured := original.WithInitial(1)
	configured.OnEnter(1, func(*core.Context) error { calls++; return nil })
	if configured == original || configured.initialized || configured.hasPending || !configured.Is(1) {
		t.Fatal("WithInitial did not return a fresh machine")
	}
	if pending, ok := original.Pending(); !original.Is(0) || !original.initialized || !ok || pending != 2 ||
		len(original.entryHooks[1]) != 1 {
		t.Fatal("WithInitial modified the original machine or its hooks")
	}
	if err := NewSystem(configured).Update(testContext(t)); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("copied hook calls = %d, want 2", calls)
	}
}

func TestHookOrderAndContext(t *testing.T) {
	m := New[string]().WithInitial("boot")
	ctx := testContext(t)
	ctx.Frame = 7
	hub := ctx.World.MustResource[*signal.Hub]()
	initialized := hub.NewReader[Initialized[string]]()
	changed := hub.NewReader[Changed[string]]()
	observer := hub.NewReader[Changed[string]]()
	var got []string
	hook := func(label, current string) core.SystemFunc {
		return func(actual *core.Context) error {
			if actual != ctx || !m.Is(current) {
				t.Fatalf("%s: wrong context or current state %q", label, m.Current())
			}
			if label == "boot:enter:1" {
				if _, ok := initialized.Next(); ok {
					t.Fatal("Initialized emitted before initial hooks completed")
				}
			}
			if label == "boot:exit:1" {
				if _, ok := changed.Next(); ok {
					t.Fatal("Changed emitted before exit hooks completed")
				}
			}
			if label == "play:enter:1" {
				if event, ok := changed.Next(); !ok || event != (Changed[string]{From: "boot", To: "play"}) {
					t.Fatalf("enter hook signal = %+v, %t", event, ok)
				}
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
	if event, ok := initialized.Next(); !ok || event.State != "boot" {
		t.Fatalf("Initialized = %+v, %t", event, ok)
	}
	if event, ok := observer.Next(); !ok || event != (Changed[string]{From: "boot", To: "play"}) {
		t.Fatalf("independent change reader = %+v, %t", event, ok)
	}
	if _, ok := initialized.Next(); ok {
		t.Fatal("initialization emitted more than once")
	}
	if _, ok := observer.Next(); ok {
		t.Fatal("change emitted without another transition")
	}
}

func TestHookRequestsAreDeferred(t *testing.T) {
	for _, action := range []string{"initial enter", "exit", "enter"} {
		t.Run(action, func(t *testing.T) {
			m := New[int]()
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
			ctx := testContext(t)
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
			m := New[string]().WithInitial("boot")
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
			ctx := testContext(t)
			hub := ctx.World.MustResource[*signal.Hub]()
			initialized := hub.NewReader[Initialized[string]]()
			changed := hub.NewReader[Changed[string]]()
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
			if _, ok := initialized.Next(); ok != (action != "initial enter") {
				t.Fatalf("Initialized after %s failure = %t", action, ok)
			}
			if event, ok := changed.Next(); ok != (action == "enter") ||
				(ok && event != (Changed[string]{From: "boot", To: "play"})) {
				t.Fatalf("Changed after %s failure = %+v, %t", action, event, ok)
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
	ctx := testContext(t)
	if err := m.Request(1); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := system.Update(ctx); err != nil {
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
	m := New[int]()
	m.OnEnter(0, nil)
	if err := NewSystem(m).Update(testContext(t)); err == nil {
		t.Fatal("nil hook did not return an error")
	}
}

func TestExitHookCanRequestTheDestination(t *testing.T) {
	m := New[int]()
	enters := 0
	m.OnExit(0, func(*core.Context) error { return m.Request(1) })
	m.OnEnter(1, func(*core.Context) error {
		enters++
		return nil
	})
	system := NewSystem(m)
	ctx := testContext(t)
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
			m := New[int]()
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
			ctx := testContext(t)
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
