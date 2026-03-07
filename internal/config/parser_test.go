package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_ValidFullConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: full-project
settings:
  runtime: docker
  log_level: info
  block_time: "2s"
  accounts: 10
  account_balance: "10000"
chains:
  ethereum:
    engine: anvil
    chain_id: 31337
  polygon:
    engine: hardhat
    chain_id: 137
services:
  ipfs:
    type: ipfs
    port: 5001
plugins:
  my-plugin:
    version: "^1.0.0"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "1", cfg.Version)
	assert.Equal(t, "full-project", cfg.Name)
	assert.Equal(t, "docker", cfg.Settings.Runtime)
	assert.Equal(t, "info", cfg.Settings.LogLevel)
	assert.Len(t, cfg.Chains, 2)
	assert.Contains(t, cfg.Chains, "ethereum")
	assert.Contains(t, cfg.Chains, "polygon")
	assert.Len(t, cfg.Services, 1)
	assert.Contains(t, cfg.Services, "ipfs")
}

func TestParse_MinimalConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: minimal
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "minimal", cfg.Name)
	assert.Equal(t, "docker", cfg.Settings.Runtime)
	assert.Equal(t, "info", cfg.Settings.LogLevel)
	assert.Equal(t, 10, cfg.Settings.Accounts)
}

func TestParse_EnvVarInterpolation_Set(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")

	t.Setenv("DOKRYPT_TEST_LOG_LEVEL", "debug")

	content := `
version: "1"
name: env-test
settings:
  log_level: ${DOKRYPT_TEST_LOG_LEVEL}
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.Settings.LogLevel)
}

func TestParse_EnvVarInterpolation_DefaultUsed(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")

	os.Unsetenv("DOKRYPT_TEST_UNSET_VAR_XYZ")

	content := `
version: "1"
name: default-test
settings:
  log_level: ${DOKRYPT_TEST_UNSET_VAR_XYZ:-warn}
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "warn", cfg.Settings.LogLevel)
}

func TestParse_EnvVarInterpolation_SetOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")

	t.Setenv("DOKRYPT_TEST_SET_VAR", "error")

	content := `
version: "1"
name: set-override-test
settings:
  log_level: ${DOKRYPT_TEST_SET_VAR:-warn}
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "error", cfg.Settings.LogLevel)
}

func TestParse_EnvVarInterpolation_UnsetNoDefault_LeftUnresolved(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")

	os.Unsetenv("TOTALLY_MISSING_VAR_ABC")

	content := `
version: "1"
name: ${TOTALLY_MISSING_VAR_ABC}
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "${TOTALLY_MISSING_VAR_ABC}", cfg.Name)
}

func TestParse_MultipleEnvVarsInSameField(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")

	t.Setenv("DOKRYPT_TEST_RUNTIME", "podman")
	t.Setenv("DOKRYPT_TEST_LEVEL", "debug")

	content := `
version: "1"
name: multi-env
settings:
  runtime: ${DOKRYPT_TEST_RUNTIME}
  log_level: ${DOKRYPT_TEST_LEVEL}
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "podman", cfg.Settings.Runtime)
	assert.Equal(t, "debug", cfg.Settings.LogLevel)
}

func TestParse_MissingFile_ReturnsError(t *testing.T) {
	_, err := Parse("/nonexistent/path/dokrypt.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestParse_EmptyPath_ReturnsError(t *testing.T) {
	_, err := Parse("")
	require.Error(t, err)
}

func TestParse_InvalidYAML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: [broken yaml
  - invalid
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	_, err := Parse(cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestParse_AppliesDefaults_Runtime(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: defaults-test
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "docker", cfg.Settings.Runtime)
	assert.Equal(t, "info", cfg.Settings.LogLevel)
	assert.Equal(t, "2s", cfg.Settings.BlockTime)
	assert.Equal(t, 10, cfg.Settings.Accounts)
	assert.Equal(t, "10000", cfg.Settings.AccountBalance)
}

func TestParse_AppliesDefaults_ChainInherits(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: chain-defaults
chains:
  local:
    engine: anvil
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)

	chain := cfg.Chains["local"]
	assert.Equal(t, uint64(31337), chain.ChainID)
	assert.Equal(t, "2s", chain.BlockTime)
	assert.Equal(t, 10, chain.Accounts)
	assert.Equal(t, "cancun", chain.Hardfork)
	assert.Equal(t, "auto", chain.Mining.Mode)
}

func TestParse_OverrideFile_MergedOnTop(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	overridePath := filepath.Join(dir, "dokrypt.override.yaml")

	base := `
version: "1"
name: base-project
settings:
  runtime: docker
  log_level: info
`
	override := `
name: override-project
settings:
  log_level: debug
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(overridePath, []byte(override), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, "override-project", cfg.Name)
	assert.Equal(t, "debug", cfg.Settings.LogLevel)
	assert.Equal(t, "docker", cfg.Settings.Runtime)
}

func TestParse_NoOverrideFile_StillWorks(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: no-override
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "no-override", cfg.Name)
}

func TestParseWithProfile_ValidProfile_OverridesSettings(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: profile-test
settings:
  runtime: docker
  log_level: info
  accounts: 10
profiles:
  staging:
    settings:
      log_level: warn
      accounts: 5
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := ParseWithProfile(cfgPath, "staging")
	require.NoError(t, err)
	assert.Equal(t, "warn", cfg.Settings.LogLevel)
	assert.Equal(t, 5, cfg.Settings.Accounts)
	assert.Equal(t, "docker", cfg.Settings.Runtime)
}

func TestParseWithProfile_EmptyProfile_NoOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: empty-profile
settings:
  log_level: info
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := ParseWithProfile(cfgPath, "")
	require.NoError(t, err)
	assert.Equal(t, "info", cfg.Settings.LogLevel)
}

func TestParseWithProfile_UnknownProfile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: bad-profile
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	_, err := ParseWithProfile(cfgPath, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
	assert.Contains(t, err.Error(), "not found")
}

func TestResolveTemplateVars_ResolvesChainVars(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"myservice": {
				Type: "custom",
				Environment: map[string]string{
					"RPC_URL": "{{chain.ethereum.rpc_url}}",
				},
			},
		},
	}

	resolver := &mockResolver{
		chains: map[string]map[string]string{
			"ethereum": {"rpc_url": "http://localhost:8545"},
		},
	}

	err := ResolveTemplateVars(cfg, resolver)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8545", cfg.Services["myservice"].Environment["RPC_URL"])
}

func TestResolveTemplateVars_ResolvesServiceVars(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"myservice": {
				Type: "custom",
				Environment: map[string]string{
					"IPFS_URL": "{{service.ipfs.api_url}}",
				},
			},
		},
	}

	resolver := &mockResolver{
		services: map[string]map[string]string{
			"ipfs": {"api_url": "http://localhost:5001"},
		},
	}

	err := ResolveTemplateVars(cfg, resolver)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:5001", cfg.Services["myservice"].Environment["IPFS_URL"])
}

func TestResolveTemplateVars_ResolvesDeployVars(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"myservice": {
				Type: "custom",
				Environment: map[string]string{
					"TOKEN_ADDR": "{{deploy.mytoken.address}}",
				},
			},
		},
	}

	resolver := &mockResolver{
		deploys: map[string]map[string]string{
			"mytoken": {"address": "0xdeadbeef"},
		},
	}

	err := ResolveTemplateVars(cfg, resolver)
	require.NoError(t, err)
	assert.Equal(t, "0xdeadbeef", cfg.Services["myservice"].Environment["TOKEN_ADDR"])
}

func TestResolveTemplateVars_NoTemplateVars_Passthrough(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"svc": {
				Type: "custom",
				Environment: map[string]string{
					"KEY": "plain-value",
				},
			},
		},
	}

	err := ResolveTemplateVars(cfg, &mockResolver{})
	require.NoError(t, err)
	assert.Equal(t, "plain-value", cfg.Services["svc"].Environment["KEY"])
}

func TestResolveTemplateVars_InvalidCategoryReturnsError(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"svc": {
				Type: "custom",
				Environment: map[string]string{
					"VAL": "{{unknown.thing.field}}",
				},
			},
		},
	}

	err := ResolveTemplateVars(cfg, &mockResolver{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

func TestResolveTemplateVars_TooFewParts_ReturnsError(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"svc": {
				Type: "custom",
				Environment: map[string]string{
					"VAL": "{{chain.only_two}}",
				},
			},
		},
	}

	err := ResolveTemplateVars(cfg, &mockResolver{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template variable")
}

type mockResolver struct {
	chains   map[string]map[string]string
	services map[string]map[string]string
	deploys  map[string]map[string]string
}

func (m *mockResolver) ResolveChain(chainName, field string) (string, error) {
	if c, ok := m.chains[chainName]; ok {
		if v, ok := c[field]; ok {
			return v, nil
		}
	}
	return "", nil
}

func (m *mockResolver) ResolveService(serviceName, field string) (string, error) {
	if s, ok := m.services[serviceName]; ok {
		if v, ok := s[field]; ok {
			return v, nil
		}
	}
	return "", nil
}

func (m *mockResolver) ResolveDeploy(label, field string) (string, error) {
	if d, ok := m.deploys[label]; ok {
		if v, ok := d[field]; ok {
			return v, nil
		}
	}
	return "", nil
}

type errResolver struct {
	chainErr   error
	serviceErr error
	deployErr  error
}

func (e *errResolver) ResolveChain(chainName, field string) (string, error) {
	return "", e.chainErr
}

func (e *errResolver) ResolveService(serviceName, field string) (string, error) {
	return "", e.serviceErr
}

func (e *errResolver) ResolveDeploy(label, field string) (string, error) {
	return "", e.deployErr
}

func TestParse_EmptyYAMLContent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(""), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "1.0", cfg.Version)
	assert.Equal(t, "docker", cfg.Settings.Runtime)
}

func TestParse_WhitespaceOnlyYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("   \n\n  \n"), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "1.0", cfg.Version)
}

func TestParse_OverrideFile_InvalidYAML_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	overridePath := filepath.Join(dir, "dokrypt.override.yaml")

	base := `
version: "1"
name: base-project
`
	badOverride := `
name: [broken yaml
  - invalid
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(overridePath, []byte(badOverride), 0644))

	_, err := Parse(cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "override")
}

func TestParse_OverrideFile_AddsChainsToBase(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	overridePath := filepath.Join(dir, "dokrypt.override.yaml")

	base := `
version: "1"
name: chain-test
chains:
  ethereum:
    engine: anvil
`
	override := `
chains:
  polygon:
    engine: hardhat
    chain_id: 137
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(base), 0644))
	require.NoError(t, os.WriteFile(overridePath, []byte(override), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Contains(t, cfg.Chains, "ethereum")
	assert.Contains(t, cfg.Chains, "polygon")
	assert.Equal(t, uint64(137), cfg.Chains["polygon"].ChainID)
}

func TestParseWithProfile_ProfileOverridesRuntime(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: rt-test
settings:
  runtime: docker
profiles:
  test:
    settings:
      runtime: podman
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := ParseWithProfile(cfgPath, "test")
	require.NoError(t, err)
	assert.Equal(t, "podman", cfg.Settings.Runtime)
}

func TestParseWithProfile_BadBaseFile_ReturnsError(t *testing.T) {
	_, err := ParseWithProfile("/nonexistent/path.yaml", "dev")
	require.Error(t, err)
}

func TestParse_MultipleEnvVarsInOneLine(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")

	t.Setenv("DOKRYPT_PART_A", "hello")
	t.Setenv("DOKRYPT_PART_B", "world")

	content := `
version: "1"
name: "${DOKRYPT_PART_A}-${DOKRYPT_PART_B}"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "hello-world", cfg.Name)
}

func TestParse_EnvVarWithEmptyDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")

	os.Unsetenv("DOKRYPT_UNDEFINED_VAR_XYZ123")

	content := `
version: "1"
name: "${DOKRYPT_UNDEFINED_VAR_XYZ123:-}"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "", cfg.Name)
}

func TestResolveTemplateVars_ChainResolverError_ReturnsError(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"svc": {
				Type: "custom",
				Environment: map[string]string{
					"RPC": "{{chain.ethereum.rpc_url}}",
				},
			},
		},
	}

	resolver := &errResolver{chainErr: fmt.Errorf("chain not started")}
	err := ResolveTemplateVars(cfg, resolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain not started")
}

func TestResolveTemplateVars_ServiceResolverError_ReturnsError(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"svc": {
				Type: "custom",
				Environment: map[string]string{
					"URL": "{{service.ipfs.api_url}}",
				},
			},
		},
	}

	resolver := &errResolver{serviceErr: fmt.Errorf("service not ready")}
	err := ResolveTemplateVars(cfg, resolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service not ready")
}

func TestResolveTemplateVars_DeployResolverError_ReturnsError(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"svc": {
				Type: "custom",
				Environment: map[string]string{
					"ADDR": "{{deploy.token.address}}",
				},
			},
		},
	}

	resolver := &errResolver{deployErr: fmt.Errorf("contract not deployed")}
	err := ResolveTemplateVars(cfg, resolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "contract not deployed")
}

func TestResolveTemplateVars_MultipleVarsInSameValue(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"svc": {
				Type: "custom",
				Environment: map[string]string{
					"COMBINED": "chain={{chain.eth.rpc_url}},deploy={{deploy.token.address}}",
				},
			},
		},
	}

	resolver := &mockResolver{
		chains:  map[string]map[string]string{"eth": {"rpc_url": "http://localhost:8545"}},
		deploys: map[string]map[string]string{"token": {"address": "0xdead"}},
	}

	err := ResolveTemplateVars(cfg, resolver)
	require.NoError(t, err)
	assert.Equal(t, "chain=http://localhost:8545,deploy=0xdead", cfg.Services["svc"].Environment["COMBINED"])
}

func TestResolveTemplateVars_NoServices_NoError(t *testing.T) {
	cfg := &Config{}
	err := ResolveTemplateVars(cfg, &mockResolver{})
	require.NoError(t, err)
}

func TestResolveTemplateVars_NilServicesMap_NoError(t *testing.T) {
	cfg := &Config{Services: nil}
	err := ResolveTemplateVars(cfg, &mockResolver{})
	require.NoError(t, err)
}

func TestResolveTemplateVars_ServiceWithNoEnvironment_NoError(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"svc": {Type: "ipfs"},
		},
	}
	err := ResolveTemplateVars(cfg, &mockResolver{})
	require.NoError(t, err)
}

func TestResolveTemplateVars_WhitespaceInTemplateVar(t *testing.T) {
	cfg := &Config{
		Services: map[string]ServiceConfig{
			"svc": {
				Type: "custom",
				Environment: map[string]string{
					"RPC": "{{ chain.eth.rpc_url }}",
				},
			},
		},
	}

	resolver := &mockResolver{
		chains: map[string]map[string]string{"eth": {"rpc_url": "http://localhost:8545"}},
	}

	err := ResolveTemplateVars(cfg, resolver)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8545", cfg.Services["svc"].Environment["RPC"])
}

func TestParse_ForkConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: fork-test
chains:
  mainnet:
    engine: anvil
    fork:
      network: mainnet
      block_number: 18000000
      rpc_url: https://rpc.example.com
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	chain := cfg.Chains["mainnet"]
	require.NotNil(t, chain.Fork)
	assert.Equal(t, "mainnet", chain.Fork.Network)
	assert.Equal(t, uint64(18000000), chain.Fork.BlockNumber)
	assert.Equal(t, "https://rpc.example.com", chain.Fork.RPCURL)
}

func TestParse_ServiceWithHealthcheck(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: health-test
services:
  myservice:
    type: custom
    port: 8080
    healthcheck:
      http: http://localhost:8080/health
      interval: 5s
      timeout: 3s
      retries: 3
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	svc := cfg.Services["myservice"]
	require.NotNil(t, svc.Healthcheck)
	assert.Equal(t, "http://localhost:8080/health", svc.Healthcheck.HTTP)
	assert.Equal(t, "5s", svc.Healthcheck.Interval)
	assert.Equal(t, "3s", svc.Healthcheck.Timeout)
	assert.Equal(t, 3, svc.Healthcheck.Retries)
}

func TestParse_HooksConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: hooks-test
hooks:
  pre_up:
    - "echo starting"
    - "npm run compile"
  post_up:
    - "echo started"
  pre_down:
    - "echo stopping"
  post_down:
    - "echo stopped"
  post_snapshot:
    - "echo snapped"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"echo starting", "npm run compile"}, cfg.Hooks.PreUp)
	assert.Equal(t, []string{"echo started"}, cfg.Hooks.PostUp)
	assert.Equal(t, []string{"echo stopping"}, cfg.Hooks.PreDown)
	assert.Equal(t, []string{"echo stopped"}, cfg.Hooks.PostDown)
	assert.Equal(t, []string{"echo snapped"}, cfg.Hooks.PostSnapshot)
}

func TestParse_TestSuites(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: suite-test
tests:
  dir: "./tests"
  timeout: "120s"
  parallel: 8
  snapshot_isolation: true
  gas_report: true
  coverage: true
  suites:
    unit:
      pattern: "test/unit/**"
      timeout: "30s"
      snapshot_isolation: false
      services:
        - ipfs
    integration:
      pattern: "test/integration/**"
      timeout: "60s"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "./tests", cfg.Tests.Dir)
	assert.Equal(t, "120s", cfg.Tests.Timeout)
	assert.Equal(t, 8, cfg.Tests.Parallel)
	assert.True(t, cfg.Tests.SnapshotIsolation)
	assert.True(t, cfg.Tests.GasReport)
	assert.True(t, cfg.Tests.Coverage)
	require.Len(t, cfg.Tests.Suites, 2)
	assert.Equal(t, "test/unit/**", cfg.Tests.Suites["unit"].Pattern)
	assert.Equal(t, []string{"ipfs"}, cfg.Tests.Suites["unit"].Services)
}

func TestParse_GenesisAccountsAndDeploy(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: genesis-test
chains:
  local:
    engine: anvil
    genesis_accounts:
      - address: "0x1234567890abcdef1234567890abcdef12345678"
        balance: "1000000"
      - address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
        balance: "500000"
    deploy:
      - artifact: "Token.sol"
        label: "mytoken"
        constructor_args:
          - "MyToken"
          - "MTK"
          - 18
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	chain := cfg.Chains["local"]
	require.Len(t, chain.GenesisAccounts, 2)
	assert.Equal(t, "1000000", chain.GenesisAccounts[0].Balance)
	require.Len(t, chain.Deploy, 1)
	assert.Equal(t, "Token.sol", chain.Deploy[0].Artifact)
	assert.Equal(t, "mytoken", chain.Deploy[0].Label)
	require.Len(t, chain.Deploy[0].ConstructorArgs, 3)
}

func TestParse_OracleFeeds(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: oracle-test
services:
  oracle:
    type: chainlink-mock
    feeds:
      - pair: ETH/USD
        price: 2000.50
        decimals: 8
        update_interval: 30s
        volatility:
          enabled: true
          max_deviation_pct: 2.5
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0644))

	cfg, err := Parse(cfgPath)
	require.NoError(t, err)
	svc := cfg.Services["oracle"]
	require.Len(t, svc.Feeds, 1)
	assert.Equal(t, "ETH/USD", svc.Feeds[0].Pair)
	assert.Equal(t, 2000.50, svc.Feeds[0].Price)
	assert.Equal(t, 8, svc.Feeds[0].Decimals)
	require.NotNil(t, svc.Feeds[0].Volatility)
	assert.True(t, svc.Feeds[0].Volatility.Enabled)
	assert.Equal(t, 2.5, svc.Feeds[0].Volatility.MaxDeviationPct)
}
