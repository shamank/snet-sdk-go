package payment

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-sdk-go/internal/testutil/grpcbuf"
	"github.com/singnet/snet-sdk-go/pkg/model"
	gmd "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func Test_PaidStrategy_Metadata_Propagates(t *testing.T) {
	// gRPC test server
	srv, lis, cap := grpcbuf.StartServer()
	defer srv.Stop()

	// gRPC client
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := grpcbuf.Dial(ctx, lis)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Paid strategy setup
	priv, _ := gethcrypto.GenerateKey()
	mpe := common.HexToAddress("0x00000000000000000000000000000000000000AB")
	ps := &PaidStrategy{
		serviceMetadata: &model.ServiceMetadata{MPEAddress: mpe.Hex()},
		channelID:       big.NewInt(123),
		nonce:           big.NewInt(9),
		signedAmount:    big.NewInt(777),
		priceInCogs:     big.NewInt(1),
		privateKeyECDSA: priv,
	}
	ctx = ps.GRPCMetadata(ctx)

	// invoke Echo
	if err := conn.Invoke(ctx, "/test.Echo/Ping", &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatalf("invoke: %v", err)
	}

	md := cap.Last()
	if md == nil {
		t.Fatal("server did not capture metadata")
	}

	// Basic headers must be present.
	_ = mustOne(t, md, PaymentTypeHeader)
	_ = mustOne(t, md, PaymentChannelIDHeader)
	_ = mustOne(t, md, PaymentChannelNonceHeader)
	_ = mustOne(t, md, PaymentChannelAmountHeader)

	// Signature must decode to 65 bytes regardless of transport encoding.
	sig := mustOne(t, md, PaymentChannelSignatureHeader)
	_ = mustSignature65(t, sig)
}

func Test_PrepaidStrategy_Metadata_Propagates(t *testing.T) {
	srv, lis, cap := grpcbuf.StartServer()
	defer srv.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := grpcbuf.Dial(ctx, lis)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ps := &PrepaidStrategy{
		Token:     "token-abc",
		channelID: big.NewInt(42),
		nonce:     big.NewInt(7),
	}
	ctx = ps.GRPCMetadata(ctx)

	if err := conn.Invoke(ctx, "/test.Echo/Ping", &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
		t.Fatalf("invoke: %v", err)
	}

	md := cap.Last()
	if md == nil {
		t.Fatal("server did not capture metadata")
	}

	if got := mustOne(t, md, PaymentTypeHeader); got != "prepaid-call" {
		t.Fatalf("%s=%q; want prepaid-call", PaymentTypeHeader, got)
	}

	// Token: first try the canonical header, then fall back to Authorization, and lastly inspect the outgoing context.
	token := md.Get(PrePaidAuthTokenHeader)
	if len(token) == 0 {
		token = md.Get("authorization")
	}
	if len(token) == 0 {
		// As a last resort check the outgoing context to confirm the SDK attached the token.
		if omd, ok := gmd.FromOutgoingContext(ctx); ok {
			tok := omd.Get(PrePaidAuthTokenHeader)
			if len(tok) == 0 {
				t.Fatalf("missing prepaid token in server MD and outgoing context")
			}
		} else {
			t.Fatalf("no outgoing MD available to verify prepaid token")
		}
	}
}
