package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dokrypt/dokrypt/internal/chain"
)

type ChainStateManager struct {
	store *Store
}

func NewChainStateManager(store *Store) *ChainStateManager {
	return &ChainStateManager{store: store}
}

func (m *ChainStateManager) ExportChainState(ctx context.Context, snapshotName string, c chain.Chain) (*ChainSnapshot, error) {
	chainDir := filepath.Join(m.store.ChainsDir(snapshotName), c.Name())
	if err := os.MkdirAll(chainDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create chain dir: %w", err)
	}

	stateFile := filepath.Join(chainDir, "state.json")
	if err := c.ExportState(ctx, stateFile); err != nil {
		return nil, fmt.Errorf("failed to export state for chain %s: %w", c.Name(), err)
	}

	return &ChainSnapshot{
		Name:        c.Name(),
		Engine:      c.Engine(),
		ChainID:     c.ChainID(),
		BlockNumber: getBlockNumber(ctx, c),
		StateFile:   filepath.Join("chains", c.Name(), "state.json"),
	}, nil
}

func (m *ChainStateManager) ImportChainState(ctx context.Context, snapshotName string, c chain.Chain, cs ChainSnapshot) error {
	statePath := filepath.Join(m.store.SnapshotDir(snapshotName), cs.StateFile)
	if _, err := os.Stat(statePath); err != nil {
		return fmt.Errorf("state file not found for chain %s: %w", cs.Name, err)
	}

	if err := c.ImportState(ctx, statePath); err != nil {
		return fmt.Errorf("failed to import state for chain %s: %w", cs.Name, err)
	}

	return nil
}

func getBlockNumber(ctx context.Context, c chain.Chain) uint64 {
	result, err := c.RPC(ctx, "eth_blockNumber")
	if err != nil {
		return 0
	}
	var hexBlock string
	if err := json.Unmarshal(result, &hexBlock); err != nil {
		return 0
	}
	var blockNum uint64
	if _, err := fmt.Sscanf(hexBlock, "0x%x", &blockNum); err != nil {
		return 0
	}
	return blockNum
}
