package abi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/sha3"
)

type ABI struct {
	Methods map[string]Method
	Events  map[string]Event
}

type Method struct {
	Name     string
	Inputs   []Argument
	Outputs  []Argument
	Selector [4]byte
}

type Event struct {
	Name   string
	Inputs []Argument
	Topic  [32]byte
}

type Argument struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Indexed bool   `json:"indexed,omitempty"`
}

type DecodedEvent struct {
	Name string
	Args map[string]any
}

type DecodedCall struct {
	Method string
	Args   map[string]any
}

type abiEntry struct {
	Type    string     `json:"type"`
	Name    string     `json:"name"`
	Inputs  []Argument `json:"inputs"`
	Outputs []Argument `json:"outputs"`
}

func Parse(abiJSON string) (*ABI, error) {
	var entries []abiEntry
	if err := json.Unmarshal([]byte(abiJSON), &entries); err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	a := &ABI{
		Methods: make(map[string]Method),
		Events:  make(map[string]Event),
	}

	for _, entry := range entries {
		switch entry.Type {
		case "function", "":
			sig := buildSignature(entry.Name, entry.Inputs)
			selector := computeSelector(sig)
			a.Methods[entry.Name] = Method{
				Name:     entry.Name,
				Inputs:   entry.Inputs,
				Outputs:  entry.Outputs,
				Selector: selector,
			}
		case "event":
			sig := buildSignature(entry.Name, entry.Inputs)
			topic := computeTopic(sig)
			a.Events[entry.Name] = Event{
				Name:   entry.Name,
				Inputs: entry.Inputs,
				Topic:  topic,
			}
		}
	}

	return a, nil
}

func (a *ABI) MethodBySelector(selector [4]byte) (*Method, bool) {
	for _, m := range a.Methods {
		if m.Selector == selector {
			return &m, true
		}
	}
	return nil, false
}

func (a *ABI) EventByTopic(topic [32]byte) (*Event, bool) {
	for _, e := range a.Events {
		if e.Topic == topic {
			return &e, true
		}
	}
	return nil, false
}

func isDynamicType(typ string) bool {
	return typ == "string" || typ == "bytes" ||
		strings.HasSuffix(typ, "[]") ||
		strings.Contains(typ, "[]")
}

func (a *ABI) DecodeCalldata(data []byte) (*DecodedCall, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("calldata too short")
	}
	var selector [4]byte
	copy(selector[:], data[:4])

	method, ok := a.MethodBySelector(selector)
	if !ok {
		return nil, fmt.Errorf("unknown function selector: 0x%s", hex.EncodeToString(selector[:]))
	}

	args := make(map[string]any)
	paramData := data[4:]

	for i, input := range method.Inputs {
		offset := i * 32
		if offset+32 > len(paramData) {
			break
		}

		word := paramData[offset : offset+32]

		if isDynamicType(input.Type) {
			dataOffset := new(big.Int).SetBytes(word).Int64()
			if int(dataOffset) >= len(paramData) {
				args[input.Name] = nil
				continue
			}

			dynData := paramData[dataOffset:]
			if len(dynData) < 32 {
				args[input.Name] = nil
				continue
			}

			length := new(big.Int).SetBytes(dynData[:32]).Int64()
			dynData = dynData[32:]

			if int(length) > len(dynData) {
				args[input.Name] = nil
				continue
			}

			switch {
			case input.Type == "string":
				args[input.Name] = string(dynData[:length])
			case input.Type == "bytes":
				args[input.Name] = "0x" + hex.EncodeToString(dynData[:length])
			default:
				elemCount := int(length)
				elems := make([]string, 0, elemCount)
				for j := 0; j < elemCount && (j+1)*32 <= len(dynData); j++ {
					elemWord := dynData[j*32 : (j+1)*32]
					elems = append(elems, "0x"+hex.EncodeToString(elemWord))
				}
				args[input.Name] = elems
			}
			continue
		}

		switch {
		case input.Type == "address":
			args[input.Name] = "0x" + hex.EncodeToString(word[12:32])
		case input.Type == "bool":
			args[input.Name] = word[31] != 0
		case input.Type == "bytes32":
			args[input.Name] = "0x" + hex.EncodeToString(word)
		case strings.HasPrefix(input.Type, "uint"):
			val := new(big.Int).SetBytes(word)
			args[input.Name] = val
		case strings.HasPrefix(input.Type, "int"):
			val := new(big.Int).SetBytes(word)
			if word[0]&0x80 != 0 {
				max := new(big.Int).Lsh(big.NewInt(1), 256)
				val.Sub(val, max)
			}
			args[input.Name] = val
		case strings.HasPrefix(input.Type, "bytes"):
			args[input.Name] = "0x" + hex.EncodeToString(word)
		default:
			args[input.Name] = "0x" + hex.EncodeToString(word)
		}
	}

	return &DecodedCall{
		Method: method.Name,
		Args:   args,
	}, nil
}

func (a *ABI) EncodeCall(method string, args ...any) ([]byte, error) {
	m, ok := a.Methods[method]
	if !ok {
		return nil, fmt.Errorf("unknown method: %s", method)
	}

	if len(args) != len(m.Inputs) {
		return nil, fmt.Errorf("expected %d arguments for %s, got %d", len(m.Inputs), method, len(args))
	}

	result := make([]byte, 4, 4+len(args)*32)
	copy(result, m.Selector[:])

	for i, arg := range args {
		word := make([]byte, 32)
		inputType := m.Inputs[i].Type

		switch {
		case strings.HasPrefix(inputType, "uint"):
			switch v := arg.(type) {
			case *big.Int:
				b := v.Bytes()
				copy(word[32-len(b):], b)
			case int64:
				val := new(big.Int).SetInt64(v)
				b := val.Bytes()
				copy(word[32-len(b):], b)
			case uint64:
				val := new(big.Int).SetUint64(v)
				b := val.Bytes()
				copy(word[32-len(b):], b)
			case int:
				val := new(big.Int).SetInt64(int64(v))
				b := val.Bytes()
				copy(word[32-len(b):], b)
			default:
				return nil, fmt.Errorf("unsupported value type for %s: %T", inputType, arg)
			}

		case inputType == "address":
			switch v := arg.(type) {
			case string:
				addr := strings.TrimPrefix(v, "0x")
				b, err := hex.DecodeString(addr)
				if err != nil {
					return nil, fmt.Errorf("invalid address: %w", err)
				}
				if len(b) != 20 {
					return nil, fmt.Errorf("address must be 20 bytes, got %d", len(b))
				}
				copy(word[12:], b)
			case []byte:
				if len(v) != 20 {
					return nil, fmt.Errorf("address must be 20 bytes, got %d", len(v))
				}
				copy(word[12:], v)
			default:
				return nil, fmt.Errorf("unsupported value type for address: %T", arg)
			}

		case inputType == "bool":
			switch v := arg.(type) {
			case bool:
				if v {
					word[31] = 1
				}
			default:
				return nil, fmt.Errorf("unsupported value type for bool: %T", arg)
			}

		case strings.HasPrefix(inputType, "int"):
			switch v := arg.(type) {
			case *big.Int:
				if v.Sign() >= 0 {
					b := v.Bytes()
					copy(word[32-len(b):], b)
				} else {
					max := new(big.Int).Lsh(big.NewInt(1), 256)
					pos := new(big.Int).Add(max, v)
					b := pos.Bytes()
					copy(word[32-len(b):], b)
				}
			case int64:
				val := new(big.Int).SetInt64(v)
				if val.Sign() >= 0 {
					b := val.Bytes()
					copy(word[32-len(b):], b)
				} else {
					max := new(big.Int).Lsh(big.NewInt(1), 256)
					pos := new(big.Int).Add(max, val)
					b := pos.Bytes()
					copy(word[32-len(b):], b)
				}
			case int:
				val := new(big.Int).SetInt64(int64(v))
				if val.Sign() >= 0 {
					b := val.Bytes()
					copy(word[32-len(b):], b)
				} else {
					max := new(big.Int).Lsh(big.NewInt(1), 256)
					pos := new(big.Int).Add(max, val)
					b := pos.Bytes()
					copy(word[32-len(b):], b)
				}
			default:
				return nil, fmt.Errorf("unsupported value type for %s: %T", inputType, arg)
			}

		case inputType == "bytes32":
			switch v := arg.(type) {
			case string:
				b, err := hex.DecodeString(strings.TrimPrefix(v, "0x"))
				if err != nil {
					return nil, fmt.Errorf("invalid bytes32: %w", err)
				}
				if len(b) > 32 {
					return nil, fmt.Errorf("bytes32 value too long: %d bytes", len(b))
				}
				copy(word[:len(b)], b)
			case [32]byte:
				copy(word[:], v[:])
			case []byte:
				if len(v) > 32 {
					return nil, fmt.Errorf("bytes32 value too long: %d bytes", len(v))
				}
				copy(word[:len(v)], v)
			default:
				return nil, fmt.Errorf("unsupported value type for bytes32: %T", arg)
			}

		default:
			return nil, fmt.Errorf("unsupported ABI type: %s", inputType)
		}

		result = append(result, word...)
	}

	return result, nil
}

func buildSignature(name string, inputs []Argument) string {
	types := make([]string, len(inputs))
	for i, inp := range inputs {
		types[i] = inp.Type
	}
	return name + "(" + strings.Join(types, ",") + ")"
}

func keccak256(data []byte) [32]byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	var result [32]byte
	h.Sum(result[:0])
	return result
}

func computeSelector(sig string) [4]byte {
	hash := keccak256([]byte(sig))
	var selector [4]byte
	copy(selector[:], hash[:4])
	return selector
}

func computeTopic(sig string) [32]byte {
	return keccak256([]byte(sig))
}
