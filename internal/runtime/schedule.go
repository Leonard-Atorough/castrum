// Package runtime provides ordered execution of systems for a core phase.
package runtime

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum/core"
)

// Entry associates a system with its registration name and group.
type Entry struct {
	// Name identifies the registration in execution errors.
	Name string
	// System runs when the entry is enabled.
	System core.System
	// Group determines the execution group for the system.
	Group string
}

// Schedule runs registered systems in order for one [core.Phase].
type Schedule struct {
	opening   []Entry
	first     []Entry
	systems   []Entry
	last      []Entry
	closing   []Entry
	usedNames map[string]bool
	groups    map[string]bool
	phase     core.Phase
}

// SetOpening registers the outermost opening entry.
func (s *Schedule) SetOpening(entry Entry) error {
	if len(s.opening) != 0 {
		return fmt.Errorf("opening system already registered for %s", s.phase)
	}
	if err := s.register(entry); err != nil {
		return err
	}
	s.opening = []Entry{entry}
	return nil
}

// SetClosing registers the outermost closing entry.
func (s *Schedule) SetClosing(entry Entry) error {
	if len(s.closing) != 0 {
		return fmt.Errorf("closing system already registered for %s", s.phase)
	}
	if err := s.register(entry); err != nil {
		return err
	}
	s.closing = []Entry{entry}
	return nil
}

// NewSchedule returns an empty schedule sharing group state with other schedules.
func NewSchedule(phase core.Phase, groups map[string]bool) *Schedule {
	return &Schedule{
		usedNames: make(map[string]bool),
		groups:    groups,
		phase:     phase,
	}
}

// Add appends an entry to the schedule.
func (s *Schedule) Add(entry Entry) error {
	if err := s.register(entry); err != nil {
		return err
	}
	s.systems = append(s.systems, entry)
	return nil
}

// AddFirst appends an entry to the first boundary.
func (s *Schedule) AddFirst(entry Entry) error {
	if err := s.register(entry); err != nil {
		return err
	}
	s.first = append(s.first, entry)
	return nil
}

// AddLast appends an entry to the last boundary.
func (s *Schedule) AddLast(entry Entry) error {
	if err := s.register(entry); err != nil {
		return err
	}
	s.last = append(s.last, entry)
	return nil
}

// AddBefore inserts an entry immediately before the named regular system.
func (s *Schedule) AddBefore(target string, entry Entry) error {
	return s.addRelative(target, entry, false)
}

// AddAfter inserts an entry immediately after the named regular system.
func (s *Schedule) AddAfter(target string, entry Entry) error {
	return s.addRelative(target, entry, true)
}

func (s *Schedule) addRelative(target string, entry Entry, after bool) error {
	index := -1
	for i, existing := range s.systems {
		if existing.Name == target {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("target system %q not found in regular systems for %s", target, s.phase)
	}
	if err := s.register(entry); err != nil {
		return err
	}
	if after {
		index++
	}
	// A running pass retains its original slice when a system inserts work.
	updated := make([]Entry, len(s.systems)+1)
	copy(updated, s.systems[:index])
	updated[index] = entry
	copy(updated[index+1:], s.systems[index:])
	s.systems = updated
	return nil
}

// Run executes each system in order. Systems added during a run take effect on
// the next run. It stops at the first error and returns it with the phase and name.
func (s *Schedule) Run(ctx *core.Context) error {
	for _, entries := range [][]Entry{s.opening, s.first, s.systems, s.last, s.closing} {
		for _, e := range entries {
			if e.Group != "" && !s.groups[e.Group] {
				continue
			}
			if err := e.System.Update(ctx); err != nil {
				return fmt.Errorf("%s system %q: %w", s.phase, e.Name, err)
			}
		}
	}
	return nil
}

func (s *Schedule) register(entry Entry) error {
	if entry.Name == "" {
		return fmt.Errorf("system name must not be empty")
	}
	if s.usedNames[entry.Name] {
		return fmt.Errorf("system with name %q already registered", entry.Name)
	}
	s.usedNames[entry.Name] = true
	if entry.Group != "" {
		if _, ok := s.groups[entry.Group]; !ok {
			s.groups[entry.Group] = true
		}
	}
	return nil
}
