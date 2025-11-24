package blockchain

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
)

func TestGetAddressFromPrivateKeyECDSA(t *testing.T) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	addr := GetAddressFromPrivateKeyECDSA(priv)
	if addr == nil {
		t.Fatal("expected non-nil address")
	}
	want := crypto.PubkeyToAddress(priv.PublicKey)
	if *addr != want {
		t.Fatalf("unexpected address: got %s want %s", addr.Hex(), want.Hex())
	}

	if GetAddressFromPrivateKeyECDSA(nil) != nil {
		t.Fatal("expected nil for nil key")
	}
}

func TestParsePrivateKeyECDSA(t *testing.T) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	hexKey := hex.EncodeToString(crypto.FromECDSA(priv))

	addr, parsedKey, err := ParsePrivateKeyECDSA(hexKey)
	if err != nil {
		t.Fatalf("ParsePrivateKeyECDSA: %v", err)
	}
	if addr != crypto.PubkeyToAddress(priv.PublicKey) {
		t.Fatalf("unexpected address: %s", addr.Hex())
	}
	if parsedKey.D.Cmp(priv.D) != 0 {
		t.Fatal("parsed key mismatch")
	}

	if _, _, err := ParsePrivateKeyECDSA("zz"); err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestBigIntToBytes(t *testing.T) {
	got := BigIntToBytes(big.NewInt(1))
	if len(got) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(got))
	}
	if got[31] != 1 {
		t.Fatalf("unexpected bytes: %x", got)
	}
}

func TestAsiToAasi(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{"1", "1000000000000000000"},
		{1.5, "1500000000000000000"},
		{int64(2), "2000000000000000000"},
		{decimal.NewFromFloat(0.25), "250000000000000000"},
	}

	for _, tc := range tests {
		got, err := AsiToAasi(tc.input)
		if err != nil {
			t.Fatalf("AsiToAasi(%v) error: %v", tc.input, err)
		}
		if got.String() != tc.expected {
			t.Fatalf("AsiToAasi(%v) = %s, want %s", tc.input, got.String(), tc.expected)
		}
	}

	if _, err := AsiToAasi("not-a-number"); err == nil {
		t.Fatal("expected error for invalid string")
	}
}

func TestAasiToAsi(t *testing.T) {
	val := AasiToAsi("1500000000000000000")
	want := decimal.RequireFromString("1.500000000000000000")
	if !val.Equal(want) {
		t.Fatalf("AasiToAsi mismatch: got %s, want %s", val, want)
	}

	bigVal := big.NewInt(2000000000000000000)
	if got := AasiToAsi(bigVal); !got.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("AasiToAsi(*big.Int) = %s, want 2", got)
	}
}

func TestUint64ToBytes(t *testing.T) {
	got := uint64ToBytes(0x0102030405060708)
	want := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if len(got) != len(want) {
		t.Fatalf("unexpected length: %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d mismatch: got %x want %x", i, got[i], want[i])
		}
	}
}
