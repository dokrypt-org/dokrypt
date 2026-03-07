package testrunner

import (
	"context"
	"fmt"

	"github.com/dokrypt/dokrypt/internal/chain"
)

type Fixture struct {
	chain      chain.Chain
	snapshotID string
}

func NewFixture(ctx context.Context, c chain.Chain) (*Fixture, error) {
	id, err := c.TakeSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create test fixture: %w", err)
	}
	return &Fixture{chain: c, snapshotID: id}, nil
}

func (f *Fixture) Revert(ctx context.Context) error {
	return f.chain.RevertSnapshot(ctx, f.snapshotID)
}

func WithFixture(ctx context.Context, c chain.Chain, fn func(ctx context.Context) error) error {
	fixture, err := NewFixture(ctx, c)
	if err != nil {
		return err
	}
	defer fixture.Revert(ctx)
	return fn(ctx)
}
