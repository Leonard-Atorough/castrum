package asset

import "fmt"

func (c *Codecs) registerDefaults() {
	for _, format := range []Format{FormatPNG, FormatJPG, FormatJPEG} {
		mustRegister(c.SetDecoder(format, decodeTexture))
	}
	mustRegister(c.SetDecoder(FormatJSON, decodeAtlasMeta))
	for _, format := range []Format{FormatTTF, FormatOTF} {
		mustRegister(c.SetDecoder(format, decodeFont))
	}
}

func mustRegister(err error) {
	if err != nil {
		panic(fmt.Sprintf("asset: default decoder registration failed: %v", err))
	}
}
