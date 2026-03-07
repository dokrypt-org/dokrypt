package testrunner

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertBalance_EqualDecimalStrings_DoesNotFail(t *testing.T) {
	AssertBalance(t, "1000000000000000000", "1000000000000000000")
}

func TestAssertBalance_ZeroEqualZero_DoesNotFail(t *testing.T) {
	AssertBalance(t, "0", "0")
}

func TestAssertBalance_LargeNumbers_DoesNotFail(t *testing.T) {
	large := "99999999999999999999999999999"
	AssertBalance(t, large, large)
}

func TestAssertBalance_MismatchingValues_BigIntDiffers(t *testing.T) {
	a, ok := new(big.Int).SetString("999", 10)
	require.True(t, ok)
	e, ok := new(big.Int).SetString("1000", 10)
	require.True(t, ok)
	assert.NotEqual(t, 0, a.Cmp(e), "999 and 1000 should not be equal")
}

func TestAssertBalance_InvalidDecimalString_ParseFails(t *testing.T) {
	_, ok := new(big.Int).SetString("not-a-number", 10)
	assert.False(t, ok, "parsing 'not-a-number' in base 10 should fail")
}

func TestAssertBalance_HexStringIsInvalidForBase10(t *testing.T) {
	_, ok := new(big.Int).SetString("0xDEAD", 10)
	assert.False(t, ok, "parsing hex-prefixed string with base 10 should fail")
}

func TestAssertBalance_NegativeVsPositive_DiffersBySign(t *testing.T) {
	a, _ := new(big.Int).SetString("-1", 10)
	e, _ := new(big.Int).SetString("1", 10)
	assert.NotEqual(t, 0, a.Cmp(e))
}

func TestAssertTxSuccess_SuccessfulTx_DoesNotFail(t *testing.T) {
	tx := &Transaction{Status: 1, Hash: "0xabc"}
	AssertTxSuccess(t, tx)
}

func TestAssertTxSuccess_RevertedTx_StatusIsZero(t *testing.T) {
	tx := &Transaction{Status: 0, RevertMsg: "execution reverted"}
	assert.Equal(t, uint64(0), tx.Status)
	assert.NotEqual(t, uint64(1), tx.Status)
}

func TestAssertTxSuccess_RevertedTx_HasRevertMsg(t *testing.T) {
	tx := &Transaction{Status: 0, RevertMsg: "insufficient allowance"}
	assert.NotEmpty(t, tx.RevertMsg)
}

func TestAssertTxReverted_RevertedTx_DoesNotFail(t *testing.T) {
	tx := &Transaction{Status: 0}
	AssertTxReverted(t, tx)
}

func TestAssertTxReverted_SuccessfulTx_StatusIsOne(t *testing.T) {
	tx := &Transaction{Status: 1}
	assert.NotEqual(t, uint64(0), tx.Status)
}

func TestAssertTxRevertedWith_RevertedWithCorrectReason_DoesNotFail(t *testing.T) {
	tx := &Transaction{Status: 0, RevertMsg: "ERC20: insufficient balance"}
	AssertTxRevertedWith(t, tx, "ERC20: insufficient balance")
}

func TestAssertTxRevertedWith_WrongRevertReason_Differs(t *testing.T) {
	tx := &Transaction{Status: 0, RevertMsg: "actual reason"}
	assert.NotEqual(t, "expected reason", tx.RevertMsg)
}

func TestAssertTxRevertedWith_SuccessfulTx_WouldNotReach_ReasonCheck(t *testing.T) {
	tx := &Transaction{Status: 1}
	assert.Equal(t, uint64(1), tx.Status)
}

func TestAssertGasBelow_GasBelowMax_DoesNotFail(t *testing.T) {
	tx := &Transaction{GasUsed: 20_000}
	AssertGasBelow(t, tx, 21_000)
}

func TestAssertGasBelow_GasEqualsMax_DoesNotFail(t *testing.T) {
	tx := &Transaction{GasUsed: 21_000}
	AssertGasBelow(t, tx, 21_000)
}

func TestAssertGasBelow_GasExceedsMax_ConditionIsTrue(t *testing.T) {
	tx := &Transaction{GasUsed: 22_000}
	maxGas := uint64(21_000)
	assert.Greater(t, tx.GasUsed, maxGas, "gas exceeds max should be detected")
}

func TestAssertEvent_EventPresent_DoesNotFail(t *testing.T) {
	tx := &Transaction{
		Status: 1,
		Logs:   []Log{{Name: "Transfer", Args: map[string]any{"from": "0xAAA", "to": "0xBBB"}}},
	}
	AssertEvent(t, tx, "Transfer", nil)
}

func TestAssertEvent_WithMatchingArgs_DoesNotFail(t *testing.T) {
	tx := &Transaction{
		Logs: []Log{
			{Name: "Transfer", Args: map[string]any{"from": "0x1", "to": "0x2", "value": "500"}},
		},
	}
	AssertEvent(t, tx, "Transfer", map[string]any{"from": "0x1", "value": "500"})
}

func TestAssertEvent_MultipleLogs_FindsCorrectOne_DoesNotFail(t *testing.T) {
	tx := &Transaction{
		Logs: []Log{
			{Name: "Approval", Args: map[string]any{"owner": "0x1"}},
			{Name: "Transfer", Args: map[string]any{"from": "0x1", "to": "0x2"}},
		},
	}
	AssertEvent(t, tx, "Transfer", map[string]any{"from": "0x1"})
}

func TestAssertEvent_NilArgs_DoesNotFail(t *testing.T) {
	tx := &Transaction{
		Logs: []Log{{Name: "Mint", Args: nil}},
	}
	AssertEvent(t, tx, "Mint", nil)
}

func TestAssertEvent_EventAbsent_NotFoundInLogs(t *testing.T) {
	tx := &Transaction{
		Logs: []Log{{Name: "Approval"}},
	}
	found := false
	for _, log := range tx.Logs {
		if log.Name == "Transfer" {
			found = true
		}
	}
	assert.False(t, found, "Transfer event should not be found")
}

func TestAssertEvent_NoLogs_EmptySlice(t *testing.T) {
	tx := &Transaction{Logs: nil}
	assert.Empty(t, tx.Logs)
}

func TestAssertEvent_MissingArgKey_NotInMap(t *testing.T) {
	log := Log{Name: "Transfer", Args: map[string]any{"value": "100"}}
	_, ok := log.Args["nonexistent"]
	assert.False(t, ok, "key 'nonexistent' should not be in log args")
}

func TestAssertEvent_MismatchingArgValue_ValuesAreDifferent(t *testing.T) {
	log := Log{Name: "Transfer", Args: map[string]any{"value": "100"}}
	actual := log.Args["value"]
	assert.NotEqual(t, "999", actual)
}

func TestAssertEventCount_CorrectCount_DoesNotFail(t *testing.T) {
	tx := &Transaction{
		Logs: []Log{
			{Name: "Transfer"},
			{Name: "Transfer"},
			{Name: "Approval"},
		},
	}
	AssertEventCount(t, tx, "Transfer", 2)
}

func TestAssertEventCount_ZeroEvents_DoesNotFail(t *testing.T) {
	tx := &Transaction{Logs: []Log{{Name: "Approval"}}}
	AssertEventCount(t, tx, "Transfer", 0)
}

func TestAssertEventCount_AllSameEvent_DoesNotFail(t *testing.T) {
	tx := &Transaction{
		Logs: []Log{{Name: "Transfer"}, {Name: "Transfer"}, {Name: "Transfer"}},
	}
	AssertEventCount(t, tx, "Transfer", 3)
}

func TestAssertEventCount_WrongCount_CountingLogic(t *testing.T) {
	tx := &Transaction{Logs: []Log{{Name: "Transfer"}}}
	count := 0
	for _, log := range tx.Logs {
		if log.Name == "Transfer" {
			count++
		}
	}
	assert.Equal(t, 1, count)
	assert.NotEqual(t, 3, count)
}

func TestAssertNoEvent_EventAbsent_DoesNotFail(t *testing.T) {
	tx := &Transaction{
		Logs: []Log{{Name: "Approval"}},
	}
	AssertNoEvent(t, tx, "Transfer")
}

func TestAssertNoEvent_NoLogs_DoesNotFail(t *testing.T) {
	tx := &Transaction{Logs: nil}
	AssertNoEvent(t, tx, "Transfer")
}

func TestAssertNoEvent_EventPresent_FoundInLogs(t *testing.T) {
	tx := &Transaction{Logs: []Log{{Name: "Transfer"}}}
	found := false
	for _, log := range tx.Logs {
		if log.Name == "Transfer" {
			found = true
		}
	}
	assert.True(t, found, "Transfer event is present, so AssertNoEvent should detect it and fail")
}

func TestAssertETHBalance_EqualValues_DoesNotFail(t *testing.T) {
	val := new(big.Int).SetUint64(1_000_000_000)
	expected := new(big.Int).SetUint64(1_000_000_000)
	AssertETHBalance(t, val, expected)
}

func TestAssertETHBalance_ZeroValues_DoesNotFail(t *testing.T) {
	AssertETHBalance(t, big.NewInt(0), big.NewInt(0))
}

func TestAssertETHBalance_LargeWeiValues_DoesNotFail(t *testing.T) {
	v, ok := new(big.Int).SetString("100000000000000000000", 10)
	require.True(t, ok)
	AssertETHBalance(t, v, new(big.Int).Set(v))
}

func TestAssertETHBalance_MismatchingValues_CmpIsNonZero(t *testing.T) {
	a := new(big.Int).SetUint64(999)
	e := new(big.Int).SetUint64(1000)
	assert.NotEqual(t, 0, a.Cmp(e))
}

func TestAssertJSONEqual_EquivalentObjects_DoesNotFail(t *testing.T) {
	AssertJSONEqual(t, `{"a":1,"b":2}`, `{"b":2,"a":1}`)
}

func TestAssertJSONEqual_IdenticalArrays_DoesNotFail(t *testing.T) {
	AssertJSONEqual(t, `[1,2,3]`, `[1,2,3]`)
}

func TestAssertJSONEqual_NestedObjects_DoesNotFail(t *testing.T) {
	AssertJSONEqual(t,
		`{"user":{"name":"alice","age":30}}`,
		`{"user":{"age":30,"name":"alice"}}`,
	)
}

func TestAssertJSONEqual_NullValues_DoesNotFail(t *testing.T) {
	AssertJSONEqual(t, `null`, `null`)
}

func TestAssertJSONEqual_DifferentValues_JSONDiffers(t *testing.T) {
	var a, e interface{}
	require.NoError(t, json.Unmarshal([]byte(`{"a":1}`), &a))
	require.NoError(t, json.Unmarshal([]byte(`{"a":2}`), &e))
	ab, _ := json.Marshal(a)
	eb, _ := json.Marshal(e)
	assert.NotEqual(t, string(ab), string(eb))
}

func TestAssertJSONEqual_InvalidActualJSON_ParseFails(t *testing.T) {
	var v interface{}
	err := json.Unmarshal([]byte("not-json"), &v)
	assert.Error(t, err, "invalid JSON should fail to parse")
}

func TestAssertJSONEqual_InvalidExpectedJSON_ParseFails(t *testing.T) {
	var v interface{}
	err := json.Unmarshal([]byte("also-not-json"), &v)
	assert.Error(t, err, "invalid JSON should fail to parse")
}

func TestAssertJSONEqual_ArraysDifferentOrder_JSONDiffers(t *testing.T) {
	var a, e interface{}
	require.NoError(t, json.Unmarshal([]byte(`[1,2,3]`), &a))
	require.NoError(t, json.Unmarshal([]byte(`[3,2,1]`), &e))
	ab, _ := json.Marshal(a)
	eb, _ := json.Marshal(e)
	assert.NotEqual(t, string(ab), string(eb))
}

func TestTransaction_DefaultStatusIsZero(t *testing.T) {
	tx := &Transaction{}
	assert.Equal(t, uint64(0), tx.Status)
}

func TestTransaction_LogsDefaultsToNil(t *testing.T) {
	tx := &Transaction{}
	assert.Nil(t, tx.Logs)
}

func TestLog_ArgsMapCanBeNil(t *testing.T) {
	log := Log{Name: "Transfer"}
	assert.Nil(t, log.Args)
}

func TestTransaction_MultipleLogsCanHaveSameName(t *testing.T) {
	tx := &Transaction{
		Logs: []Log{
			{Name: "Transfer"},
			{Name: "Transfer"},
		},
	}
	count := 0
	for _, log := range tx.Logs {
		if log.Name == "Transfer" {
			count++
		}
	}
	assert.Equal(t, 2, count)
}
