package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/actlane/actlane/packages/lanhole/internal/discovery"
	"github.com/actlane/actlane/packages/lanhole/internal/protocol"
	"github.com/actlane/actlane/packages/lanhole/internal/security"
	"github.com/actlane/actlane/packages/lanhole/internal/ticket"
	"github.com/actlane/actlane/packages/lanhole/internal/units"
)

const (
	defaultChunkSize = protocol.ChunkSize
	alpn             = "lanhole/1"
	roleSender       = "sender"
	roleReceiver     = "receiver"
)

type joinRequest struct {
	Version   int    `json:"version"`
	Role      string `json:"role"`
	SessionID string `json:"session_id"`
}

type joinResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type metadata struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	ChunkSize int64  `json:"chunk_size"`
}

type manifestMessage struct {
	ChunkSize int64    `json:"chunk_size"`
	SHA256    string   `json:"sha256"`
	Chunks    []string `json:"chunks"`
}

type acceptMessage struct {
	OK           bool  `json:"ok"`
	ResumeOffset int64 `json:"resume_offset,omitempty"`
}

type finishMessage struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type errorMessage struct {
	Message string `json:"message"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "send":
		err = runSend(os.Args[2:])
	case "recv":
		err = runRecv(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `lanhole file transfer

Usage:
  lanhole send ./actlane-pack.zip
  lanhole recv 123-orange-river-candle --out ./downloads
  lanhole send --transport tls --broker relay.example.com:443 ./file.tar.zst
  lanhole recv 'lanhole://relay.example.com:443/SESSION?transport=tls#CODE'

Commands:
  send     send a file over LAN by default or through a broker with --broker
  recv     receive a LAN code or a lanhole:// broker ticket

`)
}

func runSend(args []string) error {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	brokerAddr := fs.String("broker", "", "relay broker host:port; empty uses LAN discovery")
	transport := fs.String("transport", "tcp", "transport: tcp, tls")
	insecureTLS := fs.Bool("insecure-skip-verify", false, "skip broker TLS verification; local tests only")
	caFile := fs.String("ca-file", "", "extra CA PEM file for tls broker verification")
	code := fs.String("code", "", "override human code for tests only")
	maxSizeRaw := fs.String("max-size", "10GiB", "max plaintext file size; 0 disables")
	ttl := fs.Duration("ttl", 5*time.Minute, "LAN sender wait time; ignored when --broker is set")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: lanhole send --broker host:port ./file")
	}
	maxSize, err := units.ParseBytes(*maxSizeRaw)
	if err != nil {
		return fmt.Errorf("bad --max-size: %w", err)
	}

	path := fs.Arg(0)
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return fmt.Errorf("directories are not supported; archive it first")
	}
	if maxSize > 0 && st.Size() > maxSize {
		return fmt.Errorf("file size %d exceeds --max-size %s", st.Size(), units.FormatBytes(maxSize))
	}
	if strings.TrimSpace(*brokerAddr) == "" {
		return runLanSend(path, st, *code, *ttl)
	}

	br := ticket.NormalizeBroker(*brokerAddr)
	t, err := ticket.New(br, *transport)
	if err != nil {
		return err
	}
	if *code != "" {
		t.Code = security.NormalizeCode(*code)
	}

	fmt.Println("Share this ticket with receiver:")
	fmt.Println("  " + t.String())
	fmt.Println()
	fmt.Printf("Connecting to relay broker %s as sender via %s; waiting for receiver...\n", t.Broker, t.Transport)
	conn, err := joinRelay(context.Background(), t.Broker, t.Transport, t.SessionID, roleSender, *insecureTLS, *caFile)
	if err != nil {
		return err
	}
	defer conn.Close()
	return sendSecure(conn, t.SessionID, t.Code, path, st)
}

func runLanSend(path string, st os.FileInfo, code string, ttl time.Duration) error {
	if code == "" {
		code = generateCode()
	}
	code = security.NormalizeCode(code)
	sessionID := randomHex(16)
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	defer ln.Close()
	tcpPort := ln.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithTimeout(context.Background(), ttl)
	defer cancel()
	mdns, err := discovery.RegisterMDNS(sessionID, tcpPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: mDNS registration failed, UDP fallback only: %v\n", err)
	} else {
		defer mdns.Shutdown()
	}
	go func() {
		if err := discovery.ServeUDP(ctx, sessionID, tcpPort); err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "warning: UDP fallback disabled: %v\n", err)
		}
	}()
	fmt.Printf("Code: %s\n", code)
	fmt.Printf("Session: %s, TCP port: %d, expires: %s\n", sessionID, tcpPort, time.Now().Add(ttl).Format(time.RFC3339))
	fmt.Println("Waiting for receiver on LAN...")
	connCh := make(chan net.Conn, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			errCh <- err
			return
		}
		connCh <- conn
	}()
	select {
	case <-ctx.Done():
		return errors.New("session expired")
	case err := <-errCh:
		return err
	case conn := <-connCh:
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
		if err := sendSecure(conn, sessionID, code, path, st); err != nil {
			return err
		}
		_ = conn.SetDeadline(time.Time{})
		return nil
	}
}

func sendSecure(conn io.ReadWriteCloser, sessionID, code, path string, st os.FileInfo) error {
	fmt.Println("Preparing chunk manifest...")
	manifest, err := computeManifest(path, defaultChunkSize)
	if err != nil {
		return err
	}
	keys, fp, err := security.SenderHandshake(conn, sessionID, code)
	if err != nil {
		return err
	}
	if d, ok := conn.(interface{ SetDeadline(time.Time) error }); ok {
		_ = d.SetDeadline(time.Time{})
	}
	fmt.Println("Secure session established. Fingerprint:", fp)
	stream, err := security.NewSecureStream(conn, keys)
	if err != nil {
		return err
	}

	meta := metadata{Name: filepath.Base(path), Size: st.Size(), ChunkSize: defaultChunkSize}
	if err := stream.WriteJSON(security.KindMetadata, meta); err != nil {
		return fmt.Errorf("send metadata: %w", err)
	}
	if err := stream.WriteJSON(security.KindManifest, manifest); err != nil {
		return fmt.Errorf("send manifest: %w", err)
	}

	kind, payload, err := stream.Read()
	if err != nil {
		return fmt.Errorf("read receiver accept: %w", err)
	}
	if kind == security.KindError {
		return fmt.Errorf("receiver rejected: %s", errorText(payload))
	}
	if kind != security.KindAccept {
		return fmt.Errorf("expected accept frame, got kind %d", kind)
	}
	var acc acceptMessage
	if err := json.Unmarshal(payload, &acc); err != nil {
		return err
	}
	if !acc.OK {
		return fmt.Errorf("receiver did not accept transfer")
	}
	if acc.ResumeOffset < 0 || acc.ResumeOffset > st.Size() || acc.ResumeOffset%defaultChunkSize != 0 {
		return fmt.Errorf("bad receiver resume offset: %d", acc.ResumeOffset)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if acc.ResumeOffset > 0 {
		if _, err := f.Seek(acc.ResumeOffset, io.SeekStart); err != nil {
			return err
		}
		fmt.Printf("Receiver already has verified prefix: %d bytes\n", acc.ResumeOffset)
	}

	buf := make([]byte, defaultChunkSize)
	sent := acc.ResumeOffset
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if err := stream.Write(security.KindChunk, chunk); err != nil {
				return fmt.Errorf("send chunk: %w", err)
			}
			sent += int64(n)
			printProgress("sent", sent, st.Size())
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	fmt.Println()
	fin := finishMessage{SHA256: manifest.SHA256, Size: st.Size()}
	if err := stream.WriteJSON(security.KindFinish, fin); err != nil {
		return fmt.Errorf("send finish: %w", err)
	}
	fmt.Printf("Done. sha256=%s\n", fin.SHA256)
	return nil
}

func runRecv(args []string) error {
	fs := flag.NewFlagSet("recv", flag.ExitOnError)
	defaultBroker := fs.String("broker", "", "broker for compact SESSION:CODE form")
	transportOverride := fs.String("transport", "", "override ticket transport: tcp, tls")
	insecureTLS := fs.Bool("insecure-skip-verify", false, "skip broker TLS verification; local tests only")
	caFile := fs.String("ca-file", "", "extra CA PEM file for tls broker verification")
	outDir := fs.String("out", ".", "output directory")
	yes := fs.Bool("yes", false, "accept without interactive prompt")
	overwrite := fs.Bool("overwrite", false, "allow overwriting existing target file")
	resume := fs.Bool("resume", true, "resume from existing .part file after chunk-hash prefix verification")
	maxSizeRaw := fs.String("max-size", "10GiB", "max plaintext file size; 0 disables")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: lanhole recv 'lanhole://broker/session?transport=tls#code'")
	}
	maxSize, err := units.ParseBytes(*maxSizeRaw)
	if err != nil {
		return fmt.Errorf("bad --max-size: %w", err)
	}

	input := fs.Arg(0)
	if *defaultBroker == "" && !strings.HasPrefix(input, "lanhole://") {
		return runLanRecv(input, *outDir, *yes, *overwrite, *resume, maxSize)
	}

	br := ""
	if *defaultBroker != "" {
		br = ticket.NormalizeBroker(*defaultBroker)
	}
	t, err := ticket.Parse(input, br)
	if err != nil {
		return err
	}
	if *transportOverride != "" {
		t.Transport = strings.ToLower(*transportOverride)
	}
	fmt.Printf("Connecting to relay broker %s as receiver via %s...\n", t.Broker, t.Transport)
	conn, err := joinRelay(context.Background(), t.Broker, t.Transport, t.SessionID, roleReceiver, *insecureTLS, *caFile)
	if err != nil {
		return err
	}
	defer conn.Close()

	keys, fp, err := security.ReceiverHandshake(conn, t.SessionID, t.Code)
	if err != nil {
		return err
	}
	fmt.Println("Secure session established. Fingerprint:", fp)
	stream, err := security.NewSecureStream(conn, keys)
	if err != nil {
		return err
	}
	return receiveSecure(stream, *outDir, *yes, *overwrite, *resume, maxSize)
}

func runLanRecv(code, outDir string, yes, overwrite, resume bool, maxSize int64) error {
	code = security.NormalizeCode(code)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fmt.Println("Discovering senders by mDNS...")
	offers, _ := discovery.BrowseMDNS(ctx)
	if len(offers) == 0 {
		fmt.Println("No mDNS offers found; trying UDP broadcast fallback...")
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		offers, _ = discovery.DiscoverUDP(ctx2)
	}
	if len(offers) == 0 {
		return errors.New("no LAN offers found")
	}
	var lastErr error
	for _, offer := range offers {
		fmt.Printf("Trying %s offer %s at %s:%d...\n", offer.Source, offer.SessionID, offer.Addr, offer.Port)
		if err := recvLanOffer(offer, code, outDir, yes, overwrite, resume, maxSize); err != nil {
			lastErr = err
			fmt.Fprintf(os.Stderr, "offer failed: %v\n", err)
			continue
		}
		return nil
	}
	return fmt.Errorf("all offers failed; last error: %w", lastErr)
}

func recvLanOffer(offer discovery.Offer, code, outDir string, yes, overwrite, resume bool, maxSize int64) error {
	addr := net.JoinHostPort(offer.Addr, fmt.Sprintf("%d", offer.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	keys, fp, err := security.ReceiverHandshake(conn, offer.SessionID, code)
	if err != nil {
		return err
	}
	_ = conn.SetDeadline(time.Time{})
	fmt.Println("Secure session established. Fingerprint:", fp)
	stream, err := security.NewSecureStream(conn, keys)
	if err != nil {
		return err
	}
	return receiveSecure(stream, outDir, yes, overwrite, resume, maxSize)
}

func receiveSecure(stream *security.SecureStream, outDir string, yes, overwrite, resume bool, maxSize int64) error {
	var meta metadata
	if err := readJSONKind(stream, security.KindMetadata, &meta); err != nil {
		return err
	}
	var manifest manifestMessage
	if err := readJSONKind(stream, security.KindManifest, &manifest); err != nil {
		return err
	}
	if meta.Size < 0 || meta.ChunkSize <= 0 || manifest.ChunkSize != meta.ChunkSize {
		_ = stream.Write(security.KindError, mustJSON(errorMessage{Message: "bad metadata/manifest"}))
		return fmt.Errorf("bad metadata/manifest")
	}
	if maxSize > 0 && meta.Size > maxSize {
		_ = stream.Write(security.KindError, mustJSON(errorMessage{Message: "file exceeds receiver max size"}))
		return fmt.Errorf("incoming file size %d exceeds --max-size %s", meta.Size, units.FormatBytes(maxSize))
	}
	name := safeBaseName(meta.Name)
	if name == "" {
		_ = stream.Write(security.KindError, mustJSON(errorMessage{Message: "unsafe file name"}))
		return fmt.Errorf("empty or unsafe file name")
	}
	if int64(len(manifest.Chunks))*meta.ChunkSize < meta.Size && meta.Size > 0 {
		_ = stream.Write(security.KindError, mustJSON(errorMessage{Message: "manifest too short"}))
		return fmt.Errorf("manifest too short")
	}
	target := filepath.Join(outDir, name)
	part := target + ".part"
	if !overwrite && exists(target) {
		_ = stream.Write(security.KindError, mustJSON(errorMessage{Message: "target exists and overwrite is disabled"}))
		return fmt.Errorf("target exists: %s", target)
	}

	resumeOffset := int64(0)
	var err error
	if exists(part) {
		if resume && !overwrite {
			resumeOffset, err = verifiedPrefix(part, manifest, meta.Size)
			if err != nil {
				_ = stream.Write(security.KindError, mustJSON(errorMessage{Message: "partial file verification failed"}))
				return err
			}
			if resumeOffset == 0 {
				return fmt.Errorf("partial file exists but no valid chunk prefix found: %s", part)
			}
			if err := os.Truncate(part, resumeOffset); err != nil {
				return err
			}
		} else if !overwrite {
			_ = stream.Write(security.KindError, mustJSON(errorMessage{Message: "partial file exists"}))
			return fmt.Errorf("partial file exists: %s", part)
		}
	}

	fmt.Printf("Incoming file: %s (%d bytes)\n", name, meta.Size)
	if resumeOffset > 0 {
		fmt.Printf("Resume: verified %d existing bytes in %s\n", resumeOffset, part)
	}
	if !yes && !askYesNo("Accept transfer? [y/N] ") {
		_ = stream.Write(security.KindError, mustJSON(errorMessage{Message: "receiver declined"}))
		return fmt.Errorf("declined")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := stream.WriteJSON(security.KindAccept, acceptMessage{OK: true, ResumeOffset: resumeOffset}); err != nil {
		return fmt.Errorf("send accept: %w", err)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if resumeOffset > 0 {
		flags |= os.O_APPEND
	} else if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(part, flags, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	received := resumeOffset
	var final finishMessage
	for {
		kind, payload, err := stream.Read()
		if err != nil {
			return fmt.Errorf("read encrypted stream: %w", err)
		}
		switch kind {
		case security.KindChunk:
			if _, err := f.Write(payload); err != nil {
				return err
			}
			received += int64(len(payload))
			printProgress("received", received, meta.Size)
		case security.KindFinish:
			if err := json.Unmarshal(payload, &final); err != nil {
				return err
			}
			goto done
		case security.KindError:
			return fmt.Errorf("sender error: %s", errorText(payload))
		default:
			return fmt.Errorf("unexpected encrypted frame kind: %d", kind)
		}
	}

done:
	fmt.Println()
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if received != final.Size || received != meta.Size {
		return fmt.Errorf("size mismatch: received=%d metadata=%d final=%d", received, meta.Size, final.Size)
	}
	actual, err := fileSHA256(part)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, final.SHA256) || !strings.EqualFold(actual, manifest.SHA256) {
		return fmt.Errorf("sha256 mismatch: got=%s want=%s", actual, final.SHA256)
	}
	if overwrite {
		_ = os.Remove(target)
	}
	if err := os.Rename(part, target); err != nil {
		return err
	}
	fmt.Printf("Done. saved=%s sha256=%s\n", target, actual)
	return nil
}

type relayIO interface {
	io.ReadWriteCloser
	SetDeadline(time.Time) error
}

func joinRelay(ctx context.Context, addr, transport, sessionID, role string, insecureTLS bool, caFile string) (relayIO, error) {
	transport = strings.ToLower(strings.TrimSpace(transport))
	if transport == "" {
		transport = "tcp"
	}
	var conn relayIO
	var err error
	switch transport {
	case "tcp":
		dialer := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		c, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		conn = c
	case "tls":
		host, _, _ := net.SplitHostPort(addr)
		tlsConf, err := clientTLSConfig(host, insecureTLS, caFile)
		if err != nil {
			return nil, err
		}
		dialer := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
		c, err := tls.DialWithDialer(&dialer, "tcp", addr, tlsConf)
		if err != nil {
			return nil, err
		}
		conn = c
	default:
		return nil, fmt.Errorf("unsupported transport %q", transport)
	}

	req := joinRequest{Version: 1, Role: role, SessionID: sessionID}
	b, _ := json.Marshal(req)
	if err := protocol.WriteFrame(conn, b); err != nil {
		_ = conn.Close()
		return nil, err
	}
	payload, err := protocol.ReadFrame(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	var resp joinResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if resp.Status != "paired" {
		_ = conn.Close()
		if resp.Message == "" {
			resp.Message = "join failed"
		}
		return nil, fmt.Errorf("broker: %s", resp.Message)
	}
	return conn, nil
}

func clientTLSConfig(serverName string, insecureSkipVerify bool, caFile string) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		ServerName:         serverName,
		NextProtos:         []string{alpn},
		InsecureSkipVerify: insecureSkipVerify,
	}
	if caFile == "" {
		return cfg, nil
	}
	certPool, err := x509.SystemCertPool()
	if err != nil || certPool == nil {
		certPool = x509.NewCertPool()
	}
	pemBytes, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	certPool.AppendCertsFromPEM(pemBytes)
	cfg.RootCAs = certPool
	return cfg, nil
}

func computeManifest(path string, chunkSize int64) (manifestMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return manifestMessage{}, err
	}
	defer f.Close()
	full := sha256.New()
	buf := make([]byte, int(chunkSize))
	var chunks []string
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			_, _ = full.Write(chunk)
			ch := sha256.Sum256(chunk)
			chunks = append(chunks, hex.EncodeToString(ch[:]))
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			return manifestMessage{}, rerr
		}
	}
	return manifestMessage{ChunkSize: chunkSize, SHA256: hex.EncodeToString(full.Sum(nil)), Chunks: chunks}, nil
}

func verifiedPrefix(path string, manifest manifestMessage, totalSize int64) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var offset int64
	for _, want := range manifest.Chunks {
		remaining := totalSize - offset
		if remaining <= 0 {
			return offset, nil
		}
		expected := manifest.ChunkSize
		if remaining < expected {
			expected = remaining
		}
		buf := make([]byte, int(expected))
		if _, err := io.ReadFull(f, buf); err != nil {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return offset, nil
			}
			return offset, err
		}
		ch := sha256.Sum256(buf)
		if !strings.EqualFold(hex.EncodeToString(ch[:]), want) {
			return offset, nil
		}
		offset += expected
	}
	return offset, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func readJSONKind(stream *security.SecureStream, want security.FrameKind, out any) error {
	kind, payload, err := stream.Read()
	if err != nil {
		return err
	}
	if kind == security.KindError {
		return fmt.Errorf("peer error: %s", errorText(payload))
	}
	if kind != want {
		return fmt.Errorf("expected frame kind %d, got %d", want, kind)
	}
	return json.Unmarshal(payload, out)
}

func safeBaseName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return ""
	}
	return name
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func askYesNo(prompt string) bool {
	fmt.Print(prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes" || line == "д" || line == "да"
}

func printProgress(prefix string, done, total int64) {
	if total <= 0 {
		fmt.Printf("\r%s %d bytes", prefix, done)
		return
	}
	pct := float64(done) * 100 / float64(total)
	fmt.Printf("\r%s %d/%d bytes %.1f%%", prefix, done, total, pct)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func errorText(b []byte) string {
	var m errorMessage
	if err := json.Unmarshal(b, &m); err == nil && m.Message != "" {
		return m.Message
	}
	return string(b)
}

func generateCode() string {
	adjectives := []string{"orange", "river", "quiet", "green", "silver", "north", "bright", "cedar"}
	nouns := []string{"candle", "field", "stone", "window", "harbor", "signal", "valley", "bridge"}
	return fmt.Sprintf("%04d-%s-%s", randInt(10000), adjectives[randInt(len(adjectives))], nouns[randInt(len(nouns))])
}

func randInt(max int) int {
	if max <= 0 {
		return 0
	}
	var b [1]byte
	if _, err := rand.Read(b[:]); err != nil {
		return int(time.Now().UnixNano() % int64(max))
	}
	return int(b[0]) % max
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
