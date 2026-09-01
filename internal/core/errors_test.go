package core

import (
	"errors"
	"testing"
)

// Each wrapper type below must satisfy errors.Is/errors.As against its
// sentinel via Unwrap - that's the part copy-paste-derived error types
// tend to get wrong.
func TestErrorWrappers_Unwrap(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"WorldError", &WorldError{Op: "Create", Err: ErrInvalidEntity}, ErrInvalidEntity},
		{"EntityError", &EntityError{EntityID: 1, Op: "GetComponent", Err: ErrEntityNotFound}, ErrEntityNotFound},
		{"IndexError", &IndexError{EntityID: 1, IndexKey: "player", Op: "AddTag", Err: ErrTagNotFound}, ErrTagNotFound},
		{"HierarchyError", &HierarchyError{ParentID: 1, ChildID: 2, Op: "Add", Err: ErrEntityNotFound}, ErrEntityNotFound},
		{"SystemError", &SystemError{Name: "render", Op: "Update", Err: ErrSystemNotFound}, ErrSystemNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.err, tc.want) {
				t.Fatalf("errors.Is failed to find %v through %T", tc.want, tc.err)
			}
			if tc.err.Error() == "" {
				t.Fatal("Error() should produce a non-empty message")
			}
		})
	}
}
