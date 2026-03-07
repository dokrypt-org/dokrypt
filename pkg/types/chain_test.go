package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccount_JSONRoundTrip(t *testing.T) {
	original := Account{
		Address:    "0x1234567890abcdef1234567890abcdef12345678",
		PrivateKey: "0xdeadbeef",
		Label:      "deployer",
		Balance:    "1000000000000000000",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Account
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestAccount_JSONFieldNames(t *testing.T) {
	a := Account{
		Address:    "0xabc",
		PrivateKey: "0xkey",
		Label:      "test",
		Balance:    "100",
	}

	data, err := json.Marshal(a)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "address")
	assert.Contains(t, raw, "private_key")
	assert.Contains(t, raw, "label")
	assert.Contains(t, raw, "balance")
}

func TestAccount_JSONOmitEmpty(t *testing.T) {
	a := Account{
		Address: "0xabc",
	}

	data, err := json.Marshal(a)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "address")
	assert.NotContains(t, raw, "private_key")
	assert.NotContains(t, raw, "label")
	assert.NotContains(t, raw, "balance")
}

func TestAccount_JSONUnmarshal(t *testing.T) {
	jsonStr := `{"address":"0xabc","private_key":"0xkey","label":"test","balance":"100"}`
	var a Account
	err := json.Unmarshal([]byte(jsonStr), &a)
	require.NoError(t, err)

	assert.Equal(t, "0xabc", a.Address)
	assert.Equal(t, "0xkey", a.PrivateKey)
	assert.Equal(t, "test", a.Label)
	assert.Equal(t, "100", a.Balance)
}

func TestAccount_ZeroValue(t *testing.T) {
	var a Account
	assert.Empty(t, a.Address)
	assert.Empty(t, a.PrivateKey)
	assert.Empty(t, a.Label)
	assert.Empty(t, a.Balance)
}

func TestChainStatus_JSONRoundTrip(t *testing.T) {
	original := ChainStatus{
		Name:    "mainnet-fork",
		Engine:  "anvil",
		ChainID: 31337,
		RPCURL:  "http://localhost:8545",
		WSURL:   "ws://localhost:8546",
		Running: true,
		Healthy: true,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ChainStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestChainStatus_JSONFieldNames(t *testing.T) {
	cs := ChainStatus{
		Name:    "test",
		Engine:  "anvil",
		ChainID: 1,
		RPCURL:  "http://localhost:8545",
		WSURL:   "ws://localhost:8546",
		Running: true,
		Healthy: false,
	}

	data, err := json.Marshal(cs)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "name")
	assert.Contains(t, raw, "engine")
	assert.Contains(t, raw, "chain_id")
	assert.Contains(t, raw, "rpc_url")
	assert.Contains(t, raw, "ws_url")
	assert.Contains(t, raw, "running")
	assert.Contains(t, raw, "healthy")
}

func TestChainStatus_JSONUnmarshal(t *testing.T) {
	jsonStr := `{
		"name": "local",
		"engine": "hardhat",
		"chain_id": 31337,
		"rpc_url": "http://localhost:8545",
		"ws_url": "ws://localhost:8546",
		"running": true,
		"healthy": false
	}`

	var cs ChainStatus
	err := json.Unmarshal([]byte(jsonStr), &cs)
	require.NoError(t, err)

	assert.Equal(t, "local", cs.Name)
	assert.Equal(t, "hardhat", cs.Engine)
	assert.Equal(t, uint64(31337), cs.ChainID)
	assert.Equal(t, "http://localhost:8545", cs.RPCURL)
	assert.Equal(t, "ws://localhost:8546", cs.WSURL)
	assert.True(t, cs.Running)
	assert.False(t, cs.Healthy)
}

func TestChainStatus_ZeroValue(t *testing.T) {
	var cs ChainStatus
	assert.Empty(t, cs.Name)
	assert.Empty(t, cs.Engine)
	assert.Equal(t, uint64(0), cs.ChainID)
	assert.Empty(t, cs.RPCURL)
	assert.Empty(t, cs.WSURL)
	assert.False(t, cs.Running)
	assert.False(t, cs.Healthy)
}

func TestChainStatus_LargeChainID(t *testing.T) {
	cs := ChainStatus{
		ChainID: 999999999999,
	}

	data, err := json.Marshal(cs)
	require.NoError(t, err)

	var decoded ChainStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, uint64(999999999999), decoded.ChainID)
}

func TestTransactionReceipt_JSONRoundTrip(t *testing.T) {
	original := TransactionReceipt{
		Hash:        "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Status:      1,
		GasUsed:     21000,
		BlockNumber: 12345678,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded TransactionReceipt
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestTransactionReceipt_JSONFieldNames(t *testing.T) {
	tr := TransactionReceipt{
		Hash:        "0xabc",
		Status:      1,
		GasUsed:     21000,
		BlockNumber: 100,
	}

	data, err := json.Marshal(tr)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Contains(t, raw, "hash")
	assert.Contains(t, raw, "status")
	assert.Contains(t, raw, "gas_used")
	assert.Contains(t, raw, "block_number")
}

func TestTransactionReceipt_StatusValues(t *testing.T) {
	tests := []struct {
		name   string
		status uint64
	}{
		{"success", 1},
		{"failure", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := TransactionReceipt{Status: tc.status}
			data, err := json.Marshal(tr)
			require.NoError(t, err)

			var decoded TransactionReceipt
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)
			assert.Equal(t, tc.status, decoded.Status)
		})
	}
}

func TestTransactionReceipt_ZeroValue(t *testing.T) {
	var tr TransactionReceipt
	assert.Empty(t, tr.Hash)
	assert.Equal(t, uint64(0), tr.Status)
	assert.Equal(t, uint64(0), tr.GasUsed)
	assert.Equal(t, uint64(0), tr.BlockNumber)
}
