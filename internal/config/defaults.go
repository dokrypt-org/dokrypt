package config

func ApplyDefaults(cfg *Config) {
	if cfg.Version == "" {
		cfg.Version = "1.0"
	}

	applySettingsDefaults(&cfg.Settings)

	for name, chain := range cfg.Chains {
		applyChainDefaults(&chain, &cfg.Settings)
		cfg.Chains[name] = chain
	}

	applyTestDefaults(&cfg.Tests)
}

func applySettingsDefaults(s *Settings) {
	if s.Runtime == "" {
		s.Runtime = "docker"
	}
	if s.LogLevel == "" {
		s.LogLevel = "info"
	}
	if s.BlockTime == "" {
		s.BlockTime = "2s"
	}
	if s.Accounts == 0 {
		s.Accounts = 10
	}
	if s.AccountBalance == "" {
		s.AccountBalance = "10000"
	}
}

func applyChainDefaults(c *ChainConfig, settings *Settings) {
	if c.Engine == "" {
		c.Engine = "anvil"
	}
	if c.ChainID == 0 {
		c.ChainID = 31337
	}
	if c.BlockTime == "" {
		c.BlockTime = settings.BlockTime
	}
	if c.Accounts == 0 {
		c.Accounts = settings.Accounts
	}
	if c.AccountBalance == "" && c.Balance == "" {
		c.AccountBalance = settings.AccountBalance
	}
	if c.GasLimit == 0 {
		c.GasLimit = 30000000
	}
	if c.BaseFee == 0 {
		c.BaseFee = 1
	}
	if c.Hardfork == "" {
		c.Hardfork = "cancun"
	}
	if c.Mining.Mode == "" {
		c.Mining.Mode = "auto"
	}
}

func applyTestDefaults(t *TestConfig) {
	if t.Dir == "" {
		t.Dir = "./test"
	}
	if t.Timeout == "" {
		t.Timeout = "60s"
	}
	if t.Parallel == 0 {
		t.Parallel = 4
	}
}
