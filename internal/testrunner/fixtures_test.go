package testrunner

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"testing"

	"github.com/dokrypt/dokrypt/internal/chain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockChain struct {
	snapshotCount  int
	snapshotErr    error
	revertErr      error
	revertedWith   string
	lastSnapshotID string
}

func (m *mockChain) TakeSnapshot(_ context.Context) (string, error) {
	if m.snapshotErr != nil {
		return "", m.snapshotErr
	}
	m.snapshotCount++
	m.lastSnapshotID = "snap-" + string(rune('0'+m.snapshotCount))
	return m.lastSnapshotID, nil
}

func (m *mockChain) RevertSnapshot(_ context.Context, id string) error {
	m.revertedWith = id
	return m.revertErr
}

func (m *mockChain) Start(_ context.Context) error                            { return nil }
func (m *mockChain) Stop(_ context.Context) error                             { return nil }
func (m *mockChain) IsRunning(_ context.Context) bool                         { return true }
func (m *mockChain) Health(_ context.Context) error                           { return nil }
func (m *mockChain) Name() string                                             { return "mock" }
func (m *mockChain) ChainID() uint64                                          { return 1337 }
func (m *mockChain) RPCURL() string                                           { return "http://localhost:8545" }
func (m *mockChain) WSURL() string                                            { return "ws://localhost:8546" }
func (m *mockChain) Engine() string                                           { return "mock" }
func (m *mockChain) Accounts() []chain.Account                                { return nil }
func (m *mockChain) FundAccount(_ context.Context, _ string, _ *big.Int) error { return nil }
func (m *mockChain) ImpersonateAccount(_ context.Context, _ string) error     { return nil }
func (m *mockChain) GenerateAccounts(_ context.Context, _ int) ([]chain.Account, error) {
	return nil, nil
}
func (m *mockChain) MineBlocks(_ context.Context, _ uint64) error          { return nil }
func (m *mockChain) SetBlockTime(_ context.Context, _ uint64) error        { return nil }
func (m *mockChain) SetGasPrice(_ context.Context, _ uint64) error         { return nil }
func (m *mockChain) TimeTravel(_ context.Context, _ int64) error           { return nil }
func (m *mockChain) SetBalance(_ context.Context, _ string, _ *big.Int) error { return nil }
func (m *mockChain) SetStorageAt(_ context.Context, _ string, _ string, _ string) error {
	return nil
}
func (m *mockChain) ExportState(_ context.Context, _ string) error         { return nil }
func (m *mockChain) ImportState(_ context.Context, _ string) error         { return nil }
func (m *mockChain) Fork(_ context.Context, _ chain.ForkOptions) error     { return nil }
func (m *mockChain) ForkInfo() *chain.ForkInfo                             { return nil }
func (m *mockChain) RPC(_ context.Context, _ string, _ ...any) (json.RawMessage, error) {
	return nil, nil
}
func (m *mockChain) Logs(_ context.Context, _ bool) (io.ReadCloser, error) { return nil, nil }

func TestNewFixture_Success(t *testing.T) {
	mc := &mockChain{}
	fixture, err := NewFixture(context.Background(), mc)

	require.NoError(t, err)
	require.NotNil(t, fixture)
	assert.Equal(t, 1, mc.snapshotCount)
}

func TestNewFixture_SnapshotError(t *testing.T) {
	mc := &mockChain{snapshotErr: errors.New("snapshot failed")}
	fixture, err := NewFixture(context.Background(), mc)

	require.Error(t, err)
	assert.Nil(t, fixture)
	assert.Contains(t, err.Error(), "failed to create test fixture")
	assert.Contains(t, err.Error(), "snapshot failed")
}

func TestNewFixture_StoresSnapshotID(t *testing.T) {
	mc := &mockChain{}
	fixture, err := NewFixture(context.Background(), mc)

	require.NoError(t, err)
	assert.Equal(t, mc.lastSnapshotID, fixture.snapshotID)
}

func TestNewFixture_StoresChain(t *testing.T) {
	mc := &mockChain{}
	fixture, err := NewFixture(context.Background(), mc)

	require.NoError(t, err)
	assert.Equal(t, mc, fixture.chain)
}

func TestRevert_CallsRevertSnapshotWithCorrectID(t *testing.T) {
	mc := &mockChain{}
	fixture, err := NewFixture(context.Background(), mc)
	require.NoError(t, err)

	err = fixture.Revert(context.Background())
	require.NoError(t, err)
	assert.Equal(t, fixture.snapshotID, mc.revertedWith)
}

func TestRevert_ReturnsRevertError(t *testing.T) {
	mc := &mockChain{revertErr: errors.New("revert failed")}
	fixture, err := NewFixture(context.Background(), mc)
	require.NoError(t, err)

	err = fixture.Revert(context.Background())
	require.Error(t, err)
	assert.Equal(t, "revert failed", err.Error())
}

func TestRevert_NoErrorOnSuccess(t *testing.T) {
	mc := &mockChain{}
	fixture, err := NewFixture(context.Background(), mc)
	require.NoError(t, err)

	err = fixture.Revert(context.Background())
	assert.NoError(t, err)
}

func TestWithFixture_ExecutesFunction(t *testing.T) {
	mc := &mockChain{}
	called := false

	err := WithFixture(context.Background(), mc, func(_ context.Context) error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestWithFixture_TakesAndRevertsSnapshot(t *testing.T) {
	mc := &mockChain{}

	err := WithFixture(context.Background(), mc, func(_ context.Context) error {
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, mc.snapshotCount)
	assert.NotEmpty(t, mc.revertedWith, "RevertSnapshot should have been called")
}

func TestWithFixture_RevertsEvenOnFunctionError(t *testing.T) {
	mc := &mockChain{}

	err := WithFixture(context.Background(), mc, func(_ context.Context) error {
		return errors.New("test failure")
	})

	require.Error(t, err)
	assert.Equal(t, "test failure", err.Error())
	assert.NotEmpty(t, mc.revertedWith)
}

func TestWithFixture_ReturnsFunctionError(t *testing.T) {
	mc := &mockChain{}
	expectedErr := errors.New("some test error")

	err := WithFixture(context.Background(), mc, func(_ context.Context) error {
		return expectedErr
	})

	assert.Equal(t, expectedErr, err)
}

func TestWithFixture_SnapshotFailure_ReturnsError(t *testing.T) {
	mc := &mockChain{snapshotErr: errors.New("cannot snapshot")}

	err := WithFixture(context.Background(), mc, func(_ context.Context) error {
		t.Fatal("function should not be called when snapshot fails")
		return nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create test fixture")
}

func TestWithFixture_PassesContext(t *testing.T) {
	mc := &mockChain{}
	type ctxKey string
	ctx := context.WithValue(context.Background(), ctxKey("key"), "value")

	err := WithFixture(ctx, mc, func(innerCtx context.Context) error {
		val := innerCtx.Value(ctxKey("key"))
		assert.Equal(t, "value", val)
		return nil
	})

	require.NoError(t, err)
}

func TestWithFixture_MultipleCallsTakeMultipleSnapshots(t *testing.T) {
	mc := &mockChain{}

	for i := 0; i < 3; i++ {
		err := WithFixture(context.Background(), mc, func(_ context.Context) error {
			return nil
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 3, mc.snapshotCount)
}
