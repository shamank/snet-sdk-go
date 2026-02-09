# Quick Start Guide - Your First SingularityNET Service Call

Get started with the SingularityNET Go SDK in just 5 minutes! This guide walks you through making your first AI service call on the decentralized marketplace.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Step-by-Step Setup](#step-by-step-setup)
4. [Your First Service Call](#your-first-service-call)
5. [Understanding the Configuration](#understanding-the-configuration)
6. [Under the Hood](#under-the-hood)
7[Next Steps](#next-steps)

---

## Prerequisites

Before you begin, make sure you have:

### 1. Go Environment
- **Go 1.24+** installed
- Verify: `go version`

### 2. Ethereum RPC Endpoint
Get a free endpoint from any of these providers:
- **[Infura](https://infura.io/)** - Create account, get project ID
- **[Alchemy](https://www.alchemyapi.io/)** - Sign up for free tier
- **[QuickNode](https://www.quicknode.com/)** - Free trial available

**Example endpoint**: `https://sepolia.infura.io/v3/YOUR_PROJECT_ID`

### 3. Service Selection (Optional for Testing)
Browse available AI services:
- **Testnet Marketplace**: https://testnet.marketplace.singularitynet.io/
- Find `Organization ID` and `Service ID` for services you want to call

**Note**: Many services offer **free calls** for testing - no wallet or FET tokens needed!

### 4. Wallet & Funds (Only for Paid Services)
If you plan to use paid services:
- **ERC-20 compatible wallet** (private key)
- **Sepolia ETH** for gas (get from [faucet](https://sepoliafaucet.com/))
- **FET tokens** on Sepolia testnet

**⚠️ Security**: Never commit private keys to version control!

---

## Installation

### Install the SDK

```bash
go get -u github.com/singnet/snet-sdk-go
```

### Create a New Project

```bash
mkdir my-snet-project
cd my-snet-project
go mod init my-snet-project
go get -u github.com/singnet/snet-sdk-go
```

---

## Step-by-Step Setup

### Step 1: Import Required Packages

Create `main.go`:

```go
package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)
```

### Step 2: Configure the SDK

Add the configuration inside your `main()` function:

```go
func main() {
	// Configure SDK
	cfg := config.Config{
		RPCAddr:    "https://sepolia.infura.io/v3/YOUR_PROJECT_ID",
		PrivateKey: "",  // Leave empty for free calls
		Debug:      true,
		Network:    config.Sepolia,
	}
```

**Replace `YOUR_PROJECT_ID`** with your actual Infura project ID.

### Step 3: Initialize SDK

```go
	// Initialize SDK
	snetSDK := sdk.NewSDK(&cfg)
	defer snetSDK.Close()
```

The `defer` statement ensures proper cleanup when your program exits.

### Step 4: Create Service Client

```go
	// Create service client
	service, err := snetSDK.NewServiceClient("snet", "example-service", "default_group")
	if err != nil {
		log.Fatalf("Failed to create service client: %v", err)
	}
	defer service.Close()
```

**Parameters explained**:
- `"snet"` - Organization ID
- `"example-service"` - Service ID
- `"default_group"` - Payment group (usually `"default_group"`)

### Step 5: Prepare Input Data

```go
	// Prepare input as JSON
	inputJSON := []byte(`{"a": 7, "b": 2}`)
```

The input format depends on the service's API definition (proto files).

### Step 6: Call the Service

```go
	// Make the service call
	response, err := service.CallWithJSON("add", inputJSON)
	if err != nil {
		log.Fatalf("Service call failed: %v", err)
	}

	fmt.Printf("Response: %s\n", string(response))
}
```

---

## Your First Service Call

### Complete Example

Here's the full working code:

```go
package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	// Step 1: Configure SDK
	cfg := config.Config{
		RPCAddr:    "https://sepolia.infura.io/v3/YOUR_PROJECT_ID",
		PrivateKey: "YOUR_PRIVATE_KEY",
		Debug:      true,
		Network:    config.Sepolia,
	}

	// Step 2: Initialize SDK
	snetSDK := sdk.NewSDK(&cfg)
	defer snetSDK.Close()

	// Step 3: Create service client
	service, err := snetSDK.NewServiceClient("snet", "example-service", "default_group")
	if err != nil {
		log.Fatalf("Failed to create service client: %v", err)
	}
	defer service.Close()

	// Step 4: Prepare input
	inputJSON := []byte(`{"a": 7, "b": 2}`)

	// Step 5: Call the service
	response, err := service.CallWithJSON("add", inputJSON)
	if err != nil {
		log.Fatalf("Service call failed: %v", err)
	}

	// Step 6: Display result
	fmt.Printf("Response from service: %s\n", string(response))
	fmt.Printf("Result: 7 + 2 = %s\n", string(response))
}
```

### Run the Example

```bash
go run main.go
```

### Expected Output

```
Response from service: {"result": 9}
Result: 7 + 2 = 9
```

---

## Understanding the Configuration

### Configuration Parameters Explained

#### RPCAddr (Required)
```go
RPCAddr: "https://sepolia.infura.io/v3/YOUR_PROJECT_ID"
```
- **Purpose**: Ethereum RPC endpoint for blockchain communication
- **Format**: HTTP/HTTPS for free calls, WebSocket (wss://) for paid services
- **Examples**:
  - Infura: `https://sepolia.infura.io/v3/YOUR_PROJECT_ID`
  - Alchemy: `https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY`
  - Local: `http://localhost:8545`

#### PrivateKey (Optional)
```go
PrivateKey: "YOUR_PRIVATE_KEY"
// OR
PrivateKey: ""  // For free calls only
```
- **Purpose**: Sign transactions for paid services
- **Format**: 64 hex characters (with or without "0x" prefix)
- **Security**: 
  - ✅ Replace with your actual private key
  - ✅ Keep secure and never share
  - ❌ Never hardcode in source code
  - ❌ Never commit to version control

#### Network
```go
Network: config.Sepolia  // Testnet
// OR
Network: config.Main     // Mainnet
```
- **Purpose**: Specify blockchain network
- **Options**:
  - `config.Sepolia` - Sepolia testnet (ChainID: 11155111)
  - `config.Main` - Ethereum mainnet (ChainID: 1)

#### Debug
```go
Debug: true   // Development/testing
Debug: false  // Production
```
- **Purpose**: Enable verbose logging
- **Recommended**: `true` for development, `false` for production

### Service Client Parameters

```go
snetSDK.NewServiceClient("ORGANIZATION_ID", "SERVICE_ID", "GROUP_ID")
```

#### Organization ID
- Unique identifier for the service provider
- Example: `"snet"`, `"my-org"`
- Find on marketplace

#### Service ID
- Unique identifier for the specific service
- Example: `"example-service"`, `"image-classifier"`
- Find on marketplace

#### Group ID
- Payment group for the service
- Usually: `"default_group"`
- Some services may have multiple pricing groups

**Where to find these IDs**:
1. Browse [Testnet Marketplace](https://testnet.marketplace.singularitynet.io/)
2. Select a service
3. Find IDs in service details

---

## Under the Hood

### What Happens During a Service Call?

```
1. SDK Initialization
   ├── Connect to Ethereum RPC
   ├── Initialize blockchain contracts
   └── Set up IPFS/Lighthouse clients

2. Service Client Creation
   ├── Query Registry contract for service metadata
   ├── Fetch metadata from IPFS
   ├── Download proto files
   ├── Generate dynamic gRPC client
   └── Initialize payment strategy

3. Service Call
   ├── Serialize input to Protobuf
   ├── Handle payment (if required)
   │   ├── Free call: Use free call signer
   │   ├── Paid call: Create payment channel
   │   └── Pre-paid: Reuse existing channel
   ├── Make gRPC request to service endpoint
   ├── Receive and deserialize response
   └── Return result to caller

4. Cleanup
   ├── Close gRPC connections
   ├── Release resources
   └── Close blockchain connections
```

### Key Components

1. **Registry Contract**: Smart contract storing service metadata locations
2. **IPFS**: Decentralized storage for service definitions and proto files
3. **Proto Files**: Service API definitions (methods, inputs, outputs)
4. **Payment Channels**: Payment infrastructure for efficient micro-transactions
5. **gRPC**: Communication protocol between client and service

---

## Alternative Call Methods

### Using Map for Input/Output

Instead of JSON strings, you can use Go maps:

```go
// Input as map
inputMap := map[string]any{
	"a": 7,
	"b": 2,
}

// Call with map
responseMap, err := service.CallWithMap("add", inputMap)
if err != nil {
	log.Fatalf("Call failed: %v", err)
}

// Access result
result := responseMap["result"]
fmt.Printf("Result: %v\n", result)
```

**When to use maps**:
- ✅ Complex nested structures
- ✅ Type safety in Go code
- ✅ Easier manipulation of data

**When to use JSON**:
- ✅ Simple input/output
- ✅ Input from external sources
- ✅ Direct string manipulation

---

## Next Steps

### 🎯 Essential Reading

1. **[Configuration Guide](configuration.md)**
   - Complete configuration reference
   - Environment variables
   - Security best practices
   - Network-specific settings

2. **[Choose Payment Strategy](choose_strategy.md)**
   - Free calls vs Paid vs Pre-paid
   - When to use each strategy
   - Cost optimization

3. **[Organizations & Services](orgs_services.md)**
   - Discover available services
   - Query service metadata
   - List organizations

### 🚀 Practical Guides

4. **[Proto Files Guide](proto_files.md)**
   - Understanding service definitions
   - Available methods and parameters
   - Custom proto handling

5. **[Health Checks](healthcheck.md)**
   - Monitor service availability
   - Test connectivity
   - Service status checking

### 🔬 Advanced Topics

6. **[Training Support](training.md)**
   - Submit training jobs
   - Monitor training progress
   - Retrieve trained models

### 💡 Examples

Explore working examples in the [`/examples`](../examples) directory:
- **[free-calls](../examples/free-calls)** - Free call strategy
- **[paid-call](../examples/paid-call)** - Pay-per-call pattern
- **[pre-paid](../examples/pre-paid)** - Payment channels
- **[orgs-and-services](../examples/orgs-and-services)** - Service discovery

---

**Congratulations!** 🎉 You've completed the Quick Start guide.

**Ready for more?** Continue to [Configuration Guide](configuration.md) or explore the [Examples](../examples)!