package asset

import (
	"image"
	"io"

	_ "image/jpeg" // register JPEG decoding with image.Decode
	_ "image/png"  // register PNG decoding with image.Decode
)

// TextureData is a decoded image asset: the pixels plus their dimensions.
// Its consumers are atlas construction, which validates region bounds
// against the dimensions, and a runner, which converts the image to a GPU
// texture once. Backend-free by the package contract: nothing here holds
// a backend type.
type TextureData struct {
	Image  image.Image
	Width  int
	Height int
}

// decodeTexture decodes an image into TextureData for the png, jpg, and
// jpeg formats. Registered by default; see defaults.go.
func decodeTexture(reader io.Reader) (TextureData, error) {
	decoded, _, err := image.Decode(reader)
	if err != nil {
		return TextureData{}, err
	}
	bounds := decoded.Bounds()
	return TextureData{Image: decoded, Width: bounds.Dx(), Height: bounds.Dy()}, nil
}
