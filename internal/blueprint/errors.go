package blueprint

import "fmt"

var (
	// ErrComponentTypeNotRegistered is returned when a component type is not registered in the registry.
	ErrComponentTypeNotRegistered = &BlueprintError{Message: "component type is not registered"}
	// ErrComponentTypeAlreadyRegistered is returned when a component type is already registered in the registry.
	ErrComponentTypeAlreadyRegistered = &BlueprintError{Message: "component type is already registered"}
	// ErrBlueprintNotFound is returned when a blueprint is not found in the manager.
	ErrBlueprintNotFound = &BlueprintError{Message: "blueprint not found"}
)

type BlueprintError struct {
	Message string
	Err     error
}

func (e *BlueprintError) Error() string {
	return fmt.Sprintf("blueprint error: %s: %v", e.Message, e.Err)
}

func (e *BlueprintError) Unwrap() error {
	return e.Err
}
