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
	"testing"
	"time"

	"github.com/Leonard-Atorough/castrum"
	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"

	ebitaudio "github.com/hajimehoshi/ebiten/v2/audio"
)

func TestNewRegistersAudioCodecs(t *testing.T) {
	r, err := New(mustGame(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	loader, err := r.g.World().Resource[*asset.Loader]()
	if err != nil {
		t.Fatalf("asset loader: %v", err)
	}
	noop := func(r io.Reader) (asset.AudioData, error) {
		return asset.AudioData{}, nil
	}
	for _, format := range []asset.Format{asset.FormatMP3, asset.FormatWAV, asset.FormatOGG} {
		err := loader.RegisterDecoder(format, noop, false)
		if err == nil || !strings.Contains(err.Error(), "already exists") {
			t.Errorf("format %s: RegisterDecoder = %v, want already-exists error (codec not registered)", format, err)
		}
	}
}

func testWAV() []byte {
	const sampleRate = 44100
	pcm := make([]byte, 64*4) // 64 stereo frames, 16-bit
	for i := range pcm {
		pcm[i] = byte(i)
	}
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+len(pcm)))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
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

func newStreamTestProvider(t *testing.T) (*audioProvider, string) {
	t.Helper()
	dir := t.TempDir()
	name := filepath.Join(dir, "tone.wav")
	if err := os.WriteFile(name, testWAV(), 0o644); err != nil {
		t.Fatalf("write wav: %v", err)
	}
	audioCtx := ebitaudio.CurrentContext()
	if audioCtx == nil {
		audioCtx = ebitaudio.NewContext(44100)
	}
	return newAudioProvider(audioCtx, asset.NewLoader(os.DirFS(dir))), "tone.wav"
}

func TestCreatePlayerEager(t *testing.T) {
	p, source := newStreamTestProvider(t)
	s, err := p.createPlayer(asset.ID(source), audio.Playback{
		Playing: true, Volume: 1, Rate: 1, Loop: audio.LoopForever,
	})
	if err != nil {
		t.Fatalf("createPlayer eager: %v", err)
	}
	if s.player == nil {
		t.Fatal("eager session has no player")
	}
	if s.source != nil {
		t.Fatal("eager session must not hold an open file")
	}
	if s.loop != audio.LoopForever || s.load != audio.LoadEager {
		t.Fatalf("session = %+v, want loop/load recorded", s)
	}
}

func TestCreatePlayerStream(t *testing.T) {
	p, source := newStreamTestProvider(t)
	s, err := p.createPlayer(asset.ID(source), audio.Playback{
		Playing: true, Volume: 1, Rate: 1, Load: audio.LoadStream,
	})
	if err != nil {
		t.Fatalf("createPlayer stream: %v", err)
	}
	if s.player == nil {
		t.Fatal("stream session has no player")
	}
	if s.source == nil {
		t.Fatal("stream session must park its open file for Close")
	}
	if s.load != audio.LoadStream {
		t.Fatalf("session load = %v, want LoadStream", s.load)
	}

	s.player.PauseAndStopReading()
	if err := s.source.Close(); err != nil {
		t.Fatalf("close parked source: %v", err)
	}
}

func TestCreatePlayerErrors(t *testing.T) {
	p, _ := newStreamTestProvider(t)
	if _, err := p.createPlayer("tone.wav", audio.Playback{
		Playing: true, Volume: 1, Rate: 2,
	}); err == nil {
		t.Error("rate 2: want error")
	}
	if _, err := p.createPlayer("unknown.bin", audio.Playback{
		Playing: true, Volume: 1, Rate: 1, Load: audio.LoadStream,
	}); err == nil {
		t.Error("unknown format: want error")
	}
	if _, err := p.createPlayer("missing.wav", audio.Playback{
		Playing: true, Volume: 1, Rate: 1, Load: audio.LoadEager,
	}); err == nil {
		t.Error("missing file: want error")
	}
}

func TestCreatePlayerBalance(t *testing.T) {
	for _, load := range []audio.LoadMode{audio.LoadEager, audio.LoadStream} {
		for _, loop := range []audio.LoopMode{audio.LoopNone, audio.LoopForever} {
			for _, balance := range []float64{-1, 0, 0.5, 1} {
				t.Run(fmt.Sprintf("load=%d/loop=%d/balance=%g", load, loop, balance), func(t *testing.T) {
					p, source := newStreamTestProvider(t)
					s, err := p.createPlayer(asset.ID(source), audio.Playback{
						Volume: 0.25, Rate: 1, Load: load, Loop: loop, Balance: balance,
					})
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						s.player.PauseAndStopReading()
						if s.source != nil {
							if err := s.source.Close(); err != nil {
								t.Error(err)
							}
						}
					})
					if s.player.Volume() != 0.25 {
						t.Fatalf("player volume = %v, want 0.25", s.player.Volume())
					}
					// The player has not started, so its source can be read safely.
					original := testWAV()[44:]
					want := bytes.Clone(original)
					for i := 0; i < len(want); i += stereoFrameBytes {
						left := int16(binary.LittleEndian.Uint16(original[i:]))
						right := int16(binary.LittleEndian.Uint16(original[i+2:]))
						binary.LittleEndian.PutUint16(want[i:], uint16(int16(float64(left)*(1-math.Max(balance, 0)))))
						binary.LittleEndian.PutUint16(want[i+2:], uint16(int16(float64(right)*(1+math.Min(balance, 0)))))
					}
					if loop == audio.LoopForever {
						want = bytes.Repeat(want, 3)
					}
					got := make([]byte, len(want))
					if _, err := io.ReadFull(s.balance, got); err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(got, want) {
						t.Fatal("session reader returned incorrect balanced PCM")
					}
					if loop == audio.LoopNone {
						if n, err := s.balance.Read(make([]byte, 4)); n != 0 || err != io.EOF {
							t.Fatalf("end of play = %d, %v; want 0, EOF", n, err)
						}
					}
					if load == audio.LoadEager {
						data, err := p.loader.Load[asset.AudioData](source)
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(data.PCM, original) {
							t.Fatal("balance modified cached PCM")
						}
					}
				})
			}
		}
	}
}

func TestSyncBalanceUpdatesAndRebuilds(t *testing.T) {
	for _, load := range []audio.LoadMode{audio.LoadEager, audio.LoadStream} {
		t.Run(fmt.Sprintf("load=%d", load), func(t *testing.T) {
			p, source := newStreamTestProvider(t)
			t.Cleanup(func() {
				clear(p.touched)
				if err := p.Sweep(); err != nil {
					t.Error(err)
				}
			})
			id := core.EntityID(42)
			want := audio.Playback{Volume: 1, Rate: 1, Load: load, Loop: audio.LoopForever, Restart: true}
			if live, err := p.Sync(id, asset.ID(source), want); err != nil || !live {
				t.Fatalf("create held: live=%v, err=%v", live, err)
			}
			first := p.players[id]
			want.Restart = false
			for _, playing := range []bool{false, true, false} {
				want.Playing = playing
				want.Balance = 0.5
				want.Volume = 0.25
				if live, err := p.Sync(id, asset.ID(source), want); err != nil || !live {
					t.Fatalf("update: live=%v, err=%v", live, err)
				}
				s := p.players[id]
				if s != first || s.held == playing {
					t.Fatal("balance update replaced session or changed pause behavior")
				}
				reader, ok := s.balance.(*seekableBalanceReader)
				if !ok {
					t.Fatalf("reader = %T, want seekable balance reader", s.balance)
				}
				if got := math.Float64frombits(reader.balance.Load()); got != want.Balance {
					t.Fatalf("balance = %v, want %v", got, want.Balance)
				}
				if got := s.player.Volume(); got != want.Volume {
					t.Fatalf("volume = %v, want %v", got, want.Volume)
				}
			}
			firstSource := first.source
			want.Restart = true
			want.Balance = -1
			if live, err := p.Sync(id, asset.ID(source), want); err != nil || !live {
				t.Fatalf("restart: live=%v, err=%v", live, err)
			}
			s := p.players[id]
			if s == first {
				t.Fatal("restart did not replace session")
			}
			if got := math.Float64frombits(s.balance.(*seekableBalanceReader).balance.Load()); got != -1 {
				t.Fatalf("restart balance = %v, want -1", got)
			}
			if load == audio.LoadStream {
				if err := firstSource.Close(); err == nil {
					t.Fatal("restart did not close old stream")
				}
			}
			for _, rebuild := range []string{"load", "loop"} {
				old := s
				want.Restart = false
				if rebuild == "load" {
					if want.Load == audio.LoadEager {
						want.Load = audio.LoadStream
					} else {
						want.Load = audio.LoadEager
					}
				} else {
					want.Loop = audio.LoopNone
				}
				if live, err := p.Sync(id, asset.ID(source), want); err != nil || !live {
					t.Fatalf("%s rebuild: live=%v, err=%v", rebuild, live, err)
				}
				s = p.players[id]
				if s == old || math.Float64frombits(s.balance.(*seekableBalanceReader).balance.Load()) != -1 {
					t.Fatalf("%s rebuild did not preserve balance in a fresh session", rebuild)
				}
			}
			s.finished = true
			want.Restart = false
			want.Playing = true
			want.Balance = 1
			if live, err := p.Sync(id, asset.ID(source), want); err != nil || live {
				t.Fatalf("finished update: live=%v, err=%v", live, err)
			}
			if p.players[id] != s || !s.finished {
				t.Fatal("balance resurrected a completed play")
			}
		})
	}
}

func TestSessionBalanceErrors(t *testing.T) {
	p, source := newStreamTestProvider(t)
	for _, load := range []audio.LoadMode{audio.LoadEager, audio.LoadStream} {
		for _, balance := range []float64{math.NaN(), math.Inf(1), -2, 2} {
			if _, err := p.createPlayer(asset.ID(source), audio.Playback{
				Volume: 1, Rate: 1, Load: load, Balance: balance,
			}); err == nil || !strings.Contains(err.Error(), "balance") {
				t.Fatalf("load=%d balance=%v: error = %v", load, balance, err)
			}
		}
	}
	id := core.EntityID(43)
	want := audio.Playback{Volume: 1, Rate: 1, Restart: true}
	if _, err := p.Sync(id, asset.ID(source), want); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		clear(p.touched)
		if err := p.Sweep(); err != nil {
			t.Error(err)
		}
	})
	want.Restart = false
	want.Balance = math.NaN()
	if _, err := p.Sync(id, asset.ID(source), want); err == nil || !strings.Contains(err.Error(), source) {
		t.Fatalf("invalid update error = %v, want source-specific error", err)
	}
}

func TestInvalidBalanceClosesStream(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tone.wav"), testWAV(), 0o644); err != nil {
		t.Fatal(err)
	}
	var opens, closes int
	p, _ := newStreamTestProvider(t)
	p.loader = asset.NewLoader(closeTrackingFS{fsys: os.DirFS(dir), opens: &opens, closes: &closes})
	registerAudioCodecs(p.loader, p.audioCtx.SampleRate())
	if _, err := p.createPlayer("tone.wav", audio.Playback{
		Volume: 1, Rate: 1, Load: audio.LoadStream, Balance: math.NaN(),
	}); err == nil {
		t.Fatal("invalid balance accepted")
	}
	if opens == 0 || opens != closes {
		t.Fatalf("opens=%d, closes=%d; want all opened files closed", opens, closes)
	}
}

func TestSyncLifecycle(t *testing.T) {
	p, source := newStreamTestProvider(t)
	id := core.EntityID(7)

	live, err := p.Sync(id, asset.ID(source), audio.Playback{Playing: false, Volume: 1, Rate: 1})
	if err != nil || live {
		t.Fatalf("absent, not playing, no restart: live=%v err=%v, want false nil", live, err)
	}
	if p.players[id] != nil {
		t.Fatal("nothing should have been created")
	}

	live, err = p.Sync(id, asset.ID(source), audio.Playback{Playing: false, Volume: 1, Rate: 1, Restart: true})
	if err != nil || !live {
		t.Fatalf("restart while paused: live=%v err=%v, want true nil", live, err)
	}
	s := p.players[id]
	if s == nil || !s.held || s.finished {
		t.Fatalf("created session = %+v, want held and not finished", s)
	}

	live, err = p.Sync(id, asset.ID(source), audio.Playback{Playing: true, Volume: 1, Rate: 1})
	if err != nil || !live {
		t.Fatalf("resume: live=%v err=%v, want true nil", live, err)
	}
	if s.held {
		t.Fatal("resume left the session held")
	}

	live, err = p.Sync(id, asset.ID(source), audio.Playback{Playing: false, Volume: 1, Rate: 1})
	if err != nil || !live {
		t.Fatalf("pause: live=%v err=%v, want true (held is live)", live, err)
	}
	if !s.held {
		t.Fatal("pause left the session unheld")
	}

	// A finished play is a tombstone: nothing but a rebuild touches it.
	s.finished = true
	live, err = p.Sync(id, asset.ID(source), audio.Playback{Playing: true, Volume: 1, Rate: 1})
	if err != nil || live {
		t.Fatalf("finished tombstone with Playing: live=%v err=%v, want false nil", live, err)
	}
	if !s.finished {
		t.Fatal("a finished play must not resurrect")

	}

	old := s
	live, err = p.Sync(id, asset.ID(source), audio.Playback{Playing: false, Volume: 1, Rate: 1, Restart: true})
	if err != nil || !live {
		t.Fatalf("restart over tombstone: live=%v err=%v, want true nil", live, err)
	}
	if p.players[id] == old {
		t.Fatal("restart must replace the session")
	}
	if s := p.players[id]; !s.held || s.finished {
		t.Fatalf("rebuilt session = %+v, want held and fresh", s)
	}
}

func TestSyncRebuildsOnLoopChange(t *testing.T) {
	p, source := newStreamTestProvider(t)
	id := core.EntityID(9)

	if _, err := p.Sync(id, asset.ID(source), audio.Playback{Playing: false, Volume: 1, Rate: 1, Restart: true}); err != nil {
		t.Fatalf("create: %v", err)
	}
	old := p.players[id]

	if _, err := p.Sync(id, asset.ID(source), audio.Playback{
		Playing: false, Volume: 1, Rate: 1, Loop: audio.LoopForever,
	}); err != nil {
		t.Fatalf("loop change: %v", err)
	}
	if p.players[id] == old {
		t.Fatal("a loop change must rebuild the session")
	}
	if p.players[id].loop != audio.LoopForever {
		t.Fatal("rebuilt session did not record the new loop mode")
	}
}

func TestSweepReleasesUnsyncedSessions(t *testing.T) {
	p, source := newStreamTestProvider(t)
	id1, id2 := core.EntityID(1), core.EntityID(2)

	for _, id := range []core.EntityID{id1, id2} {
		if _, err := p.Sync(id, asset.ID(source), audio.Playback{
			Playing: false, Volume: 1, Rate: 1, Restart: true, Load: audio.LoadStream,
		}); err != nil {
			t.Fatalf("create %d: %v", id, err)
		}
	}
	if err := p.Sweep(); err != nil {
		t.Fatalf("sweep 1: %v", err)
	}
	if p.players[id1] == nil || p.players[id2] == nil {
		t.Fatal("both sessions were synced; sweep must keep both")
	}

	staleSource := p.players[id2].source
	if _, err := p.Sync(id1, asset.ID(source), audio.Playback{Playing: false, Volume: 1, Rate: 1}); err != nil {
		t.Fatalf("touch id1: %v", err)
	}
	if err := p.Sweep(); err != nil {
		t.Fatalf("sweep 2: %v", err)
	}
	if p.players[id2] != nil {
		t.Fatal("untouched session must be swept")
	}
	if p.players[id1] == nil {
		t.Fatal("synced session must survive the sweep")
	}
	if err := staleSource.Close(); err == nil {
		t.Fatal("sweep must have closed the swept session's parked file")
	}
}

// closeTrackingFS counts file opens and closes over a real directory,
// so the streaming path's parked file is observable.
type closeTrackingFS struct {
	fsys   fs.FS
	opens  *int
	closes *int
}

func (c closeTrackingFS) Open(name string) (fs.File, error) {
	f, err := c.fsys.Open(name)
	if err != nil {
		return nil, err
	}
	*c.opens++
	return &trackingFile{File: f, closes: c.closes}, nil
}

type trackingFile struct {
	fs.File
	closes   *int
	closeErr error
}

func (f *trackingFile) Close() error {
	*f.closes++
	return errors.Join(f.File.Close(), f.closeErr)
}

// Seek forwards to the wrapped file so the streaming path sees the
// underlying directory file's seekability.
func (f *trackingFile) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := f.File.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("trackingFile: wrapped file %T is not seekable", f.File)
	}
	return seeker.Seek(offset, whence)
}

// TestDefaultRunnerAudioPlaybackAndCleanup regression-checks the
// default runner's audio path end to end: both load modes play
// through the engine's audio system, a paused play holds, and
// destroying a playing entity releases its player and closes the
// streaming file on the next sweep.
func TestDefaultRunnerAudioPlaybackAndCleanup(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tone.wav"), testWAV(), 0o644); err != nil {
		t.Fatalf("write wav: %v", err)
	}
	var opens, closes int
	fsys := closeTrackingFS{fsys: os.DirFS(dir), opens: &opens, closes: &closes}

	g, err := castrum.New(castrum.WithFilesystem(fsys))
	if err != nil {
		t.Fatalf("castrum.New: %v", err)
	}
	if _, err := New(g); err != nil { // default wiring: codecs, provider, engine.audio
		t.Fatalf("ebitrun.New: %v", err)
	}
	world := g.World()

	// One eager play and one streaming loop, as games build them.
	shot := audio.NewSource("tone.wav")
	shot.Load = audio.LoadEager
	shot.Balance = -1
	shotEntity, err := world.NewEntity(shot)
	if err != nil {
		t.Fatal(err)
	}
	music := audio.NewSource("tone.wav")
	music.Load = audio.LoadStream
	music.Loop = audio.LoopForever
	music.Balance = 0.5
	musicEntity, err := world.NewEntity(music)
	if err != nil {
		t.Fatal(err)
	}

	if err := g.Startup(); err != nil {
		t.Fatal(err)
	}
	advance := func(frames int) {
		t.Helper()
		for range frames {
			if err := g.Advance(17 * time.Millisecond); err != nil {
				t.Fatal(err)
			}
		}
	}
	advance(3)

	controller, err := world.Resource[audio.Controller]()
	if err != nil {
		t.Fatalf("audio controller: %v", err)
	}
	provider, ok := controller.(*audioProvider)
	if !ok {
		t.Fatalf("controller = %T, want the runner's audioProvider", controller)
	}
	t.Cleanup(func() {
		clear(provider.touched)
		if err := provider.Sweep(); err != nil {
			t.Error(err)
		}
	})

	// Both load modes play: the eager session owns no file, the
	// streaming session parks its open one.
	shotSession := provider.players[shotEntity.ID()]
	if shotSession == nil || shotSession.load != audio.LoadEager || shotSession.source != nil {
		t.Fatalf("eager session = %+v, want live with no open file", shotSession)
	}
	if !shotSession.player.IsPlaying() {
		t.Error("eager play should be playing")
	}
	musicSession := provider.players[musicEntity.ID()]
	if musicSession == nil || musicSession.load != audio.LoadStream || musicSession.source == nil {
		t.Fatalf("stream session = %+v, want live with its open file", musicSession)
	}
	if !musicSession.player.IsPlaying() {
		t.Error("stream play should be playing")
	}
	for _, tt := range []struct {
		session *audioSession
		want    float64
	}{{shotSession, -1}, {musicSession, 0.5}} {
		if got := math.Float64frombits(tt.session.balance.(*seekableBalanceReader).balance.Load()); got != tt.want {
			t.Errorf("resolved balance = %v, want %v", got, tt.want)
		}
	}
	if open := opens - closes; open != 1 {
		t.Errorf("open files = %d during playback, want the stream's one", open)
	}

	// A paused play holds its session.
	if err := musicEntity.Update(world, func(s *audio.Source) {
		s.Paused = true
		s.Balance = -0.5
	}); err != nil {
		t.Fatal(err)
	}
	advance(2)
	if !musicSession.held {
		t.Error("a paused play should hold its session")
	}
	if provider.players[musicEntity.ID()] != musicSession {
		t.Fatal("paused balance update replaced session")
	}
	if got := math.Float64frombits(musicSession.balance.(*seekableBalanceReader).balance.Load()); got != -0.5 {
		t.Errorf("paused balance = %v, want -0.5", got)
	}
	if err := musicEntity.Update(world, func(s *audio.Source) { s.Restart() }); err != nil {
		t.Fatal(err)
	}
	advance(1)
	restarted := provider.players[musicEntity.ID()]
	if restarted == musicSession || math.Float64frombits(restarted.balance.(*seekableBalanceReader).balance.Load()) != -0.5 {
		t.Fatal("source restart did not preserve balance in a new session")
	}

	// Destroying the streaming entity cleans up: the session goes
	// and the parked file closes on the next sweep.
	if err := world.DestroyEntity(musicEntity); err != nil {
		t.Fatal(err)
	}
	advance(2)
	if _, still := provider.players[musicEntity.ID()]; still {
		t.Error("destroyed stream play should release its session")
	}
	if open := opens - closes; open != 0 {
		t.Errorf("open files = %d after destroy, want the stream file closed", open)
	}

	// The eager play survives the music's death and releases when
	// its own entity goes.
	if _, still := provider.players[shotEntity.ID()]; !still {
		t.Fatal("the eager play should still be live")
	}
	if err := world.DestroyEntity(shotEntity); err != nil {
		t.Fatal(err)
	}
	advance(2)
	if _, still := provider.players[shotEntity.ID()]; still {
		t.Error("destroyed eager play should release its session")
	}

	stream := audio.NewSource("tone.wav")
	stream.Load = audio.LoadStream
	stream.Balance = -0.5
	persistent, err := world.NewEntity(stream)
	if err != nil {
		t.Fatal(err)
	}
	oneShot, err := audio.OneShot(world, stream)
	if err != nil {
		t.Fatal(err)
	}
	g.Pause()
	tick := g.Context().Tick
	advance(1)
	sessions := make(map[core.EntityID]*audioSession)
	for _, entity := range []*core.Entity{persistent, oneShot} {
		s := provider.players[entity.ID()]
		if s == nil || s.held || !s.player.IsPlaying() || s.source == nil {
			t.Fatalf("simulation pause held or prevented audio: %+v", s)
		}
		sessions[entity.ID()] = s
		// Simulate the backend reporting playback stopped, without waiting
		// for an audio device or depending on wall-clock playback.
		s.player.PauseAndStopReading()
	}
	advance(1)
	if source, ok := persistent.Component[audio.Source](world); !ok || !source.Completed || !persistent.IsAlive() {
		t.Fatalf("completed persistent source = %+v, present=%v", source, ok)
	}
	if oneShot.IsAlive() {
		t.Fatal("completed one-shot was not reclaimed during simulation pause")
	}
	for id, s := range sessions {
		if provider.players[id] != s || !s.finished || s.player != nil || s.balance != nil || s.source != nil {
			t.Fatalf("completed session %d retained playback resources or lost its tombstone: %+v", id, s)
		}
	}
	if open := opens - closes; open != 0 {
		t.Fatalf("open files after completion = %d, want 0", open)
	}
	completedOpens, completedCloses := opens, closes
	advance(1)
	if provider.players[oneShot.ID()] != nil {
		t.Fatal("one-shot tombstone was not swept on the next frame")
	}
	if provider.players[persistent.ID()] != sessions[persistent.ID()] ||
		opens != completedOpens || closes != completedCloses {
		t.Fatal("completed source replayed or its resources were released twice")
	}
	if g.Context().Tick != tick {
		t.Fatal("audio completion required simulation ticks")
	}
	if err := persistent.Update(world, func(s *audio.Source) { s.Restart() }); err != nil {
		t.Fatal(err)
	}
	advance(1)
	fresh := provider.players[persistent.ID()]
	if fresh == nil || fresh == sessions[persistent.ID()] || fresh.finished ||
		fresh.source == nil || !fresh.player.IsPlaying() {
		t.Fatalf("restart did not create fresh streaming playback: %+v", fresh)
	}
	if source, _ := persistent.Component[audio.Source](world); source.Completed {
		t.Fatal("restart left the persistent source completed")
	}
	if got := math.Float64frombits(fresh.balance.(*seekableBalanceReader).balance.Load()); got != stream.Balance {
		t.Fatalf("restart balance = %v, want %v", got, stream.Balance)
	}
	if opens != completedOpens+1 || opens-closes != 1 {
		t.Fatalf("restart opens=%d closes=%d, want one new open stream", opens, closes)
	}
	closeErr := errors.New("stream close failed")
	file, ok := fresh.source.(*trackingFile)
	if !ok {
		t.Fatalf("stream source = %T, want trackingFile", fresh.source)
	}
	file.closeErr = closeErr
	fresh.player.PauseAndStopReading()
	if err := g.Advance(17 * time.Millisecond); !errors.Is(err, closeErr) || !strings.Contains(err.Error(), "tone.wav") {
		t.Fatalf("completion close error = %v, want wrapped source-specific error", err)
	}
	advance(1)
	if err := world.DestroyEntity(persistent); err != nil {
		t.Fatal(err)
	}
	advance(1)
	if opens != closes {
		t.Fatalf("open files after restarted source destruction = %d, want 0", opens-closes)
	}
	if closes != completedCloses+1 {
		t.Fatal("failed close was attempted more than once")
	}
}
