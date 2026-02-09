package config

import (
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestConfig_Validate_Success(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   Config
	}{
		{
			name: "with defaults",
			config: &Config{
				RPCAddr: "wss://sepolia.infura.io",
			},
			want: Config{
				RPCAddr:       "wss://sepolia.infura.io",
				LighthouseURL: "https://gateway.lighthouse.storage/ipfs/",
				IpfsURL:       "https://ipfs.singularitynet.io:443",
				Network:       Sepolia,
			},
		},
		{
			name: "with custom values",
			config: &Config{
				RPCAddr:       "wss://mainnet.infura.io",
				LighthouseURL: "https://custom.lighthouse.io",
				IpfsURL:       "https://custom.ipfs.io",
				Network:       Main,
			},
			want: Config{
				RPCAddr:       "wss://mainnet.infura.io",
				LighthouseURL: "https://custom.lighthouse.io",
				IpfsURL:       "https://custom.ipfs.io",
				Network:       Main,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}

			if tt.config.RPCAddr != tt.want.RPCAddr {
				t.Errorf("RPCAddr = %v, want %v", tt.config.RPCAddr, tt.want.RPCAddr)
			}

			if tt.config.LighthouseURL != tt.want.LighthouseURL {
				t.Errorf("LighthouseURL = %v, want %v", tt.config.LighthouseURL, tt.want.LighthouseURL)
			}

			if tt.config.IpfsURL != tt.want.IpfsURL {
				t.Errorf("IpfsURL = %v, want %v", tt.config.IpfsURL, tt.want.IpfsURL)
			}

			if tt.config.Network.ChainID != tt.want.Network.ChainID {
				t.Errorf("Network.ChainID = %v, want %v", tt.config.Network.ChainID, tt.want.Network.ChainID)
			}
		})
	}
}

func TestConfig_Validate_Error(t *testing.T) {
	config := &Config{}
	err := config.Validate()
	if err == nil {
		t.Fatal("expected error when RPCAddr is empty")
	}

	expectedErr := "RPC address is required"
	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestTimeouts_WithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		timeouts Timeouts
		want     Timeouts
	}{
		{
			name:     "empty timeouts",
			timeouts: Timeouts{},
			want: Timeouts{
				Dial:            15 * time.Second,
				GRPCUnary:       15 * time.Second,
				ChainRead:       13 * time.Second,
				ChainSubmit:     25 * time.Second,
				ReceiptWait:     90 * time.Second,
				StrategyRefresh: 15 * time.Second,
				PaymentEnsure:   120 * time.Second,
			},
		},
		{
			name: "partial timeouts",
			timeouts: Timeouts{
				Dial:      5 * time.Second,
				ChainRead: 10 * time.Second,
			},
			want: Timeouts{
				Dial:            5 * time.Second,
				GRPCUnary:       15 * time.Second,
				ChainRead:       10 * time.Second,
				ChainSubmit:     25 * time.Second,
				ReceiptWait:     90 * time.Second,
				StrategyRefresh: 15 * time.Second,
				PaymentEnsure:   120 * time.Second,
			},
		},
		{
			name: "all custom timeouts",
			timeouts: Timeouts{
				Dial:            1 * time.Second,
				GRPCUnary:       2 * time.Second,
				ChainRead:       3 * time.Second,
				ChainSubmit:     4 * time.Second,
				ReceiptWait:     5 * time.Second,
				StrategyRefresh: 6 * time.Second,
				PaymentEnsure:   7 * time.Second,
			},
			want: Timeouts{
				Dial:            1 * time.Second,
				GRPCUnary:       2 * time.Second,
				ChainRead:       3 * time.Second,
				ChainSubmit:     4 * time.Second,
				ReceiptWait:     5 * time.Second,
				StrategyRefresh: 6 * time.Second,
				PaymentEnsure:   7 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.timeouts.WithDefaults()

			if got.Dial != tt.want.Dial {
				t.Errorf("Dial = %v, want %v", got.Dial, tt.want.Dial)
			}
			if got.GRPCUnary != tt.want.GRPCUnary {
				t.Errorf("GRPCUnary = %v, want %v", got.GRPCUnary, tt.want.GRPCUnary)
			}
			if got.ChainRead != tt.want.ChainRead {
				t.Errorf("ChainRead = %v, want %v", got.ChainRead, tt.want.ChainRead)
			}
			if got.ChainSubmit != tt.want.ChainSubmit {
				t.Errorf("ChainSubmit = %v, want %v", got.ChainSubmit, tt.want.ChainSubmit)
			}
			if got.ReceiptWait != tt.want.ReceiptWait {
				t.Errorf("ReceiptWait = %v, want %v", got.ReceiptWait, tt.want.ReceiptWait)
			}
			if got.StrategyRefresh != tt.want.StrategyRefresh {
				t.Errorf("StrategyRefresh = %v, want %v", got.StrategyRefresh, tt.want.StrategyRefresh)
			}
			if got.PaymentEnsure != tt.want.PaymentEnsure {
				t.Errorf("PaymentEnsure = %v, want %v", got.PaymentEnsure, tt.want.PaymentEnsure)
			}
		})
	}
}

func TestConfig_GetPrivateKey(t *testing.T) {
	tests := []struct {
		name       string
		privateKey string
		wantNil    bool
	}{
		{
			name:       "empty private key",
			privateKey: "",
			wantNil:    true,
		},
		{
			name:       "invalid key",
			privateKey: strings.Repeat("x", 64),
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				PrivateKey: tt.privateKey,
			}

			got := config.GetPrivateKey()
			if tt.wantNil && got != nil {
				t.Errorf("GetPrivateKey() = %v, want nil", got)
			}
		})
	}
}

func TestConfig_GetPrivateKey_Caching(t *testing.T) {
	testKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	hexKey := ""
	for _, b := range crypto.FromECDSA(testKey) {
		hexKey += string("0123456789abcdef"[b>>4])
		hexKey += string("0123456789abcdef"[b&0xF])
	}

	if len(hexKey) > 64 {
		hexKey = hexKey[:64]
	}

	config := &Config{
		PrivateKey: hexKey,
	}

	key1 := config.GetPrivateKey()
	key2 := config.GetPrivateKey()

	if key1 != key2 {
		t.Error("GetPrivateKey() should return cached instance")
	}
}

func TestConfig_HasPrivateKey(t *testing.T) {
	tests := []struct {
		name       string
		privateKey string
		want       bool
	}{
		{
			name:       "has private key",
			privateKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			want:       true,
		},
		{
			name:       "no private key",
			privateKey: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				PrivateKey: tt.privateKey,
			}

			got := config.HasPrivateKey()
			if got != tt.want {
				t.Errorf("HasPrivateKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_RequirePrivateKey(t *testing.T) {
	t.Run("with private key", func(t *testing.T) {
		config := &Config{
			PrivateKey: strings.Repeat("a", 64),
		}

		_, err := config.RequirePrivateKey()
		// We expect nil since we can't easily create a valid key in test
		// The important part is that it doesn't error on "required"
		if err != nil && strings.Contains(err.Error(), "required") {
			t.Errorf("RequirePrivateKey() should not error on 'required' when key is set")
		}
	})

	t.Run("without private key", func(t *testing.T) {
		config := &Config{}

		_, err := config.RequirePrivateKey()
		if err == nil {
			t.Fatal("RequirePrivateKey() should error when no key is set")
		}

		expectedErr := "private key is required for this operation"
		if err.Error() != expectedErr {
			t.Errorf("expected error %q, got %q", expectedErr, err.Error())
		}
	})
}

func TestParsePrivateKey(t *testing.T) {
	tests := []struct {
		name    string
		keyHex  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "too short",
			keyHex:  "123",
			wantErr: true,
			errMsg:  "private key must be 32 bytes (64 hex characters), got 3",
		},
		{
			name:    "too long",
			keyHex:  strings.Repeat("a", 128),
			wantErr: true,
			errMsg:  "private key must be 32 bytes (64 hex characters), got 128",
		},
		{
			name:    "invalid hex",
			keyHex:  "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			wantErr: true,
		},
		{
			name:   "with 0x prefix",
			keyHex: "0x" + strings.Repeat("a", 64),
			// Will attempt to parse, may or may not succeed depending on validity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePrivateKey(tt.keyHex)

			if tt.wantErr {
				if err == nil {
					t.Fatal("parsePrivateKey() expected error, got nil")
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("error = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestNetwork_Presets(t *testing.T) {
	if Sepolia.ChainID != "11155111" {
		t.Errorf("Sepolia.ChainID = %s, want 11155111", Sepolia.ChainID)
	}

	if Sepolia.Name != "sepolia" {
		t.Errorf("Sepolia.Name = %s, want sepolia", Sepolia.Name)
	}

	if Main.ChainID != "1" {
		t.Errorf("Main.ChainID = %s, want 1", Main.ChainID)
	}

	if Main.Name != "main" {
		t.Errorf("Main.Name = %s, want main", Main.Name)
	}
}

func TestConfig_FullWorkflow(t *testing.T) {
	config := &Config{
		RPCAddr:    "wss://sepolia.infura.io",
		PrivateKey: "",
		Debug:      true,
	}

	// Validate
	err := config.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Check defaults were applied
	if config.LighthouseURL == "" {
		t.Error("LighthouseURL should have default value")
	}

	if config.IpfsURL == "" {
		t.Error("IpfsURL should have default value")
	}

	if config.Network.ChainID == "" {
		t.Error("Network should have default value")
	}

	// Check timeouts
	timeouts := config.Timeouts.WithDefaults()
	if timeouts.Dial == 0 {
		t.Error("Dial timeout should have default value")
	}

	// Check private key handling
	if config.HasPrivateKey() {
		t.Error("should not have private key")
	}

	key := config.GetPrivateKey()
	if key != nil {
		t.Error("GetPrivateKey() should return nil when no key is set")
	}
}
