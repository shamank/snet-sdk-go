package grpc

import (
	"context"
	"testing"
	"time"
)

func TestDialEndpoint_Timeout(t *testing.T) {
	// Non-routable IP should hang until timeout; we verify we return quickly.
	ctx := context.Background()
	start := time.Now()
	_, err := DialEndpoint(ctx, "10.255.255.1:65535", 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected error")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("call exceeded 1s: %v", elapsed)
	}
}
