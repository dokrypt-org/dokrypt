package network

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

type ProxyMeta struct {
	Service   string
	Chain     string
	Protocol  string
}

type ProxyInfo struct {
	HostPort  int
	Target    string
	Meta      ProxyMeta
	StartedAt time.Time
	Conns     int64
}

type activeProxy struct {
	listener  net.Listener
	target    string
	meta      ProxyMeta
	startedAt time.Time
	conns     int64
	cancel    context.CancelFunc
	mu        sync.Mutex
}

type PortProxy struct {
	proxies map[int]*activeProxy
	mu      sync.RWMutex
}

func NewPortProxy() *PortProxy {
	return &PortProxy{
		proxies: make(map[int]*activeProxy),
	}
}

func (p *PortProxy) Forward(ctx context.Context, hostPort int, target string, meta ProxyMeta) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.proxies[hostPort]; exists {
		return fmt.Errorf("port %d is already in use by proxy", hostPort)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", hostPort))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", hostPort, err)
	}

	proxyCtx, cancel := context.WithCancel(ctx)
	ap := &activeProxy{
		listener:  listener,
		target:    target,
		meta:      meta,
		startedAt: time.Now(),
		cancel:    cancel,
	}
	p.proxies[hostPort] = ap

	slog.Info("port proxy started", "host_port", hostPort, "target", target, "service", meta.Service)

	go func() {
		defer listener.Close()
		for {
			select {
			case <-proxyCtx.Done():
				return
			default:
			}

			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-proxyCtx.Done():
					return
				default:
					slog.Debug("proxy accept error", "port", hostPort, "error", err)
					continue
				}
			}

			ap.mu.Lock()
			ap.conns++
			ap.mu.Unlock()

			go handleProxyConn(proxyCtx, conn, target)
		}
	}()

	return nil
}

func handleProxyConn(ctx context.Context, client net.Conn, target string) {
	defer client.Close()

	backend, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		slog.Debug("proxy backend dial failed", "target", target, "error", err)
		return
	}
	defer backend.Close()

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(backend, client)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(client, backend)
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}

func (p *PortProxy) Stop(hostPort int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	ap, ok := p.proxies[hostPort]
	if !ok {
		return fmt.Errorf("no proxy on port %d", hostPort)
	}

	ap.cancel()
	ap.listener.Close()
	delete(p.proxies, hostPort)
	slog.Info("port proxy stopped", "host_port", hostPort)
	return nil
}

func (p *PortProxy) StopAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for port, ap := range p.proxies {
		ap.cancel()
		ap.listener.Close()
		delete(p.proxies, port)
	}
	return nil
}

func (p *PortProxy) List() []ProxyInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]ProxyInfo, 0, len(p.proxies))
	for port, ap := range p.proxies {
		ap.mu.Lock()
		conns := ap.conns
		ap.mu.Unlock()
		result = append(result, ProxyInfo{
			HostPort:  port,
			Target:    ap.target,
			Meta:      ap.meta,
			StartedAt: ap.startedAt,
			Conns:     conns,
		})
	}
	return result
}
