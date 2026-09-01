package texture

import "testing"

func TestStore_AddAndGetTexture(t *testing.T) {
	s := NewStore()
	tx := &Texture{Path: "square.png", Width: 4, Height: 4}

	s.AddTexture("square", tx)

	got, err := s.GetTexture("square")
	if err != nil {
		t.Fatalf("GetTexture failed: %v", err)
	}
	if got != tx {
		t.Fatal("expected to retrieve the added texture")
	}
}

func TestStore_GetTexture_NotFound(t *testing.T) {
	s := NewStore()
	if _, err := s.GetTexture("missing"); err == nil {
		t.Fatal("expected an error for an unregistered texture name")
	}
}
