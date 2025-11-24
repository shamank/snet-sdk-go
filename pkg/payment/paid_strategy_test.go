package payment

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-sdk-go/pkg/model"
	gmd "google.golang.org/grpc/metadata"
)

// TestPaidStrategy_GRPCMetadata_HeadersAndSignature ensures metadata includes the expected headers and a valid signature.
func TestPaidStrategy_GRPCMetadata_HeadersAndSignature(t *testing.T) {
	priv := mustKey(t)

	channelID := big.NewInt(123)
	nonce := big.NewInt(9)
	amount := big.NewInt(777)
	mpe := common.HexToAddress("0x00000000000000000000000000000000000000Ab")

	ps := &PaidStrategy{
		serviceMetadata: &model.ServiceMetadata{MPEAddress: mpe.Hex()},
		channelID:       channelID,
		nonce:           nonce,
		signedAmount:    amount,
		privateKeyECDSA: priv,
	}

	ctx := ps.GRPCMetadata(context.Background())

	md, ok := gmd.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("no metadata in context")
	}

	if got := mustOne(t, md, PaymentTypeHeader); got != "escrow" {
		t.Fatalf("%s=%q; want escrow", PaymentTypeHeader, got)
	}
	if got := mustOne(t, md, PaymentChannelIDHeader); got != channelID.String() {
		t.Fatalf("%s=%q; want %s", PaymentChannelIDHeader, got, channelID.String())
	}
	if got := mustOne(t, md, PaymentChannelNonceHeader); got != nonce.String() {
		t.Fatalf("%s=%q; want %s", PaymentChannelNonceHeader, got, nonce.String())
	}
	if got := mustOne(t, md, PaymentChannelAmountHeader); got != amount.String() {
		t.Fatalf("%s=%q; want %s", PaymentChannelAmountHeader, got, amount.String())
	}

	sig := mustOne(t, md, PaymentChannelSignatureHeader)
	_ = mustSignature65(t, sig) // Validates that the signature decodes to 65 bytes.
}
