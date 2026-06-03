package security

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/actlane/actlane/packages/lanhole/internal/protocol"
	"golang.org/x/crypto/chacha20poly1305"
)

type FrameKind byte

const (
	KindMetadata FrameKind = 1
	KindAccept   FrameKind = 2
	KindChunk    FrameKind = 3
	KindFinish   FrameKind = 4
	KindManifest FrameKind = 5
	KindError    FrameKind = 255
)

type SecureStream struct {
	rw     io.ReadWriter
	send   cipherBox
	recv   cipherBox
	seqOut uint64
	seqIn  uint64
}

type cipherBox struct {
	key         []byte
	noncePrefix []byte
	aead        interface {
		NonceSize() int
		Overhead() int
		Seal(dst, nonce, plaintext, additionalData []byte) []byte
		Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error)
	}
}

func NewSecureStream(rw io.ReadWriter, keys DirectionKeys) (*SecureStream, error) {
	send, err := newBox(keys.Send)
	if err != nil {
		return nil, fmt.Errorf("send AEAD: %w", err)
	}
	recv, err := newBox(keys.Recv)
	if err != nil {
		return nil, fmt.Errorf("recv AEAD: %w", err)
	}
	return &SecureStream{rw: rw, send: send, recv: recv}, nil
}

func newBox(m StreamMaterial) (cipherBox, error) {
	if len(m.Key) != chacha20poly1305.KeySize {
		return cipherBox{}, fmt.Errorf("bad key size: %d", len(m.Key))
	}
	if len(m.NoncePrefix) != 16 {
		return cipherBox{}, fmt.Errorf("bad nonce prefix size: %d", len(m.NoncePrefix))
	}
	aead, err := chacha20poly1305.NewX(m.Key)
	if err != nil {
		return cipherBox{}, err
	}
	return cipherBox{key: m.Key, noncePrefix: m.NoncePrefix, aead: aead}, nil
}

func (s *SecureStream) Write(kind FrameKind, plaintext []byte) error {
	if len(plaintext) > protocol.MaxFrameSize {
		return fmt.Errorf("plaintext frame too large: %d", len(plaintext))
	}
	seq := s.seqOut
	s.seqOut++
	nonce := makeNonce(s.send.noncePrefix, seq)
	ad := additionalData(kind, seq)
	sealed := s.send.aead.Seal(nil, nonce, plaintext, ad)
	payload := make([]byte, 1+8+len(sealed))
	payload[0] = byte(kind)
	binary.BigEndian.PutUint64(payload[1:9], seq)
	copy(payload[9:], sealed)
	return protocol.WriteFrame(s.rw, payload)
}

func (s *SecureStream) WriteJSON(kind FrameKind, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.Write(kind, b)
}

func (s *SecureStream) Read() (FrameKind, []byte, error) {
	payload, err := protocol.ReadFrame(s.rw)
	if err != nil {
		return 0, nil, err
	}
	if len(payload) < 1+8+s.recv.aead.Overhead() {
		return 0, nil, fmt.Errorf("encrypted frame too short")
	}
	kind := FrameKind(payload[0])
	seq := binary.BigEndian.Uint64(payload[1:9])
	if seq != s.seqIn {
		return 0, nil, fmt.Errorf("unexpected encrypted frame sequence: got %d want %d", seq, s.seqIn)
	}
	s.seqIn++
	nonce := makeNonce(s.recv.noncePrefix, seq)
	ad := additionalData(kind, seq)
	plain, err := s.recv.aead.Open(nil, nonce, payload[9:], ad)
	if err != nil {
		return 0, nil, fmt.Errorf("decrypt frame %d: %w", kind, err)
	}
	return kind, plain, nil
}

func makeNonce(prefix []byte, seq uint64) []byte {
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	copy(nonce[:16], prefix)
	binary.BigEndian.PutUint64(nonce[16:], seq)
	return nonce
}

func additionalData(kind FrameKind, seq uint64) []byte {
	var ad [9]byte
	ad[0] = byte(kind)
	binary.BigEndian.PutUint64(ad[1:], seq)
	return ad[:]
}
