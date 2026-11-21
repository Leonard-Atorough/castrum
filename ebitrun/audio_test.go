package ebitrun

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
	"github.com/Leonard-Atorough/castrum/signal"
	ebitaudio "github.com/hajimehoshi/ebiten/v2/audio"
)

func TestNewInstallsAudioCodecsOnSharedCodecs(t *testing.T) {
	codecs := asset.NewCodecs()
	game := castrum.New(castrum.WithAssetCodecs(codecs))
	mustRunner(t, game)
	for _, format := range []asset.Format{asset.FormatMP3, asset.FormatWAV, asset.FormatOGG} {
		if !codecs.HasDecoder[asset.AudioData](format) {
			t.Errorf("runner did not install a decoder for %q on the shared codecs", format)
		}
	}
	loader := game.World().MustResource[*asset.Loader]()
	data, err := loader.LoadReader[asset.AudioData](bytes.NewReader(testWAV()), asset.FormatWAV)
	if err != nil {
		t.Fatal(err)
	}
	if data.SampleRate != ebitaudio.CurrentContext().SampleRate() {
		t.Fatalf("decoded rate = %d, want %d", data.SampleRate, ebitaudio.CurrentContext().SampleRate())
	}
}

func TestNewKeepsUserAudioDecoder(t *testing.T) {
	codecs := asset.NewCodecs()
	if err := codecs.SetDecoder(asset.FormatWAV, func(io.Reader) (asset.AudioData, error) {
		return asset.AudioData{SampleRate: 123}, nil
	}); err != nil {
		t.Fatal(err)
	}
	game := castrum.New(castrum.WithAssetCodecs(codecs))
	mustRunner(t, game)
	loader := game.World().MustResource[*asset.Loader]()
	value, err := loader.LoadReader[asset.AudioData](bytes.NewReader(nil), asset.FormatWAV)
	if err != nil || value.SampleRate != 123 {
		t.Fatalf("user decoder was replaced: %+v, %v", value, err)
	}
}

func testWAV() []byte {
	sampleRate := 44100
	if ctx := ebitaudio.CurrentContext(); ctx != nil {
		sampleRate = ctx.SampleRate()
	}
	pcm := make([]byte, 64*4)
	for i := range pcm {
		pcm[i] = byte(i)
	}
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+len(pcm)))
	buf.WriteString("WAVEfmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(2))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate*4))
	binary.Write(&buf, binary.LittleEndian, uint16(4))
	binary.Write(&buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(len(pcm)))
	buf.Write(pcm)
	return buf.Bytes()
}

type closeTrackingFS struct {
	fsys     fs.FS
	opens    atomic.Int64
	closes   atomic.Int64
	closeErr error
}

func (c *closeTrackingFS) Open(name string) (fs.File, error) {
	f, err := c.fsys.Open(name)
	if err != nil {
		return nil, err
	}
	c.opens.Add(1)
	return &trackingFile{File: f, owner: c, closeErr: c.closeErr}, nil
}

type trackingFile struct {
	fs.File
	owner    *closeTrackingFS
	closeErr error
}

func (f *trackingFile) Close() error {
	f.owner.closes.Add(1)
	return errors.Join(f.File.Close(), f.closeErr)
}

func (f *trackingFile) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := f.File.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("tracking file %T is not seekable", f.File)
	}
	return seeker.Seek(offset, whence)
}

type audioProviderHarness struct {
	provider *audioProvider
	files    *closeTrackingFS
	dir      string
}

func newAudioProviderHarness(t *testing.T) *audioProviderHarness {
	t.Helper()
	ctx := ebitaudio.CurrentContext()
	if ctx == nil {
		ctx = ebitaudio.NewContext(44100)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tone.wav"), testWAV(), 0o644); err != nil {
		t.Fatal(err)
	}
	files := &closeTrackingFS{fsys: os.DirFS(dir)}
	codecs := asset.NewCodecs()
	loader, err := asset.NewLoader(files, asset.LoaderOptions{Codecs: codecs})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := newAudioProvider(ctx, loader, codecs)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		clear(provider.touched)
		if err := provider.Sweep(); err != nil {
			t.Errorf("cleanup audio sessions: %v", err)
		}
	})
	return &audioProviderHarness{provider: provider, files: files, dir: dir}
}

func TestAudioLoadingAndCaching(t *testing.T) {
	h := newAudioProviderHarness(t)
	want := audio.Playback{Volume: 1, Rate: 1, Restart: true}
	for _, id := range []core.EntityID{1, 2} {
		if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || !live {
			t.Fatalf("eager sync %d: live=%v, err=%v", id, live, err)
		}
		if h.provider.players[id].source != nil {
			t.Fatal("eager player retained an open file")
		}
	}
	if h.files.opens.Load() != 1 || h.files.closes.Load() != 1 {
		t.Fatalf("eager opens=%d closes=%d, want one cached decode", h.files.opens.Load(), h.files.closes.Load())
	}
	want.Load = audio.LoadStream
	if live, err := h.provider.Sync(3, "tone.wav", want); err != nil || !live {
		t.Fatalf("stream sync: live=%v, err=%v", live, err)
	}
	if h.provider.players[3].source == nil || h.files.opens.Load() != 2 || h.files.closes.Load() != 1 {
		t.Fatal("stream did not retain its own open file")
	}
	if err := h.provider.Sweep(); err != nil {
		t.Fatal(err)
	}
	if err := h.provider.Sweep(); err != nil {
		t.Fatal(err)
	}
	if h.files.opens.Load() != h.files.closes.Load() {
		t.Fatalf("stream not closed: opens=%d closes=%d", h.files.opens.Load(), h.files.closes.Load())
	}
}

func TestAudioLoadFailuresCloseStreams(t *testing.T) {
	h := newAudioProviderHarness(t)
	if err := os.WriteFile(filepath.Join(h.dir, "bad.wav"), []byte("not a wav"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.dir, "unknown.bin"), []byte("unknown"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		id   asset.ID
		want audio.Playback
	}{
		{"missing eager", "missing.wav", audio.Playback{Volume: 1, Rate: 1, Restart: true}},
		{"invalid stream", "bad.wav", audio.Playback{Volume: 1, Rate: 1, Restart: true, Load: audio.LoadStream}},
		{"unknown format", "unknown.bin", audio.Playback{Volume: 1, Rate: 1, Restart: true, Load: audio.LoadStream}},
		{"invalid balance", "tone.wav", audio.Playback{Volume: 1, Rate: 1, Restart: true, Load: audio.LoadStream, Balance: math.NaN()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := h.provider.Sync(1, tc.id, tc.want); err == nil {
				t.Fatal("invalid source or playback was accepted")
			}
			if h.files.opens.Load() != h.files.closes.Load() {
				t.Fatalf("failed creation leaked a file: opens=%d closes=%d", h.files.opens.Load(), h.files.closes.Load())
			}
			if h.provider.players[1] != nil {
				t.Fatal("failed creation retained a session")
			}
		})
	}
}

func TestAudioTrackLoopAndBalance(t *testing.T) {
	h := newAudioProviderHarness(t)
	want := audio.Playback{Volume: 0.25, Rate: 1, Restart: true, Loop: audio.LoopTrack, Balance: -1}
	if live, err := h.provider.Sync(1, "tone.wav", want); err != nil || !live {
		t.Fatalf("sync: live=%v, err=%v", live, err)
	}
	session := h.provider.players[1]
	pcm := testWAV()[44:]
	got := make([]byte, len(pcm)*2)
	if _, err := io.ReadFull(session.balance, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got[:len(pcm)], got[len(pcm):]) {
		t.Fatal("track loop did not repeat the decoded samples")
	}
	if !bytes.Equal(got[:2], pcm[:2]) || got[2] != 0 || got[3] != 0 {
		t.Fatalf("left balance did not preserve left and mute right: %v", got[:4])
	}
	data, err := h.provider.loader.Load[asset.AudioData]("tone.wav")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data.PCM, pcm) || session.player.Volume() != 0.25 {
		t.Fatal("balance modified cached PCM or lost volume")
	}
}

func TestAudioSyncLifecycle(t *testing.T) {
	h := newAudioProviderHarness(t)
	id := core.EntityID(1)
	want := audio.Playback{Volume: 1, Rate: 1}
	if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || live {
		t.Fatalf("absent paused play: live=%v, err=%v", live, err)
	}
	want.Restart, want.Loop = true, audio.LoopTrack
	if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || !live {
		t.Fatalf("paused start: live=%v, err=%v", live, err)
	}
	session := h.provider.players[id]
	if !session.held {
		t.Fatal("paused start did not hold playback")
	}
	want.Restart, want.Playing = false, true
	if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || !live || session.held {
		t.Fatalf("resume: live=%v, held=%v, err=%v", live, session.held, err)
	}
	want.Playing = false
	if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || !live || !session.held {
		t.Fatalf("pause: live=%v, held=%v, err=%v", live, session.held, err)
	}
	want.Loop = audio.LoopNone
	if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || !live {
		t.Fatalf("switch to non-looping: live=%v, err=%v", live, err)
	}
	session = h.provider.players[id]
	want.Playing = true
	if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || !live {
		t.Fatalf("start non-looping: live=%v, err=%v", live, err)
	}
	session.player.PauseAndStopReading()
	if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || live {
		t.Fatalf("completion: live=%v, err=%v", live, err)
	}
	if !session.finished || session.player != nil || session.balance != nil {
		t.Fatalf("completed session retained resources: %+v", session)
	}
	if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || live || h.provider.players[id] != session {
		t.Fatalf("completed session replayed: live=%v, err=%v", live, err)
	}
	want.Restart, want.Playing = true, false
	if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || !live || h.provider.players[id] == session {
		t.Fatalf("restart: live=%v, err=%v", live, err)
	}
}

func TestAudioLiveUpdatesAndRebuilds(t *testing.T) {
	h := newAudioProviderHarness(t)
	id := core.EntityID(1)
	want := audio.Playback{Volume: 1, Rate: 1, Restart: true, Load: audio.LoadStream, Loop: audio.LoopTrack}
	if _, err := h.provider.Sync(id, "tone.wav", want); err != nil {
		t.Fatal(err)
	}
	session := h.provider.players[id]
	want.Restart, want.Volume, want.Balance = false, 0.25, 0.5
	for _, playing := range []bool{false, true, false} {
		want.Playing = playing
		if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || !live {
			t.Fatalf("live update: live=%v, err=%v", live, err)
		}
		if h.provider.players[id] != session || session.held == playing || session.player.Volume() != 0.25 {
			t.Fatalf("live update replaced or misconfigured session: %+v", session)
		}
		balance := session.balance.(*seekableBalanceReader)
		if got := math.Float64frombits(balance.balance.Load()); got != 0.5 {
			t.Fatalf("balance = %v, want 0.5", got)
		}
	}
	for _, change := range []string{"restart", "loop", "load"} {
		old := session
		switch change {
		case "restart":
			want.Restart = true
		case "loop":
			want.Restart = false
			want.Loop = audio.LoopNone
		case "load":
			want.Load = audio.LoadEager
		}
		if live, err := h.provider.Sync(id, "tone.wav", want); err != nil || !live {
			t.Fatalf("%s rebuild: live=%v, err=%v", change, live, err)
		}
		session = h.provider.players[id]
		if session == old || !session.held || session.player.Volume() != 0.25 {
			t.Fatalf("%s did not rebuild the held session", change)
		}
		balance := session.balance.(*seekableBalanceReader)
		if got := math.Float64frombits(balance.balance.Load()); got != 0.5 {
			t.Fatalf("%s lost balance: %v", change, got)
		}
		if old.source != nil {
			t.Fatalf("%s retained the old streaming source", change)
		}
	}
}

func TestAudioSyncRejectsUnsupportedRates(t *testing.T) {
	h := newAudioProviderHarness(t)
	id := core.EntityID(1)
	want := audio.Playback{Volume: 1, Rate: 2, Restart: true, Load: audio.LoadStream}
	if _, err := h.provider.Sync(id, "tone.wav", want); err == nil || !strings.Contains(err.Error(), "tone.wav") {
		t.Fatalf("unsupported initial rate = %v", err)
	}
	want.Rate = 1
	if _, err := h.provider.Sync(id, "tone.wav", want); err != nil {
		t.Fatal(err)
	}
	old := h.provider.players[id]
	closes := h.files.closes.Load()
	want.Rate, want.Loop = math.NaN(), audio.LoopTrack
	if _, err := h.provider.Sync(id, "tone.wav", want); err == nil || !strings.Contains(err.Error(), "playback rate") {
		t.Fatalf("unsupported update rate = %v", err)
	}
	if h.provider.players[id] != old || old.source == nil || h.files.closes.Load() != closes {
		t.Fatal("unsupported rate changed the existing session")
	}
}

func TestAudioSweepReleasesUntouchedSessions(t *testing.T) {
	h := newAudioProviderHarness(t)
	want := audio.Playback{Volume: 1, Rate: 1, Restart: true, Load: audio.LoadStream}
	for _, id := range []core.EntityID{1, 2} {
		if _, err := h.provider.Sync(id, "tone.wav", want); err != nil {
			t.Fatal(err)
		}
	}
	first := h.provider.players[1]
	if err := h.provider.Sweep(); err != nil {
		t.Fatal(err)
	}
	want.Restart = false
	if _, err := h.provider.Sync(1, "tone.wav", want); err != nil {
		t.Fatal(err)
	}
	if err := h.provider.Sweep(); err != nil {
		t.Fatal(err)
	}
	if h.provider.players[1] != first || h.provider.players[2] != nil {
		t.Fatal("sweep did not preserve the touched session and remove the untouched session")
	}
	if h.files.opens.Load() != 2 || h.files.closes.Load() != 1 {
		t.Fatalf("sweep opens=%d closes=%d, want one live stream", h.files.opens.Load(), h.files.closes.Load())
	}
}

func TestAudioCloseErrorsPropagate(t *testing.T) {
	h := newAudioProviderHarness(t)
	closeErr := errors.New("close failed")
	h.files.closeErr = closeErr
	want := audio.Playback{Volume: 1, Rate: 1, Restart: true, Load: audio.LoadStream, Balance: math.NaN()}
	if _, err := h.provider.Sync(1, "tone.wav", want); !errors.Is(err, closeErr) || !strings.Contains(err.Error(), "balance") {
		t.Fatalf("creation error = %v, want balance and close errors", err)
	}
	if h.files.opens.Load() != h.files.closes.Load() {
		t.Fatal("failed creation did not close its stream")
	}
	want.Balance = 0
	if _, err := h.provider.Sync(1, "tone.wav", want); err != nil {
		t.Fatal(err)
	}
	if err := h.provider.Sweep(); err != nil {
		t.Fatal(err)
	}
	if err := h.provider.Sweep(); !errors.Is(err, closeErr) {
		t.Fatalf("sweep error = %v, want close error", err)
	}
	if err := h.provider.Sweep(); err != nil {
		t.Fatalf("retry sweep: %v", err)
	}
	if h.files.opens.Load() != h.files.closes.Load() {
		t.Fatal("failed close was attempted more than once")
	}
}

func TestRunnerAudioEntityRemoval(t *testing.T) {
	ctx := ebitaudio.CurrentContext()
	if ctx == nil {
		ebitaudio.NewContext(44100)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tone.wav"), testWAV(), 0o644); err != nil {
		t.Fatal(err)
	}
	files := &closeTrackingFS{fsys: os.DirFS(dir)}
	game := castrum.New(castrum.WithFilesystem(files))
	mustRunner(t, game)
	provider, ok := game.World().MustResource[*audio.Backend]().Controller().(*audioProvider)
	if !ok {
		t.Fatal("runner did not attach its audio provider")
	}
	t.Cleanup(func() {
		clear(provider.touched)
		if err := provider.Sweep(); err != nil {
			t.Errorf("cleanup audio sessions: %v", err)
		}
	})
	settings := audio.DefaultSettings()
	settings.Load, settings.Loop = audio.LoadStream, audio.LoopTrack
	player, err := audio.NewPlayer(audio.Track("tone.wav"), settings)
	if err != nil {
		t.Fatal(err)
	}
	first, err := game.World().NewEntity(player)
	if err != nil {
		t.Fatal(err)
	}
	second, err := game.World().NewEntity(player)
	if err != nil {
		t.Fatal(err)
	}
	if err := game.Startup(); err != nil {
		t.Fatal(err)
	}
	if err := game.Advance(17 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	survivor := provider.players[second.ID()]
	if provider.players[first.ID()] == nil || survivor == nil || files.opens.Load()-files.closes.Load() != 2 {
		t.Fatal("runner did not start both streaming sessions")
	}
	if err := game.World().DestroyEntity(first); err != nil {
		t.Fatal(err)
	}
	if err := game.Advance(17 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if provider.players[first.ID()] != nil || provider.players[second.ID()] != survivor {
		t.Fatal("entity removal did not release only the removed player's session")
	}
	if files.opens.Load()-files.closes.Load() != 1 {
		t.Fatal("entity removal did not close its stream")
	}
}

func TestRunnerAudioCompletionReclaimsOneShot(t *testing.T) {
	ctx := ebitaudio.CurrentContext()
	if ctx == nil {
		ebitaudio.NewContext(44100)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tone.wav"), testWAV(), 0o644); err != nil {
		t.Fatal(err)
	}
	files := &closeTrackingFS{fsys: os.DirFS(dir)}
	game := castrum.New(castrum.WithFilesystem(files))
	mustRunner(t, game)
	world := game.World()
	provider, ok := world.MustResource[*audio.Backend]().Controller().(*audioProvider)
	if !ok {
		t.Fatal("runner did not attach its audio provider")
	}
	t.Cleanup(func() {
		clear(provider.touched)
		if err := provider.Sweep(); err != nil {
			t.Errorf("cleanup audio sessions: %v", err)
		}
	})
	finished := world.MustResource[*signal.Hub]().NewReader[audio.Finished]()
	var observed []audio.Finished
	if err := game.AddSystem(core.PhasePostFrame, "record finished", core.SystemFunc(func(ctx *core.Context) error {
		for event, ok := finished.Next(); ok; event, ok = finished.Next() {
			observed = append(observed, event)
		}
		return nil
	})); err != nil {
		t.Fatal(err)
	}

	streamSettings := audio.DefaultSettings()
	streamSettings.Load = audio.LoadStream
	streamed, err := audio.NewPlayer(audio.Track("tone.wav"), streamSettings)
	if err != nil {
		t.Fatal(err)
	}
	persistent, err := world.NewEntity(streamed)
	if err != nil {
		t.Fatal(err)
	}
	oneshot, err := audio.OneShot(world, audio.Track("tone.wav"), audio.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}

	if err := game.Startup(); err != nil {
		t.Fatal(err)
	}
	if err := game.Advance(17 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	streamSession := provider.players[persistent.ID()]
	shotSession := provider.players[oneshot.ID()]
	if streamSession == nil || shotSession == nil || !streamSession.player.IsPlaying() || !shotSession.player.IsPlaying() {
		t.Fatal("runner did not start both plays")
	}
	if files.opens.Load() != 2 || files.closes.Load() != 1 {
		t.Fatalf("opens=%d closes=%d, want a closed eager read and one live stream", files.opens.Load(), files.closes.Load())
	}

	streamSession.player.PauseAndStopReading()
	shotSession.player.PauseAndStopReading()
	if err := game.Advance(17 * time.Millisecond); err != nil {
		t.Fatal(err)
	}

	if !world.HasEntity(persistent) {
		t.Fatal("completed persistent play was destroyed")
	}
	if player, ok := persistent.Component[audio.Player](world); !ok || !player.Completed() {
		t.Fatalf("completed player = %+v, present=%v", player, ok)
	}
	if world.HasEntity(oneshot) {
		t.Fatal("completed one-shot was not reclaimed")
	}
	signals := make(map[core.Entity]asset.ID)
	for _, event := range observed {
		signals[event.Entity] = event.Audio
	}
	if len(signals) != 2 || signals[persistent] != "tone.wav" || signals[oneshot] != "tone.wav" {
		t.Fatalf("finished signals = %v from %d events", signals, len(observed))
	}
	if got := provider.players[persistent.ID()]; got != streamSession || !got.finished || got.player != nil || got.balance != nil || got.source != nil {
		t.Fatalf("completed stream session = %+v, want a released tombstone", got)
	}
	if files.opens.Load() != files.closes.Load() {
		t.Fatal("completion did not close the streamed file")
	}

	opens := files.opens.Load()
	closes := files.closes.Load()
	if err := game.Advance(17 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if provider.players[oneshot.ID()] != nil {
		t.Fatal("one-shot session was not swept after reclamation")
	}
	if provider.players[persistent.ID()] != streamSession {
		t.Fatal("completed persistent play replayed or lost its tombstone")
	}
	if files.opens.Load() != opens || files.closes.Load() != closes {
		t.Fatal("completed plays reopened or closed files a second time")
	}
	if len(observed) != 2 {
		t.Fatal("completion signals were emitted more than once")
	}
}
