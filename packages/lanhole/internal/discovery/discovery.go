package discovery

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	Service = "_lanhole._tcp"
	Domain  = "local."
	UDPPort = 37373
)

type Offer struct {
	SessionID string
	Addr      string
	Port      int
	Source    string
}

func RegisterMDNS(sessionID string, port int) (*zeroconf.Server, error) {
	instance := "lanhole-" + sessionID
	text := []string{
		"v=1",
		"sid=" + sessionID,
		"note=metadata-after-pake",
	}
	srv, err := zeroconf.Register(instance, Service, Domain, port, text, nil)
	if err != nil {
		return nil, fmt.Errorf("register mdns: %w", err)
	}
	srv.TTL(120)
	return srv, nil
}

func BrowseMDNS(ctx context.Context) ([]Offer, error) {
	resolver, err := zeroconf.NewResolver()
	if err != nil {
		return nil, fmt.Errorf("new mdns resolver: %w", err)
	}
	entries := make(chan *zeroconf.ServiceEntry)
	var offers []Offer
	go func() {
		for e := range entries {
			sid := textValue(e.Text, "sid")
			if sid == "" || e.Port <= 0 {
				continue
			}
			addr := firstAddr(e)
			if addr == "" {
				continue
			}
			offers = append(offers, Offer{SessionID: sid, Addr: addr, Port: e.Port, Source: "mdns"})
		}
	}()
	if err := resolver.Browse(ctx, Service, Domain, entries); err != nil {
		return nil, fmt.Errorf("browse mdns: %w", err)
	}
	<-ctx.Done()
	return offers, nil
}

func firstAddr(e *zeroconf.ServiceEntry) string {
	if len(e.AddrIPv4) > 0 {
		return e.AddrIPv4[0].String()
	}
	if len(e.AddrIPv6) > 0 {
		return e.AddrIPv6[0].String()
	}
	return ""
}

func textValue(txt []string, key string) string {
	prefix := key + "="
	for _, s := range txt {
		if strings.HasPrefix(s, prefix) {
			return strings.TrimPrefix(s, prefix)
		}
	}
	return ""
}

func ServeUDP(ctx context.Context, sessionID string, tcpPort int) error {
	pc, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", UDPPort))
	if err != nil {
		return fmt.Errorf("udp fallback listen: %w", err)
	}
	defer pc.Close()
	go func() {
		<-ctx.Done()
		_ = pc.Close()
	}()
	buf := make([]byte, 1024)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		msg := strings.TrimSpace(string(buf[:n]))
		if msg != "LANHOLE_DISCOVER_V1" {
			continue
		}
		resp := fmt.Sprintf("LANHOLE_OFFER_V1 sid=%s port=%d", sessionID, tcpPort)
		_, _ = pc.WriteTo([]byte(resp), addr)
	}
}

func DiscoverUDP(ctx context.Context) ([]Offer, error) {
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return nil, fmt.Errorf("udp fallback listen: %w", err)
	}
	defer conn.Close()
	udp, ok := conn.(*net.UDPConn)
	if !ok {
		return nil, fmt.Errorf("not udp conn")
	}
	if err := udp.SetWriteBuffer(2048); err != nil {
		return nil, err
	}
	broadcast := &net.UDPAddr{IP: net.IPv4bcast, Port: UDPPort}
	for i := 0; i < 3; i++ {
		_, _ = udp.WriteTo([]byte("LANHOLE_DISCOVER_V1"), broadcast)
		time.Sleep(200 * time.Millisecond)
	}
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	seen := map[string]bool{}
	var offers []Offer
	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return offers, nil
			}
			return offers, err
		}
		msg := strings.TrimSpace(string(buf[:n]))
		if !strings.HasPrefix(msg, "LANHOLE_OFFER_V1 ") {
			continue
		}
		sid, port := parseUDPResponse(msg)
		if sid == "" || port == 0 {
			continue
		}
		host, _, err := net.SplitHostPort(addr.String())
		if err != nil {
			continue
		}
		key := sid + "@" + host + ":" + strconv.Itoa(port)
		if seen[key] {
			continue
		}
		seen[key] = true
		offers = append(offers, Offer{SessionID: sid, Addr: host, Port: port, Source: "udp"})
	}
}

func parseUDPResponse(msg string) (string, int) {
	var sid string
	var port int
	for _, field := range strings.Fields(msg) {
		if strings.HasPrefix(field, "sid=") {
			sid = strings.TrimPrefix(field, "sid=")
		}
		if strings.HasPrefix(field, "port=") {
			p, _ := strconv.Atoi(strings.TrimPrefix(field, "port="))
			port = p
		}
	}
	return sid, port
}
