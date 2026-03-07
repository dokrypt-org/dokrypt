package config

import "time"

type Config struct {
	Version  string                    `yaml:"version"`
	Name     string                    `yaml:"name"`
	Settings Settings                  `yaml:"settings"`
	Profiles map[string]Profile        `yaml:"profiles"`
	Chains   map[string]ChainConfig    `yaml:"chains"`
	Services map[string]ServiceConfig  `yaml:"services"`
	Plugins  map[string]PluginConfig   `yaml:"plugins"`
	Tests    TestConfig                `yaml:"tests"`
	Hooks    HookConfig                `yaml:"hooks"`
}

type Settings struct {
	Runtime        string `yaml:"runtime"`
	LogLevel       string `yaml:"log_level"`
	BlockTime      string `yaml:"block_time"`
	Accounts       int    `yaml:"accounts"`
	AccountBalance string `yaml:"account_balance"`
	Telemetry      bool   `yaml:"telemetry"`
}

type Profile struct {
	Settings Settings `yaml:"settings"`
}

type ChainConfig struct {
	Engine         string              `yaml:"engine"`
	ChainID        uint64              `yaml:"chain_id"`
	Fork           *ForkConfig         `yaml:"fork,omitempty"`
	BlockTime      string              `yaml:"block_time"`
	Accounts       int                 `yaml:"accounts"`
	AccountBalance string              `yaml:"account_balance"`
	Balance        string              `yaml:"balance"`
	GasLimit       uint64              `yaml:"gas_limit"`
	BaseFee        uint64              `yaml:"base_fee"`
	Hardfork       string              `yaml:"hardfork"`
	CodeSizeLimit  uint64              `yaml:"code_size_limit"`
	AutoImpersonate bool              `yaml:"auto_impersonate"`
	Mining         MiningConfig        `yaml:"mining"`
	GenesisAccounts []GenesisAccount  `yaml:"genesis_accounts"`
	Deploy         []DeployConfig      `yaml:"deploy"`
}

func (c ChainConfig) GetBalance() string {
	if c.Balance != "" {
		return c.Balance
	}
	return c.AccountBalance
}

type ForkConfig struct {
	Network     string `yaml:"network"`
	BlockNumber uint64 `yaml:"block_number"`
	RPCURL      string `yaml:"rpc_url"`
}

type MiningConfig struct {
	Mode     string `yaml:"mode"`     // auto, interval, manual
	Interval string `yaml:"interval"`
}

type GenesisAccount struct {
	Address string `yaml:"address"`
	Balance string `yaml:"balance"`
}

type DeployConfig struct {
	Artifact        string `yaml:"artifact"`
	ConstructorArgs []any  `yaml:"constructor_args"`
	Label           string `yaml:"label"`
}

type ServiceConfig struct {
	Type           string            `yaml:"type"`
	Implementation string            `yaml:"implementation,omitempty"`
	Image          string            `yaml:"image,omitempty"`
	Command        []string          `yaml:"command,omitempty"`
	Chain          string            `yaml:"chain,omitempty"`
	Chains         []string          `yaml:"chains,omitempty"`
	Port           int               `yaml:"port,omitempty"`
	Ports          map[string]int    `yaml:"ports,omitempty"`
	Volumes        []string          `yaml:"volumes,omitempty"`
	Environment    map[string]string `yaml:"environment,omitempty"`
	DependsOn      []string          `yaml:"depends_on,omitempty"`
	Build          *BuildConfig      `yaml:"build,omitempty"`
	Healthcheck    *HealthcheckConfig `yaml:"healthcheck,omitempty"`
	Features       map[string]bool   `yaml:"features,omitempty"`

	APIPort     int    `yaml:"api_port,omitempty"`
	GatewayPort int    `yaml:"gateway_port,omitempty"`
	PinOnAdd    *bool  `yaml:"pin_on_add,omitempty"`
	MaxStorage  string `yaml:"max_storage,omitempty"`

	Schema      string `yaml:"schema,omitempty"`
	StartBlock  uint64 `yaml:"start_block,omitempty"`
	GraphQLPort int    `yaml:"graphql_port,omitempty"`
	AdminPort   int    `yaml:"admin_port,omitempty"`
	IPFS        string `yaml:"ipfs,omitempty"`
	Config      string `yaml:"config,omitempty"`

	Feeds []OracleFeedConfig `yaml:"feeds,omitempty"`

	RelayDelay         string `yaml:"relay_delay,omitempty"`
	ConfirmationBlocks int    `yaml:"confirmation_blocks,omitempty"`

	Dashboards []string `yaml:"dashboards,omitempty"`

	DripAmount string `yaml:"drip_amount,omitempty"`
	Cooldown   string `yaml:"cooldown,omitempty"`
}

type BuildConfig struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

type HealthcheckConfig struct {
	HTTP     string        `yaml:"http,omitempty"`
	TCP      string        `yaml:"tcp,omitempty"`
	Interval string        `yaml:"interval,omitempty"`
	Timeout  string        `yaml:"timeout,omitempty"`
	Retries  int           `yaml:"retries,omitempty"`
}

type OracleFeedConfig struct {
	Pair           string               `yaml:"pair"`
	Price          float64              `yaml:"price"`
	Decimals       int                  `yaml:"decimals"`
	UpdateInterval string               `yaml:"update_interval"`
	Volatility     *VolatilityConfig    `yaml:"volatility,omitempty"`
}

type VolatilityConfig struct {
	Enabled         bool    `yaml:"enabled"`
	MaxDeviationPct float64 `yaml:"max_deviation_pct"`
}

type PluginConfig struct {
	Version string         `yaml:"version"`
	Config  map[string]any `yaml:"config"`
}

type TestConfig struct {
	Dir               string                   `yaml:"dir"`
	Timeout           string                   `yaml:"timeout"`
	Parallel          int                      `yaml:"parallel"`
	SnapshotIsolation bool                     `yaml:"snapshot_isolation"`
	GasReport         bool                     `yaml:"gas_report"`
	Coverage          bool                     `yaml:"coverage"`
	Suites            map[string]TestSuiteConfig `yaml:"suites"`
}

type TestSuiteConfig struct {
	Pattern           string   `yaml:"pattern"`
	Timeout           string   `yaml:"timeout"`
	SnapshotIsolation bool     `yaml:"snapshot_isolation"`
	Services          []string `yaml:"services"`
}

type HookConfig struct {
	PreUp        []string `yaml:"pre_up"`
	PostUp       []string `yaml:"post_up"`
	PreDown      []string `yaml:"pre_down"`
	PostDown     []string `yaml:"post_down"`
	PostSnapshot []string `yaml:"post_snapshot"`
}

func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	return time.ParseDuration(s)
}
