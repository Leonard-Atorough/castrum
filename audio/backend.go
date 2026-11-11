package audio

import (
	"fmt"
	"reflect"
)

// SystemName is the name of the game's audio reconciliation system.
const SystemName = "engine.audio"

// Backend holds the controller used by the game's audio system.
// Its zero value has no controller and is suitable for headless games.
type Backend struct {
	controller Controller
}

// Attach installs a controller. It returns an error if controller is nil
// or a controller is already attached. An attachment during the frame
// phase takes effect on the next frame pass.
//
// Attach must not run concurrently with game updates.
func (b *Backend) Attach(controller Controller) error {
	if controller == nil {
		return fmt.Errorf("audio: controller must not be nil")
	}
	value := reflect.ValueOf(controller)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return fmt.Errorf("audio: controller must not be nil")
		}
	}
	if b.controller != nil {
		return fmt.Errorf("audio: controller already attached")
	}
	b.controller = controller
	return nil
}

// Controller returns the attached controller, or nil if none is attached.
func (b *Backend) Controller() Controller {
	return b.controller
}
