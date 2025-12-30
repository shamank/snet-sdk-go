package blockchain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestGetTransactOpts(t *testing.T) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	chainID := big.NewInt(1)

	opts, err := GetTransactOpts(chainID, priv)
	if err != nil {
		t.Fatalf("GetTransactOpts failed: %v", err)
	}

	if opts == nil {
		t.Fatal("expected non-nil TransactOpts")
	}

	if opts.From != crypto.PubkeyToAddress(priv.PublicKey) {
		t.Fatalf("unexpected From address: got %s, want %s",
			opts.From.Hex(),
			crypto.PubkeyToAddress(priv.PublicKey).Hex())
	}
}

func TestGetTransactOpts_NilKey(t *testing.T) {
	chainID := big.NewInt(1)

	defer func() {
		if r := recover(); r != nil {
			// Expected panic on nil key
		}
	}()

	_, _ = GetTransactOpts(chainID, nil)
}

func TestGetTransactOpts_NilChainID(t *testing.T) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	opts, err := GetTransactOpts(nil, priv)
	if err == nil {
		t.Fatal("expected error for nil chainID")
	}
	if opts != nil {
		t.Fatal("expected nil opts on error")
	}
}

func TestGetTransactOpts_DifferentChainIDs(t *testing.T) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	testCases := []struct {
		name    string
		chainID *big.Int
	}{
		{"Mainnet", big.NewInt(1)},
		{"Sepolia", big.NewInt(11155111)},
		{"Custom", big.NewInt(999)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := GetTransactOpts(tc.chainID, priv)
			if err != nil {
				t.Fatalf("GetTransactOpts failed for %s: %v", tc.name, err)
			}
			if opts == nil {
				t.Fatalf("expected non-nil opts for %s", tc.name)
			}
		})
	}
}

func TestEVMClient_GetTransactOpts_NilPrivateKey(t *testing.T) {
	evm := &EVMClient{}

	opts, err := evm.GetTransactOpts(nil)
	if err == nil {
		t.Fatal("expected error for nil private key")
	}
	if opts != nil {
		t.Fatal("expected nil opts")
	}

	expectedErr := "private key is required for transactions"
	if err.Error() != expectedErr {
		t.Fatalf("unexpected error: got %q, want %q", err.Error(), expectedErr)
	}
}
