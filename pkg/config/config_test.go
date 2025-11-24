package config

import (
	"testing"
	"time"
)

// TestConfigValidate_AppliesDefaults verifies that Validate applies default values
// for LighthouseURL, IpfsURL, and Network when they are not explicitly set.
func TestConfigValidate_AppliesDefaults(t *testing.T) {
	cfg := &Config{
		RPCAddr: "https://rpc.example",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	if cfg.LighthouseURL != "https://gateway.lighthouse.storage/ipfs/" {
		t.Fatalf("unexpected LighthouseURL: %s", cfg.LighthouseURL)
	}
	if cfg.IpfsURL != "https://ipfs.singularitynet.io:443" {
		t.Fatalf("unexpected IpfsURL: %s", cfg.IpfsURL)
	}
	if cfg.Network != Sepolia {
		t.Fatalf("expected default Sepolia network, got %#v", cfg.Network)
	}
}

// TestConfigValidate_RequiresRPC verifies that Validate returns an error
// when RPCAddr is not provided.
func TestConfigValidate_RequiresRPC(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing RPC address")
	}
}

// TestTimeoutsWithDefaults verifies that WithDefaults preserves explicitly set
// timeout values and fills in defaults for zero values.
func TestTimeoutsWithDefaults(t *testing.T) {
	in := Timeouts{
		Dial:        time.Second,
		ChainSubmit: 42 * time.Second,
	}

	out := in.WithDefaults()

	// Provided values should be kept.
	if out.Dial != time.Second {
		t.Fatalf("Dial overwritten: got %v", out.Dial)
	}
	if out.ChainSubmit != 42*time.Second {
		t.Fatalf("ChainSubmit overwritten: got %v", out.ChainSubmit)
	}

	// Zero values filled with defaults.
	if out.GRPCUnary != 12*time.Second {
		t.Fatalf("GRPCUnary default mismatch: %v", out.GRPCUnary)
	}
	if out.ChainRead != 13*time.Second {
		t.Fatalf("ChainRead default mismatch: %v", out.ChainRead)
	}
	if out.ReceiptWait != 90*time.Second {
		t.Fatalf("ReceiptWait default mismatch: %v", out.ReceiptWait)
	}
	if out.StrategyRefresh != 14*time.Second {
		t.Fatalf("StrategyRefresh default mismatch: %v", out.StrategyRefresh)
	}
	if out.PaymentEnsure != 120*time.Second {
		t.Fatalf("PaymentEnsure default mismatch: %v", out.PaymentEnsure)
	}
}
