package training

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-sdk-go/pkg/blockchain"
)

func mustGenerateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	return k
}

func TestNewAuth(t *testing.T) {
	addr := "0x1234567890123456789012345678901234567890"
	msg := "test_method"
	blockNumber := uint64(12345)
	signature := []byte("test_signature")

	auth := newAuth(addr, msg, blockNumber, signature)

	if auth == nil {
		t.Fatal("expected non-nil auth")
	}

	if auth.SignerAddress != addr {
		t.Fatalf("expected SignerAddress=%s, got %s", addr, auth.SignerAddress)
	}

	if auth.Message != msg {
		t.Fatalf("expected Message=%s, got %s", msg, auth.Message)
	}

	if auth.CurrentBlock != blockNumber {
		t.Fatalf("expected CurrentBlock=%d, got %d", blockNumber, auth.CurrentBlock)
	}

	if !bytes.Equal(auth.Signature, signature) {
		t.Fatalf("expected Signature=%v, got %v", signature, auth.Signature)
	}
}

func TestGetSignature(t *testing.T) {
	priv := mustGenerateKey(t)
	methodName := "create_model"
	blockNumber := big.NewInt(12345)

	signature, err := getSignature(methodName, blockNumber, priv)
	if err != nil {
		t.Fatalf("getSignature failed: %v", err)
	}

	if len(signature) != 65 {
		t.Fatalf("expected signature length 65, got %d", len(signature))
	}

	message := bytes.Join([][]byte{
		[]byte(methodName),
		crypto.PubkeyToAddress(priv.PublicKey).Bytes(),
		math.U256Bytes(blockNumber),
	}, nil)

	hash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(message),
	)

	pubKey, err := crypto.SigToPub(hash, signature)
	if err != nil {
		t.Fatalf("failed to recover public key: %v", err)
	}

	expectedAddr := crypto.PubkeyToAddress(priv.PublicKey)
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	if expectedAddr != recoveredAddr {
		t.Fatalf("signature verification failed: expected %s, got %s",
			expectedAddr.Hex(), recoveredAddr.Hex())
	}
}

func TestGetSignature_DifferentMethods(t *testing.T) {
	priv := mustGenerateKey(t)
	blockNumber := big.NewInt(100)

	methods := []string{
		"get_all_models",
		"get_model",
		"create_model",
		"update_model",
		"delete_model",
	}

	signatures := make(map[string][]byte)

	for _, method := range methods {
		sig, err := getSignature(method, blockNumber, priv)
		if err != nil {
			t.Fatalf("getSignature(%s) failed: %v", method, err)
		}
		signatures[method] = sig
	}

	for i, method1 := range methods {
		for j, method2 := range methods {
			if i != j {
				if bytes.Equal(signatures[method1], signatures[method2]) {
					t.Fatalf("signatures for %s and %s should be different",
						method1, method2)
				}
			}
		}
	}
}

func TestGetSignature_DifferentBlocks(t *testing.T) {
	priv := mustGenerateKey(t)
	methodName := "test_method"

	sig1, err := getSignature(methodName, big.NewInt(100), priv)
	if err != nil {
		t.Fatalf("getSignature failed: %v", err)
	}

	sig2, err := getSignature(methodName, big.NewInt(101), priv)
	if err != nil {
		t.Fatalf("getSignature failed: %v", err)
	}

	if bytes.Equal(sig1, sig2) {
		t.Fatal("signatures for different block numbers should be different")
	}
}

func TestGetSignature_DifferentKeys(t *testing.T) {
	priv1 := mustGenerateKey(t)
	priv2 := mustGenerateKey(t)
	methodName := "test_method"
	blockNumber := big.NewInt(100)

	sig1, err := getSignature(methodName, blockNumber, priv1)
	if err != nil {
		t.Fatalf("getSignature failed: %v", err)
	}

	sig2, err := getSignature(methodName, blockNumber, priv2)
	if err != nil {
		t.Fatalf("getSignature failed: %v", err)
	}

	if bytes.Equal(sig1, sig2) {
		t.Fatal("signatures from different keys should be different")
	}
}

func TestNewTrainingClient(t *testing.T) {
	priv := mustGenerateKey(t)
	currentBlockNumber := func() (*big.Int, error) {
		return big.NewInt(100), nil
	}

	client := &TrainingClient{
		privateKey:         priv,
		currentBlockNumber: currentBlockNumber,
	}

	if client.privateKey != priv {
		t.Fatal("private key not set correctly")
	}

	if client.currentBlockNumber == nil {
		t.Fatal("currentBlockNumber function not set")
	}

	block, err := client.currentBlockNumber()
	if err != nil {
		t.Fatalf("currentBlockNumber() failed: %v", err)
	}

	if block.Cmp(big.NewInt(100)) != 0 {
		t.Fatalf("expected block 100, got %s", block)
	}
}
