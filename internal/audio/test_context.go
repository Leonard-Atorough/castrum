package audio

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const testSampleRate = 44100

var (
	testCtxOnce sync.Once
	testCtx     *audio.Context
)

func initTestAudioContext() *audio.Context {
	testCtxOnce.Do(func() {
		testCtx = audio.NewContext(testSampleRate)
	})
	return testCtx
}

func testAudioContext() *audio.Context {
	return initTestAudioContext()
}
