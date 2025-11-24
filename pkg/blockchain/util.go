package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// maxUint256 is the maximum uint256 value (2^256 - 1). Useful for setting
// ERC-20 allowances to "unlimited".
var maxUint256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

// GetAddressFromPrivateKeyECDSA derives the Ethereum address from the given
// ECDSA private key. It returns nil if the key is nil or its public part cannot
// be asserted to *ecdsa.PublicKey.
func GetAddressFromPrivateKeyECDSA(privateKeyECDSA *ecdsa.PrivateKey) *common.Address {
	if privateKeyECDSA == nil {
		return nil
	}
	publicKey := privateKeyECDSA.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil
	}
	addr := crypto.PubkeyToAddress(*publicKeyECDSA)
	return &addr
}

// ParsePrivateKeyECDSA parses a hex-encoded ECDSA private key and returns the
// corresponding Ethereum address together with the private key object.
// It returns an error if the hex string is invalid or the public key cannot be
// derived from the private key.
func ParsePrivateKeyECDSA(privateKey string) (common.Address, *ecdsa.PrivateKey, error) {
	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return common.Address{}, nil, err
	}

	publicKey := privateKeyECDSA.Public()

	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, nil, errors.New("failed to get public key")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address, privateKeyECDSA, nil
}

// BigIntToBytes converts a *big.Int value to a 32-byte big-endian slice, using
// the same formatting that Ethereum commonly applies to integers in ABI/keccak
// contexts (common.BigToHash).
func BigIntToBytes(value *big.Int) []byte {
	return common.BigToHash(value).Bytes()
}

// AsiToAasi converts an ASI amount to its smallest unit AASI (18 decimals).
//
// Supported input types for iamount: string, float64, int64, decimal.Decimal,
// *decimal.Decimal. Any other type results in an error.
//
// The returned value is a *big.Int representing amount * 10^18.
func AsiToAasi(iamount any) (asi *big.Int, err error) {
	base := 10
	amount := decimal.NewFromFloat(0)
	switch v := iamount.(type) {
	case string:
		amount, err = decimal.NewFromString(v)
		if err != nil {
			zap.L().Error("Failed to convert string to decimal", zap.Error(err))
			return nil, err
		}
	case float64:
		amount = decimal.NewFromFloat(v)
	case int64:
		amount = decimal.NewFromFloat(float64(v))
	case decimal.Decimal:
		amount = v
	case *decimal.Decimal:
		amount = *v
	default:
		zap.L().Error("Unsupported type")

	}
	dec, pow := float64(10), float64(18)
	mul := decimal.NewFromFloat(dec).Pow(decimal.NewFromFloat(pow))
	result := amount.Mul(mul)

	asi = new(big.Int)
	asi.SetString(result.String(), base)

	return
}

// AasiToAsi converts an AASI amount (smallest unit, 18 decimals) into ASI as
// a decimal.Decimal with 18 digits of precision.
//
// Supported input types for ivalue: string, *big.Int, int.
// Any other type results in decimal.Zero and logs an error.
func AasiToAsi(ivalue any) decimal.Decimal {
	value := new(big.Int)
	base := 10
	switch v := ivalue.(type) {
	case string:
		value.SetString(v, base)
	case *big.Int:
		value = v
	case int:
		value.SetInt64(int64(v))
	default:
		zap.L().Error("Unsupported type")
		return decimal.Zero
	}
	dec, pow := float64(10), float64(18)
	mul := decimal.NewFromFloat(dec).Pow(decimal.NewFromFloat(pow))
	num, err := decimal.NewFromString(value.String())
	if err != nil {
		zap.L().Error("Failed to convert string to decimal", zap.Error(err))
	}
	precision := int32(18)
	result := num.DivRound(mul, precision)

	return result
}

// uint64ToBytes encodes a uint64 as an 8-byte big-endian slice.
func uint64ToBytes(val uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, val)
	return buf
}

// StringToBytes32 returns a right-padded [32]byte containing at most the first
// 32 bytes of the provided string.
func StringToBytes32(str string) [32]byte {
	var byte32 [32]byte
	copy(byte32[:], str)
	return byte32
}

// GetSignature produces an Ethereum-compatible personal-sign (EIP-191 style)
// signature over the given message. It hashes the payload as
// keccak256("\x19Ethereum Signed Message:\n32" || keccak256(message)) and
// signs with the provided ECDSA private key.
//
// Returns the 65-byte signature (R||S||V). On signing error it logs and returns nil.
func GetSignature(message []byte, privateKeyECDSA *ecdsa.PrivateKey) []byte {
	hash := crypto.Keccak256(
		HashPrefix32Bytes,
		crypto.Keccak256(message),
	)

	signature, err := crypto.Sign(hash, privateKeyECDSA)
	if err != nil {
		zap.L().Error("Failed to sign message", zap.Error(err))
	}

	return signature
}

// Bytes32ArrayToStrings converts an array of [32]byte values into a slice of strings,
// trimming trailing NUL bytes on the right of each element.
func Bytes32ArrayToStrings(arr [][32]byte) []string {
	result := make([]string, len(arr))
	for i, b := range arr {
		// b[:] is the 32-byte slice; trim trailing '\x00'.
		clean := bytes.TrimRight(b[:], "\x00")
		result[i] = string(clean)
	}
	return result
}
