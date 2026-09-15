package atlas

import (
	"fmt"
	"maps"
	"strings"

	"github.com/leonard-atorough/castrum/assets"
)

// Builder constructs an [Atlas] by slicing regions from a texture image.
// A Builder is obtained from [Service.NewBuilder]; there is no standalone
// constructor, because every Builder must be wired to a store that the
// texture provider can read from.
type Builder struct {
	regions          map[string]AtlasRegion
	texW, texH       int
	atlasID, assetID string
	store            *Service
}

// NewBuilder creates a Builder wired to this Service. Build will register
// the atlas with the Service so it is retrievable by the texture provider.
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

// SliceRegion adds a named region at the given pixel coordinates. Returns
// an error if the name is already in use, the region is out of texture
// bounds, or it overlaps an existing region. The Builder is returned on
// error to allow continued chaining, but the failed region is not added.
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

// GridSlice divides the texture into a uniform grid of w x h tiles. The
// nameFunc receives the zero-based tile index (left-to-right, top-to-bottom)
// and returns the region name. Returns all errors collected during the
// slice; the Builder is still returned to allow further chaining.
func (b *Builder) GridSlice(w, h int, prefix string) (*Builder, []error) {
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
			name := fmt.Sprintf("%s_%d", prefix, idx)
			if _, err := b.SliceRegion(name, x*w, y*h, w, h); err != nil {
				errs.Errors = append(errs.Errors, err)
			}
			idx++
		}
	}
	return b, errs.Errors
}

// FromMeta adds regions from atlas metadata. Returns all errors collected
// during the slice; the Builder is still returned to allow further chaining.
func (b *Builder) FromMeta(meta assets.AtlasMeta) (*Builder, []error) {
	var errs AtlasErrorList
	for _, region := range meta.Regions {
		if _, err := b.SliceRegion(region.Name, region.X, region.Y, region.W, region.H); err != nil {
			errs.Errors = append(errs.Errors, err)
		}
	}
	return b, errs.Errors
}

// Build creates the Atlas and registers it with the Service's store. The
// returned Atlas is a copy; modifying it does not affect the store entry.
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
	if err := b.store.Set(b.atlasID, atlas); err != nil {
		return nil, fmt.Errorf("atlas build: %w", err)
	}

	return atlas, nil
}

// AtlasError is the error type returned by Builder methods.
type AtlasError struct {
	Message string
}

func (e *AtlasError) Error() string {
	return e.Message
}

// AtlasErrorList is a collection of errors returned by methods that
// process multiple regions.
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
