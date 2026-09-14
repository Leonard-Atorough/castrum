package atlas

import (
	"fmt"
	"maps"
	"strings"

	"github.com/leonard-atorough/castrum/assets"
	"github.com/leonard-atorough/castrum/geom"
)

type AtlasRegion struct {
	Name       string
	X, Y, W, H int
}

func (a *AtlasRegion) Bounds() geom.Rect {
	return geom.Rect{
		Min: geom.Vector2{X: float64(a.X), Y: float64(a.Y)},
		Max: geom.Vector2{X: float64(a.X + a.W), Y: float64(a.Y + a.H)},
	}
}

type ID string

type Atlas struct {
	id         ID
	assetID    string
	texW, texH int
	regions    map[string]AtlasRegion
}

func NewTextureAtlas(id, assetID string, texW, texH int, regions map[string]AtlasRegion) *Atlas {
	return &Atlas{
		id:      ID(id),
		assetID: assetID,
		texW:    texW,
		texH:    texH,
		regions: regions,
	}
}

func (a *Atlas) ID() ID {
	return a.id
}

func (a *Atlas) AssetID() string {
	return a.assetID
}

func (a *Atlas) Dimensions() (int, int) {
	return a.texW, a.texH
}

func (a *Atlas) Region(name string) (AtlasRegion, bool) {
	region, exists := a.regions[name]
	return region, exists
}

func (a *Atlas) Regions() map[string]AtlasRegion {
	copied := make(map[string]AtlasRegion, len(a.regions))
	maps.Copy(copied, a.regions)
	return copied
}

type AtlasStorer interface {
	Set(string, string, any) error
}

type Builder struct {
	regions          map[string]AtlasRegion
	texW, texH       int
	atlasID, assetID string
	store            AtlasStorer
}

func NewBuilder(id string, assetID string, texW, texH int, store AtlasStorer) (*Builder, error) {
	if texW <= 0 || texH <= 0 {
		return nil, &AtlasError{Message: "invalid texture dimensions"}
	}
	return &Builder{
		regions: make(map[string]AtlasRegion),
		texW:    texW,
		texH:    texH,
		atlasID: id,
		assetID: assetID,
		store:   store,
	}, nil
}

func (b *Builder) SliceRegion(name string, x, y, w, h int) (*Builder, error) {
	if _, exists := b.regions[name]; exists {
		return b, &AtlasError{Message: "region already exists"}
	}

	// validate that the region is within the texture bounds
	if x < 0 || y < 0 || w <= 0 || h <= 0 || x+w > b.texW || y+h > b.texH {
		return b, &AtlasError{Message: "region out of bounds"}
	}

	// validate that the region does not overlap with existing regions using their bounds
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
			// call slice region
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
	// iterate over the regions in the meta and slice them
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
