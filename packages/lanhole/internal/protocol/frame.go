package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	MaxFrameSize = 8 << 20 // 8 MiB; chunks are smaller.
	ChunkSize    = 256 << 10
)

func WriteFrame(w io.Writer, b []byte) error {
	if len(b) > MaxFrameSize {
		return fmt.Errorf("frame too large: %d", len(b))
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(b)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

func ReadFrame(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > MaxFrameSize {
		return nil, fmt.Errorf("frame too large: %d", n)
	}
	b := make([]byte, n)
	_, err := io.ReadFull(r, b)
	return b, err
}
