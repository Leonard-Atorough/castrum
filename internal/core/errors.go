package core

import (
	"errors"
	"fmt"

	"github.com/leonard-atorough/castrum/ecs"
)

// Sentinel errors for common ECS operations.
var (
	// ErrEntityNotFound is returned when an operation references an entity that does not exist.
	ErrEntityNotFound = errors.New("entity not found")
	// ErrInvalidEntity is returned when an entity has invalid state or properties.
	ErrInvalidEntity = errors.New("invalid entity")
)

// WorldError wraps errors that occur at the world level.
type WorldError struct {
	Op  string // operation that failed (e.g., "Spawn", "Destroy")
	Err error  // underlying error
}

func (e *WorldError) Error() string {
	return fmt.Sprintf("world error: %s: %v", e.Op, e.Err)
}

func (e *WorldError) Unwrap() error {
	return e.Err
}

// EntityError wraps errors related to a specific entity.
type EntityError struct {
	EntityID ecs.EntityID // ID of the entity that caused the error
	Op       string       // operation that failed
	Err      error        // underlying error
}

func (e *EntityError) Error() string {
	return fmt.Sprintf("entity %d: %s: %v", e.EntityID, e.Op, e.Err)
}

func (e *EntityError) Unwrap() error {
	return e.Err
}

// IndexError wraps errors from the entity index (components, tags, templates).
type IndexError struct {
	EntityID ecs.EntityID // ID of the entity involved
	IndexKey string   // component type, tag name, or template name
	Op       string   // operation that failed
	Err      error    // underlying error
}

func (e *IndexError) Error() string {
	return fmt.Sprintf("index error: %s for %q on entity %d: %v", e.Op, e.IndexKey, e.EntityID, e.Err)
}

func (e *IndexError) Unwrap() error {
	return e.Err
}

// HierarchyError wraps errors from hierarchy operations.
type HierarchyError struct {
	ParentID ecs.EntityID
	ChildID  ecs.EntityID
	Op       string
	Err      error
}

func (e *HierarchyError) Error() string {
	return fmt.Sprintf("hierarchy error: %s for parent %d and child %d: %v", e.Op, e.ParentID, e.ChildID, e.Err)
}

func (e *HierarchyError) Unwrap() error {
	return e.Err
}
