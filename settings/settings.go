// Package settings provides typed groups of player preferences.
// Groups hold desired values; applying and persisting them is separate.
package settings

import (
	"fmt"
	"reflect"
)

// Group holds preferences validated by the policy passed to
// [New]. Copies of a group share its state. Groups are not safe for concurrent
// use.
type Group[T any] struct {
	state *groupState[T]
}

type groupState[T any] struct {
	value       T
	policy      Policy[T]
	updating    bool
	policyDepth int
}

// Validator checks a proposed preference value without mutating it.
type Validator[T any] interface {
	Validate(value T) error
}

// ValidatorFunc adapts a non-nil function to [Validator].
type ValidatorFunc[T any] func(T) error

// Validate calls f with value.
func (f ValidatorFunc[T]) Validate(value T) error {
	return f(value)
}

// Cloner copies all mutable data reachable from a preference value without
// mutating its argument.
type Cloner[T any] interface {
	Clone(value T) T
}

// ClonerFunc adapts a non-nil function to [Cloner].
type ClonerFunc[T any] func(T) T

// Clone calls f with value.
func (f ClonerFunc[T]) Clone(value T) T {
	return f(value)
}

// Policy defines optional validation and copying for a group. Policy methods
// must not replace the group. Set and Update reject mutations of the same
// group while a policy method is running. Other side effects are not rolled
// back.
type Policy[T any] struct {
	// Validator checks initial and replacement values. Nil accepts all values.
	Validator Validator[T]
	// Cloner copies values at the group boundary. Nil uses an ordinary copy,
	// which is safe only when values contain no mutable references.
	Cloner Cloner[T]
}

// New constructs a group after validating initial with policy.Validator, if
// non-nil. It returns an error for a typed nil policy or invalid initial value.
func New[T any](initial T, policy Policy[T]) (*Group[T], error) {
	if isTypedNil(policy.Validator) || isTypedNil(policy.Cloner) {
		return nil, fmt.Errorf("settings: policy must not contain typed nil implementations")
	}
	s := &groupState[T]{policy: policy}
	value := s.copy(initial)
	if err := s.check(value); err != nil {
		return nil, err
	}
	s.value = value
	return &Group[T]{state: s}, nil
}

func isTypedNil(value any) bool {
	if value == nil {
		return false
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func (s *groupState[T]) copy(value T) T {
	if s.policy.Cloner != nil {
		s.policyDepth++
		defer func() { s.policyDepth-- }()
		return s.policy.Cloner.Clone(value)
	}
	return value
}

func (s *groupState[T]) check(value T) error {
	if s.policy.Validator != nil {
		s.policyDepth++
		defer func() { s.policyDepth-- }()
		return s.policy.Validator.Validate(value)
	}
	return nil
}

// Value returns a copy of the preferences. It returns an error if the group
// was not constructed with [New] or is replaced by its copy policy.
func (g *Group[T]) Value() (T, error) {
	if g == nil || g.state == nil {
		var zero T
		return zero, fmt.Errorf("settings: group is not initialized")
	}
	state := g.state
	value := state.copy(state.value)
	if g.state != state {
		var zero T
		return zero, fmt.Errorf("settings: group was replaced during its copy policy")
	}
	return value, nil
}

// Set validates and replaces the preferences. Invalid values leave the group
// unchanged. Set returns an error for an uninitialized group, a nested mutation,
// or a group replaced by its policy.
func (g *Group[T]) Set(value T) error {
	if g == nil || g.state == nil {
		return fmt.Errorf("settings: group is not initialized")
	}
	state := g.state
	if state.updating {
		return fmt.Errorf("settings: cannot set a group during its update callback")
	}
	if state.policyDepth != 0 {
		return fmt.Errorf("settings: cannot set a group during its policy callback")
	}
	value = state.copy(value)
	if g.state != state {
		return fmt.Errorf("settings: group was replaced during its copy policy")
	}
	err := state.check(value)
	if g.state != state {
		return fmt.Errorf("settings: group was replaced during its validation policy")
	}
	if err != nil {
		return err
	}
	state.value = value
	return nil
}

// Update applies f to a copy of the current value and sets its result.
// It returns an error if f is nil, the group is uninitialized, or validation
// fails. The callback must not replace the group or mutate it through Set or
// Update; nested mutations return an error. Update also returns an error if
// called during a policy method. Callback side effects are not rolled back.
func (g *Group[T]) Update(f func(T) T) error {
	if f == nil {
		return fmt.Errorf("settings: update callback must not be nil")
	}
	if g == nil || g.state == nil {
		return fmt.Errorf("settings: group is not initialized")
	}
	state := g.state
	if state.updating {
		return fmt.Errorf("settings: cannot update a group during its update callback")
	}
	if state.policyDepth != 0 {
		return fmt.Errorf("settings: cannot update a group during its policy callback")
	}
	v, err := g.Value()
	if err != nil {
		return err
	}
	result := func() T {
		state.updating = true
		defer func() { state.updating = false }()
		return f(v)
	}()
	if g.state != state {
		return fmt.Errorf("settings: group was replaced during its update callback")
	}
	return g.Set(result)
}
