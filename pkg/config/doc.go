// Package config provides configuration management for the SingularityNET SDK.
//
// This package defines the Config structure that controls all SDK behavior including
// network settings, RPC endpoints, storage gateways, authentication, and timeouts.
//
// # Basic Configuration
//
// The minimum required configuration needs an RPC endpoint and network:
//
//	cfg := &config.Config{
//		RPCAddr: "https://sepolia.infura.io/v3/YOUR_PROJECT_ID",
//		Network: config.Sepolia,
//	}
//
// # Network Selection
//
// Two predefined networks are available:
//
//	config.Sepolia - Ethereum Sepolia testnet (ChainID: 11155111)
//	config.Main    - Ethereum mainnet (ChainID: 1)
//
// Custom networks can be defined:
//
//	customNet := config.Network{
//		ChainID: "12345",
//		Name:    "custom-network",
//	}
//
// # RPC Endpoints
//
// The RPC endpoint protocol depends on your payment strategy:
//
//   - Free calls: HTTP/HTTPS endpoints work fine
//     Example: "https://sepolia.infura.io/v3/PROJECT_ID"
//
//   - Paid/Prepaid calls: WebSocket (WSS/WS) required for event subscriptions
//     Example: "wss://sepolia.infura.io/ws/v3/PROJECT_ID"
//
// # Private Key
//
// Private key is required for:
//   - Paid and prepaid payment operations
//   - Creating/managing organizations
//   - Creating/managing services
//   - Any blockchain write operations
//
// The key should be hex-encoded without the "0x" prefix:
//
//	cfg.PrivateKey = "abcdef1234567890..." // 64 hex characters
//
// Replace with your actual private key:
//
//	cfg.PrivateKey = "YOUR_PRIVATE_KEY"
//
// # Storage Gateways
//
// Service metadata is stored on IPFS/Lighthouse. Default gateways are provided:
//
//	IpfsURL:       "https://ipfs.singularitynet.io:443"
//	LighthouseURL: "https://gateway.lighthouse.storage/ipfs/"
//
// Custom gateways can be configured:
//
//	cfg.IpfsURL = "http://localhost:5001"
//	cfg.LighthouseURL = "https://custom-gateway.example.com/ipfs/"
//
// # Timeouts
//
// All operations have configurable timeouts. The Timeouts struct provides granular control:
//
//	cfg.Timeouts = config.Timeouts{
//		Dial:            10 * time.Second,  // Connection timeout
//		GRPCUnary:       30 * time.Second,  // RPC call timeout
//		GRPCStream:      120 * time.Second, // Streaming RPC timeout
//		ChainRead:       15 * time.Second,  // Blockchain read timeout
//		ChainSubmit:     60 * time.Second,  // Transaction submission timeout
//		ReceiptWait:     180 * time.Second, // Transaction confirmation timeout
//		StrategyRefresh: 30 * time.Second,  // Payment strategy refresh timeout
//		PaymentEnsure:   120 * time.Second, // Payment channel setup timeout
//	}
//
// Zero values are replaced with sensible defaults via WithDefaults().
//
// # Debug Mode
//
// Enable debug logging for troubleshooting:
//
//	cfg.Debug = true
//
// This enables verbose output about:
//   - Blockchain transactions
//   - Service metadata fetching
//   - gRPC invocations
//   - Payment channel operations
//
// # Registry Address
//
// By default, the SDK uses the standard SingularityNET registry contract.
// For testing or custom deployments, specify a custom registry:
//
//	cfg.RegistryAddr = "0x1234567890abcdef1234567890abcdef12345678"
//
// # Configuration Validation
//
// Always call Validate() to apply defaults and check required fields:
//
//	cfg := &config.Config{...}
//	if err := cfg.Validate(); err != nil {
//		log.Fatalf("Invalid config: %v", err)
//	}
//
// Validate() will:
//   - Set default storage URLs if not provided
//   - Set default network to Sepolia if not provided
//   - Return error if RPCAddr is empty
//
// # Complete Example
//
//	import (
//		"time"
//		"github.com/shamank/snet-sdk-go/pkg/config"
//	)
//
//	func loadConfig() (*config.Config, error) {
//		cfg := &config.Config{
//			Network:    config.Sepolia,
//			RPCAddr:    "https://sepolia.infura.io/v3/YOUR_PROJECT_ID",
//			PrivateKey: "YOUR_PRIVATE_KEY",
//			Debug:      true,
//			Timeouts: config.Timeouts{
//				Dial:        10 * time.Second,
//				GRPCUnary:   30 * time.Second,
//				ChainRead:   15 * time.Second,
//			},
//		}
//
//		return cfg, cfg.Validate()
//	}
//
// # Configuration Pattern
//
// A common pattern is to replace placeholders with actual values:
//
//	cfg := &config.Config{
//		RPCAddr:    "wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID",
//		PrivateKey: "YOUR_PRIVATE_KEY",
//		Network:    config.Main,
//		Debug:      false,
//	}
//
//	return cfg, cfg.Validate()
//
// # Thread Safety
//
// Config instances should be created once and not modified after passing to SDK.NewSDK().
// The Config is read-only during SDK operations.
//
// # See Also
//
//   - sdk.NewSDK() for SDK initialization
//   - examples/quick-start for basic configuration
//   - wiki/configuration.md for detailed configuration guide
package config
