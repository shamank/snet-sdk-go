package payment

import (
	"context"
	"math/big"
	"net"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type channelStateServer struct {
	UnimplementedPaymentChannelStateServiceServer
	lastRequest *ChannelStateRequest
	reply       *ChannelStateReply
}

func (s *channelStateServer) GetChannelState(ctx context.Context, req *ChannelStateRequest) (*ChannelStateReply, error) {
	s.lastRequest = req
	return s.reply, nil
}

func TestGetChannelStateFromDaemon(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("network operations not permitted in sandbox")
		}
		t.Fatalf("listen: %v", err)
	}
	server := grpc.NewServer()
	svc := &channelStateServer{
		reply: &ChannelStateReply{
			CurrentNonce:        big.NewInt(3).Bytes(),
			CurrentSignedAmount: big.NewInt(100).Bytes(),
		},
	}
	RegisterPaymentChannelStateServiceServer(server, svc)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(func() {
		server.Stop()
		_ = lis.Close()
	})

	priv := mustKey(t)

	channelID := big.NewInt(42)
	block := big.NewInt(1000)
	mpeAddr := common.HexToAddress("0x00000000000000000000000000000000000000aa")

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("network operations not permitted in sandbox")
		}
		t.Fatalf("dial: %v", err)
	}
	conn.Connect()
	defer conn.Close()

	ctx := context.Background()
	resp, err := GetChannelStateFromDaemon(conn, ctx, mpeAddr, channelID, block, priv)
	if err != nil {
		t.Fatalf("GetChannelStateFromDaemon error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if new(big.Int).SetBytes(resp.CurrentNonce).Cmp(big.NewInt(3)) != 0 {
		t.Fatalf("unexpected nonce: %v", resp.CurrentNonce)
	}
	if svc.lastRequest == nil {
		t.Fatal("server did not receive request")
	}
	if svc.lastRequest.CurrentBlock != block.Uint64() {
		t.Fatalf("block number mismatch: %d", svc.lastRequest.CurrentBlock)
	}
	if len(svc.lastRequest.Signature) != 65 {
		t.Fatalf("expected 65-byte signature, got %d", len(svc.lastRequest.Signature))
	}
}
