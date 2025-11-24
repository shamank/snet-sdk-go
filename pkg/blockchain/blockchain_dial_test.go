package blockchain

import (
	"testing"
	"time"
)

func TestInitEvm_Unreachable(t *testing.T) {
	start := time.Now()
	_, err := InitEvm("devnet", "http://127.0.0.1:1", "", nil)
	if err == nil {
		t.Fatal("expected error dialing")
	}
	if time.Since(start) > 6*time.Second {
		t.Fatalf("InitEvm took too long")
	}
}
