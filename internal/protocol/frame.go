package protocol

import (
	"encoding/binary"
	"io"
)

// Frame sends and reads length-prefixed binary frames (4-byte big-endian length + body).
const maxFrameSize = 1024 * 1024 // 1MB

// WriteFrame writes len(body) as 4-byte BE then body to w.
func WriteFrame(w io.Writer, body []byte) error {
	var h [4]byte
	binary.BigEndian.PutUint32(h[:], uint32(len(body)))
	if _, err := w.Write(h[:]); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}

// ReadFrame reads a length-prefixed frame from r into a buffer; returns the body slice.
func ReadFrame(r io.Reader, buf []byte) ([]byte, error) {
	var h [4]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(h[:])
	if n > maxFrameSize {
		return nil, io.ErrShortBuffer
	}
	if cap(buf) < int(n) {
		buf = make([]byte, n)
	} else {
		buf = buf[:n]
	}
	_, err := io.ReadFull(r, buf)
	return buf, err
}
