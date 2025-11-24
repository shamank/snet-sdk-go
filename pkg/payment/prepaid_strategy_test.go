package payment

import (
	"context"
	"math/big"
	"testing"

	gmd "google.golang.org/grpc/metadata"
)

// TestPrepaidStrategy_GRPCMetadata_Headers verifies that the prepaid strategy populates all required headers.
func TestPrepaidStrategy_GRPCMetadata_Headers(t *testing.T) {
	ps := &PrepaidStrategy{
		Token:     "token-abc",
		channelID: big.NewInt(42),
		nonce:     big.NewInt(7),
	}

	ctx := ps.GRPCMetadata(context.Background())

	md, ok := gmd.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("no metadata in context")
	}

	if got := mustOne(t, md, PaymentTypeHeader); got != "prepaid-call" {
		t.Fatalf("%s=%q; want prepaid-call", PaymentTypeHeader, got)
	}
	if got := mustOne(t, md, PrePaidAuthTokenHeader); got != "token-abc" {
		t.Fatalf("%s=%q; want token-abc", PrePaidAuthTokenHeader, got)
	}
	if got := mustOne(t, md, PaymentChannelIDHeader); got != "42" {
		t.Fatalf("%s=%q; want 42", PaymentChannelIDHeader, got)
	}
	if got := mustOne(t, md, PaymentChannelNonceHeader); got != "7" {
		t.Fatalf("%s=%q; want 7", PaymentChannelNonceHeader, got)
	}
}
