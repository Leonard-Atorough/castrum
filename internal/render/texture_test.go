package render

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/leonard-atorough/castrum/assets"
)

func TestTextureProviderInvalidationListenerRemovesImage(t *testing.T) {
	loader := assets.NewLoader(nil)

	provider := NewTextureProvider(loader, nil)
	id := assets.ID("texture.png")
	provider.images[id] = &textureResource{image: ebiten.NewImage(1, 1), width: 1, height: 1}

	loader.Invalidate(id)

	provider.mu.RLock()
	_, ok := provider.images[id]
	provider.mu.RUnlock()
	if ok {
		t.Fatal("texture provider image survived loader invalidation")
	}
}
