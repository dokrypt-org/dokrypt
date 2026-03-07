package ipfs

type IPFSConfig struct {
	APIPort         int
	GatewayPort     int
	EnableFilestore bool
}

func DefaultIPFSConfig() *IPFSConfig {
	return &IPFSConfig{
		APIPort:         5001,
		GatewayPort:     8080,
		EnableFilestore: true,
	}
}

func (c *IPFSConfig) InitCommands() [][]string {
	return [][]string{
		{"ipfs", "config", "--json", "API.HTTPHeaders.Access-Control-Allow-Origin", `["*"]`},
		{"ipfs", "config", "--json", "API.HTTPHeaders.Access-Control-Allow-Methods", `["PUT","POST","GET"]`},
		{"ipfs", "config", "--json", "Experimental.UrlstoreEnabled", "true"},
		{"ipfs", "config", "--json", "Experimental.FilestoreEnabled", "true"},
		{"ipfs", "config", "--json", "Swarm.DisableNatPortMap", "true"},
		{"ipfs", "config", "--json", "Swarm.RelayClient.Enabled", "false"},
		{"ipfs", "config", "--json", "Swarm.RelayService.Enabled", "false"},
	}
}
