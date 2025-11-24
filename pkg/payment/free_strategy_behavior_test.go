package payment

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/grpc"
	gmd "google.golang.org/grpc/metadata"
)

type mockFreeCallClient struct {
	token      *FreeCallToken
	available  uint64
	lastToken  *GetFreeCallTokenRequest
	lastStatus *FreeCallStateRequest
}

func (m *mockFreeCallClient) GetFreeCallsAvailable(ctx context.Context, in *FreeCallStateRequest, opts ...grpc.CallOption) (*FreeCallStateReply, error) {
	m.lastStatus = in
	return &FreeCallStateReply{FreeCallsAvailable: m.available}, nil
}

func (m *mockFreeCallClient) GetFreeCallToken(ctx context.Context, in *GetFreeCallTokenRequest, opts ...grpc.CallOption) (*FreeCallToken, error) {
	m.lastToken = in
	return m.token, nil
}

func TestFreeStrategyRefreshAndMetadata(t *testing.T) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	mock := &mockFreeCallClient{
		token: &FreeCallToken{Token: []byte("token123")},
	}

	strategy := &FreeStrategy{
		Token:            nil,
		orgID:            "org",
		groupID:          "group",
		serviceID:        "service",
		signerPrivateKey: priv,
		signerAddress:    crypto.PubkeyToAddress(priv.PublicKey),
		stateClient:      mock,
		blockNumber: func(ctx context.Context) (*big.Int, error) {
			return big.NewInt(55), nil
		},
	}

	if err := strategy.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh error: %v", err)
	}

	if string(strategy.Token) != "token123" {
		t.Fatalf("unexpected token: %s", strategy.Token)
	}
	if mock.lastToken == nil || mock.lastToken.CurrentBlock != 55 {
		t.Fatalf("expected token request to capture block number, got %#v", mock.lastToken)
	}

	ctx := strategy.GRPCMetadata(context.Background())
	if ctx == nil {
		t.Fatal("expected metadata context")
	}

	md, ok := gmd.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("no metadata in context")
	}
	if got := mustOne(t, md, PaymentTypeHeader); got != "free-call" {
		t.Fatalf("PaymentType header mismatch: %s", got)
	}
	if got := mustOne(t, md, FreeCallAuthTokenHeader); got != "token123" {
		t.Fatalf("token header mismatch: %s", got)
	}

	mock.available = 7
	available, err := strategy.GetFreeCallsAvailable(context.Background())
	if err != nil {
		t.Fatalf("GetFreeCallsAvailable error: %v", err)
	}
	if available != 7 {
		t.Fatalf("expected 7 free calls, got %d", available)
	}
	if mock.lastStatus == nil || mock.lastStatus.CurrentBlock != 55 {
		t.Fatalf("expected free call status request to capture block, got %#v", mock.lastStatus)
	}
}
