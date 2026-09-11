package input

import "testing"

func TestActionMap_ResolvesBindings(t *testing.T) {
	moveUp := Action("move_up")
	mapper := NewActionMap(Bindings{
		moveUp: {
			{Key: "arrow_up"},
			{Key: "w"},
		},
	})

	mapper.Update(map[string]KeyState{
		"arrowup": {Held: true, Duration: 0.25},
	}, Modifiers{})

	if !mapper.ActionHeld(moveUp) {
		t.Fatal("expected move_up to be held")
	}
	if mapper.ActionPressed(moveUp) {
		t.Fatal("did not expect move_up to be pressed")
	}
}

func TestActionMap_RequiresModifiers(t *testing.T) {
	zoomIn := Action("zoom_in")
	mapper := NewActionMap(Bindings{
		zoomIn: {{Key: "z", Modifiers: Modifiers{Ctrl: true}}},
	})

	mapper.Update(map[string]KeyState{"z": {Pressed: true, Held: true}}, Modifiers{})
	if mapper.ActionPressed(zoomIn) {
		t.Fatal("expected zoom_in to require Ctrl")
	}

	mapper.Update(map[string]KeyState{"z": {Pressed: true, Held: true}}, Modifiers{Ctrl: true})
	if !mapper.ActionPressed(zoomIn) {
		t.Fatal("expected zoom_in with Ctrl to be pressed")
	}
}

func TestActionMap_ResolvesMultipleBindings(t *testing.T) {
	jump := Action("jump")
	mapper := NewActionMap(Bindings{jump: {{Key: "space"}, {Key: "j"}}})

	mapper.Update(map[string]KeyState{"j": {Released: true}}, Modifiers{})
	if !mapper.ActionReleased(jump) {
		t.Fatal("expected jump to be released through its second binding")
	}
}
