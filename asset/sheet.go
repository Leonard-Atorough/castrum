package asset

import (
	"fmt"
	"image"
	"io"
	"io/fs"
	"maps"
	"path"
	"strings"
	"unicode/utf8"
)

// Sheet describes named regions of one texture. Texture paths are relative
// to the loader's root, not to the document. Its zero value is invalid.
//
// Sheet uses ordinary JSON fields. Its default decoder validates metadata
// before caching, without loading the texture. Use ValidateBounds during
// setup to check every region against the image. Treat cached sheets and
// sheets retained by MemorySheet as read-only, including their region maps.
type Sheet struct {
	// Texture is the slash-separated, root-relative texture asset path.
	Texture ID `json:"texture"`
	// Regions maps region names to pixel rectangles within Texture.
	Regions map[string]SheetRegion `json:"regions"`
}

// SheetRegion describes a rectangle in texture pixel coordinates.
type SheetRegion struct {
	// X is the left coordinate in pixels.
	X int `json:"x"`
	// Y is the top coordinate in pixels.
	Y int `json:"y"`
	// W is the width in pixels.
	W int `json:"w"`
	// H is the height in pixels.
	H int `json:"h"`
}

// NewSheet creates a validated sheet, cleaning texture and copying regions.
// It does not load the image; use [Sheet.ValidateBounds] to check its extent.
func NewSheet(texture ID, regions map[string]SheetRegion) (*Sheet, error) {
	sheet := Sheet{Texture: texture, Regions: regions}
	if err := sheet.Validate(); err != nil {
		return nil, err
	}
	cleaned, err := sheet.validateHeader()
	if err != nil {
		return nil, err
	}
	sheet.Texture = cleaned
	sheet.Regions = maps.Clone(regions)
	return &sheet, nil
}

// Validate checks the texture path and every region without loading pixels.
// It rejects empty or escaping paths, empty region maps, empty or invalid
// UTF-8 names, negative coordinates, non-positive sizes, and overflowing
// rectangles. It does not change the sheet.
func (s Sheet) Validate() error {
	if _, err := s.validateHeader(); err != nil {
		return err
	}
	for name, r := range s.Regions {
		if err := validateSheetRegion(name, r); err != nil {
			return err
		}
	}
	return nil
}

// Region returns a named region's pixel bounds. It rejects missing names and
// invalid geometry without scanning other regions or loading the texture.
func (s Sheet) Region(name string) (image.Rectangle, error) {
	r, ok := s.Regions[name]
	if !ok {
		return image.Rectangle{}, fmt.Errorf("asset: region %q not found for texture %q", name, s.Texture)
	}
	if err := validateSheetRegion(name, r); err != nil {
		return image.Rectangle{}, err
	}
	return sheetRegionRect(r), nil
}

// ValidateBounds checks the sheet's metadata and whether every region fits
// within the texture's actual pixel bounds. Use it during setup to detect
// image mismatches before rendering.
func (s Sheet) ValidateBounds(bounds image.Rectangle) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if bounds.Empty() {
		return fmt.Errorf("asset: sheet texture %q has empty bounds %v", s.Texture, bounds)
	}
	for name, r := range s.Regions {
		if !sheetRegionRect(r).In(bounds) {
			return fmt.Errorf("asset: region %q falls outside texture %q bounds %v", name, s.Texture, bounds)
		}
	}
	return nil
}

// Save validates s and writes it as JSON through saver, regardless of name's
// extension. It does not save the texture, choose storage, or invalidate caches.
// It uses the value-type Sheet encoder supplied by [NewCodecs].
func (s *Sheet) Save(saver *Saver, name string) error {
	if s == nil {
		return fmt.Errorf("asset: cannot save a nil sheet")
	}
	if saver == nil {
		return fmt.Errorf("asset: sheet saver must not be nil")
	}
	if err := s.Validate(); err != nil {
		return err
	}
	return saver.Save(name, *s, WithFormat(FormatJSON))
}

func decodeSheetJSON(reader io.Reader) (Sheet, error) {
	sheet, err := decodeJSON[Sheet](reader)
	if err != nil {
		return Sheet{}, err
	}
	if err := sheet.Validate(); err != nil {
		return Sheet{}, err
	}
	return sheet, nil
}

func (s Sheet) validateHeader() (ID, error) {
	texture, err := cleanSheetPath(s.Texture)
	if err != nil {
		return "", fmt.Errorf("asset: sheet texture: %w", err)
	}
	if len(s.Regions) == 0 {
		return "", fmt.Errorf("asset: sheet texture %q must have at least one region", texture)
	}
	return texture, nil
}

func validateSheetRegion(name string, r SheetRegion) error {
	if name == "" || !utf8.ValidString(name) {
		return fmt.Errorf("asset: sheet region name must be non-empty valid UTF-8")
	}
	maxInt := int(^uint(0) >> 1)
	if r.X < 0 || r.Y < 0 || r.W <= 0 || r.H <= 0 ||
		r.X > maxInt-r.W || r.Y > maxInt-r.H {
		return fmt.Errorf("asset: sheet region %q has invalid or overflowing geometry", name)
	}
	return nil
}

func sheetRegionRect(r SheetRegion) image.Rectangle {
	return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
}

func cleanSheetPath(name ID) (ID, error) {
	cleaned := path.Clean(string(name))
	if name == "" || cleaned == "." || !fs.ValidPath(cleaned) || strings.ContainsAny(string(name), "\\:") {
		return "", fmt.Errorf("invalid root-relative asset path %q", name)
	}
	return ID(cleaned), nil
}

// SheetRef identifies a JSON sheet file or retains a sheet in memory.
// It is comparable and may be shared by sprites and animation clips.
// Its zero value is invalid; use [FileSheet] or [MemorySheet].
type SheetRef struct {
	file  ID
	sheet *Sheet
}

// FileSheet refers to a JSON document at name, relative to the loader's root.
// It cleans the slash-separated path without loading anything. Invalid paths
// are reported by [SheetRef.Validate] or [SheetRef.Resolve].
func FileSheet(name ID) SheetRef {
	if name != "" && !strings.ContainsAny(string(name), "\\:") {
		name = ID(path.Clean(string(name)))
	}
	return SheetRef{file: name}
}

// MemorySheet retains sheet without copying it. The sheet and its region map
// must remain read-only while references are in use. Nil and zero sheets are
// invalid; use [Sheet.Validate] to check all regions before sharing.
func MemorySheet(sheet *Sheet) SheetRef {
	return SheetRef{sheet: sheet}
}

// Validate checks the reference without I/O. For memory sheets it checks the
// texture path and a non-empty region map, not every region. It does not check
// a file's existence, codec registration, or texture bounds.
func (r SheetRef) Validate() error {
	if r.sheet != nil {
		_, err := r.sheet.validateHeader()
		return err
	}
	if _, err := cleanSheetPath(r.file); err != nil {
		return fmt.Errorf("asset: sheet file reference: %w", err)
	}
	return nil
}

// Resolve returns the referenced sheet with a cleaned texture path. File
// references load JSON through loader and follow its cache policy; memory
// references need no loader. Returned values share read-only region maps.
//
// Resolve checks the texture path and a non-empty region map without loading
// pixels or scanning all regions. Use [Sheet.Region] for a selected rectangle
// or [Sheet.ValidateBounds] to check the complete sheet during setup.
func (r SheetRef) Resolve(loader *Loader) (Sheet, error) {
	if err := r.Validate(); err != nil {
		return Sheet{}, err
	}
	var sheet Sheet
	if r.sheet != nil {
		sheet = *r.sheet
	} else {
		if loader == nil {
			return Sheet{}, fmt.Errorf("asset: sheet file %q requires an asset loader", r.file)
		}
		var err error
		sheet, err = loader.Load[Sheet](string(r.file), WithFormat(FormatJSON))
		if err != nil {
			return Sheet{}, fmt.Errorf("asset: sheet file %q: %w", r.file, err)
		}
	}
	texture, err := sheet.validateHeader()
	if err != nil {
		return Sheet{}, fmt.Errorf("asset: sheet file %q: %w", r.file, err)
	}
	sheet.Texture = texture
	return sheet, nil
}
