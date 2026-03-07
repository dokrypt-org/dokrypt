package common

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/sha3"
)

func secp256k1Curve() *secp256k1.KoblitzCurve {
	return secp256k1.S256()
}

func GenerateKeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(secp256k1Curve(), rand.Reader)
}

func DeriveKey(seed []byte, index int) (*ecdsa.PrivateKey, error) {
	h := sha256.New()
	h.Write(seed)
	indexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(indexBytes, uint64(index))
	h.Write(indexBytes)
	keyBytes := h.Sum(nil)

	curve := secp256k1Curve()
	k := new(big.Int).SetBytes(keyBytes)
	k.Mod(k, curve.Params().N)

	if k.Sign() == 0 {
		k.SetInt64(1)
	}

	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = curve
	priv.D = k
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(k.Bytes())

	return priv, nil
}

func AddressFromPrivateKey(key *ecdsa.PrivateKey) string {
	pubBytes := make([]byte, 65)
	pubBytes[0] = 0x04
	xBytes := key.PublicKey.X.Bytes()
	yBytes := key.PublicKey.Y.Bytes()
	copy(pubBytes[1+(32-len(xBytes)):33], xBytes)
	copy(pubBytes[33+(32-len(yBytes)):65], yBytes)

	hash := keccak256(pubBytes[1:])

	addrBytes := hash[12:]

	return toChecksumAddress(addrBytes)
}

func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

func toChecksumAddress(addr []byte) string {
	lowHex := hex.EncodeToString(addr)

	hashBytes := keccak256([]byte(lowHex))

	var result strings.Builder
	result.WriteString("0x")
	for i, c := range lowHex {
		byteVal := hashBytes[i/2]
		var nibble byte
		if i%2 == 0 {
			nibble = byteVal >> 4
		} else {
			nibble = byteVal & 0x0f
		}

		if nibble >= 8 && c >= 'a' && c <= 'f' {
			result.WriteByte(byte(c - 32)) // uppercase
		} else {
			result.WriteByte(byte(c))
		}
	}

	return result.String()
}

func PrivateKeyFromHex(hexKey string) (*ecdsa.PrivateKey, error) {
	hexKey = strings.TrimPrefix(hexKey, "0x")
	hexKey = strings.TrimPrefix(hexKey, "0X")

	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}
	if len(keyBytes) == 0 {
		return nil, fmt.Errorf("empty private key")
	}

	curve := secp256k1Curve()
	k := new(big.Int).SetBytes(keyBytes)

	if k.Sign() == 0 {
		return nil, fmt.Errorf("private key is zero")
	}
	if k.Cmp(curve.Params().N) >= 0 {
		return nil, fmt.Errorf("private key exceeds curve order")
	}

	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = curve
	priv.D = k
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(k.Bytes())

	return priv, nil
}

func PrivateKeyToHex(key *ecdsa.PrivateKey) string {
	b := key.D.Bytes()
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return "0x" + hex.EncodeToString(padded)
}

var DefaultSeed = []byte("dokrypt-default-seed-v1")

func GenerateAccounts(seed []byte, count int) ([]AccountInfo, error) {
	if seed == nil {
		seed = DefaultSeed
	}

	labels := []string{"deployer"}
	for i := 1; i < count; i++ {
		labels = append(labels, fmt.Sprintf("user%d", i))
	}

	accounts := make([]AccountInfo, 0, count)
	for i := range count {
		key, err := DeriveKey(seed, i)
		if err != nil {
			return nil, fmt.Errorf("failed to derive key for index %d: %w", i, err)
		}

		label := ""
		if i < len(labels) {
			label = labels[i]
		}

		accounts = append(accounts, AccountInfo{
			Address:    AddressFromPrivateKey(key),
			PrivateKey: PrivateKeyToHex(key),
			Label:      label,
		})
	}

	return accounts, nil
}

type AccountInfo struct {
	Address    string
	PrivateKey string
	Label      string
}
