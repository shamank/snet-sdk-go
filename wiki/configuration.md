## SDK Configuration Guide

The SingularityNET SDK requires proper configuration to connect to the Ethereum network, interact with services, and manage payments. This guide covers all configuration parameters.

## Table of Contents

1. [Configuration Parameters](#configuration-parameters)
2. [Configuration Examples](#configuration-examples)

---

## Configuration Parameters

### Complete Config Structure

```go
type Config struct {
    Network       Network   // Target blockchain network
    RPCAddr       string    // Ethereum RPC endpoint URL
    RegistryAddr  string    // Registry contract address (optional)
    PrivateKey    string    // Hex-encoded ECDSA private key
    LighthouseURL string    // Filecoin gateway URL
    IpfsURL       string    // IPFS HTTP API endpoint
    Debug         bool      // Enable verbose logging
    Timeouts      Timeouts  // Operation timeout settings
}
```

### Parameter Descriptions

#### Network
- **Type**: `Network` struct
- **Required**: No (defaults to Sepolia)
- **Description**: Specifies the target blockchain network (chain ID and name)
- **Predefined Networks**:
  - `config.Sepolia` - Ethereum Sepolia testnet (ChainID: 11155111)
  - `config.Main` - Ethereum mainnet (ChainID: 1)

```go
Network: config.Sepolia  // For testing
Network: config.Main     // For production
```

#### RPCAddr
- **Type**: `string`
- **Required**: Yes
- **Description**: Ethereum RPC endpoint URL for blockchain communication
- **Protocol Requirements**:
  - **Paid/Pre-paid strategies**: Must use WebSocket (`wss://` or `ws://`) for event subscriptions
  - **Free call strategy**: Can use HTTP/HTTPS

#### RegistryAddr
- **Type**: `string`
- **Required**: No
- **Description**: Custom registry contract address (uses network default if empty)
- **Use Case**: Override default registry for testing or custom deployments
- **Example**:
  ```go
  RegistryAddr: "0x1234567890123456789012345678901234567890"
  ```

#### PrivateKey
- **Type**: `string`
- **Required**: No (required for paid operations)
- **Description**: Hex-encoded ECDSA private key for transaction signing
- **Format**: 64 hex characters (with or without "0x" prefix)
- **Security**: Never hardcode in production - use environment variables
- **Examples**:
  ```go
  PrivateKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
  PrivateKey: "0x0123456789abcdef..."  // With 0x prefix also accepted
  PrivateKey: ""                        // Read-only mode (free calls only)
  ```

#### LighthouseURL
- **Type**: `string`
- **Required**: No
- **Default**: `https://gateway.lighthouse.storage/ipfs/`
- **Description**: HTTP gateway for accessing Filecoin-backed content
- **Example**:
  ```go
  LighthouseURL: "https://gateway.lighthouse.storage/ipfs/"
  ```

#### IpfsURL
- **Type**: `string`
- **Required**: No
- **Default**: `https://ipfs.singularitynet.io:443`
- **Description**: IPFS HTTP API endpoint for reading metadata files
- **Example**:
  ```go
  IpfsURL: "https://ipfs.singularitynet.io:443"
  IpfsURL: "http://localhost:5001"  // Local IPFS node
  ```

#### Debug
- **Type**: `bool`
- **Required**: No
- **Default**: `false`
- **Description**: Enables verbose logging for debugging
- **Example**:
  ```go
  Debug: true   // Development
  Debug: false  // Production
  ```

#### Timeouts
- **Type**: `Timeouts` struct
- **Required**: No (uses defaults)
- **Description**: Configures operation timeouts. See [Timeout Configuration](#timeouts)

---

## Configuration Examples

### Minimal Configuration (Free Calls)

```go
package main

import (
	"github.com/shamank/snet-sdk-go/pkg/config"
	"github.com/shamank/snet-sdk-go/pkg/sdk"
)

func main() {
	cfg := config.Config{
		RPCAddr: "https://sepolia.infura.io/v3/YOUR_PROJECT_ID",
		Debug:   true,
	}

	snetSDK := sdk.NewSDK(&cfg)
	defer snetSDK.Close()

	// Use SDK for free calls only
}
```

### Development Configuration (Sepolia Testnet)

```go
import (
	"github.com/shamank/snet-sdk-go/pkg/config"
	"github.com/shamank/snet-sdk-go/pkg/sdk"
)

func main() {
	cfg := config.Config{
		Network:    config.Sepolia,
		RPCAddr:    "wss://sepolia.infura.io/ws/v3/YOUR_PROJECT_ID",
		PrivateKey: "YOUR_PRIVATE_KEY",
		Debug:      true,
		Timeouts: config.Timeouts{
			GRPCUnary:   20 * time.Second,
			ChainSubmit: 30 * time.Second,
		},
	}

	snetSDK := sdk.NewSDK(&cfg)
	defer snetSDK.Close()

	// Development and testing
}
```

### Production Configuration (Mainnet)

```go
import (
	"time"
	"github.com/shamank/snet-sdk-go/pkg/config"
	"github.com/shamank/snet-sdk-go/pkg/sdk"
)

func main() {
	cfg := config.Config{
		Network:    config.Main,
		RPCAddr:    "wss://sepolia.infura.io/ws/v3/YOUR_PROJECT_ID",
		PrivateKey: "YOUR_PRIVATE_KEY",
		Debug:      false,
		Timeouts: config.Timeouts{
			Dial:            10 * time.Second,
			GRPCUnary:       30 * time.Second,
			ChainRead:       15 * time.Second,
			ChainSubmit:     60 * time.Second,
			ReceiptWait:     120 * time.Second,
			StrategyRefresh: 10 * time.Second,
			PaymentEnsure:   180 * time.Second,
		},
	}

	snetSDK := sdk.NewSDK(&cfg)
	defer snetSDK.Close()

	// Production service calls
}
```