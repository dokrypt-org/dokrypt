package config

import (
	"fmt"
	"strings"

	"github.com/dokrypt/dokrypt/internal/common"
)

func Validate(cfg *Config) error {
	var errs []string

	if cfg.Version == "" {
		errs = append(errs, "version is required")
	}

	if cfg.Name == "" {
		errs = append(errs, "name is required")
	}

	errs = append(errs, validateSettings(&cfg.Settings)...)

	errs = append(errs, validateChains(cfg.Chains)...)

	errs = append(errs, validateServices(cfg.Services, cfg.Chains)...)

	errs = append(errs, validatePortConflicts(cfg)...)

	errs = append(errs, validateDependencies(cfg)...)

	errs = append(errs, validatePlugins(cfg.Plugins)...)

	if len(errs) > 0 {
		return common.NewError(common.ErrConfigValidation,
			fmt.Sprintf("configuration validation failed:\n  - %s", strings.Join(errs, "\n  - ")))
	}

	return nil
}

func validateSettings(s *Settings) []string {
	var errs []string

	validRuntimes := map[string]bool{"docker": true, "podman": true}
	if !validRuntimes[s.Runtime] {
		errs = append(errs, fmt.Sprintf("invalid runtime %q (must be docker or podman)", s.Runtime))
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[s.LogLevel] {
		errs = append(errs, fmt.Sprintf("invalid log_level %q", s.LogLevel))
	}

	if _, err := ParseDuration(s.BlockTime); err != nil {
		errs = append(errs, fmt.Sprintf("invalid block_time %q: %v", s.BlockTime, err))
	}

	if s.Accounts < 1 {
		errs = append(errs, "accounts must be at least 1")
	}

	return errs
}

func validateChains(chains map[string]ChainConfig) []string {
	var errs []string

	validEngines := map[string]bool{"anvil": true, "hardhat": true, "geth": true}
	validHardforks := map[string]bool{"london": true, "shanghai": true, "cancun": true}
	validMiningModes := map[string]bool{"auto": true, "interval": true, "manual": true}

	for name, chain := range chains {
		if !validEngines[chain.Engine] {
			errs = append(errs, fmt.Sprintf("chain %q: invalid engine %q (must be anvil, hardhat, or geth)", name, chain.Engine))
		}
		if chain.Hardfork != "" && !validHardforks[chain.Hardfork] {
			errs = append(errs, fmt.Sprintf("chain %q: invalid hardfork %q", name, chain.Hardfork))
		}
		if chain.Mining.Mode != "" && !validMiningModes[chain.Mining.Mode] {
			errs = append(errs, fmt.Sprintf("chain %q: invalid mining mode %q", name, chain.Mining.Mode))
		}
		if chain.BlockTime != "" {
			if _, err := ParseDuration(chain.BlockTime); err != nil {
				errs = append(errs, fmt.Sprintf("chain %q: invalid block_time %q: %v", name, chain.BlockTime, err))
			}
		}
	}

	return errs
}

func validateServices(services map[string]ServiceConfig, chains map[string]ChainConfig) []string {
	var errs []string

	validTypes := map[string]bool{
		"ipfs": true, "subgraph": true, "ponder": true,
		"blockscout": true, "otterscan": true,
		"chainlink-mock": true, "pyth-mock": true,
		"grafana": true, "prometheus": true,
		"faucet": true, "mock-bridge": true, "custom": true,
	}

	for name, svc := range services {
		if !validTypes[svc.Type] {
			errs = append(errs, fmt.Sprintf("service %q: invalid type %q", name, svc.Type))
		}

		if svc.Chain != "" {
			if _, ok := chains[svc.Chain]; !ok {
				errs = append(errs, fmt.Sprintf("service %q: references undefined chain %q", name, svc.Chain))
			}
		}
		for _, chainRef := range svc.Chains {
			if _, ok := chains[chainRef]; !ok {
				errs = append(errs, fmt.Sprintf("service %q: references undefined chain %q", name, chainRef))
			}
		}

		for _, dep := range svc.DependsOn {
			if _, ok := services[dep]; !ok {
				if _, ok := chains[dep]; !ok {
					errs = append(errs, fmt.Sprintf("service %q: depends on undefined service/chain %q", name, dep))
				}
			}
		}
	}

	return errs
}

func validatePortConflicts(cfg *Config) []string {
	var errs []string
	portMap := make(map[int]string) // port -> service name

	for name, svc := range cfg.Services {
		ports := collectServicePorts(name, &svc)
		for _, p := range ports {
			if existing, ok := portMap[p]; ok {
				errs = append(errs, fmt.Sprintf("port %d: conflict between %q and %q", p, existing, name))
			} else {
				portMap[p] = name
			}
		}
	}

	return errs
}

func collectServicePorts(name string, svc *ServiceConfig) []int {
	var ports []int
	if svc.Port > 0 {
		ports = append(ports, svc.Port)
	}
	for _, p := range svc.Ports {
		ports = append(ports, p)
	}
	if svc.APIPort > 0 {
		ports = append(ports, svc.APIPort)
	}
	if svc.GatewayPort > 0 {
		ports = append(ports, svc.GatewayPort)
	}
	if svc.GraphQLPort > 0 {
		ports = append(ports, svc.GraphQLPort)
	}
	if svc.AdminPort > 0 {
		ports = append(ports, svc.AdminPort)
	}
	return ports
}

func validateDependencies(cfg *Config) []string {
	var errs []string

	adj := make(map[string][]string)

	for name, svc := range cfg.Services {
		deps := svc.DependsOn
		if svc.Chain != "" {
			deps = append(deps, svc.Chain)
		}
		adj[name] = deps
	}

	visited := make(map[string]int) // 0=unvisited, 1=in-progress, 2=done
	var path []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		if visited[node] == 2 {
			return false
		}
		if visited[node] == 1 {
			cycleStart := 0
			for i, n := range path {
				if n == node {
					cycleStart = i
					break
				}
			}
			cycle := append(path[cycleStart:], node)
			errs = append(errs, fmt.Sprintf("dependency cycle detected: %s", strings.Join(cycle, " -> ")))
			return true
		}

		visited[node] = 1
		path = append(path, node)

		for _, dep := range adj[node] {
			if dfs(dep) {
				return true
			}
		}

		path = path[:len(path)-1]
		visited[node] = 2
		return false
	}

	for name := range cfg.Services {
		if visited[name] == 0 {
			dfs(name)
		}
	}

	return errs
}

func validatePlugins(plugins map[string]PluginConfig) []string {
	var errs []string

	for name, plugin := range plugins {
		if plugin.Version == "" {
			errs = append(errs, fmt.Sprintf("plugin %q: version is required", name))
			continue
		}
		if _, err := common.ParseConstraint(plugin.Version); err != nil {
			errs = append(errs, fmt.Sprintf("plugin %q: invalid version constraint %q: %v", name, plugin.Version, err))
		}
	}

	return errs
}
