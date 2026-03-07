package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_BasicConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: test-project
settings:
  runtime: docker
  log_level: info
  accounts: 10
  account_balance: "10000"
chains:
  ethereum:
    engine: anvil
    chain_id: 31337
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(cfgPath)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if cfg.Name != "test-project" {
		t.Errorf("Name = %q, want %q", cfg.Name, "test-project")
	}
	if cfg.Settings.Accounts != 10 {
		t.Errorf("Accounts = %d, want 10", cfg.Settings.Accounts)
	}
	chain, ok := cfg.Chains["ethereum"]
	if !ok {
		t.Fatal("missing chain 'ethereum'")
	}
	if chain.Engine != "anvil" {
		t.Errorf("Engine = %q, want %q", chain.Engine, "anvil")
	}
	if chain.ChainID != 31337 {
		t.Errorf("ChainID = %d, want 31337", chain.ChainID)
	}
}

func TestParse_EnvVarInterpolation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	os.Setenv("TEST_CHAIN_ID", "42")
	defer os.Unsetenv("TEST_CHAIN_ID")
	content := `
version: "1"
name: env-test
chains:
  test:
    engine: anvil
    chain_id: ${TEST_CHAIN_ID}
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(cfgPath)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if cfg.Chains["test"].ChainID != 42 {
		t.Errorf("ChainID = %d, want 42", cfg.Chains["test"].ChainID)
	}
}

func TestParse_EnvVarWithDefault(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	os.Unsetenv("UNSET_VAR_FOR_TEST")
	content := `
version: "1"
name: default-test
settings:
  log_level: ${UNSET_VAR_FOR_TEST:-warn}
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(cfgPath)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if cfg.Settings.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.Settings.LogLevel, "warn")
	}
}

func TestParse_NonexistentFile(t *testing.T) {
	_, err := Parse("/nonexistent/path/dokrypt.yaml")
	if err == nil {
		t.Error("Parse() should return error for nonexistent file")
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: [invalid yaml
  - broken
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Parse(cfgPath)
	if err == nil {
		t.Error("Parse() should return error for invalid YAML")
	}
}

func TestParse_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: minimal
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(cfgPath)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if cfg.Settings.Runtime != "docker" {
		t.Errorf("Runtime = %q, want %q", cfg.Settings.Runtime, "docker")
	}
	if cfg.Settings.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.Settings.LogLevel, "info")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Version:  "1",
		Name:     "test",
		Settings: Settings{Runtime: "docker", LogLevel: "info", BlockTime: "2s", Accounts: 10},
		Chains: map[string]ChainConfig{
			"eth": {Engine: "anvil", ChainID: 31337},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Errorf("Validate() error: %v", err)
	}
}

func TestValidate_MissingName(t *testing.T) {
	cfg := &Config{
		Version:  "1",
		Settings: Settings{Runtime: "docker", LogLevel: "info", BlockTime: "2s", Accounts: 10},
	}
	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() should return error for missing name")
	}
}

func TestApplyDefaults_SetsRuntime(t *testing.T) {
	cfg := &Config{}
	ApplyDefaults(cfg)
	if cfg.Settings.Runtime != "docker" {
		t.Errorf("Runtime = %q, want %q", cfg.Settings.Runtime, "docker")
	}
	if cfg.Settings.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.Settings.LogLevel, "info")
	}
}

func TestApplyDefaults_DoesNotOverwrite(t *testing.T) {
	cfg := &Config{
		Settings: Settings{Runtime: "podman", LogLevel: "debug"},
	}
	ApplyDefaults(cfg)
	if cfg.Settings.Runtime != "podman" {
		t.Errorf("Runtime = %q, want %q (should not overwrite)", cfg.Settings.Runtime, "podman")
	}
	if cfg.Settings.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q (should not overwrite)", cfg.Settings.LogLevel, "debug")
	}
}

func TestParseWithProfile_AppliesProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: profile-test
settings:
  runtime: docker
  log_level: info
profiles:
  staging:
    settings:
      log_level: warn
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseWithProfile(cfgPath, "staging")
	if err != nil {
		t.Fatalf("ParseWithProfile() error: %v", err)
	}
	if cfg.Settings.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q after profile applied", cfg.Settings.LogLevel, "warn")
	}
}

func TestParseWithProfile_UnknownProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	content := `
version: "1"
name: profile-test
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ParseWithProfile(cfgPath, "nonexistent")
	if err == nil {
		t.Error("ParseWithProfile() should return error for unknown profile")
	}
}

func TestParse_OverrideFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "dokrypt.yaml")
	overridePath := filepath.Join(dir, "dokrypt.override.yaml")

	base := `
version: "1"
name: base
settings:
  runtime: docker
  log_level: info
`
	override := `
settings:
  log_level: debug
`
	if err := os.WriteFile(cfgPath, []byte(base), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overridePath, []byte(override), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(cfgPath)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if cfg.Settings.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q after override", cfg.Settings.LogLevel, "debug")
	}
}

func TestChainConfig_GetBalance_ReturnsBalance_WhenBothSet(t *testing.T) {
	c := ChainConfig{Balance: "5000", AccountBalance: "10000"}
	assert.Equal(t, "5000", c.GetBalance(), "Balance field should take priority over AccountBalance")
}

func TestChainConfig_GetBalance_ReturnsAccountBalance_WhenBalanceEmpty(t *testing.T) {
	c := ChainConfig{AccountBalance: "10000"}
	assert.Equal(t, "10000", c.GetBalance())
}

func TestChainConfig_GetBalance_ReturnsEmpty_WhenBothEmpty(t *testing.T) {
	c := ChainConfig{}
	assert.Equal(t, "", c.GetBalance())
}

func TestChainConfig_GetBalance_ReturnsBalance_WhenAccountBalanceEmpty(t *testing.T) {
	c := ChainConfig{Balance: "7777"}
	assert.Equal(t, "7777", c.GetBalance())
}

func TestParseDuration_ValidDuration(t *testing.T) {
	d, err := ParseDuration("2s")
	require.NoError(t, err)
	assert.Equal(t, 2*time.Second, d)
}

func TestParseDuration_EmptyString_ReturnsZero(t *testing.T) {
	d, err := ParseDuration("")
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), d)
}

func TestParseDuration_InvalidString_ReturnsError(t *testing.T) {
	_, err := ParseDuration("not-a-duration")
	require.Error(t, err)
}

func TestParseDuration_Milliseconds(t *testing.T) {
	d, err := ParseDuration("500ms")
	require.NoError(t, err)
	assert.Equal(t, 500*time.Millisecond, d)
}

func TestParseDuration_Minutes(t *testing.T) {
	d, err := ParseDuration("5m")
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, d)
}

func TestParseDuration_Hours(t *testing.T) {
	d, err := ParseDuration("1h")
	require.NoError(t, err)
	assert.Equal(t, time.Hour, d)
}

func TestParseDuration_Complex(t *testing.T) {
	d, err := ParseDuration("1h30m")
	require.NoError(t, err)
	assert.Equal(t, time.Hour+30*time.Minute, d)
}

func TestParseDuration_NegativeDuration(t *testing.T) {
	d, err := ParseDuration("-5s")
	require.NoError(t, err)
	assert.Equal(t, -5*time.Second, d)
}

func TestParseDuration_JustNumber_ReturnsError(t *testing.T) {
	_, err := ParseDuration("123")
	require.Error(t, err)
}
