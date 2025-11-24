package payment

import (
	"bytes"
	"math/big"
	"testing"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
)

func TestFreeStrategy_msgForFreeCall(t *testing.T) {
	priv := mustKey(t)
	addr := gethcrypto.PubkeyToAddress(priv.PublicKey)

	f := &FreeStrategy{
		Token:            []byte("tok"),
		orgID:            "orgX",
		groupID:          "grpY",
		serviceID:        "svcZ",
		signerPrivateKey: priv,
		signerAddress:    addr,
	}

	block := uint64(12345)
	msg := f.msgForFreeCall(block)

	want := bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature),
		[]byte(f.signerAddress.Hex()),
		[]byte(f.orgID),
		[]byte(f.serviceID),
		[]byte(f.groupID),
		bigIntToBytes(big.NewInt(int64(block))),
		f.Token,
	}, nil)

	if !bytes.Equal(msg, want) {
		t.Fatalf("unexpected message payload\nwant: %x\n got: %x", want, msg)
	}
}
