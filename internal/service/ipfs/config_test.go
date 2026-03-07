package ipfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultIPFSConfig(t *testing.T) {
	cfg := DefaultIPFSConfig()
	require.NotNil(t, cfg)

	assert.Equal(t, 5001, cfg.APIPort)
	assert.Equal(t, 8080, cfg.GatewayPort)
	assert.True(t, cfg.EnableFilestore)
}

func TestInitCommands(t *testing.T) {
	cfg := DefaultIPFSConfig()
	cmds := cfg.InitCommands()

	require.NotEmpty(t, cmds)
	assert.Len(t, cmds, 7)

	for _, cmd := range cmds {
		require.GreaterOrEqual(t, len(cmd), 4)
		assert.Equal(t, "ipfs", cmd[0])
		assert.Equal(t, "config", cmd[1])
		assert.Equal(t, "--json", cmd[2])
	}

	t.Run("CORS allow origin", func(t *testing.T) {
		cmd := cmds[0]
		assert.Equal(t, "API.HTTPHeaders.Access-Control-Allow-Origin", cmd[3])
		assert.Equal(t, `["*"]`, cmd[4])
	})

	t.Run("CORS allow methods", func(t *testing.T) {
		cmd := cmds[1]
		assert.Equal(t, "API.HTTPHeaders.Access-Control-Allow-Methods", cmd[3])
		assert.Equal(t, `["PUT","POST","GET"]`, cmd[4])
	})

	t.Run("URL store enabled", func(t *testing.T) {
		cmd := cmds[2]
		assert.Equal(t, "Experimental.UrlstoreEnabled", cmd[3])
		assert.Equal(t, "true", cmd[4])
	})

	t.Run("filestore enabled", func(t *testing.T) {
		cmd := cmds[3]
		assert.Equal(t, "Experimental.FilestoreEnabled", cmd[3])
		assert.Equal(t, "true", cmd[4])
	})

	t.Run("disable NAT port map", func(t *testing.T) {
		cmd := cmds[4]
		assert.Equal(t, "Swarm.DisableNatPortMap", cmd[3])
		assert.Equal(t, "true", cmd[4])
	})

	t.Run("relay client disabled", func(t *testing.T) {
		cmd := cmds[5]
		assert.Equal(t, "Swarm.RelayClient.Enabled", cmd[3])
		assert.Equal(t, "false", cmd[4])
	})

	t.Run("relay service disabled", func(t *testing.T) {
		cmd := cmds[6]
		assert.Equal(t, "Swarm.RelayService.Enabled", cmd[3])
		assert.Equal(t, "false", cmd[4])
	})
}

func TestIPFSConfigStruct(t *testing.T) {
	cfg := &IPFSConfig{
		APIPort:         9001,
		GatewayPort:     9080,
		EnableFilestore: false,
	}
	assert.Equal(t, 9001, cfg.APIPort)
	assert.Equal(t, 9080, cfg.GatewayPort)
	assert.False(t, cfg.EnableFilestore)

	cmds := cfg.InitCommands()
	assert.Len(t, cmds, 7)
}
