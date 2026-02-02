// Package config defines the runtime configuration for the SDK, including
// Ethereum network settings, RPC endpoint, storage gateways, debug mode,
// and operation timeouts. It also provides validation and defaulting helpers.
package config

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// Config holds all SDK settings required to initialize blockchain and service clients.
// Use Validate to fill implicit defaults and to check for required fields.
type Config struct {
	// Network selects the target chain (chain ID and human-readable name).
	Network Network `json:"network" yaml:"network"`
	// RPCAddr is the Ethereum WebSocket endpoint URL (required).
	// For paid and prepaid payment strategies, must use wss:// or ws:// protocol
	// because these strategies require event subscriptions from the blockchain.
	// Free call strategies may work with HTTP/HTTPS endpoints.
	RPCAddr string `json:"rpc_addr" yaml:"rpc_addr"`
	// RegistryAddr is the registry contract address (optional).
	RegistryAddr string `json:"registry_addr" yaml:"registry_addr"`
	// PrivateKey is the hex-encoded ECDSA private key used for signed operations
	// (optional if you only do free calls / read-only operations).
	PrivateKey string `json:"private_key" yaml:"private_key"`
	// LighthouseURL is the HTTP gateway used to fetch Filecoin-backed content.
	// Default: https://gateway.lighthouse.storage/ipfs/
	LighthouseURL string `json:"lighthouse_url" yaml:"lighthouse_url"`
	// IpfsURL is the HTTP API endpoint of the IPFS node used to read files.
	// Default: https://ipfs.singularitynet.io:443
	IpfsURL string `json:"ipfs_url" yaml:"ipfs_url"`
	// Debug enables verbose logging.
	Debug bool `json:"debug" yaml:"debug"`
	// Timeouts configures per-operation timeouts. See Timeouts.WithDefaults for defaults.
	Timeouts Timeouts `json:"timeouts" yaml:"timeouts"`

	// privateKeyECDSA is the parsed ECDSA private key (lazy-loaded on first access)
	privateKeyECDSA *ecdsa.PrivateKey
}

// Network describes a blockchain network (chain ID and name). ChainID is used
// for EIP-155 signing; Name is informational.
type Network struct {
	ChainID string `json:"chain_id"`
	Name    string `json:"network_name"`
}

// Sepolia is a predefined Network for Ethereum Sepolia testnet.
var Sepolia = Network{
	ChainID: "11155111",
	Name:    "sepolia",
}

// Main is a predefined Network for Ethereum mainnet.
var Main = Network{
	ChainID: "1",
	Name:    "main",
}

// Timeouts controls SDK operation deadlines.
// Zero values will be replaced by sane defaults in WithDefaults.
type Timeouts struct {
	Dial            time.Duration // gRPC/Web3 dial/connect
	GRPCUnary       time.Duration // RPC
	GRPCStream      time.Duration // RPC stream
	ChainRead       time.Duration // eth_call, balance etc
	ChainSubmit     time.Duration // send tx
	ReceiptWait     time.Duration // wait tx
	StrategyRefresh time.Duration // refresh strategy
	PaymentEnsure   time.Duration // ensure payment channel
}

// Validate normalizes the configuration by applying implicit defaults for
// LighthouseURL, IpfsURL and Network (defaults to Sepolia) and verifies that
// RPCAddr is provided.
// Returns an error when RPCAddr is empty.
func (c *Config) Validate() error {

	if c.LighthouseURL == "" {
		c.LighthouseURL = "https://gateway.lighthouse.storage/ipfs/"
	}

	if c.IpfsURL == "" {
		c.IpfsURL = "https://ipfs.singularitynet.io:443"
	}

	if c.Network.ChainID == "" {
		c.Network = Sepolia
	}

	if c.RPCAddr == "" {
		return errors.New("RPC address is required")
	}

	return nil
}

// WithDefaults returns a copy of t with zero values replaced by defaults:
//
//	Dial:            5s
//	GRPCUnary:       5s
//	ChainRead:       12s
//	ChainSubmit:     25s
//	ReceiptWait:     90s
//	StrategyRefresh: 5s
//	PaymentEnsure:   120s
func (t Timeouts) WithDefaults() Timeouts {
	tt := t
	if tt.Dial == 0 {
		tt.Dial = 15 * time.Second
	}
	if tt.GRPCUnary == 0 {
		tt.GRPCUnary = 15 * time.Second
	}
	if tt.GRPCStream == 0 {
		tt.GRPCStream = 100 * time.Second
	}
	if tt.ChainRead == 0 {
		tt.ChainRead = 13 * time.Second
	}
	if tt.ChainSubmit == 0 {
		tt.ChainSubmit = 25 * time.Second
	}
	if tt.ReceiptWait == 0 {
		tt.ReceiptWait = 90 * time.Second
	}
	if tt.StrategyRefresh == 0 {
		tt.StrategyRefresh = 15 * time.Second
	}
	if tt.PaymentEnsure == 0 {
		tt.PaymentEnsure = 120 * time.Second
	}
	return tt
}

// GetPrivateKey returns the parsed ECDSA private key.
// It parses the hex string on first call and caches the result.
// Returns nil if PrivateKey is empty (read-only mode).
func (c *Config) GetPrivateKey() *ecdsa.PrivateKey {
	// If key is not set - this is normal for read-only operations
	if c.PrivateKey == "" {
		return nil
	}

	// If already parsed - return cache
	if c.privateKeyECDSA != nil {
		return c.privateKeyECDSA
	}

	// Parse key
	key, err := parsePrivateKey(c.PrivateKey)
	if err != nil {
		return nil
	}

	c.privateKeyECDSA = key
	return c.privateKeyECDSA
}

// parsePrivateKey converts a hex-encoded private key string to *ecdsa.PrivateKey.
// It handles both formats: with and without "0x" prefix.
func parsePrivateKey(keyHex string) (*ecdsa.PrivateKey, error) {
	// Remove 0x prefix if present
	keyHex = strings.TrimPrefix(keyHex, "0x")

	// Check length (must be 64 hex characters = 32 bytes)
	if len(keyHex) != 64 {
		return nil, fmt.Errorf("private key must be 32 bytes (64 hex characters), got %d", len(keyHex))
	}

	// Parse using go-ethereum crypto
	privateKey, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hex private key: %w", err)
	}

	return privateKey, nil
}

// HasPrivateKey returns true if a private key is configured.
func (c *Config) HasPrivateKey() bool {
	return c.PrivateKey != ""
}

// RequirePrivateKey returns the private key or an error if not configured.
func (c *Config) RequirePrivateKey() (*ecdsa.PrivateKey, error) {
	if !c.HasPrivateKey() {
		return nil, fmt.Errorf("private key is required for this operation")
	}
	return c.GetPrivateKey(), nil
}
