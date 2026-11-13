// Package spritesheet builds asset.Sheet region maps from texture bounds.
// It describes slicing without loading, saving, or modifying image pixels.
package spritesheet

import (
	"fmt"
	"image"

	"github.com/Leonard-Atorough/castrum/asset"
)

// Grid creates a sheet of equally sized regions within bounds, named prefix_0,
// prefix_1, and so on in row-major order. Bounds must be non-empty and
// non-negative, tile dimensions positive and evenly divide bounds, and prefix
// non-empty. Texture is a root-relative asset path; Grid performs no I/O.
func Grid(texture asset.ID, bounds image.Rectangle, tileWidth, tileHeight int, prefix string) (*asset.Sheet, error) {
	if bounds.Empty() || bounds.Min.X < 0 || bounds.Min.Y < 0 {
		return nil, fmt.Errorf("spritesheet: invalid grid bounds %v", bounds)
	}
	if tileWidth <= 0 || tileHeight <= 0 || prefix == "" {
		return nil, fmt.Errorf("spritesheet: grid requires positive tile dimensions and a non-empty prefix")
	}
	if bounds.Dx()%tileWidth != 0 || bounds.Dy()%tileHeight != 0 {
		return nil, fmt.Errorf("spritesheet: tiles %dx%d do not evenly divide bounds %v", tileWidth, tileHeight, bounds)
	}
	rows, cols := bounds.Dy()/tileHeight, bounds.Dx()/tileWidth
	if rows > int(^uint(0)>>1)/cols {
		return nil, fmt.Errorf("spritesheet: grid region count overflows int")
	}
	regions := make(map[string]asset.SheetRegion, rows*cols)
	for y := range rows {
		for x := range cols {
			regions[fmt.Sprintf("%s_%d", prefix, y*cols+x)] = asset.SheetRegion{
				X: bounds.Min.X + x*tileWidth, Y: bounds.Min.Y + y*tileHeight,
				W: tileWidth, H: tileHeight,
			}
		}
	}
	return asset.NewSheet(texture, regions)
}
