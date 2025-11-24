package payment

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	gmd "google.golang.org/grpc/metadata"
)

// mustKey generates a secp256k1 private key via go-ethereum helpers.
func mustKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := gethcrypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return k
}

// mustOne extracts exactly one value for the given metadata key.
func mustOne(t *testing.T, md gmd.MD, key string) string {
	t.Helper()
	vals := md.Get(key)
	if len(vals) != 1 {
		t.Fatalf("metadata[%s] = %v; want exactly 1", key, vals)
	}
	return vals[0]
}

// mustSignature65 decodes the signature from base64, hex, or raw bytes and ensures it is 65 bytes long.
func mustSignature65(t *testing.T, s string) []byte {
	t.Helper()
	// Try base64 → hex → raw bytes.
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		if len(b) == 65 {
			return b
		}
	}
	ss := strings.TrimSpace(s)
	ssLower := strings.TrimPrefix(strings.ToLower(ss), "0x")
	if b, err := hex.DecodeString(ssLower); err == nil {
		if len(b) == 65 {
			return b
		}
	}
	// Finally interpret the string as raw binary data.
	raw := []byte(s)
	if len(raw) == 65 {
		return raw
	}
	t.Fatalf("signature: unsupported format/length (got %d bytes); expected base64/hex/raw 65B", len(raw))
	return nil
}
