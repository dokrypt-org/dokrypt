package network

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPortProxy(t *testing.T) {
	p := NewPortProxy()
	assert.NotNil(t, p)
	assert.NotNil(t, p.proxies)
	assert.Empty(t, p.List())
}

func TestPortProxy_Forward_Success(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer backend.Close()
	backendAddr := backend.Addr().String()

	p := NewPortProxy()
	defer p.StopAll()

	hostListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	hostPort := hostListener.Addr().(*net.TCPAddr).Port
	hostListener.Close() // free the port for Forward to use

	meta := ProxyMeta{Service: "rpc", Chain: "eth", Protocol: "tcp"}
	err = p.Forward(context.Background(), hostPort, backendAddr, meta)
	require.NoError(t, err)

	list := p.List()
	require.Len(t, list, 1)
	assert.Equal(t, hostPort, list[0].HostPort)
	assert.Equal(t, backendAddr, list[0].Target)
	assert.Equal(t, meta, list[0].Meta)
	assert.False(t, list[0].StartedAt.IsZero())
}

func TestPortProxy_Forward_DuplicatePort(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer backend.Close()

	p := NewPortProxy()
	defer p.StopAll()

	hostListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	hostPort := hostListener.Addr().(*net.TCPAddr).Port
	hostListener.Close()

	meta := ProxyMeta{Service: "rpc"}
	err = p.Forward(context.Background(), hostPort, backend.Addr().String(), meta)
	require.NoError(t, err)

	err = p.Forward(context.Background(), hostPort, backend.Addr().String(), meta)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("port %d is already in use", hostPort))
}

func TestPortProxy_Forward_InvalidPort(t *testing.T) {
	p := NewPortProxy()
	err := p.Forward(context.Background(), -1, "127.0.0.1:9999", ProxyMeta{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen")
}

func TestPortProxy_Stop_Success(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer backend.Close()

	p := NewPortProxy()
	hostListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	hostPort := hostListener.Addr().(*net.TCPAddr).Port
	hostListener.Close()

	err = p.Forward(context.Background(), hostPort, backend.Addr().String(), ProxyMeta{})
	require.NoError(t, err)

	err = p.Stop(hostPort)
	assert.NoError(t, err)
	assert.Empty(t, p.List())
}

func TestPortProxy_Stop_NotFound(t *testing.T) {
	p := NewPortProxy()
	err := p.Stop(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no proxy on port 99999")
}

func TestPortProxy_StopAll(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer backend.Close()

	p := NewPortProxy()
	for i := 0; i < 3; i++ {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		err = p.Forward(context.Background(), port, backend.Addr().String(), ProxyMeta{Service: fmt.Sprintf("svc-%d", i)})
		require.NoError(t, err)
	}

	assert.Len(t, p.List(), 3)
	err = p.StopAll()
	assert.NoError(t, err)
	assert.Empty(t, p.List())
}

func TestPortProxy_StopAll_Empty(t *testing.T) {
	p := NewPortProxy()
	err := p.StopAll()
	assert.NoError(t, err)
}

func TestPortProxy_List_Empty(t *testing.T) {
	p := NewPortProxy()
	list := p.List()
	assert.NotNil(t, list)
	assert.Len(t, list, 0)
}

func TestPortProxy_List_ContainsMeta(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer backend.Close()

	p := NewPortProxy()
	defer p.StopAll()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	meta := ProxyMeta{Service: "ws", Chain: "sol", Protocol: "tcp"}
	err = p.Forward(context.Background(), port, backend.Addr().String(), meta)
	require.NoError(t, err)

	list := p.List()
	require.Len(t, list, 1)
	assert.Equal(t, "ws", list[0].Meta.Service)
	assert.Equal(t, "sol", list[0].Meta.Chain)
	assert.Equal(t, "tcp", list[0].Meta.Protocol)
}

func TestPortProxy_TrafficForwarding(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer backend.Close()

	go func() {
		for {
			conn, err := backend.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil {
					return
				}
				c.Write(buf[:n])
			}(conn)
		}
	}()

	p := NewPortProxy()
	defer p.StopAll()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	proxyPort := l.Addr().(*net.TCPAddr).Port
	l.Close()

	err = p.Forward(context.Background(), proxyPort, backend.Addr().String(), ProxyMeta{Service: "echo"})
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort), 2*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	msg := []byte("hello proxy")
	_, err = conn.Write(msg)
	require.NoError(t, err)

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "hello proxy", string(buf[:n]))
}

func TestPortProxy_ConnCountIncremented(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer backend.Close()

	go func() {
		for {
			conn, err := backend.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	p := NewPortProxy()
	defer p.StopAll()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	proxyPort := l.Addr().(*net.TCPAddr).Port
	l.Close()

	err = p.Forward(context.Background(), proxyPort, backend.Addr().String(), ProxyMeta{})
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 3; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", proxyPort), time.Second)
		if err != nil {
			continue
		}
		conn.Close()
	}

	time.Sleep(100 * time.Millisecond)

	list := p.List()
	require.Len(t, list, 1)
	assert.GreaterOrEqual(t, list[0].Conns, int64(1))
}

func TestProxyMeta_Fields(t *testing.T) {
	m := ProxyMeta{Service: "rpc", Chain: "eth", Protocol: "grpc"}
	assert.Equal(t, "rpc", m.Service)
	assert.Equal(t, "eth", m.Chain)
	assert.Equal(t, "grpc", m.Protocol)
}

func TestProxyInfo_Fields(t *testing.T) {
	now := time.Now()
	info := ProxyInfo{
		HostPort:  8545,
		Target:    "10.0.0.1:8545",
		Meta:      ProxyMeta{Service: "rpc"},
		StartedAt: now,
		Conns:     42,
	}
	assert.Equal(t, 8545, info.HostPort)
	assert.Equal(t, "10.0.0.1:8545", info.Target)
	assert.Equal(t, "rpc", info.Meta.Service)
	assert.Equal(t, now, info.StartedAt)
	assert.Equal(t, int64(42), info.Conns)
}

func TestPortProxy_Forward_ContextCancel(t *testing.T) {
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer backend.Close()

	p := NewPortProxy()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	err = p.Forward(ctx, port, backend.Addr().String(), ProxyMeta{})
	require.NoError(t, err)

	cancel()
	time.Sleep(100 * time.Millisecond)

	p.StopAll()
}
