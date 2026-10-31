// Package state provides generic state machines whose transitions are driven
// by scheduled system updates.
package state

import (
	"errors"
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
)

// ErrTransitionPending reports that a machine already has a different
// transition request waiting to be processed.
var ErrTransitionPending = errors.New("state transition already pending")

// Machine tracks a comparable state value and queues requested transitions
// until its system processes them. Its zero value starts in S's zero state.
// Access a machine only from the game loop's goroutine.
type Machine[S comparable] struct {
	current     S
	pending     S
	hasPending  bool
	initialized bool
	entryHooks  map[S][]core.SystemFunc
	exitHooks   map[S][]core.SystemFunc
}

// New returns a machine whose initial state is initial.
func New[S comparable](initial S) *Machine[S] {
	return &Machine[S]{
		current:    initial,
		pending:    initial,
		entryHooks: make(map[S][]core.SystemFunc),
		exitHooks:  make(map[S][]core.SystemFunc),
	}
}

// Current returns the machine's current state.
func (m *Machine[S]) Current() S {
	return m.current
}

// Is reports whether value is the current state.
func (m *Machine[S]) Is(value S) bool {
	return m.current == value
}

// Pending returns the accepted destination waiting to be processed, if any.
func (m *Machine[S]) Pending() (S, bool) {
	return m.pending, m.hasPending
}

// OnEnter registers a hook that runs when value is entered, including the
// initial state during startup. An error stops the remaining hooks and is
// returned by the state system. A nil hook returns an error when run.
func (m *Machine[S]) OnEnter(value S, hook core.SystemFunc) {
	if m.entryHooks == nil {
		m.entryHooks = make(map[S][]core.SystemFunc)
	}
	m.entryHooks[value] = append(m.entryHooks[value], hook)
}

// OnExit registers a hook that runs before leaving value. An error stops the
// transition and is returned by the state system. A nil hook returns an error
// when run.
func (m *Machine[S]) OnExit(value S, hook core.SystemFunc) {
	if m.exitHooks == nil {
		m.exitHooks = make(map[S][]core.SystemFunc)
	}
	m.exitHooks[value] = append(m.exitHooks[value], hook)
}

// Request accepts next as the pending destination without running hooks.
// Requests for the current state are no-ops; an identical pending request
// coalesces, while a different pending destination returns
// [ErrTransitionPending]. A successful request means it was accepted, not
// that the transition completed.
func (m *Machine[S]) Request(next S) error {
	if m.current == next {
		return nil
	}
	if m.hasPending {
		if m.pending == next {
			return nil
		}
		return fmt.Errorf("state: %w: pending %v, requested %v", ErrTransitionPending, m.pending, next)
	}
	m.pending = next
	m.hasPending = true
	return nil
}

func (m *Machine[S]) apply(ctx *core.Context) error {
	if !m.initialized {
		m.initialized = true
		return runHooks("enter", m.current, m.entryHooks[m.current], ctx)
	}
	if !m.hasPending {
		return nil
	}
	next := m.pending
	m.hasPending = false
	if m.current == next {
		return nil
	}
	if err := runHooks("exit", m.current, m.exitHooks[m.current], ctx); err != nil {
		return err
	}
	m.current = next
	return runHooks("enter", m.current, m.entryHooks[m.current], ctx)
}

func runHooks[S comparable](action string, value S, hooks []core.SystemFunc, ctx *core.Context) error {
	for _, hook := range hooks {
		if hook == nil {
			return fmt.Errorf("state: %s hook for %v: hook must not be nil", action, value)
		}
		if err := hook(ctx); err != nil {
			return fmt.Errorf("state: %s hook for %v: %w", action, value, err)
		}
	}
	return nil
}

// SystemName is the registered name for the state system.
const SystemName = "state"

// NewSystem returns a system that runs initial enter hooks once, then applies
// pending transitions on later updates. Register the same system in
// [core.PhasePreStartup] and [core.PhasePreFrame] so initial hooks precede
// startup scene operations and later transitions precede frame scene
// operations. Requests made by hooks are deferred until the next update.
//
// Hook errors are returned from Update. An exit-hook error prevents the
// transition; an enter-hook error is reported after the transition. Hook side
// effects are not rolled back. Failed transitions are consumed, and failed
// hooks are not retried automatically. Requests queued by hooks remain pending
// even if a hook fails. A nil machine returns an error from Update.
func NewSystem[S comparable](machine *Machine[S]) core.System {
	return core.SystemFunc(func(ctx *core.Context) error {
		if machine == nil {
			return errors.New("state: machine must not be nil")
		}
		return machine.apply(ctx)
	})
}
