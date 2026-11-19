package ebitrun

import (
	"fmt"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
)

// NewAssetCodecs returns Castrum's defaults with JSON codecs for
// [DisplayPreferences]. Use it for preference loaders and savers before
// constructing a runner.
//
// Each call returns independent codecs. Custom registrations are visible to
// facades built from these codecs, including ones built earlier. Audio data
// decoders are installed on these codecs by [New] at the effective audio
// sample rate, for each format that has no decoder yet; a decoder set before
// [New] is kept.
func NewAssetCodecs() *asset.Codecs {
	codecs := castrum.NewAssetCodecs()
	if err := codecs.SetJSON[DisplayPreferences](); err != nil {
		panic(fmt.Sprintf("castrum/ebiten: register display preferences: %v", err))
	}
	return codecs
}
