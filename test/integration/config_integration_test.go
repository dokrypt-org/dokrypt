package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dokrypt/dokrypt/internal/config"
)

func TestTemplateConfigs(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "internal", "template", "builtin", "templates")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		t.Skipf("templates directory not found: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			cfgPath := filepath.Join(templatesDir, entry.Name(), "dokrypt.yaml")
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				t.Skipf("no dokrypt.yaml in template %s", entry.Name())
			}
			cfg, err := config.Parse(cfgPath)
			if err != nil {
				t.Fatalf("Parse(%s) error: %v", cfgPath, err)
			}
			config.ApplyDefaults(cfg)
			if err := config.Validate(cfg); err != nil {
				t.Errorf("Validate(%s) error: %v", cfgPath, err)
			}
		})
	}
}

func TestTemplateConfigs_HaveRequiredFields(t *testing.T) {
	templatesDir := filepath.Join("..", "..", "internal", "template", "builtin", "templates")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		t.Skipf("templates directory not found: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			cfgPath := filepath.Join(templatesDir, entry.Name(), "dokrypt.yaml")
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				t.Skipf("no dokrypt.yaml in template %s", entry.Name())
			}
			cfg, err := config.Parse(cfgPath)
			if err != nil {
				t.Fatalf("Parse(%s) error: %v", cfgPath, err)
			}
			if cfg.Name == "" {
				t.Errorf("template %s: name is empty", entry.Name())
			}
			if cfg.Version == "" {
				t.Errorf("template %s: version is empty", entry.Name())
			}
		})
	}
}

func TestTemplateConfigs_ChainsHaveValidEngines(t *testing.T) {
	validEngines := map[string]bool{"anvil": true, "hardhat": true, "geth": true}

	templatesDir := filepath.Join("..", "..", "internal", "template", "builtin", "templates")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		t.Skipf("templates directory not found: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			cfgPath := filepath.Join(templatesDir, entry.Name(), "dokrypt.yaml")
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				t.Skipf("no dokrypt.yaml in template %s", entry.Name())
			}
			cfg, err := config.Parse(cfgPath)
			if err != nil {
				t.Fatalf("Parse(%s) error: %v", cfgPath, err)
			}
			for chainName, chain := range cfg.Chains {
				if !validEngines[chain.Engine] {
					t.Errorf("template %s, chain %s: invalid engine %q",
						entry.Name(), chainName, chain.Engine)
				}
			}
		})
	}
}
