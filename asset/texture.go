package asset

import (
	"image"
	"io"

	_ "image/jpeg" // register JPEG decoding with image.Decode
	_ "image/png"  // register PNG decoding with image.Decode
)

// TextureData holds a decoded image and its dimensions. Atlas registration
// uses the dimensions to validate regions; runners convert the image to a
// backend texture.
type TextureData struct {
	// Image is the decoded image.
	Image image.Image
	// Width is the image width in pixels.
	Width int
	// Height is the image height in pixels.
	Height int
}

func decodeTexture(reader io.Reader) (TextureData, error) {
	decoded, _, err := image.Decode(reader)
	if err != nil {
		return TextureData{}, err
	}
	bounds := decoded.Bounds()
	return TextureData{Image: decoded, Width: bounds.Dx(), Height: bounds.Dy()}, nil
}
