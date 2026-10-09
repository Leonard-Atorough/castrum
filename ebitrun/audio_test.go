package ebitrun

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ebitaudio "github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/Leonard-Atorough/castrum/asset"
	"github.com/Leonard-Atorough/castrum/audio"
	"github.com/Leonard-Atorough/castrum/core"
)

func TestNewRegistersAudioCodecs(t *testing.T) {
	r, err := New(mustGame())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	server, err := r.g.World().Resource[*asset.Server]()
	if err != nil {
		t.Fatalf("asset server: %v", err)
	}
	noop := func(r io.Reader) (asset.AudioData, error) {
		return asset.AudioData{}, nil
	}
	for _, format := range []asset.Format{asset.FormatMP3, asset.FormatWAV, asset.FormatOGG} {
		err := server.RegisterDecoder(format, noop, false)
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
	return newAudioProvider(audioCtx, asset.New(os.DirFS(dir))), "tone.wav"
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

	stale := p.players[id2]
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
	if err := stale.source.Close(); err == nil {
		t.Fatal("sweep must have closed the swept session's parked file")
	}
}
