package ebitrun

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sync"
	"testing"
)

func balancePCM(samples ...int16) []byte {
	pcm := make([]byte, len(samples)*2)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(sample))
	}
	return pcm
}

func mustBalanceReader(t *testing.T, source io.Reader, balance float64) balanceStream {
	t.Helper()
	r, err := newBalanceReader(source, balance)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestBalanceReaderSamples(t *testing.T) {
	input := balancePCM(1000, -1000, -32768, 32767, -3, 3)
	original := bytes.Clone(input)
	tests := []struct {
		name    string
		balance float64
		want    []byte
	}{
		{"unchanged", 0, input},
		{"left", -1, balancePCM(1000, 0, -32768, 0, -3, 0)},
		{"right", 1, balancePCM(0, -1000, 0, 32767, 0, 3)},
		{"half left", -0.5, balancePCM(1000, -500, -32768, 16383, -3, 1)},
		{"half right", 0.5, balancePCM(500, -1000, -16384, 32767, -1, 3)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := mustBalanceReader(t, bytes.NewReader(input), tt.balance)
			got, err := io.ReadAll(r)
			if err != nil || !bytes.Equal(got, tt.want) {
				t.Fatalf("ReadAll = %v, %v; want %v, nil", got, err, tt.want)
			}
			if !bytes.Equal(input, original) {
				t.Fatal("reader modified shared PCM")
			}
		})
	}
}

type fragmentedPCMReader struct {
	source io.Reader
	size   int
}

func (r fragmentedPCMReader) Read(p []byte) (int, error) {
	return r.source.Read(p[:min(len(p), r.size)])
}

func TestBalanceReaderFragmentedReads(t *testing.T) {
	input := bytes.Repeat(balancePCM(-32768, 32767, 1234, -5678), 1100)
	want := bytes.Repeat(balancePCM(-32768, 16383, 1234, -2839), 1100)
	for _, inputSize := range []int{1, 2, 3, 4, 7, 4096} {
		for _, outputSize := range []int{1, 2, 3, 4, 7, 5000} {
			r := mustBalanceReader(t, fragmentedPCMReader{bytes.NewReader(input), inputSize}, -0.5)
			buf := make([]byte, outputSize)
			var got []byte
			for {
				n, err := r.Read(buf)
				got = append(got, buf[:n]...)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("input %d/output %d: %v", inputSize, outputSize, err)
				}
				if n == 0 {
					t.Fatal("reader made no progress")
				}
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("input %d/output %d: incorrect PCM", inputSize, outputSize)
			}
			if _, ok := r.(io.Seeker); ok {
				t.Fatal("non-seekable input exposed Seek")
			}
		}
	}
}

type endingPCMReader struct {
	pcm []byte
	err error
}

func (r *endingPCMReader) Read(p []byte) (int, error) {
	n := copy(p, r.pcm)
	r.pcm = r.pcm[n:]
	if len(r.pcm) == 0 {
		return n, r.err
	}
	return n, nil
}

func TestBalanceReaderErrors(t *testing.T) {
	readErr := errors.New("read failed")
	for _, tt := range []struct {
		name string
		pcm  []byte
		err  error
		want error
	}{
		{"empty", nil, io.EOF, nil},
		{"EOF with frame", balancePCM(12, 34), io.EOF, nil},
		{"read failure with frame", balancePCM(12, 34), readErr, readErr},
		{"read failure with partial frame", []byte{1, 2, 3}, readErr, readErr},
		{"one trailing byte", append(balancePCM(12, 34), 1), io.EOF, io.ErrUnexpectedEOF},
		{"two trailing bytes", []byte{1, 2}, io.EOF, io.ErrUnexpectedEOF},
		{"three trailing bytes", []byte{1, 2, 3}, io.EOF, io.ErrUnexpectedEOF},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := mustBalanceReader(t, &endingPCMReader{tt.pcm, tt.err}, 0)
			got, err := io.ReadAll(r)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			want := tt.pcm[:len(tt.pcm)/stereoFrameBytes*stereoFrameBytes]
			if !bytes.Equal(got, want) {
				t.Fatalf("PCM = %v, want complete frames %v", got, want)
			}
			if n, err := r.Read(nil); n != 0 || err != nil {
				t.Fatalf("empty read = %d, %v; want 0, nil", n, err)
			}
		})
	}
}

func TestBalanceReaderValidationAndUpdates(t *testing.T) {
	for _, balance := range []float64{-1.01, 1.01, math.NaN(), math.Inf(-1), math.Inf(1)} {
		if _, err := newBalanceReader(bytes.NewReader(nil), balance); err == nil {
			t.Fatalf("accepted balance %v", balance)
		}
	}
	input := bytes.Repeat(balancePCM(100, 200), 3)
	r := mustBalanceReader(t, bytes.NewReader(input), -1)
	if err := r.setBalance(math.NaN()); err == nil {
		t.Fatal("accepted invalid update")
	}
	buf := make([]byte, 4)
	if n, err := r.Read(buf); n != 4 || err != nil || !bytes.Equal(buf, balancePCM(100, 0)) {
		t.Fatalf("invalid update changed setting: %v, %d, %v", buf, n, err)
	}
	if err := r.setBalance(1); err != nil {
		t.Fatal(err)
	}
	if n, err := r.Read(buf); n != 4 || err != nil || !bytes.Equal(buf, balancePCM(0, 200)) {
		t.Fatalf("update not applied: %v, %d, %v", buf, n, err)
	}
	other := mustBalanceReader(t, bytes.NewReader(input), 0)
	if n, err := other.Read(buf); n != 4 || err != nil || !bytes.Equal(buf, balancePCM(100, 200)) {
		t.Fatal("sessions do not have independent balance")
	}
}

func TestBalanceReaderSeek(t *testing.T) {
	input := balancePCM(1000, -1000, -32768, 32767, -3, 3)
	want := balancePCM(500, -1000, -16384, 32767, -1, 3)
	for offset := range len(input) + 1 {
		r := mustBalanceReader(t, bytes.NewReader(input), 0.5)
		seeker := r.(io.Seeker)
		if _, err := io.ReadFull(r, make([]byte, 1)); err != nil {
			t.Fatal(err)
		}
		if pos, err := seeker.Seek(0, io.SeekCurrent); pos != 1 || err != nil {
			t.Fatalf("logical position = %d, %v; want 1, nil", pos, err)
		}
		if pos, err := seeker.Seek(int64(offset), io.SeekStart); pos != int64(offset) || err != nil {
			t.Fatalf("Seek = %d, %v; want %d, nil", pos, err, offset)
		}
		got, err := io.ReadAll(r)
		if err != nil || !bytes.Equal(got, want[offset:]) {
			t.Fatalf("offset %d: PCM = %v, %v; want %v", offset, got, err, want[offset:])
		}
		if pos, err := seeker.Seek(-4, io.SeekEnd); pos != 8 || err != nil {
			t.Fatalf("SeekEnd = %d, %v; want 8, nil", pos, err)
		}
		got, err = io.ReadAll(r)
		if err != nil || !bytes.Equal(got, want[8:]) {
			t.Fatalf("seek after EOF = %v, %v", got, err)
		}
	}
}

func TestBalanceReaderFailedSeekPreservesPendingOutput(t *testing.T) {
	r := mustBalanceReader(t, bytes.NewReader(balancePCM(100, 200)), -1)
	if _, err := io.ReadFull(r, make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := r.(io.Seeker).Seek(-1, io.SeekStart); err == nil {
		t.Fatal("negative seek succeeded")
	}
	got, err := io.ReadAll(r)
	if err != nil || !bytes.Equal(got, balancePCM(100, 0)[1:]) {
		t.Fatalf("failed seek lost output: %v, %v", got, err)
	}
}

func TestBalanceReaderConcurrentUpdates(t *testing.T) {
	r := mustBalanceReader(t, bytes.NewReader(bytes.Repeat(balancePCM(100, -200), 10000)), 0)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range 10000 {
			if err := r.setBalance(float64(i%3 - 1)); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	_, err := io.Copy(io.Discard, r)
	wg.Wait()
	if err != nil {
		t.Fatal(err)
	}
}

func TestBalanceReaderUpdatePreservesPendingFrame(t *testing.T) {
	r := mustBalanceReader(t, bytes.NewReader(bytes.Repeat(balancePCM(100, 200), 2)), -1)
	var first [1]byte
	if _, err := io.ReadFull(r, first[:]); err != nil {
		t.Fatal(err)
	}
	if err := r.setBalance(1); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	want := append(balancePCM(100, 0)[1:], balancePCM(0, 200)...)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("updated PCM = %v, %v; want %v", got, err, want)
	}
}

type stalledPCMReader struct {
	step int
}

func (r *stalledPCMReader) Read(p []byte) (int, error) {
	r.step++
	switch r.step {
	case 1:
		return copy(p, []byte{100}), nil
	case 2:
		return 0, nil
	default:
		return copy(p, []byte{0, 200, 0}), io.EOF
	}
}

func TestBalanceReaderStalledReadRetainsPartialFrame(t *testing.T) {
	r := mustBalanceReader(t, &stalledPCMReader{}, -1)
	var buf [4]byte
	if n, err := r.Read(buf[:]); n != 0 || err != nil {
		t.Fatalf("stalled Read = %d, %v; want 0, nil", n, err)
	}
	got, err := io.ReadAll(r)
	if err != nil || !bytes.Equal(got, balancePCM(100, 0)) {
		t.Fatalf("read after stall = %v, %v", got, err)
	}
}

type alignmentFailReader struct {
	*bytes.Reader
	err error
}

func (r alignmentFailReader) Seek(offset int64, whence int) (int64, error) {
	if offset < 0 && whence == io.SeekCurrent {
		return 0, r.err
	}
	return r.Reader.Seek(offset, whence)
}

func TestBalanceReaderAlignmentError(t *testing.T) {
	seekErr := errors.New("alignment failed")
	r := mustBalanceReader(t, alignmentFailReader{
		Reader: bytes.NewReader(balancePCM(100, 200)),
		err:    seekErr,
	}, 1)
	if _, err := r.(io.Seeker).Seek(1, io.SeekStart); !errors.Is(err, seekErr) {
		t.Fatalf("Seek error = %v, want %v", err, seekErr)
	}
	if n, err := r.Read(make([]byte, 4)); n != 0 || !errors.Is(err, seekErr) {
		t.Fatalf("Read after failed alignment = %d, %v", n, err)
	}
	if _, err := r.(io.Seeker).Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(r)
	if err != nil || !bytes.Equal(got, balancePCM(0, 200)) {
		t.Fatalf("read after recovery = %v, %v", got, err)
	}
}
