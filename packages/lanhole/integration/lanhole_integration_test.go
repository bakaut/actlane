package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLanSendRecvLocalhost(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	root := filepath.Clean("..")
	bin := filepath.Join(t.TempDir(), "lanhole")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/lanhole")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	inDir := t.TempDir()
	outDir := t.TempDir()
	input := filepath.Join(inDir, "actlane-pack.zip")
	content := []byte(strings.Repeat("lanhole-lan-integration-test\n", 4096))
	if err := os.WriteFile(input, content, 0o600); err != nil {
		t.Fatal(err)
	}

	send := exec.CommandContext(ctx, bin, "send", "--code", "123-orange-river", input)
	var sendOut bytes.Buffer
	send.Stdout = &sendOut
	send.Stderr = &sendOut
	if err := send.Start(); err != nil {
		t.Fatalf("start send: %v", err)
	}
	defer func() {
		_ = send.Process.Kill()
		_ = send.Wait()
	}()

	time.Sleep(time.Second)

	recv := exec.CommandContext(ctx, bin, "recv", "--yes", "--out", outDir, "123-orange-river")
	var recvOut bytes.Buffer
	recv.Stdout = &recvOut
	recv.Stderr = &recvOut
	if err := recv.Run(); err != nil {
		t.Fatalf("recv failed: %v\nsend:\n%s\nrecv:\n%s", err, sendOut.String(), recvOut.String())
	}
	if err := send.Wait(); err != nil {
		t.Fatalf("send failed: %v\n%s", err, sendOut.String())
	}

	got, err := os.ReadFile(filepath.Join(outDir, "actlane-pack.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("received payload differs")
	}
	h := sha256.Sum256(got)
	if !strings.Contains(recvOut.String(), hex.EncodeToString(h[:])) {
		t.Fatalf("receiver output did not contain final sha256; output=%s", recvOut.String())
	}
}
