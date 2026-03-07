package common

import (
	"encoding/hex"
	"strings"
	"testing"

	"golang.org/x/crypto/sha3"
)

func TestDeriveKey_Deterministic(t *testing.T) {
	seed := []byte("test-seed-abc")

	key1, err := DeriveKey(seed, 0)
	if err != nil {
		t.Fatalf("DeriveKey(seed, 0) error: %v", err)
	}

	key2, err := DeriveKey(seed, 0)
	if err != nil {
		t.Fatalf("DeriveKey(seed, 0) error: %v", err)
	}

	if key1.D.Cmp(key2.D) != 0 {
		t.Error("DeriveKey with same seed and index should produce identical keys")
	}
}

func TestDeriveKey_DifferentIndicesProduceDifferentKeys(t *testing.T) {
	seed := []byte("test-seed-abc")

	key0, err := DeriveKey(seed, 0)
	if err != nil {
		t.Fatalf("DeriveKey(seed, 0) error: %v", err)
	}

	key1, err := DeriveKey(seed, 1)
	if err != nil {
		t.Fatalf("DeriveKey(seed, 1) error: %v", err)
	}

	if key0.D.Cmp(key1.D) == 0 {
		t.Error("Different indices should produce different keys")
	}
}

func TestDeriveKey_DifferentSeedsProduceDifferentKeys(t *testing.T) {
	key1, err := DeriveKey([]byte("seed-a"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	key2, err := DeriveKey([]byte("seed-b"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	if key1.D.Cmp(key2.D) == 0 {
		t.Error("Different seeds should produce different keys")
	}
}

func TestDeriveKey_ResultIsValidKey(t *testing.T) {
	key, err := DeriveKey([]byte("any-seed"), 42)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	if key == nil {
		t.Fatal("DeriveKey returned nil")
	}
	if key.D.Sign() == 0 {
		t.Error("key D scalar should be non-zero")
	}
	if key.PublicKey.X == nil {
		t.Error("public key X should not be nil")
	}
	if key.PublicKey.Y == nil {
		t.Error("public key Y should not be nil")
	}
}

func TestDeriveKey_UsesSecp256k1Curve(t *testing.T) {
	key, err := DeriveKey([]byte("curve-check"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	params := key.PublicKey.Curve.Params()
	if params.Name != "secp256k1" {
		t.Errorf("expected curve secp256k1, got %s", params.Name)
	}
}

func TestDeriveKey_MultipleIndicesAllDifferent(t *testing.T) {
	seed := []byte("determinism-test")
	seen := make(map[string]bool)

	for i := range 10 {
		key, err := DeriveKey(seed, i)
		if err != nil {
			t.Fatalf("DeriveKey(seed, %d) error: %v", i, err)
		}
		hex := key.D.Text(16)
		if seen[hex] {
			t.Errorf("duplicate key at index %d", i)
		}
		seen[hex] = true
	}
}

func TestDeriveKey_DefaultSeed(t *testing.T) {
	key, err := DeriveKey(DefaultSeed, 0)
	if err != nil {
		t.Fatalf("DeriveKey(DefaultSeed, 0) error: %v", err)
	}
	if key == nil {
		t.Fatal("DeriveKey returned nil")
	}
}

func TestAddressFromPrivateKey_HasHexPrefix(t *testing.T) {
	key, err := DeriveKey([]byte("addr-seed"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	addr := AddressFromPrivateKey(key)
	if !strings.HasPrefix(addr, "0x") {
		t.Errorf("address should start with 0x, got: %s", addr)
	}
}

func TestAddressFromPrivateKey_IsValidLength(t *testing.T) {
	key, err := DeriveKey([]byte("addr-seed"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	addr := AddressFromPrivateKey(key)
	if len(addr) != 42 {
		t.Errorf("Address length = %d, want 42, got: %s", len(addr), addr)
	}
}

func TestAddressFromPrivateKey_IsHexString(t *testing.T) {
	key, err := DeriveKey([]byte("addr-seed"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	addr := AddressFromPrivateKey(key)
	hexPart := strings.TrimPrefix(addr, "0x")
	for _, c := range hexPart {
		if !isHexChar(c) {
			t.Errorf("non-hex character %q in address %s", c, addr)
		}
	}
}

func TestAddressFromPrivateKey_Deterministic(t *testing.T) {
	key, err := DeriveKey([]byte("deterministic-addr"), 5)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	addr1 := AddressFromPrivateKey(key)
	addr2 := AddressFromPrivateKey(key)
	if addr1 != addr2 {
		t.Errorf("AddressFromPrivateKey not deterministic: %s != %s", addr1, addr2)
	}
}

func TestAddressFromPrivateKey_DifferentKeysProduceDifferentAddresses(t *testing.T) {
	seed := []byte("addr-diff")
	key0, err := DeriveKey(seed, 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	key1, err := DeriveKey(seed, 1)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	addr0 := AddressFromPrivateKey(key0)
	addr1 := AddressFromPrivateKey(key1)
	if addr0 == addr1 {
		t.Error("Different keys should produce different addresses")
	}
}

func TestAddressFromPrivateKey_IsEIP55Checksummed(t *testing.T) {
	key, err := DeriveKey([]byte("checksum-test"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	addr := AddressFromPrivateKey(key)

	if !isValidEIP55Checksum(addr) {
		t.Errorf("address %s does not pass EIP-55 checksum validation", addr)
	}
}

func TestAddressFromPrivateKey_UsesKeccak256(t *testing.T) {
	key, err := DeriveKey([]byte("keccak-test"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	pubBytes := make([]byte, 64)
	xBytes := key.PublicKey.X.Bytes()
	yBytes := key.PublicKey.Y.Bytes()
	copy(pubBytes[32-len(xBytes):32], xBytes)
	copy(pubBytes[64-len(yBytes):64], yBytes)

	h := sha3.NewLegacyKeccak256()
	h.Write(pubBytes)
	hash := h.Sum(nil)

	expectedLower := hex.EncodeToString(hash[12:])
	addr := AddressFromPrivateKey(key)
	addrLower := strings.ToLower(strings.TrimPrefix(addr, "0x"))

	if addrLower != expectedLower {
		t.Errorf("address mismatch:\n  got:  %s\n  want: 0x%s (lowercased)", addr, expectedLower)
	}
}

func TestAddressFromPrivateKey_MultipleAddressesAllChecksummed(t *testing.T) {
	seed := []byte("multi-checksum")
	for i := range 10 {
		key, err := DeriveKey(seed, i)
		if err != nil {
			t.Fatalf("DeriveKey(seed, %d) error: %v", i, err)
		}
		addr := AddressFromPrivateKey(key)
		if !isValidEIP55Checksum(addr) {
			t.Errorf("account[%d] address %s is not EIP-55 checksummed", i, addr)
		}
	}
}

func TestPrivateKeyToHex_HasHexPrefix(t *testing.T) {
	key, err := DeriveKey([]byte("hex-seed"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	hex := PrivateKeyToHex(key)
	if !strings.HasPrefix(hex, "0x") {
		t.Errorf("private key hex should start with 0x, got: %s", hex)
	}
}

func TestPrivateKeyToHex_IsValidHex(t *testing.T) {
	key, err := DeriveKey([]byte("hex-seed"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	h := PrivateKeyToHex(key)
	hexPart := strings.TrimPrefix(h, "0x")
	for _, c := range hexPart {
		if !isHexChar(c) {
			t.Errorf("non-hex character %q in private key hex", c)
		}
	}
}

func TestPrivateKeyToHex_Is64HexChars(t *testing.T) {
	key, err := DeriveKey([]byte("hex-length"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	h := PrivateKeyToHex(key)
	hexPart := strings.TrimPrefix(h, "0x")
	if len(hexPart) != 64 {
		t.Errorf("private key hex length = %d, want 64 (32 bytes), got: %s", len(hexPart), h)
	}
}

func TestPrivateKeyToHex_Deterministic(t *testing.T) {
	key, err := DeriveKey([]byte("hex-seed"), 3)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	h1 := PrivateKeyToHex(key)
	h2 := PrivateKeyToHex(key)
	if h1 != h2 {
		t.Errorf("PrivateKeyToHex not deterministic: %s != %s", h1, h2)
	}
}

func TestGenerateAccounts_CorrectCount(t *testing.T) {
	accounts, err := GenerateAccounts([]byte("gen-seed"), 5)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}
	if len(accounts) != 5 {
		t.Fatalf("GenerateAccounts() returned %d accounts, want 5", len(accounts))
	}
}

func TestGenerateAccounts_ZeroCount(t *testing.T) {
	accounts, err := GenerateAccounts([]byte("gen-seed"), 0)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}
	if len(accounts) != 0 {
		t.Errorf("GenerateAccounts(0) returned %d accounts, want 0", len(accounts))
	}
}

func TestGenerateAccounts_FirstLabelIsDeployer(t *testing.T) {
	accounts, err := GenerateAccounts([]byte("gen-seed"), 3)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}
	if len(accounts) == 0 {
		t.Fatal("GenerateAccounts returned empty slice")
	}
	if accounts[0].Label != "deployer" {
		t.Errorf("accounts[0].Label = %q, want %q", accounts[0].Label, "deployer")
	}
}

func TestGenerateAccounts_SubsequentLabels(t *testing.T) {
	accounts, err := GenerateAccounts([]byte("gen-seed"), 4)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}

	expected := []string{"deployer", "user1", "user2", "user3"}
	for i, want := range expected {
		if accounts[i].Label != want {
			t.Errorf("accounts[%d].Label = %q, want %q", i, accounts[i].Label, want)
		}
	}
}

func TestGenerateAccounts_AddressesNonEmpty(t *testing.T) {
	accounts, err := GenerateAccounts(DefaultSeed, 5)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}
	for i, acc := range accounts {
		if acc.Address == "" {
			t.Errorf("Account[%d].Address is empty", i)
		}
		if !strings.HasPrefix(acc.Address, "0x") {
			t.Errorf("Account[%d].Address %q missing 0x prefix", i, acc.Address)
		}
	}
}

func TestGenerateAccounts_AddressesAreChecksummed(t *testing.T) {
	accounts, err := GenerateAccounts(DefaultSeed, 5)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}
	for i, acc := range accounts {
		if len(acc.Address) != 42 {
			t.Errorf("Account[%d].Address length = %d, want 42", i, len(acc.Address))
		}
		if !isValidEIP55Checksum(acc.Address) {
			t.Errorf("Account[%d].Address %s is not EIP-55 checksummed", i, acc.Address)
		}
	}
}

func TestGenerateAccounts_PrivateKeysNonEmpty(t *testing.T) {
	accounts, err := GenerateAccounts(DefaultSeed, 5)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}
	for i, acc := range accounts {
		if acc.PrivateKey == "" {
			t.Errorf("Account[%d].PrivateKey is empty", i)
		}
		if !strings.HasPrefix(acc.PrivateKey, "0x") {
			t.Errorf("Account[%d].PrivateKey %q missing 0x prefix", i, acc.PrivateKey)
		}
	}
}

func TestGenerateAccounts_PrivateKeysAre64HexChars(t *testing.T) {
	accounts, err := GenerateAccounts(DefaultSeed, 5)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}
	for i, acc := range accounts {
		hexPart := strings.TrimPrefix(acc.PrivateKey, "0x")
		if len(hexPart) != 64 {
			t.Errorf("Account[%d].PrivateKey hex length = %d, want 64", i, len(hexPart))
		}
	}
}

func TestGenerateAccounts_Deterministic(t *testing.T) {
	seed := []byte("deterministic-gen")

	accounts1, err := GenerateAccounts(seed, 5)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}

	accounts2, err := GenerateAccounts(seed, 5)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}

	for i := range accounts1 {
		if accounts1[i].Address != accounts2[i].Address {
			t.Errorf("Account[%d] not deterministic: %s != %s", i, accounts1[i].Address, accounts2[i].Address)
		}
		if accounts1[i].PrivateKey != accounts2[i].PrivateKey {
			t.Errorf("Account[%d] private key not deterministic", i)
		}
	}
}

func TestGenerateAccounts_AllAddressesUnique(t *testing.T) {
	accounts, err := GenerateAccounts([]byte("unique-seed"), 10)
	if err != nil {
		t.Fatalf("GenerateAccounts() error: %v", err)
	}

	seen := make(map[string]bool)
	for _, acc := range accounts {
		if seen[acc.Address] {
			t.Errorf("duplicate address: %s", acc.Address)
		}
		seen[acc.Address] = true
	}
}

func TestGenerateAccounts_NilSeedUsesDefault(t *testing.T) {
	accounts, err := GenerateAccounts(nil, 3)
	if err != nil {
		t.Fatalf("GenerateAccounts(nil, 3) error: %v", err)
	}
	if len(accounts) != 3 {
		t.Errorf("GenerateAccounts(nil, 3) returned %d accounts, want 3", len(accounts))
	}
}

func TestGenerateAccounts_NilSeedSameAsDefaultSeed(t *testing.T) {
	withNil, err := GenerateAccounts(nil, 3)
	if err != nil {
		t.Fatalf("GenerateAccounts(nil) error: %v", err)
	}

	withDefault, err := GenerateAccounts(DefaultSeed, 3)
	if err != nil {
		t.Fatalf("GenerateAccounts(DefaultSeed) error: %v", err)
	}

	for i := range withNil {
		if withDefault[i].Address != withNil[i].Address {
			t.Errorf("Account[%d]: nil seed address %s != default seed address %s",
				i, withNil[i].Address, withDefault[i].Address)
		}
	}
}

func TestGenerateKeyPair_ReturnsNonNilKey(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}
	if key == nil {
		t.Fatal("GenerateKeyPair() returned nil key")
	}
}

func TestGenerateKeyPair_ProducesUniqueKeys(t *testing.T) {
	key1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	key2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	if key1.D.Cmp(key2.D) == 0 {
		t.Error("Two random key pairs should not share the same D scalar")
	}
}

func TestGenerateKeyPair_UsesSecp256k1(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	params := key.PublicKey.Curve.Params()
	if params.Name != "secp256k1" {
		t.Errorf("expected curve secp256k1, got %s", params.Name)
	}
}

func TestGenerateKeyPair_AddressIsValid(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	addr := AddressFromPrivateKey(key)
	if len(addr) != 42 {
		t.Errorf("Address length = %d, want 42", len(addr))
	}
	if !strings.HasPrefix(addr, "0x") {
		t.Errorf("Address should start with 0x, got: %s", addr)
	}
	if !isValidEIP55Checksum(addr) {
		t.Errorf("Address %s is not EIP-55 checksummed", addr)
	}
}

func TestPrivateKeyFromHex_ValidKeyWith0xPrefix(t *testing.T) {
	original, err := DeriveKey([]byte("fromhex-seed"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	hexStr := PrivateKeyToHex(original)
	parsed, err := PrivateKeyFromHex(hexStr)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex(%q) error: %v", hexStr, err)
	}

	if original.D.Cmp(parsed.D) != 0 {
		t.Error("PrivateKeyFromHex should produce the same key as the original")
	}
}

func TestPrivateKeyFromHex_ValidKeyWithout0xPrefix(t *testing.T) {
	original, err := DeriveKey([]byte("fromhex-noprefix"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	hexStr := PrivateKeyToHex(original)
	noPrefixHex := strings.TrimPrefix(hexStr, "0x")

	parsed, err := PrivateKeyFromHex(noPrefixHex)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex(%q) error: %v", noPrefixHex, err)
	}

	if original.D.Cmp(parsed.D) != 0 {
		t.Error("PrivateKeyFromHex should parse key without 0x prefix")
	}
}

func TestPrivateKeyFromHex_ValidKeyWith0XPrefix(t *testing.T) {
	original, err := DeriveKey([]byte("fromhex-0X"), 0)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	hexStr := PrivateKeyToHex(original)
	uppercasePrefix := "0X" + strings.TrimPrefix(hexStr, "0x")

	parsed, err := PrivateKeyFromHex(uppercasePrefix)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex(%q) error: %v", uppercasePrefix, err)
	}

	if original.D.Cmp(parsed.D) != 0 {
		t.Error("PrivateKeyFromHex should handle 0X prefix")
	}
}

func TestPrivateKeyFromHex_InvalidHexString(t *testing.T) {
	_, err := PrivateKeyFromHex("0xZZZZnotvalidhex")
	if err == nil {
		t.Error("PrivateKeyFromHex should return error for invalid hex")
	}
}

func TestPrivateKeyFromHex_EmptyString(t *testing.T) {
	_, err := PrivateKeyFromHex("")
	if err == nil {
		t.Error("PrivateKeyFromHex should return error for empty string")
	}
}

func TestPrivateKeyFromHex_ZeroKey(t *testing.T) {
	_, err := PrivateKeyFromHex("0x0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Error("PrivateKeyFromHex should return error for zero key")
	}
}

func TestPrivateKeyFromHex_OnlyPrefix(t *testing.T) {
	_, err := PrivateKeyFromHex("0x")
	if err == nil {
		t.Error("PrivateKeyFromHex should return error for just '0x'")
	}
}

func TestPrivateKeyFromHex_ValidShortKey(t *testing.T) {
	key, err := PrivateKeyFromHex("0x01")
	if err != nil {
		t.Fatalf("PrivateKeyFromHex('0x01') error: %v", err)
	}
	if key.D.Int64() != 1 {
		t.Errorf("expected D=1, got %d", key.D.Int64())
	}
}

func TestPrivateKeyFromHex_AddressDerivedCorrectly(t *testing.T) {
	original, err := DeriveKey([]byte("roundtrip-addr"), 3)
	if err != nil {
		t.Fatalf("DeriveKey error: %v", err)
	}

	hexStr := PrivateKeyToHex(original)
	parsed, err := PrivateKeyFromHex(hexStr)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex error: %v", err)
	}

	addr1 := AddressFromPrivateKey(original)
	addr2 := AddressFromPrivateKey(parsed)
	if addr1 != addr2 {
		t.Errorf("addresses should match after round-trip: %s != %s", addr1, addr2)
	}
}

func TestPrivateKeyFromHex_UsesSecp256k1(t *testing.T) {
	key, err := PrivateKeyFromHex("0x0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("PrivateKeyFromHex error: %v", err)
	}

	params := key.PublicKey.Curve.Params()
	if params.Name != "secp256k1" {
		t.Errorf("expected curve secp256k1, got %s", params.Name)
	}
}

func TestPrivateKeyFromHex_PublicKeyIsPopulated(t *testing.T) {
	key, err := PrivateKeyFromHex("0x0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatalf("PrivateKeyFromHex error: %v", err)
	}

	if key.PublicKey.X == nil || key.PublicKey.Y == nil {
		t.Error("public key X and Y should not be nil")
	}
}

func TestPrivateKeyRoundTrip_MultipleKeys(t *testing.T) {
	seed := []byte("roundtrip-seed")
	for i := range 10 {
		original, err := DeriveKey(seed, i)
		if err != nil {
			t.Fatalf("DeriveKey(seed, %d) error: %v", i, err)
		}

		hexStr := PrivateKeyToHex(original)
		parsed, err := PrivateKeyFromHex(hexStr)
		if err != nil {
			t.Fatalf("PrivateKeyFromHex(%q) error: %v", hexStr, err)
		}

		if original.D.Cmp(parsed.D) != 0 {
			t.Errorf("round-trip failed for key at index %d: D values differ", i)
		}
	}
}

func isHexChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isValidEIP55Checksum(addr string) bool {
	if len(addr) != 42 || !strings.HasPrefix(addr, "0x") {
		return false
	}

	hexPart := addr[2:]
	lowered := strings.ToLower(hexPart)

	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(lowered))
	hashBytes := h.Sum(nil)

	for i, c := range hexPart {
		byteVal := hashBytes[i/2]
		var nibble byte
		if i%2 == 0 {
			nibble = byteVal >> 4
		} else {
			nibble = byteVal & 0x0f
		}

		if c >= 'a' && c <= 'f' {
			if nibble >= 8 {
				return false // should have been uppercased
			}
		} else if c >= 'A' && c <= 'F' {
			if nibble < 8 {
				return false // should have been lowercased
			}
		}
	}

	return true
}
