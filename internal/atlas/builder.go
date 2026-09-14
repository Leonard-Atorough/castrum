package atlas

import (
	"fmt"
	"maps"
	"strings"

	"github.com/leonard-atorough/castrum/assets"
)

type Builder struct {
	regions          map[string]AtlasRegion
	texW, texH       int
	atlasID, assetID string
	store            *Service
}

// NewBuilder creates a Builder wired to this Service. Build() will register the
// atlas with the Service so it is retrievable by the texture provider. This is
// the only way to obtain a Builder — the store cannot be nil or substituted.
func (s *Service) NewBuilder(id, assetID string, texW, texH int) (*Builder, error) {
	if texW <= 0 || texH <= 0 {
		return nil, &AtlasError{Message: "invalid texture dimensions"}
	}
	return &Builder{
		regions: make(map[string]AtlasRegion),
		texW:    texW,
		texH:    texH,
		atlasID: id,
		assetID: assetID,
		store:   s,
	}, nil
}

func (b *Builder) SliceRegion(name string, x, y, w, h int) (*Builder, error) {
	if _, exists := b.regions[name]; exists {
		return b, &AtlasError{Message: "region already exists"}
	}

	if x < 0 || y < 0 || w <= 0 || h <= 0 || x+w > b.texW || y+h > b.texH {
		return b, &AtlasError{Message: "region out of bounds"}
	}

	newRegion := AtlasRegion{
		Name: name,
		X:    x,
		Y:    y,
		W:    w,
		H:    h,
	}
	newBounds := newRegion.Bounds()
	for _, existing := range b.regions {
		if existing.Bounds().Intersects(newBounds) {
			return b, &AtlasError{Message: "region overlaps with existing region"}
		}
	}

	b.regions[name] = newRegion
	return b, nil
}

func (b *Builder) GridSlice(w, h int, nameFunc func(idx int) string) (*Builder, []error) {
	var errs AtlasErrorList
	if w <= 0 || h <= 0 {
		return b, []error{&AtlasError{Message: "tile dimensions must be positive"}}
	}
	if b.texW%w != 0 || b.texH%h != 0 {
		return b, []error{&AtlasError{Message: fmt.Sprintf(
			"texture dimensions (%d,%d) not evenly divisible by tile dimensions (%d,%d)",
			b.texW, b.texH, w, h)}}
	}
	rows := b.texH / h
	cols := b.texW / w
	idx := 0
	for y := range rows {
		for x := range cols {
			name := nameFunc(idx)
			if _, err := b.SliceRegion(name, x*w, y*h, w, h); err != nil {
				errs.Errors = append(errs.Errors, err)
			}
			idx++
		}
	}
	return b, errs.Errors
}

func (b *Builder) FromMeta(meta assets.AtlasMeta) (*Builder, error) {
	for _, region := range meta.Regions {
		if _, err := b.SliceRegion(region.Name, region.X, region.Y, region.W, region.H); err != nil {
			return b, err
		}
	}
	return b, nil
}

func (b *Builder) Build() (*Atlas, error) {
	atlasRegions := make(map[string]AtlasRegion)
	maps.Copy(atlasRegions, b.regions)
	atlas := NewTextureAtlas(
		b.atlasID,
		b.assetID,
		b.texW,
		b.texH,
		atlasRegions,
	)
	if err := b.store.Set(b.atlasID, b.assetID, atlas); err != nil {
		return nil, &AtlasError{Message: err.Error()}
	}

	return atlas, nil
}

type AtlasError struct {
	Message string
}

func (e *AtlasError) Error() string {
	return e.Message
}

type AtlasErrorList struct {
	Errors []error
}

func (e *AtlasErrorList) Error() string {
	var msg strings.Builder
	for _, err := range e.Errors {
		msg.WriteString(err.Error())
		msg.WriteString("; ")
	}
	return msg.String()
}
