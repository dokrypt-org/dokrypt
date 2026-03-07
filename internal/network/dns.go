package network

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
)

type DNSServer struct {
	records  map[string]string // hostname -> IP
	mu       sync.RWMutex
	port     int
	bindAddr string // IP address to bind to (default "0.0.0.0")
	ttl      uint32 // TTL in seconds for DNS responses (default 60)
	listener net.PacketConn
	done     chan struct{}
}

func NewDNSServer(port int) *DNSServer {
	if port == 0 {
		port = 15353
	}
	return &DNSServer{
		records:  make(map[string]string),
		port:     port,
		bindAddr: "0.0.0.0",
		ttl:      60,
		done:     make(chan struct{}),
	}
}

func (d *DNSServer) SetBindAddr(addr string) {
	d.bindAddr = addr
}

func (d *DNSServer) SetTTL(ttl uint32) {
	d.ttl = ttl
}

func (d *DNSServer) BindAddr() string {
	return d.bindAddr
}

func (d *DNSServer) TTL() uint32 {
	return d.ttl
}

func (d *DNSServer) Start() error {
	addr := fmt.Sprintf("%s:%d", d.bindAddr, d.port)
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start DNS server on %s: %w", addr, err)
	}
	d.listener = conn
	slog.Info("DNS server started", "addr", addr)

	go d.serve()
	return nil
}

func (d *DNSServer) Stop() error {
	close(d.done)
	if d.listener != nil {
		return d.listener.Close()
	}
	return nil
}

func (d *DNSServer) serve() {
	buf := make([]byte, 512)
	for {
		select {
		case <-d.done:
			return
		default:
		}

		n, addr, err := d.listener.ReadFrom(buf)
		if err != nil {
			select {
			case <-d.done:
				return
			default:
				slog.Debug("DNS read error", "error", err)
				continue
			}
		}

		go d.handleQuery(buf[:n], addr)
	}
}

func (d *DNSServer) handleQuery(query []byte, addr net.Addr) {
	if len(query) < 12 {
		return
	}

	name := extractDNSName(query[12:])
	if name == "" {
		return
	}

	ip, found := d.Resolve(name)
	if !found {
		resp := buildDNSResponse(query, nil, d.ttl)
		d.listener.WriteTo(resp, addr)
		return
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return
	}

	resp := buildDNSResponse(query, parsedIP.To4(), d.ttl)
	d.listener.WriteTo(resp, addr)
}

func (d *DNSServer) AddRecord(hostname, ip string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	hostname = strings.TrimSuffix(hostname, ".")
	d.records[hostname] = ip
	slog.Debug("DNS record added", "hostname", hostname, "ip", ip)
}

func (d *DNSServer) RemoveRecord(hostname string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	hostname = strings.TrimSuffix(hostname, ".")
	delete(d.records, hostname)
}

func (d *DNSServer) Resolve(hostname string) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	hostname = strings.TrimSuffix(hostname, ".")

	if ip, ok := d.records[hostname]; ok {
		return ip, true
	}

	parts := strings.SplitN(hostname, ".", 2)
	if len(parts) == 2 {
		wildcard := "*." + parts[1]
		if ip, ok := d.records[wildcard]; ok {
			return ip, true
		}
	}

	return "", false
}

func (d *DNSServer) ListRecords() map[string]string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make(map[string]string, len(d.records))
	for k, v := range d.records {
		result[k] = v
	}
	return result
}

func (d *DNSServer) Port() int {
	return d.port
}

func extractDNSName(data []byte) string {
	var parts []string
	i := 0
	for i < len(data) {
		length := int(data[i])
		if length == 0 {
			break
		}
		i++
		if i+length > len(data) {
			return ""
		}
		parts = append(parts, string(data[i:i+length]))
		i += length
	}
	return strings.Join(parts, ".")
}

func buildDNSResponse(query []byte, ip net.IP, ttl uint32) []byte {
	if len(query) < 12 {
		return nil
	}

	resp := make([]byte, len(query))
	copy(resp, query)

	resp[2] = 0x81 // QR=1, Opcode=0, AA=1
	if ip == nil {
		resp[3] = 0x83 // RCODE=3 (NXDOMAIN)
		return resp
	}
	resp[3] = 0x80 // RCODE=0 (NOERROR)

	resp[6] = 0x00
	resp[7] = 0x01

	answer := []byte{
		0xC0, 0x0C, // Pointer to name in question
		0x00, 0x01, // Type A
		0x00, 0x01, // Class IN
		byte(ttl >> 24), byte(ttl >> 16), byte(ttl >> 8), byte(ttl), // TTL
		0x00, 0x04, // RDLENGTH = 4
	}
	answer = append(answer, ip...)
	resp = append(resp, answer...)

	return resp
}
