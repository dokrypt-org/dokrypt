package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDNSServer_DefaultPort(t *testing.T) {
	d := NewDNSServer(0)
	assert.Equal(t, 15353, d.Port())
	assert.Equal(t, "0.0.0.0", d.BindAddr())
	assert.Equal(t, uint32(60), d.TTL())
	assert.NotNil(t, d.records)
}

func TestNewDNSServer_CustomPort(t *testing.T) {
	d := NewDNSServer(5300)
	assert.Equal(t, 5300, d.Port())
}

func TestDNSServer_SetBindAddr(t *testing.T) {
	d := NewDNSServer(0)
	d.SetBindAddr("127.0.0.1")
	assert.Equal(t, "127.0.0.1", d.BindAddr())
}

func TestDNSServer_SetTTL(t *testing.T) {
	d := NewDNSServer(0)
	d.SetTTL(120)
	assert.Equal(t, uint32(120), d.TTL())
}

func TestDNSServer_Port(t *testing.T) {
	d := NewDNSServer(9999)
	assert.Equal(t, 9999, d.Port())
}

func TestDNSServer_AddRecord(t *testing.T) {
	d := NewDNSServer(0)
	d.AddRecord("myservice.dokrypt.local", "10.0.0.1")
	records := d.ListRecords()
	assert.Equal(t, "10.0.0.1", records["myservice.dokrypt.local"])
}

func TestDNSServer_AddRecord_TrailingDot(t *testing.T) {
	d := NewDNSServer(0)
	d.AddRecord("myservice.dokrypt.local.", "10.0.0.1")
	records := d.ListRecords()
	assert.Equal(t, "10.0.0.1", records["myservice.dokrypt.local"])
	_, hasDot := records["myservice.dokrypt.local."]
	assert.False(t, hasDot)
}

func TestDNSServer_RemoveRecord(t *testing.T) {
	d := NewDNSServer(0)
	d.AddRecord("a.dokrypt.local", "10.0.0.1")
	d.AddRecord("b.dokrypt.local", "10.0.0.2")
	d.RemoveRecord("a.dokrypt.local")
	records := d.ListRecords()
	assert.Len(t, records, 1)
	assert.Equal(t, "10.0.0.2", records["b.dokrypt.local"])
}

func TestDNSServer_RemoveRecord_TrailingDot(t *testing.T) {
	d := NewDNSServer(0)
	d.AddRecord("a.dokrypt.local", "10.0.0.1")
	d.RemoveRecord("a.dokrypt.local.")
	assert.Len(t, d.ListRecords(), 0)
}

func TestDNSServer_ListRecords_Empty(t *testing.T) {
	d := NewDNSServer(0)
	records := d.ListRecords()
	assert.Empty(t, records)
}

func TestDNSServer_ListRecords_IsACopy(t *testing.T) {
	d := NewDNSServer(0)
	d.AddRecord("a.dokrypt.local", "10.0.0.1")
	records := d.ListRecords()
	records["injected"] = "evil"
	assert.Len(t, d.ListRecords(), 1) // original unchanged
}

func TestDNSServer_Resolve_DirectMatch(t *testing.T) {
	d := NewDNSServer(0)
	d.AddRecord("eth.dokrypt.local", "10.0.0.5")

	ip, ok := d.Resolve("eth.dokrypt.local")
	assert.True(t, ok)
	assert.Equal(t, "10.0.0.5", ip)
}

func TestDNSServer_Resolve_TrailingDot(t *testing.T) {
	d := NewDNSServer(0)
	d.AddRecord("eth.dokrypt.local", "10.0.0.5")

	ip, ok := d.Resolve("eth.dokrypt.local.")
	assert.True(t, ok)
	assert.Equal(t, "10.0.0.5", ip)
}

func TestDNSServer_Resolve_WildcardMatch(t *testing.T) {
	d := NewDNSServer(0)
	d.AddRecord("*.dokrypt.local", "10.0.0.99")

	ip, ok := d.Resolve("anything.dokrypt.local")
	assert.True(t, ok)
	assert.Equal(t, "10.0.0.99", ip)
}

func TestDNSServer_Resolve_DirectOverridesWildcard(t *testing.T) {
	d := NewDNSServer(0)
	d.AddRecord("*.dokrypt.local", "10.0.0.99")
	d.AddRecord("specific.dokrypt.local", "10.0.0.1")

	ip, ok := d.Resolve("specific.dokrypt.local")
	assert.True(t, ok)
	assert.Equal(t, "10.0.0.1", ip)
}

func TestDNSServer_Resolve_NotFound(t *testing.T) {
	d := NewDNSServer(0)
	ip, ok := d.Resolve("unknown.host")
	assert.False(t, ok)
	assert.Equal(t, "", ip)
}

func TestDNSServer_Resolve_NoWildcardForSingleLabel(t *testing.T) {
	d := NewDNSServer(0)
	d.AddRecord("*.dokrypt.local", "10.0.0.99")

	_, ok := d.Resolve("singlelabel")
	assert.False(t, ok)
}

func TestDNSServer_StartStop(t *testing.T) {
	d := NewDNSServer(0) // 0 -> default 15353, but we override with an ephemeral approach
	d.SetBindAddr("127.0.0.1")
	d.port = 0 // will be assigned by the OS
	err := d.Start()
	require.NoError(t, err)
	require.NotNil(t, d.listener)

	err = d.Stop()
	assert.NoError(t, err)
}

func TestDNSServer_Stop_NilListener(t *testing.T) {
	d := NewDNSServer(0)
	err := d.Stop()
	assert.NoError(t, err)
}

func TestExtractDNSName_Simple(t *testing.T) {
	data := encodeDNSName("eth.dokrypt.local")
	name := extractDNSName(data)
	assert.Equal(t, "eth.dokrypt.local", name)
}

func TestExtractDNSName_SingleLabel(t *testing.T) {
	data := encodeDNSName("localhost")
	name := extractDNSName(data)
	assert.Equal(t, "localhost", name)
}

func TestExtractDNSName_Empty(t *testing.T) {
	name := extractDNSName([]byte{0x00})
	assert.Equal(t, "", name)
}

func TestExtractDNSName_Truncated(t *testing.T) {
	name := extractDNSName([]byte{0x05, 'a', 'b'})
	assert.Equal(t, "", name)
}

func TestExtractDNSName_EmptyData(t *testing.T) {
	name := extractDNSName([]byte{})
	assert.Equal(t, "", name)
}

func TestBuildDNSResponse_NOERROR(t *testing.T) {
	query := makeDNSQuery("test.dokrypt.local")
	ip := net.ParseIP("10.0.0.1").To4()
	resp := buildDNSResponse(query, ip, 120)
	require.NotNil(t, resp)

	assert.Equal(t, byte(0x81), resp[2])
	assert.Equal(t, byte(0x80), resp[3])
	assert.Equal(t, byte(0x00), resp[6])
	assert.Equal(t, byte(0x01), resp[7])

	answerStart := len(query)
	ttlBytes := resp[answerStart+6 : answerStart+10]
	ttl := uint32(ttlBytes[0])<<24 | uint32(ttlBytes[1])<<16 | uint32(ttlBytes[2])<<8 | uint32(ttlBytes[3])
	assert.Equal(t, uint32(120), ttl)

	assert.Equal(t, ip, net.IP(resp[len(resp)-4:]))
}

func TestBuildDNSResponse_NXDOMAIN(t *testing.T) {
	query := makeDNSQuery("notfound.dokrypt.local")
	resp := buildDNSResponse(query, nil, 60)
	require.NotNil(t, resp)

	assert.Equal(t, byte(0x83), resp[3])
	assert.Equal(t, len(query), len(resp))
}

func TestBuildDNSResponse_TooShort(t *testing.T) {
	resp := buildDNSResponse([]byte{0x00, 0x01}, nil, 60)
	assert.Nil(t, resp)
}

func TestBuildDNSResponse_TTLZero(t *testing.T) {
	query := makeDNSQuery("x.dokrypt.local")
	ip := net.ParseIP("1.2.3.4").To4()
	resp := buildDNSResponse(query, ip, 0)
	require.NotNil(t, resp)
	answerStart := len(query)
	ttlBytes := resp[answerStart+6 : answerStart+10]
	ttl := uint32(ttlBytes[0])<<24 | uint32(ttlBytes[1])<<16 | uint32(ttlBytes[2])<<8 | uint32(ttlBytes[3])
	assert.Equal(t, uint32(0), ttl)
}

func TestBuildDNSResponse_HighTTL(t *testing.T) {
	query := makeDNSQuery("x.dokrypt.local")
	ip := net.ParseIP("1.2.3.4").To4()
	resp := buildDNSResponse(query, ip, 86400)
	require.NotNil(t, resp)
	answerStart := len(query)
	ttlBytes := resp[answerStart+6 : answerStart+10]
	ttl := uint32(ttlBytes[0])<<24 | uint32(ttlBytes[1])<<16 | uint32(ttlBytes[2])<<8 | uint32(ttlBytes[3])
	assert.Equal(t, uint32(86400), ttl)
}

func TestDNSServer_EndToEnd(t *testing.T) {
	d := NewDNSServer(0)
	d.SetBindAddr("127.0.0.1")
	d.port = 0 // ephemeral
	d.AddRecord("myapp.dokrypt.local", "10.0.0.42")

	err := d.Start()
	require.NoError(t, err)
	defer d.Stop()

	localAddr := d.listener.LocalAddr().(*net.UDPAddr)

	conn, err := net.Dial("udp", localAddr.String())
	require.NoError(t, err)
	defer conn.Close()

	query := makeDNSQuery("myapp.dokrypt.local")
	_, err = conn.Write(query)
	require.NoError(t, err)

	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	require.NoError(t, err)

	resp := buf[:n]
	assert.Equal(t, byte(0x80), resp[3])
	assert.Equal(t, net.ParseIP("10.0.0.42").To4(), net.IP(resp[n-4:]))
}

func TestDNSServer_EndToEnd_NXDOMAIN(t *testing.T) {
	d := NewDNSServer(0)
	d.SetBindAddr("127.0.0.1")
	d.port = 0

	err := d.Start()
	require.NoError(t, err)
	defer d.Stop()

	localAddr := d.listener.LocalAddr().(*net.UDPAddr)
	conn, err := net.Dial("udp", localAddr.String())
	require.NoError(t, err)
	defer conn.Close()

	query := makeDNSQuery("unknown.dokrypt.local")
	_, err = conn.Write(query)
	require.NoError(t, err)

	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	require.NoError(t, err)

	resp := buf[:n]
	assert.Equal(t, byte(0x83), resp[3]) // NXDOMAIN
}

func encodeDNSName(name string) []byte {
	var buf []byte
	labels := splitLabels(name)
	for _, l := range labels {
		buf = append(buf, byte(len(l)))
		buf = append(buf, []byte(l)...)
	}
	buf = append(buf, 0x00) // terminator
	return buf
}

func splitLabels(name string) []string {
	var labels []string
	start := 0
	for i := 0; i < len(name); i++ {
		if name[i] == '.' {
			labels = append(labels, name[start:i])
			start = i + 1
		}
	}
	if start < len(name) {
		labels = append(labels, name[start:])
	}
	return labels
}

func makeDNSQuery(name string) []byte {
	header := []byte{
		0xAA, 0xBB, // Transaction ID
		0x01, 0x00, // Flags: standard query
		0x00, 0x01, // Questions: 1
		0x00, 0x00, // Answer RRs: 0
		0x00, 0x00, // Authority RRs: 0
		0x00, 0x00, // Additional RRs: 0
	}
	question := encodeDNSName(name)
	question = append(question, 0x00, 0x01, 0x00, 0x01)
	return append(header, question...)
}
