package ebitrun

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync/atomic"
)

const stereoFrameBytes = 4

// balanceStream permits concurrent balance updates, but reads and seeks must
// be serialized by its owner, just as for the underlying stream.
type balanceStream interface {
	io.Reader
	setBalance(float64) error
}

// balanceReader processes signed 16-bit little-endian stereo frames without
// modifying the source. Scaling truncates toward zero. Unread output and an
// incomplete input frame share buf; buffered output retains its original gains.
type balanceReader struct {
	source  io.Reader
	balance atomic.Uint64
	buf     [4096]byte
	start   int
	end     int
	tail    int
	skip    int
	err     error
}

type seekableBalanceReader struct {
	*balanceReader
	seeker io.Seeker
}

func newBalanceReader(source io.Reader, balance float64) (balanceStream, error) {
	r := &balanceReader{source: source}
	if err := r.setBalance(balance); err != nil {
		return nil, err
	}
	if seeker, ok := source.(io.Seeker); ok {
		return &seekableBalanceReader{balanceReader: r, seeker: seeker}, nil
	}
	return r, nil
}

func (r *balanceReader) setBalance(balance float64) error {
	if math.IsNaN(balance) || balance < -1 || balance > 1 {
		return fmt.Errorf("castrum/ebiten: audio: balance must be finite and in [-1, 1], got %v", balance)
	}
	r.balance.Store(math.Float64bits(balance))
	return nil
}

func (r *balanceReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	balance := math.Float64frombits(r.balance.Load())
	left, right := 1-math.Max(balance, 0), 1+math.Min(balance, 0)
	written := 0
	for {

		n := copy(p[written:], r.buf[r.start:r.end])
		written += n
		r.start += n
		if r.start < r.end {
			return written, nil
		}
		if r.err != nil {
			return written, r.err
		}
		if written == len(p) {
			return written, nil
		}

		copy(r.buf[:], r.buf[r.end:r.end+r.tail])
		// Read no more than the remaining output, rounded up to a whole frame.
		size := min(len(r.buf), (len(p)-written+r.skip+stereoFrameBytes-1)/stereoFrameBytes*stereoFrameBytes)
		n, err := r.source.Read(r.buf[r.tail:size])
		total := r.tail + n
		r.start, r.end = 0, total-total%stereoFrameBytes
		r.tail = total % stereoFrameBytes
		r.err = err
		if err == io.EOF && r.tail != 0 {
			r.err = fmt.Errorf("castrum/ebiten: audio: incomplete stereo PCM frame: %w", io.ErrUnexpectedEOF)
		}
		if balance != 0 {
			for i := 0; i < r.end; i += stereoFrameBytes {
				leftSample := int16(binary.LittleEndian.Uint16(r.buf[i:]))
				rightSample := int16(binary.LittleEndian.Uint16(r.buf[i+2:]))
				binary.LittleEndian.PutUint16(r.buf[i:], uint16(int16(float64(leftSample)*left)))
				binary.LittleEndian.PutUint16(r.buf[i+2:], uint16(int16(float64(rightSample)*right)))
			}
		}
		if r.end != 0 {
			r.start = r.skip
			r.skip = 0
		}
		if n == 0 && err == nil {
			return written, nil
		}
	}
}

func (r *seekableBalanceReader) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekCurrent {
		offset -= int64(r.end - r.start + r.tail - r.skip)
	}
	position, err := r.seeker.Seek(offset, whence)
	if err != nil {
		return position, err
	}
	r.start, r.end, r.tail, r.skip, r.err = 0, 0, 0, 0, nil
	// A byte seek can land inside a sample. Read its whole frame so channel
	// scaling stays correct, then discard the prefix before the requested byte.
	if prefix := position % stereoFrameBytes; prefix != 0 {
		if _, err := r.seeker.Seek(-prefix, io.SeekCurrent); err != nil {
			r.err = fmt.Errorf("castrum/ebiten: audio: align balance reader seek: %w", err)
			return position, r.err
		}
		r.skip = int(prefix)
	}
	return position, nil
}
