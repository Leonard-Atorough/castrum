package audio

import "testing"

func TestBackendAttach(t *testing.T) {
	var backend Backend
	if backend.Controller() != nil {
		t.Fatal("new backend has a controller")
	}
	if err := backend.Attach(nil); err == nil {
		t.Fatal("accepted nil controller")
	}
	var typedNil *fakeController
	if err := backend.Attach(typedNil); err == nil {
		t.Fatal("accepted typed nil controller")
	}
	controller := &fakeController{}
	if err := backend.Attach(controller); err != nil {
		t.Fatal(err)
	}
	if backend.Controller() != controller {
		t.Fatal("did not retain the attached controller")
	}
	if err := backend.Attach(&fakeController{}); err == nil {
		t.Fatal("accepted a second controller")
	}
	if backend.Controller() != controller {
		t.Fatal("failed attachment replaced the controller")
	}
}
