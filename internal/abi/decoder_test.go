package abi

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const erc20ABI = `[
	{
		"type": "function",
		"name": "transfer",
		"inputs": [
			{"name": "to", "type": "address"},
			{"name": "amount", "type": "uint256"}
		],
		"outputs": [
			{"name": "", "type": "bool"}
		]
	},
	{
		"type": "function",
		"name": "balanceOf",
		"inputs": [
			{"name": "account", "type": "address"}
		],
		"outputs": [
			{"name": "", "type": "uint256"}
		]
	},
	{
		"type": "event",
		"name": "Transfer",
		"inputs": [
			{"name": "from", "type": "address", "indexed": true},
			{"name": "to", "type": "address", "indexed": true},
			{"name": "value", "type": "uint256", "indexed": false}
		]
	},
	{
		"type": "event",
		"name": "Approval",
		"inputs": [
			{"name": "owner", "type": "address", "indexed": true},
			{"name": "spender", "type": "address", "indexed": true},
			{"name": "value", "type": "uint256", "indexed": false}
		]
	}
]`

func mustHexBytes(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

func TestParse_ValidABI(t *testing.T) {
	a, err := Parse(erc20ABI)

	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Len(t, a.Methods, 2)
	assert.Len(t, a.Events, 2)
	assert.Contains(t, a.Methods, "transfer")
	assert.Contains(t, a.Methods, "balanceOf")
	assert.Contains(t, a.Events, "Transfer")
	assert.Contains(t, a.Events, "Approval")
}

func TestParse_MethodInputsAndOutputs(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	transfer := a.Methods["transfer"]
	assert.Equal(t, "transfer", transfer.Name)
	assert.Len(t, transfer.Inputs, 2)
	assert.Equal(t, "to", transfer.Inputs[0].Name)
	assert.Equal(t, "address", transfer.Inputs[0].Type)
	assert.Equal(t, "amount", transfer.Inputs[1].Name)
	assert.Equal(t, "uint256", transfer.Inputs[1].Type)
	assert.Len(t, transfer.Outputs, 1)
	assert.Equal(t, "bool", transfer.Outputs[0].Type)
}

func TestParse_EventInputs(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	ev := a.Events["Transfer"]
	assert.Equal(t, "Transfer", ev.Name)
	assert.Len(t, ev.Inputs, 3)
	assert.True(t, ev.Inputs[0].Indexed)
	assert.True(t, ev.Inputs[1].Indexed)
	assert.False(t, ev.Inputs[2].Indexed)
}

func TestParse_SelectorIsComputed(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	transfer := a.Methods["transfer"]
	assert.Equal(t, [4]byte{0xa9, 0x05, 0x9c, 0xbb}, transfer.Selector)

	balanceOf := a.Methods["balanceOf"]
	assert.Equal(t, [4]byte{0x70, 0xa0, 0x82, 0x31}, balanceOf.Selector)
}

func TestParse_EventTopicIsComputed(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	ev := a.Events["Transfer"]
	expected := keccak256([]byte("Transfer(address,address,uint256)"))
	assert.Equal(t, expected, ev.Topic)
}

func TestParse_EmptyTypeIsTreatedAsFunction(t *testing.T) {
	abiJSON := `[{"name": "foo", "inputs": [], "outputs": []}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)
	assert.Contains(t, a.Methods, "foo")
}

func TestParse_EmptyABI(t *testing.T) {
	a, err := Parse(`[]`)
	require.NoError(t, err)
	assert.Empty(t, a.Methods)
	assert.Empty(t, a.Events)
}

func TestParse_InvalidJSON(t *testing.T) {
	_, err := Parse(`not json at all`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse ABI")
}

func TestParse_SkipsNonFunctionNonEventEntries(t *testing.T) {
	abiJSON := `[
		{"type": "constructor", "name": "", "inputs": [{"name":"x","type":"uint256"}]},
		{"type": "fallback", "name": ""},
		{"type": "receive", "name": ""},
		{"type": "function", "name": "doStuff", "inputs": [], "outputs": []}
	]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)
	assert.Len(t, a.Methods, 1)
	assert.Contains(t, a.Methods, "doStuff")
	assert.Empty(t, a.Events)
}

func TestMethodBySelector_Found(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	selector := [4]byte{0xa9, 0x05, 0x9c, 0xbb}
	m, ok := a.MethodBySelector(selector)
	require.True(t, ok)
	assert.Equal(t, "transfer", m.Name)
}

func TestMethodBySelector_NotFound(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	selector := [4]byte{0xde, 0xad, 0xbe, 0xef}
	m, ok := a.MethodBySelector(selector)
	assert.False(t, ok)
	assert.Nil(t, m)
}

func TestEventByTopic_Found(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	topic := keccak256([]byte("Transfer(address,address,uint256)"))
	ev, ok := a.EventByTopic(topic)
	require.True(t, ok)
	assert.Equal(t, "Transfer", ev.Name)
}

func TestEventByTopic_NotFound(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	topic := [32]byte{0xff}
	ev, ok := a.EventByTopic(topic)
	assert.False(t, ok)
	assert.Nil(t, ev)
}

func TestIsDynamicType(t *testing.T) {
	tests := []struct {
		typ      string
		expected bool
	}{
		{"string", true},
		{"bytes", true},
		{"uint256[]", true},
		{"address[]", true},
		{"uint256", false},
		{"address", false},
		{"bool", false},
		{"bytes32", false},
		{"int128", false},
	}
	for _, tc := range tests {
		t.Run(tc.typ, func(t *testing.T) {
			assert.Equal(t, tc.expected, isDynamicType(tc.typ))
		})
	}
}

func TestBuildSignature(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		inputs   []Argument
		expected string
	}{
		{
			name:     "no args",
			funcName: "foo",
			inputs:   nil,
			expected: "foo()",
		},
		{
			name:     "single arg",
			funcName: "balanceOf",
			inputs:   []Argument{{Name: "account", Type: "address"}},
			expected: "balanceOf(address)",
		},
		{
			name:     "multiple args",
			funcName: "transfer",
			inputs:   []Argument{{Name: "to", Type: "address"}, {Name: "amount", Type: "uint256"}},
			expected: "transfer(address,uint256)",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, buildSignature(tc.funcName, tc.inputs))
		})
	}
}

func TestEncodeCall_AddressAndUint(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	addr := "0x000000000000000000000000000000000000dead"
	amount := big.NewInt(1000)

	data, err := a.EncodeCall("transfer", addr, amount)
	require.NoError(t, err)

	assert.Equal(t, []byte{0xa9, 0x05, 0x9c, 0xbb}, data[:4])

	assert.Len(t, data, 68)
}

func TestEncodeCall_UnknownMethod(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	_, err = a.EncodeCall("nonExistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown method")
}

func TestEncodeCall_WrongArgCount(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	_, err = a.EncodeCall("transfer", "0x0000000000000000000000000000000000000001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected 2 arguments")
}

func TestEncodeCall_UintVariants(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setVal","inputs":[{"name":"v","type":"uint256"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	tests := []struct {
		name string
		val  any
	}{
		{"big.Int", big.NewInt(42)},
		{"int64", int64(42)},
		{"uint64", uint64(42)},
		{"int", int(42)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := a.EncodeCall("setVal", tc.val)
			require.NoError(t, err)
			assert.Len(t, data, 36) // 4 + 32
		})
	}
}

func TestEncodeCall_UintUnsupportedType(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setVal","inputs":[{"name":"v","type":"uint256"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	_, err = a.EncodeCall("setVal", "not a number")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported value type")
}

func TestEncodeCall_AddressFromBytes(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	addrBytes := make([]byte, 20)
	addrBytes[19] = 0x01

	data, err := a.EncodeCall("balanceOf", addrBytes)
	require.NoError(t, err)
	assert.Equal(t, byte(0x01), data[4+31])
}

func TestEncodeCall_AddressInvalidHex(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	_, err = a.EncodeCall("balanceOf", "0xZZZZ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address")
}

func TestEncodeCall_AddressWrongLength(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	_, err = a.EncodeCall("balanceOf", "0x0102030405")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "address must be 20 bytes")
}

func TestEncodeCall_AddressBytesWrongLength(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	_, err = a.EncodeCall("balanceOf", []byte{0x01, 0x02})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "address must be 20 bytes")
}

func TestEncodeCall_AddressUnsupportedType(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	_, err = a.EncodeCall("balanceOf", 12345)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported value type for address")
}

func TestEncodeCall_Bool(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setFlag","inputs":[{"name":"flag","type":"bool"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	data, err := a.EncodeCall("setFlag", true)
	require.NoError(t, err)
	assert.Equal(t, byte(1), data[4+31])

	data, err = a.EncodeCall("setFlag", false)
	require.NoError(t, err)
	assert.Equal(t, byte(0), data[4+31])
}

func TestEncodeCall_BoolUnsupportedType(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setFlag","inputs":[{"name":"flag","type":"bool"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	_, err = a.EncodeCall("setFlag", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported value type for bool")
}

func TestEncodeCall_SignedInt(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setInt","inputs":[{"name":"v","type":"int256"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	data, err := a.EncodeCall("setInt", big.NewInt(100))
	require.NoError(t, err)
	assert.Len(t, data, 36)

	data, err = a.EncodeCall("setInt", big.NewInt(-1))
	require.NoError(t, err)
	for i := 4; i < 36; i++ {
		assert.Equal(t, byte(0xff), data[i], "byte at offset %d", i)
	}
}

func TestEncodeCall_SignedIntVariants(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setInt","inputs":[{"name":"v","type":"int256"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	_, err = a.EncodeCall("setInt", int64(10))
	require.NoError(t, err)

	data, err := a.EncodeCall("setInt", int64(-1))
	require.NoError(t, err)
	for i := 4; i < 36; i++ {
		assert.Equal(t, byte(0xff), data[i])
	}

	_, err = a.EncodeCall("setInt", int(10))
	require.NoError(t, err)

	data, err = a.EncodeCall("setInt", int(-1))
	require.NoError(t, err)
	for i := 4; i < 36; i++ {
		assert.Equal(t, byte(0xff), data[i])
	}
}

func TestEncodeCall_SignedIntUnsupportedType(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setInt","inputs":[{"name":"v","type":"int256"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	_, err = a.EncodeCall("setInt", "not a number")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported value type")
}

func TestEncodeCall_Bytes32FromString(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setHash","inputs":[{"name":"h","type":"bytes32"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	hash := "0x" + hex.EncodeToString(make([]byte, 32))
	data, err := a.EncodeCall("setHash", hash)
	require.NoError(t, err)
	assert.Len(t, data, 36)
}

func TestEncodeCall_Bytes32FromArray(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setHash","inputs":[{"name":"h","type":"bytes32"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	var h [32]byte
	h[0] = 0xab
	data, err := a.EncodeCall("setHash", h)
	require.NoError(t, err)
	assert.Equal(t, byte(0xab), data[4])
}

func TestEncodeCall_Bytes32FromSlice(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setHash","inputs":[{"name":"h","type":"bytes32"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	h := []byte{0xab, 0xcd}
	data, err := a.EncodeCall("setHash", h)
	require.NoError(t, err)
	assert.Equal(t, byte(0xab), data[4])
	assert.Equal(t, byte(0xcd), data[5])
}

func TestEncodeCall_Bytes32InvalidHex(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setHash","inputs":[{"name":"h","type":"bytes32"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	_, err = a.EncodeCall("setHash", "0xZZZZZZ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bytes32")
}

func TestEncodeCall_Bytes32TooLong(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setHash","inputs":[{"name":"h","type":"bytes32"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	_, err = a.EncodeCall("setHash", make([]byte, 33))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bytes32 value too long")
}

func TestEncodeCall_Bytes32StringTooLong(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setHash","inputs":[{"name":"h","type":"bytes32"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	_, err = a.EncodeCall("setHash", "0x"+hex.EncodeToString(make([]byte, 33)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bytes32 value too long")
}

func TestEncodeCall_Bytes32UnsupportedType(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setHash","inputs":[{"name":"h","type":"bytes32"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	_, err = a.EncodeCall("setHash", 12345)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported value type for bytes32")
}

func TestEncodeCall_UnsupportedABIType(t *testing.T) {
	abiJSON := `[{"type":"function","name":"weird","inputs":[{"name":"x","type":"tuple"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	_, err = a.EncodeCall("weird", "something")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported ABI type")
}

func TestDecodeCalldata_TransferCall(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	addr := "0x000000000000000000000000000000000000dEaD"
	amount := big.NewInt(1000)

	encoded, err := a.EncodeCall("transfer", addr, amount)
	require.NoError(t, err)

	decoded, err := a.DecodeCalldata(encoded)
	require.NoError(t, err)
	assert.Equal(t, "transfer", decoded.Method)
	assert.Contains(t, decoded.Args, "to")
	assert.Contains(t, decoded.Args, "amount")

	assert.Equal(t, "0x000000000000000000000000000000000000dead", decoded.Args["to"])
	decodedAmount, ok := decoded.Args["amount"].(*big.Int)
	require.True(t, ok)
	assert.Equal(t, 0, amount.Cmp(decodedAmount))
}

func TestDecodeCalldata_TooShort(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	_, err = a.DecodeCalldata([]byte{0x01, 0x02})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "calldata too short")
}

func TestDecodeCalldata_UnknownSelector(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	_, err = a.DecodeCalldata([]byte{0xde, 0xad, 0xbe, 0xef})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown function selector")
}

func TestDecodeCalldata_TruncatedParams(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	data := make([]byte, 4+32+16)
	copy(data[:4], []byte{0xa9, 0x05, 0x9c, 0xbb}) // transfer selector

	decoded, err := a.DecodeCalldata(data)
	require.NoError(t, err)
	assert.Equal(t, "transfer", decoded.Method)
	assert.Contains(t, decoded.Args, "to")
}

func TestDecodeCalldata_BoolType(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setFlag","inputs":[{"name":"flag","type":"bool"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	dataTrue, err := a.EncodeCall("setFlag", true)
	require.NoError(t, err)

	decoded, err := a.DecodeCalldata(dataTrue)
	require.NoError(t, err)
	assert.Equal(t, true, decoded.Args["flag"])

	dataFalse, err := a.EncodeCall("setFlag", false)
	require.NoError(t, err)

	decoded, err = a.DecodeCalldata(dataFalse)
	require.NoError(t, err)
	assert.Equal(t, false, decoded.Args["flag"])
}

func TestDecodeCalldata_Bytes32Type(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setHash","inputs":[{"name":"h","type":"bytes32"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	var h [32]byte
	h[0] = 0xab
	h[31] = 0xcd
	data, err := a.EncodeCall("setHash", h)
	require.NoError(t, err)

	decoded, err := a.DecodeCalldata(data)
	require.NoError(t, err)

	hexVal, ok := decoded.Args["h"].(string)
	require.True(t, ok)
	assert.True(t, len(hexVal) > 0)
	assert.Equal(t, "0x", hexVal[:2])
}

func TestDecodeCalldata_SignedIntPositive(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setInt","inputs":[{"name":"v","type":"int256"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	data, err := a.EncodeCall("setInt", big.NewInt(42))
	require.NoError(t, err)

	decoded, err := a.DecodeCalldata(data)
	require.NoError(t, err)

	val, ok := decoded.Args["v"].(*big.Int)
	require.True(t, ok)
	assert.Equal(t, big.NewInt(42), val)
}

func TestDecodeCalldata_SignedIntNegative(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setInt","inputs":[{"name":"v","type":"int256"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	data, err := a.EncodeCall("setInt", big.NewInt(-1))
	require.NoError(t, err)

	decoded, err := a.DecodeCalldata(data)
	require.NoError(t, err)

	val, ok := decoded.Args["v"].(*big.Int)
	require.True(t, ok)
	assert.Equal(t, big.NewInt(-1), val)
}

func TestDecodeCalldata_BytesNType(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setData","inputs":[{"name":"d","type":"bytes4"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	selector := computeSelector("setData(bytes4)")
	calldata := make([]byte, 36)
	copy(calldata[:4], selector[:])
	calldata[4] = 0xde
	calldata[5] = 0xad
	calldata[6] = 0xbe
	calldata[7] = 0xef

	decoded, err := a.DecodeCalldata(calldata)
	require.NoError(t, err)
	hexVal, ok := decoded.Args["d"].(string)
	require.True(t, ok)
	assert.Equal(t, "0x", hexVal[:2])
}

func TestDecodeCalldata_DefaultType(t *testing.T) {
	abiJSON := `[{"type":"function","name":"doThing","inputs":[{"name":"x","type":"fixed128x18"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	selector := computeSelector("doThing(fixed128x18)")
	calldata := make([]byte, 36)
	copy(calldata[:4], selector[:])
	calldata[35] = 0x01

	decoded, err := a.DecodeCalldata(calldata)
	require.NoError(t, err)
	hexVal, ok := decoded.Args["x"].(string)
	require.True(t, ok)
	assert.True(t, len(hexVal) > 2)
}

func TestDecodeCalldata_StringParam(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setName","inputs":[{"name":"name","type":"string"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	selector := computeSelector("setName(string)")

	calldata := make([]byte, 0, 4+32+32+32)
	calldata = append(calldata, selector[:]...)

	offset := make([]byte, 32)
	offset[31] = 32
	calldata = append(calldata, offset...)

	length := make([]byte, 32)
	length[31] = 5
	calldata = append(calldata, length...)

	strData := make([]byte, 32)
	copy(strData, "hello")
	calldata = append(calldata, strData...)

	decoded, err := a.DecodeCalldata(calldata)
	require.NoError(t, err)
	assert.Equal(t, "hello", decoded.Args["name"])
}

func TestDecodeCalldata_BytesDynamicParam(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setData","inputs":[{"name":"data","type":"bytes"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	selector := computeSelector("setData(bytes)")

	calldata := make([]byte, 0, 4+32+32+32)
	calldata = append(calldata, selector[:]...)

	offset := make([]byte, 32)
	offset[31] = 32
	calldata = append(calldata, offset...)

	length := make([]byte, 32)
	length[31] = 3
	calldata = append(calldata, length...)

	bytesData := make([]byte, 32)
	bytesData[0] = 0xaa
	bytesData[1] = 0xbb
	bytesData[2] = 0xcc
	calldata = append(calldata, bytesData...)

	decoded, err := a.DecodeCalldata(calldata)
	require.NoError(t, err)
	assert.Equal(t, "0xaabbcc", decoded.Args["data"])
}

func TestDecodeCalldata_DynamicArray(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setArr","inputs":[{"name":"arr","type":"uint256[]"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	selector := computeSelector("setArr(uint256[])")

	calldata := make([]byte, 0, 4+32+32+64)
	calldata = append(calldata, selector[:]...)

	offset := make([]byte, 32)
	offset[31] = 32
	calldata = append(calldata, offset...)

	length := make([]byte, 32)
	length[31] = 2
	calldata = append(calldata, length...)

	elem0 := make([]byte, 32)
	elem0[31] = 10
	calldata = append(calldata, elem0...)

	elem1 := make([]byte, 32)
	elem1[31] = 20
	calldata = append(calldata, elem1...)

	decoded, err := a.DecodeCalldata(calldata)
	require.NoError(t, err)
	arr, ok := decoded.Args["arr"].([]string)
	require.True(t, ok)
	assert.Len(t, arr, 2)
}

func TestDecodeCalldata_DynamicOffsetOutOfBounds(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setName","inputs":[{"name":"name","type":"string"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	selector := computeSelector("setName(string)")
	calldata := make([]byte, 0, 4+32)
	calldata = append(calldata, selector[:]...)

	offset := make([]byte, 32)
	offset[31] = 255
	calldata = append(calldata, offset...)

	decoded, err := a.DecodeCalldata(calldata)
	require.NoError(t, err)
	assert.Nil(t, decoded.Args["name"])
}

func TestDecodeCalldata_DynamicDataTooShortForLength(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setName","inputs":[{"name":"name","type":"string"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	selector := computeSelector("setName(string)")
	calldata := make([]byte, 0, 4+32+16) // only 16 bytes at offset, not enough for length word
	calldata = append(calldata, selector[:]...)

	offset := make([]byte, 32)
	offset[31] = 32
	calldata = append(calldata, offset...)

	calldata = append(calldata, make([]byte, 16)...)

	decoded, err := a.DecodeCalldata(calldata)
	require.NoError(t, err)
	assert.Nil(t, decoded.Args["name"])
}

func TestDecodeCalldata_DynamicLengthExceedsData(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setName","inputs":[{"name":"name","type":"string"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	selector := computeSelector("setName(string)")
	calldata := make([]byte, 0, 4+32+32)
	calldata = append(calldata, selector[:]...)

	offset := make([]byte, 32)
	offset[31] = 32
	calldata = append(calldata, offset...)

	length := make([]byte, 32)
	length[31] = 100
	calldata = append(calldata, length...)

	decoded, err := a.DecodeCalldata(calldata)
	require.NoError(t, err)
	assert.Nil(t, decoded.Args["name"])
}

func TestEncodeDecodeRoundTrip_Address(t *testing.T) {
	a, err := Parse(erc20ABI)
	require.NoError(t, err)

	addr := "0x0000000000000000000000000000000000000001"
	encoded, err := a.EncodeCall("balanceOf", addr)
	require.NoError(t, err)

	decoded, err := a.DecodeCalldata(encoded)
	require.NoError(t, err)
	assert.Equal(t, "balanceOf", decoded.Method)
	assert.Equal(t, addr, decoded.Args["account"])
}

func TestEncodeDecodeRoundTrip_Uint256(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setVal","inputs":[{"name":"v","type":"uint256"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	val := new(big.Int).SetUint64(123456789)
	encoded, err := a.EncodeCall("setVal", val)
	require.NoError(t, err)

	decoded, err := a.DecodeCalldata(encoded)
	require.NoError(t, err)
	decodedVal, ok := decoded.Args["v"].(*big.Int)
	require.True(t, ok)
	assert.Equal(t, 0, val.Cmp(decodedVal))
}

func TestEncodeDecodeRoundTrip_Bool(t *testing.T) {
	abiJSON := `[{"type":"function","name":"setFlag","inputs":[{"name":"f","type":"bool"}],"outputs":[]}]`
	a, err := Parse(abiJSON)
	require.NoError(t, err)

	for _, b := range []bool{true, false} {
		encoded, err := a.EncodeCall("setFlag", b)
		require.NoError(t, err)

		decoded, err := a.DecodeCalldata(encoded)
		require.NoError(t, err)
		assert.Equal(t, b, decoded.Args["f"])
	}
}

func TestKeccak256_KnownValue(t *testing.T) {
	hash := keccak256([]byte{})
	expected := mustHexBytes(t, "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	assert.Equal(t, expected, hash[:])
}

func TestComputeSelector_KnownValue(t *testing.T) {
	sel := computeSelector("transfer(address,uint256)")
	assert.Equal(t, [4]byte{0xa9, 0x05, 0x9c, 0xbb}, sel)
}

func TestComputeTopic_KnownValue(t *testing.T) {
	topic := computeTopic("Transfer(address,address,uint256)")
	hash := keccak256([]byte("Transfer(address,address,uint256)"))
	assert.Equal(t, hash, topic)
}
