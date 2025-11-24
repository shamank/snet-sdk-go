//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/singnet/snet-sdk-go/pkg/blockchain"
)

func TestETHClientChainID(t *testing.T) {
	rpc := os.Getenv("ETH_RPC_URL")
	if rpc == "" {
		t.Skip("ETH_RPC_URL not set")
	}
	cli, err := blockchain.InitEvm("e2e", rpc, "")
	if err != nil {
		t.Fatalf("InitEvm error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := cli.Client.ChainID(ctx)
	if err != nil {
		t.Fatalf("ChainID error: %v", err)
	}
	if id == nil {
		t.Fatal("nil chain id")
	}
}
