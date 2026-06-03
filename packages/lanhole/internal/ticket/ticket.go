package ticket

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net"
	"net/url"
	"strings"
)

type Ticket struct {
	Broker    string
	Transport string
	SessionID string
	Code      string
}

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func New(broker string, transport string) (Ticket, error) {
	sid, err := randomBase32(16)
	if err != nil {
		return Ticket{}, err
	}
	code, err := RandomCode()
	if err != nil {
		return Ticket{}, err
	}
	transport = normalizeTransport(transport)
	return Ticket{Broker: broker, Transport: transport, SessionID: sid, Code: code}, nil
}

func (t Ticket) String() string {
	q := url.Values{}
	if t.Transport != "" {
		q.Set("transport", normalizeTransport(t.Transport))
	}
	u := &url.URL{
		Scheme:   "lanhole",
		Host:     t.Broker,
		Path:     "/" + t.SessionID,
		Fragment: t.Code,
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func Parse(s string, defaultBroker string) (Ticket, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "lanhole://") {
		u, err := url.Parse(s)
		if err != nil {
			return Ticket{}, err
		}
		if u.Scheme != "lanhole" || u.Host == "" || strings.Trim(u.Path, "/") == "" || u.Fragment == "" {
			return Ticket{}, fmt.Errorf("bad lanhole ticket")
		}
		return Ticket{Broker: u.Host, Transport: normalizeTransport(u.Query().Get("transport")), SessionID: strings.Trim(u.Path, "/"), Code: u.Fragment}, nil
	}

	// Compact fallback: SESSION:CODE with broker supplied by --broker.
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 && defaultBroker != "" {
		return Ticket{Broker: defaultBroker, Transport: "tcp", SessionID: parts[0], Code: parts[1]}, nil
	}
	return Ticket{}, fmt.Errorf("expected lanhole://broker/session?transport=tcp|tls#code or SESSION:CODE with --broker")
}

func NormalizeBroker(addr string) string {
	addr = strings.TrimSpace(addr)
	for _, p := range []string{"tcp://", "tls://", "http://", "https://"} {
		addr = strings.TrimPrefix(addr, p)
	}
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return net.JoinHostPort(addr, "8443")
	}
	return addr
}

func normalizeTransport(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if t == "" {
		return "tcp"
	}
	if t != "tcp" && t != "tls" {
		return "tcp"
	}
	return t
}

func randomBase32(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToLower(b32.EncodeToString(b)), nil
}

func RandomCode() (string, error) {
	var raw [2]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	num := 1000 + ((int(raw[0])<<8)|int(raw[1]))%9000
	words := []string{
		"amber", "river", "candle", "forest", "stone", "orbit", "paper", "winter",
		"signal", "harbor", "silver", "garden", "rocket", "window", "pepper", "orange",
		"mirror", "violet", "island", "coffee", "planet", "bridge", "dragon", "pencil",
	}
	pick := func() string {
		var b [1]byte
		_, _ = rand.Read(b[:])
		return words[int(b[0])%len(words)]
	}
	return fmt.Sprintf("%04d-%s-%s-%s", num, pick(), pick(), pick()), nil
}
