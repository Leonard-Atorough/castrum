package audioassets

import (
	"io"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

func TestEmbeddedRecordings(t *testing.T) {
	for _, name := range []string{Magic, Music} {
		t.Run(name, func(t *testing.T) {
			file, err := Files.Open(name)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if _, ok := file.(io.Seeker); !ok {
				t.Fatal("embedded recording is not seekable")
			}
			var decoded io.Reader
			if name == Magic {
				decoded, err = wav.DecodeWithSampleRate(44100, file)
			} else {
				decoded, err = mp3.DecodeWithSampleRate(44100, file)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.ReadFull(decoded, make([]byte, 4096)); err != nil {
				t.Fatalf("decode stereo PCM: %v", err)
			}
		})
	}
}
