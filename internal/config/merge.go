package config

func MergeConfigs(base, override *Config) *Config {
	result := *base

	if override.Version != "" {
		result.Version = override.Version
	}
	if override.Name != "" {
		result.Name = override.Name
	}

	mergeSettings(&result.Settings, &override.Settings)

	if override.Profiles != nil {
		if result.Profiles == nil {
			result.Profiles = make(map[string]Profile)
		}
		for k, v := range override.Profiles {
			result.Profiles[k] = v
		}
	}

	if override.Chains != nil {
		if result.Chains == nil {
			result.Chains = make(map[string]ChainConfig)
		}
		for k, v := range override.Chains {
			if existing, ok := result.Chains[k]; ok {
				result.Chains[k] = mergeChainConfig(existing, v)
			} else {
				result.Chains[k] = v
			}
		}
	}

	if override.Services != nil {
		if result.Services == nil {
			result.Services = make(map[string]ServiceConfig)
		}
		for k, v := range override.Services {
			result.Services[k] = v
		}
	}

	if override.Plugins != nil {
		if result.Plugins == nil {
			result.Plugins = make(map[string]PluginConfig)
		}
		for k, v := range override.Plugins {
			result.Plugins[k] = v
		}
	}

	return &result
}

func mergeSettings(base, override *Settings) {
	if override.Runtime != "" {
		base.Runtime = override.Runtime
	}
	if override.LogLevel != "" {
		base.LogLevel = override.LogLevel
	}
	if override.BlockTime != "" {
		base.BlockTime = override.BlockTime
	}
	if override.Accounts != 0 {
		base.Accounts = override.Accounts
	}
	if override.AccountBalance != "" {
		base.AccountBalance = override.AccountBalance
	}
	if override.Telemetry {
		base.Telemetry = true
	}
}

func mergeChainConfig(base, override ChainConfig) ChainConfig {
	if override.Engine != "" {
		base.Engine = override.Engine
	}
	if override.ChainID != 0 {
		base.ChainID = override.ChainID
	}
	if override.Fork != nil {
		base.Fork = override.Fork
	}
	if override.BlockTime != "" {
		base.BlockTime = override.BlockTime
	}
	if override.Accounts != 0 {
		base.Accounts = override.Accounts
	}
	if override.AccountBalance != "" {
		base.AccountBalance = override.AccountBalance
	}
	if override.GasLimit != 0 {
		base.GasLimit = override.GasLimit
	}
	if override.BaseFee != 0 {
		base.BaseFee = override.BaseFee
	}
	if override.Hardfork != "" {
		base.Hardfork = override.Hardfork
	}
	if override.Balance != "" {
		base.Balance = override.Balance
	}
	if override.CodeSizeLimit != 0 {
		base.CodeSizeLimit = override.CodeSizeLimit
	}
	if override.AutoImpersonate {
		base.AutoImpersonate = true
	}
	if override.Mining.Mode != "" {
		base.Mining = override.Mining
	}
	if len(override.GenesisAccounts) > 0 {
		base.GenesisAccounts = override.GenesisAccounts
	}
	if len(override.Deploy) > 0 {
		base.Deploy = override.Deploy
	}
	return base
}
