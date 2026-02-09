# SingularityNET Go SDK - Documentation Hub

Welcome to the comprehensive documentation for the **SingularityNET Go SDK**! This SDK enables seamless integration with the SingularityNET AI marketplace, allowing you to discover, interact with, and pay for decentralized AI services on the blockchain.

---

## 🚀 Overview

The SingularityNET Go SDK is a powerful toolkit for building applications that leverage decentralized AI services. It provides:

- **Smart Contract Integration**: Native bindings for SingularityNET contracts on Ethereum
- **Dynamic Service Discovery**: Automatic fetching and parsing of service definitions
- **Flexible Payment Options**: Support for free calls, pay-per-call, and pre-paid channels
- **IPFS & Lighthouse Support**: Decentralized metadata storage and retrieval
- **gRPC Communication**: High-performance service invocation with dynamic proto handling
- **Training Support**: Enable model training workflows with AI services

**Supported Networks**: Ethereum Mainnet, Sepolia Testnet  
**Minimum Go Version**: 1.24+

---

## ✨ Key Features

| Feature | Description | Status |
|---------|-------------|--------|
| **Smart Contract Bindings** | Interact with Registry, MPE, and Token contracts | ✅ Done |
| **IPFS Integration** | Fetch service metadata from IPFS | ✅ Done |
| **Lighthouse Support** | Alternative decentralized storage via Filecoin | ✅ Done |
| **Dynamic gRPC** | Automatic proto file fetching and client generation | ✅ Done |
| **Payment Strategies** | Free-call, paid-call, and pre-paid channel support | ✅ Done |
| **Service & Org Management** | List organizations, services, and groups | ✅ Done |
| **Health Checks** | Monitor service availability (gRPC, HTTP, JSONRPC) | ✅ Done |
| **Training API** | Submit training jobs to AI services | ✅ Done |
| **Comprehensive Examples** | Real-world usage patterns and tutorials | ✅ Done |

---

## 📚 Learning Path

### 🟢 Getting Started (Beginners)

Start here if you're new to SingularityNET or the SDK:

1. **[Quick Start Guide](quick_start.md)** - Get up and running in 5 minutes
2. **[Configuration Guide](configuration.md)** - Set up your environment properly
3. **[Organizations & Services](orgs_services.md)** - Discover available AI services

### 🟡 Intermediate Topics

Once you're comfortable with basics:

4. **[Payment Strategies](choose_strategy.md)** - Choose the right payment method
5. **[Proto Files](proto_files.md)** - Work with service definitions

### 🔴 Advanced Usage

For production deployments and complex scenarios:

7. **[Health Checks](healthcheck.md)** - Monitor service availability
8. **[Training Support](training.md)** - Submit model training jobs

---

## 🔗 Quick Links

### Documentation
- [Quick Start Guide](quick_start.md) - Your first service call
- [Configuration Reference](configuration.md) - All config parameters
- [Choose Payment Strategy](choose_strategy.md) - Free vs Paid vs Pre-paid
- [Organizations & Services](orgs_services.md) - Service discovery
- [Proto Files](proto_files.md) - Working with service definitions
- [Health Checks](healthcheck.md) - Service monitoring
- [Training Guide](training.md) - Model training workflows

### Code Examples

Explore practical examples in the [`/examples`](../examples) directory:

- **[quick-start](../examples/quick-start)** - Basic service call
- **[free-calls](../examples/free-calls)** - Using free call strategy
- **[paid-call](../examples/paid-call)** - Pay-per-call pattern
- **[pre-paid](../examples/pre-paid)** - Payment channel management
- **[orgs-and-services](../examples/orgs-and-services)** - Service discovery
- **[proto-files](../examples/proto-files)** - Proto file handling
- **[healthcheck](../examples/healthcheck)** - Service health monitoring
- **[training](../examples/training)** - Model training submission

### API Documentation

- **[GoDoc](https://pkg.go.dev/github.com/shamank/snet-sdk-go)** - Complete API reference
- **[GitHub Repository](https://github.com/shamank/snet-sdk-go)** - Source code and issues

### External Resources

- **[SingularityNET Website](https://singularitynet.io/)** - Official platform
- **[Marketplace](https://marketplace.singularitynet.io/)** - Browse AI services (Mainnet)
- **[Testnet Marketplace](https://testnet.marketplace.singularitynet.io/)** - Test services (Sepolia)
- **[Developer Portal](https://dev.singularitynet.io/)** - Platform documentation

---

## 🎯 Common Use Cases

### Quick Service Call
```go
import (
    "github.com/shamank/snet-sdk-go/pkg/config"
    "github.com/shamank/snet-sdk-go/pkg/sdk"
)

cfg := config.Config{
    RPCAddr: "wss://sepolia.infura.io/ws/v3/YOUR_PROJECT_ID",
    Network: config.Sepolia,
    Debug:   true,
}

snetSDK := sdk.NewSDK(&cfg)
defer snetSDK.Close()

service, _ := snetSDK.NewServiceClient("snet", "example-service", "default_group")
defer service.Close()

response, _ := service.CallWithJSON("predict", []byte(`{"input": "data"}`))
```

**Learn more**: [Quick Start Guide](quick_start.md)

### Payment Channel Management
```go
// Use pre-paid strategy for multiple calls
service.SetPrePaidStrategy()

// Make multiple calls efficiently
for i := 0; i < 10; i++ {
    response, err := service.CallWithJSON("process", input)
    // Handle response
}
```

**Learn more**: [Payment Strategies](choose_strategy.md)

### Service Discovery
```go
// List all organizations
orgs, _ := snetSDK.ListOrganizations()

// Get services in an organization
services, _ := snetSDK.ListServices("snet")

// Get service details
metadata, _ := snetSDK.GetServiceMetadata("snet", "example-service")
```

**Learn more**: [Organizations & Services](orgs_services.md)

---

## 🛠️ Installation

### Prerequisites
- Go 1.24 or higher
- Ethereum wallet (for paid services)
- RPC endpoint (Infura, Alchemy, etc.)

### Install SDK
```bash
go get -u github.com/shamank/snet-sdk-go
```

### Verify Installation
```bash
cd examples/quick-start
go run main.go
```

**Detailed guide**: [Quick Start](quick_start.md)

---

## 🔑 Configuration Essentials

### Minimal Configuration (Free Calls)
```go
cfg := config.Config{
    RPCAddr: "https://sepolia.infura.io/v3/YOUR_PROJECT_ID",
}
```

### Full Configuration (Production)
```go
cfg := config.Config{
    Network:       config.Main,
    RPCAddr:       "wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID",
    PrivateKey:    "YOUR_PRIVATE_KEY",
    Debug:         false,
    IpfsURL:       "https://ipfs.singularitynet.io:443",
    LighthouseURL: "https://gateway.lighthouse.storage/ipfs/",
}
```

**Complete reference**: [Configuration Guide](configuration.md)

---

## 💡 Payment Strategies Explained

| Strategy | Use Case | Requires Key | Gas Costs |
|----------|----------|--------------|-----------|
| **Free Call** | Testing, free services | No | None |
| **Paid Call** | One-off calls | Yes | Per call |
| **Pre-Paid** | Multiple calls, production | Yes | Once (channel) |

```go
// Free calls (default)
service, _ := snetSDK.NewServiceClient("org", "service", "group")

// Switch to paid strategy
service.SetPaidPaymentStrategy()

// Switch to pre-paid (most efficient)
service.SetPrePaidStrategy()
```

**Detailed comparison**: [Choose Strategy](choose_strategy.md)

---

## 🆘 Getting Help

### Documentation
- Browse [Examples](../examples) for code patterns

### Community Support
- **GitHub Issues**: [Report bugs or request features](https://github.com/shamank/snet-sdk-go/issues)
- **Forum**: [Developer discussions](https://community.singularitynet.io/)

---

## 📖 Documentation Map

```
wiki/
├── README.md              ← You are here (Documentation Hub)
├── quick_start.md         ← Start here for first service call
├── configuration.md       ← All configuration parameters
├── choose_strategy.md     ← Payment strategy selection
├── orgs_services.md       ← Service discovery and metadata
├── proto_files.md         ← Proto file handling
├── healthcheck.md         ← Service health monitoring
└── training.md            ← Model training workflows
```

---

## 🎓 Next Steps

### New Users
👉 Start with the **[Quick Start Guide](quick_start.md)** to make your first service call

### Exploring Services
👉 Learn about **[Organizations & Services](orgs_services.md)** to discover available AI services

### Production Deployment
👉 Review **[Configuration Guide](configuration.md)**

### Advanced Features
👉 Explore **[Training Support](training.md)**

---

## 📄 License

This SDK is released under the MIT License.

---

**Happy coding with SingularityNET! 🚀**

For questions or feedback, please open an issue on [GitHub](https://github.com/shamank/snet-sdk-go/issues).
