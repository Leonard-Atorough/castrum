package ebitrun

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/Leonard-Atorough/castrum/asset"
)

// FontProvider caches Ebitengine drawing faces for loaded fonts. Text
// measurements come from [asset.FontData.Measure], keeping drawing aligned
// with the engine's measured bounds.
//
// [New] provides FontProvider as a resource for the engine renderer and
// user draw functions.
type FontProvider struct {
	server *asset.Server
	// mu protects both caches.
	mu      sync.RWMutex
	sources map[asset.ID]*text.GoTextFaceSource
	faces   map[faceKey]*text.GoTextFace
}

type faceKey struct {
	font asset.ID
	size float64
}

func newFontProvider(server *asset.Server) *FontProvider {
	return &FontProvider{
		server:  server,
		sources: make(map[asset.ID]*text.GoTextFaceSource),
		faces:   make(map[faceKey]*text.GoTextFace),
	}
}

// Face returns the cached drawing face for font at size, creating it from the
// loaded font asset as needed. Errors identify the font.
func (p *FontProvider) Face(font asset.ID, size float64) (*text.GoTextFace, error) {
	key := faceKey{font: font, size: size}
	p.mu.RLock()
	face, ok := p.faces[key]
	p.mu.RUnlock()
	if ok {
		return face, nil
	}

	source, err := p.source(font)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if existing, ok := p.faces[key]; ok {
		return existing, nil
	}
	face = &text.GoTextFace{Source: source, Size: size}
	p.faces[key] = face
	return face, nil
}

func (p *FontProvider) source(font asset.ID) (*text.GoTextFaceSource, error) {
	p.mu.RLock()
	source, ok := p.sources[font]
	p.mu.RUnlock()
	if ok {
		return source, nil
	}

	data, err := p.server.Load[asset.FontData](string(font))
	if err != nil {
		return nil, fmt.Errorf("font %q: %w", font, err)
	}
	source, err = text.NewGoTextFaceSource(bytes.NewReader(data.Raw))
	if err != nil {
		return nil, fmt.Errorf("font %q: %w", font, err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if existing, ok := p.sources[font]; ok {
		return existing, nil
	}
	p.sources[font] = source
	return source, nil
}
