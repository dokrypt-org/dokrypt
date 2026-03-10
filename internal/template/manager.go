package template

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dokrypt/dokrypt/internal/template/builtin"
	"gopkg.in/yaml.v3"
)

type Manager struct {
	globalDir string // ~/.dokrypt/templates
	builtins  map[string]*Info
}

func NewManager(globalDir string) *Manager {
	m := &Manager{
		globalDir: globalDir,
		builtins:  make(map[string]*Info),
	}
	m.registerBuiltins()
	return m
}

func DefaultManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".dokrypt", "templates")
	return NewManager(dir), nil
}

func (m *Manager) registerBuiltins() {
	templates := []Template{
		{
			Name: "evm-basic", Version: "2.0.0",
			Description: "Starter EVM environment with Counter and ERC-20 token",
			Author: "dokrypt", Category: "basic", Difficulty: "beginner",
			Premium: false, Price: "free",
			Tags: []string{"evm", "ethereum", "basic", "starter", "erc20"},
			Chains: []string{"ethereum"}, Services: nil,
			License: "Apache-2.0",
		},
		{
			Name: "evm-defi", Version: "2.0.0",
			Description: "Full DeFi stack: AMM, lending vault, staking, oracle",
			Author: "dokrypt", Category: "defi", Difficulty: "advanced",
			Premium: false, Price: "free",
			Tags: []string{"defi", "evm", "amm", "lending", "staking", "oracle"},
			Chains: []string{"ethereum"}, Services: []string{"ipfs", "subgraph", "blockscout", "chainlink-mock"},
			License: "Apache-2.0",
		},
		{
			Name: "evm-nft", Version: "2.0.0",
			Description: "NFT collection with marketplace, royalties, and IPFS",
			Author: "dokrypt", Category: "nft", Difficulty: "intermediate",
			Premium: false, Price: "free",
			Tags: []string{"nft", "evm", "erc721", "marketplace", "ipfs"},
			Chains: []string{"ethereum"}, Services: []string{"ipfs", "blockscout"},
			License: "Apache-2.0",
		},
		{
			Name: "evm-dao", Version: "2.0.0",
			Description: "DAO governance: governor, timelock, treasury, voting",
			Author: "dokrypt", Category: "dao", Difficulty: "advanced",
			Premium: false, Price: "free",
			Tags: []string{"dao", "governance", "voting", "treasury"},
			Chains: []string{"ethereum"}, Services: nil,
			License: "Apache-2.0",
		},
		{
			Name: "evm-token", Version: "2.0.0",
			Description: "Token toolkit: ERC-20, vesting, staking, multisig",
			Author: "dokrypt", Category: "token", Difficulty: "intermediate",
			Premium: false, Price: "free",
			Tags: []string{"token", "erc20", "vesting", "staking", "multisig"},
			Chains: []string{"ethereum"}, Services: nil,
			License: "Apache-2.0",
		},
		{
			Name: "evm-arbitrum", Version: "2.0.0",
			Description: "Arbitrum L2 development: bridge messaging, token gateway, L1/L2 interop",
			Author: "dokrypt", Category: "l2", Difficulty: "intermediate",
			Premium: false, Price: "free",
			Tags: []string{"arbitrum", "l2", "bridge", "gateway", "stylus", "orbit"},
			Chains: []string{"arbitrum"}, Services: []string{"blockscout"},
			License: "Apache-2.0",
		},
	}
	for _, t := range templates {
		t := t
		m.builtins[t.Name] = &Info{Template: t, BuiltIn: true}
	}
}

func (m *Manager) List() []*Info {
	result := make([]*Info, 0, len(m.builtins))
	for _, info := range m.builtins {
		result = append(result, info)
	}

	entries, err := os.ReadDir(m.globalDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			metaPath := filepath.Join(m.globalDir, entry.Name(), "template.yaml")
			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}
			var t Template
			if err := yaml.Unmarshal(data, &t); err != nil {
				slog.Warn("invalid template", "path", metaPath, "error", err)
				continue
			}
			result = append(result, &Info{
				Template: t,
				Path:     filepath.Join(m.globalDir, entry.Name()),
			})
		}
	}

	return result
}

func (m *Manager) Get(name string) (*Info, error) {
	if info, ok := m.builtins[name]; ok {
		return info, nil
	}

	metaPath := filepath.Join(m.globalDir, name, "template.yaml")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("template %q not found", name)
	}

	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("invalid template %q: %w", name, err)
	}

	return &Info{
		Template: t,
		Path:     filepath.Join(m.globalDir, name),
	}, nil
}

func (m *Manager) GetFS(name string) (fs.FS, error) {
	if _, ok := m.builtins[name]; ok {
		return builtin.TemplateFS(name)
	}

	metaPath := filepath.Join(m.globalDir, name, "template.yaml")
	if _, err := os.Stat(metaPath); err != nil {
		return nil, fmt.Errorf("template %q not found", name)
	}
	return os.DirFS(filepath.Join(m.globalDir, name)), nil
}
