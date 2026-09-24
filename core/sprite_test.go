package core

import (
	"image/color"
	"strings"
	"testing"
)

func TestSpriteValidate(t *testing.T) {
	valid := Sprite{Opacity: 1, Layer: 31, Tint: color.White}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Sprite rejected: %v", err)
	}
	for name, sprite := range map[string]Sprite{
		"negative opacity": {Opacity: -0.1},
		"opacity over one": {Opacity: 1.1},
		"layer over 31":    {Layer: 32},
	} {
		if err := sprite.Validate(); err == nil {
			t.Errorf("Sprite with %s should fail validation", name)
		}
	}
}

func TestSpriteSourceVariantsValidate(t *testing.T) {
	if err := (AtlasSprite{Atlas: "sprites", Region: "player"}).Validate(); err != nil {
		t.Fatalf("valid AtlasSprite rejected: %v", err)
	}
	if err := (AtlasSprite{}).Validate(); err == nil {
		t.Error("empty AtlasSprite should fail validation")
	}
	if err := (TextureSprite{Texture: "bg/sky.png"}).Validate(); err != nil {
		t.Fatalf("valid TextureSprite rejected: %v", err)
	}
	if err := (TextureSprite{}).Validate(); err == nil {
		t.Error("empty TextureSprite should fail validation")
	}
}

// The Validatable hook fires through the public surface: spawn, add, and
// set all reject invalid components, and the error names the component
// type.
func TestComponentValidationAtSpawn(t *testing.T) {
	w := NewWorld()

	if _, err := w.NewEntity(Sprite{Opacity: 2}); err == nil {
		t.Fatal("spawn with an invalid Sprite should error")
	} else if !strings.Contains(err.Error(), "Sprite") {
		t.Errorf("error %v should name the component type", err)
	}

	entity, err := w.NewEntity(Sprite{})
	if err != nil {
		t.Fatalf("spawn with a valid Sprite: %v", err)
	}
	if err := entity.AddComponent(w, AtlasSprite{}); err == nil {
		t.Fatal("adding an invalid AtlasSprite should error")
	}
	if err := entity.AddComponent(w, AtlasSprite{Atlas: "sprites", Region: "player"}); err != nil {
		t.Fatalf("adding a valid AtlasSprite: %v", err)
	}
	if err := entity.SetComponent(w, Sprite{Layer: 99}); err == nil {
		t.Fatal("setting an invalid Sprite should error")
	}
	if err := entity.SetComponent(w, Sprite{Layer: 3}); err != nil {
		t.Fatalf("setting a valid Sprite: %v", err)
	}
}
