package grpc

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/shamank/snet-sdk-go/internal/testutil/grpcbuf"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// echoProto is a minimal proto definition used for testing dynamic gRPC invocation.
const echoProto = `
syntax = "proto3";
package test;
import "google/protobuf/empty.proto";
service Echo {
  rpc Ping(google.protobuf.Empty) returns (google.protobuf.Empty);
}
`

// echoServer is a test implementation of the Echo service.
type echoServer struct {
	grpcbuf.EchoServer
}

// Ping implements the Echo.Ping RPC method for testing.
func (s *echoServer) Ping(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// startEchoServer starts a test gRPC server listening on a random port.
// It returns the server address and a cleanup function.
func startEchoServer(t *testing.T) (string, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("network operations not permitted in sandbox")
		}
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	srv.RegisterService(&grpcbuf.EchoServiceDesc, &echoServer{})
	go func() { _ = srv.Serve(lis) }()

	return lis.Addr().String(), func() {
		srv.Stop()
		_ = lis.Close()
	}
}

func TestClientCallVariants(t *testing.T) {
	addr, cleanup := startEchoServer(t)
	defer cleanup()

	client := NewClient(addr, map[string]string{"echo.proto": echoProto})
	if client == nil {
		t.Fatal("client should not be nil")
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	t.Run("CallWithJSON", func(t *testing.T) {
		resp, err := client.CallWithJSON(ctx, "Ping", []byte(`{}`))
		if err != nil {
			t.Fatalf("CallWithJSON error: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(resp, &m); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if len(m) != 0 {
			t.Fatalf("expected empty JSON response, got %v", m)
		}
	})

	t.Run("CallWithMap", func(t *testing.T) {
		resp, err := client.CallWithMap(ctx, "Ping", map[string]any{})
		if err != nil {
			t.Fatalf("CallWithMap error: %v", err)
		}
		if len(resp) != 0 {
			t.Fatalf("expected empty map response, got %v", resp)
		}
	})

	t.Run("CallWithProto", func(t *testing.T) {
		msg, err := client.CallWithProto(ctx, "Ping", &emptypb.Empty{})
		if err != nil {
			t.Fatalf("CallWithProto error: %v", err)
		}
		if !proto.Equal(msg, &emptypb.Empty{}) {
			t.Fatalf("unexpected proto response: %v", msg)
		}
	})
}
