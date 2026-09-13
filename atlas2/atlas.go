package atlas2

import (
	"maps"

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

type Atlas struct {
	id         string
	assetID    string
	texW, texH int
	regions    map[string]AtlasRegion
}

func NewTextureAtlas(id, assetID string, texW, texH int) *Atlas {
	return &Atlas{
		id:      id,
		assetID: assetID,
		texW:    texW,
		texH:    texH,
		regions: make(map[string]AtlasRegion),
	}
}

func (a *Atlas) ID() string {
	return a.id
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

type Builder struct {
	atlas       *Atlas
	regions     map[string]AtlasRegion
	texW, texH  int
	id, assetID string
}

func NewBuilder(id string, assetID string, texW, texH int) *Builder {
	atlas := NewTextureAtlas(id, assetID, texW, texH)
	return &Builder{
		atlas:   atlas,
		regions: make(map[string]AtlasRegion),
		texW:    texW,
		texH:    texH,
		id:      id,
		assetID: assetID,
	}
}

func (b *Builder) SliceRegion(name string, x, y, w, h int) (*Builder, error) {
	if _, exists := b.regions[name]; exists {
		return &Builder{
			atlas:   b.atlas,
			regions: b.regions,
			texW:    b.texW,
			texH:    b.texH,
			id:      b.id,
			assetID: b.assetID,
		}, &AtlasError{Message: "region already exists"}
	}

	// validate that the region is within the texture bounds
	if x < 0 || y < 0 || w <= 0 || h <= 0 || x+w > b.texW || y+h > b.texH {
		return &Builder{
			atlas:   b.atlas,
			regions: b.regions,
			texW:    b.texW,
			texH:    b.texH,
			id:      b.id,
			assetID: b.assetID,
		}, &AtlasError{Message: "region out of bounds"}
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
			return &Builder{
				atlas:   b.atlas,
				regions: b.regions,
				texW:    b.texW,
				texH:    b.texH,
				id:      b.id,
				assetID: b.assetID,
			}, &AtlasError{Message: "region overlaps with existing region"}
		}
	}

	b.regions[name] = newRegion
	return &Builder{
		atlas:   b.atlas,
		regions: b.regions,
		texW:    b.texW,
		texH:    b.texH,
		id:      b.id,
		assetID: b.assetID,
	}, nil
}

func (b *Builder) GridSlice(w, h int, nameFunc func(idx int) string) (*Builder, []error) {
	var errs AtlasErrorList
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
	return &Builder{
		atlas:   b.atlas,
		regions: b.regions,
		texW:    b.texW,
		texH:    b.texH,
		id:      b.id,
		assetID: b.assetID,
	}, errs.Errors
}

func (b *Builder) FromMeta(meta assets.AtlasMeta) (*Builder, error) {
	// iterate over the regions in the meta and slice them
	for _, region := range meta.Regions {
		if _, err := b.SliceRegion(region.Name, region.X, region.Y, region.W, region.H); err != nil {
			return &Builder{
				atlas:   b.atlas,
				regions: b.regions,
				texW:    b.texW,
				texH:    b.texH,
				id:      b.id,
				assetID: b.assetID,
			}, err
		}
	}
	return &Builder{
		atlas:   b.atlas,
		regions: b.regions,
		texW:    b.texW,
		texH:    b.texH,
		id:      b.id,
		assetID: b.assetID,
	}, nil
}

func (b *Builder) Build() (*Atlas, error) {
	return &Atlas{
		id:      b.id,
		assetID: b.assetID,
		texW:    b.texW,
		texH:    b.texH,
		regions: b.regions,
	}, nil
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
	var msg string
	for _, err := range e.Errors {
		msg += err.Error() + "; "
	}
	return msg
}
