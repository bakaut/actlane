package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/actlane/actlane/packages/lanhole/internal/protocol"
	"github.com/schollz/pake/v3"
)

const pakeCurve = "siec"

type Role int

const (
	Sender Role = iota
	Receiver
)

type DirectionKeys struct {
	Send StreamMaterial
	Recv StreamMaterial
}

type StreamMaterial struct {
	Key         []byte // 32 bytes for XChaCha20-Poly1305
	NoncePrefix []byte // 16 bytes; nonce = prefix || seq64
}

func NormalizeCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func SenderHandshake(rw io.ReadWriter, sessionID, code string) (DirectionKeys, string, error) {
	p, err := pake.InitCurve([]byte(passwordMaterial(sessionID, code)), 0, pakeCurve)
	if err != nil {
		return DirectionKeys{}, "", fmt.Errorf("init sender PAKE: %w", err)
	}
	if err := protocol.WriteFrame(rw, p.Bytes()); err != nil {
		return DirectionKeys{}, "", fmt.Errorf("send PAKE message: %w", err)
	}
	peer, err := protocol.ReadFrame(rw)
	if err != nil {
		return DirectionKeys{}, "", fmt.Errorf("read receiver PAKE message: %w", err)
	}
	if err := p.Update(peer); err != nil {
		return DirectionKeys{}, "", fmt.Errorf("update sender PAKE: %w", err)
	}
	sessionKey, err := p.SessionKey()
	if err != nil {
		return DirectionKeys{}, "", fmt.Errorf("sender session key: %w", err)
	}
	keys := deriveDirectionKeys(sessionKey, sessionID, Sender)
	confirmKey := deriveConfirmKey(sessionKey, sessionID)
	if err := protocol.WriteFrame(rw, confirm(confirmKey, sessionID, "sender")); err != nil {
		return DirectionKeys{}, "", fmt.Errorf("write sender key confirmation: %w", err)
	}
	got, err := protocol.ReadFrame(rw)
	if err != nil {
		return DirectionKeys{}, "", fmt.Errorf("read receiver key confirmation: %w", err)
	}
	if !hmac.Equal(got, confirm(confirmKey, sessionID, "receiver")) {
		return DirectionKeys{}, "", fmt.Errorf("receiver key confirmation failed: probably wrong code or MITM")
	}
	return keys, Fingerprint(sessionKey, sessionID), nil
}

func ReceiverHandshake(rw io.ReadWriter, sessionID, code string) (DirectionKeys, string, error) {
	p, err := pake.InitCurve([]byte(passwordMaterial(sessionID, code)), 1, pakeCurve)
	if err != nil {
		return DirectionKeys{}, "", fmt.Errorf("init receiver PAKE: %w", err)
	}
	peer, err := protocol.ReadFrame(rw)
	if err != nil {
		return DirectionKeys{}, "", fmt.Errorf("read sender PAKE message: %w", err)
	}
	if err := p.Update(peer); err != nil {
		return DirectionKeys{}, "", fmt.Errorf("update receiver PAKE: %w", err)
	}
	if err := protocol.WriteFrame(rw, p.Bytes()); err != nil {
		return DirectionKeys{}, "", fmt.Errorf("send receiver PAKE message: %w", err)
	}
	sessionKey, err := p.SessionKey()
	if err != nil {
		return DirectionKeys{}, "", fmt.Errorf("receiver session key: %w", err)
	}
	keys := deriveDirectionKeys(sessionKey, sessionID, Receiver)
	confirmKey := deriveConfirmKey(sessionKey, sessionID)
	got, err := protocol.ReadFrame(rw)
	if err != nil {
		return DirectionKeys{}, "", fmt.Errorf("read sender key confirmation: %w", err)
	}
	if !hmac.Equal(got, confirm(confirmKey, sessionID, "sender")) {
		return DirectionKeys{}, "", fmt.Errorf("sender key confirmation failed: probably wrong code or MITM")
	}
	if err := protocol.WriteFrame(rw, confirm(confirmKey, sessionID, "receiver")); err != nil {
		return DirectionKeys{}, "", fmt.Errorf("write receiver key confirmation: %w", err)
	}
	return keys, Fingerprint(sessionKey, sessionID), nil
}

func passwordMaterial(sessionID, code string) string {
	return "lanhole-v1|" + sessionID + "|" + NormalizeCode(code)
}

func deriveDirectionKeys(sessionKey []byte, sessionID string, role Role) DirectionKeys {
	salt := sha256.Sum256([]byte("lanhole-v1|" + sessionID))
	s2r := hkdfSHA256(sessionKey, salt[:], []byte("stream sender-to-receiver"), 48)
	r2s := hkdfSHA256(sessionKey, salt[:], []byte("stream receiver-to-sender"), 48)
	mk := func(b []byte) StreamMaterial {
		return StreamMaterial{Key: append([]byte(nil), b[:32]...), NoncePrefix: append([]byte(nil), b[32:48]...)}
	}
	if role == Sender {
		return DirectionKeys{Send: mk(s2r), Recv: mk(r2s)}
	}
	return DirectionKeys{Send: mk(r2s), Recv: mk(s2r)}
}

func deriveConfirmKey(sessionKey []byte, sessionID string) []byte {
	salt := sha256.Sum256([]byte("lanhole-v1-confirm|" + sessionID))
	return hkdfSHA256(sessionKey, salt[:], []byte("key confirmation"), 32)
}

func confirm(key []byte, sessionID string, side string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("lanhole-v1-confirm|"))
	mac.Write([]byte(sessionID))
	mac.Write([]byte("|"))
	mac.Write([]byte(side))
	return mac.Sum(nil)
}

func Fingerprint(sessionKey []byte, sessionID string) string {
	salt := sha256.Sum256([]byte("lanhole-v1-fingerprint|" + sessionID))
	fp := hkdfSHA256(sessionKey, salt[:], []byte("human fingerprint"), 8)
	return strings.ToUpper(hex.EncodeToString(fp[:4]) + "-" + hex.EncodeToString(fp[4:]))
}
