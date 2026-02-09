// Package sdk provides the high-level entry point for interacting with SingularityNET services.
//
// The SDK simplifies AI service discovery, invocation, and payment management by abstracting
// the complexities of blockchain interactions, distributed storage, and dynamic gRPC communication.
//
// # Quick Start
//
// Create an SDK instance with configuration, then create service clients:
//
//	import (
//		"github.com/singnet/snet-sdk-go/pkg/config"
//		"github.com/singnet/snet-sdk-go/pkg/sdk"
//	)
//
//	func main() {
//		cfg := &config.Config{
//			RPCAddr:    "https://sepolia.infura.io/v3/YOUR_PROJECT_ID",
//			PrivateKey: "YOUR_PRIVATE_KEY",
//			Network:    config.Sepolia,
//			Debug:      true,
//		}
//
//		// Initialize SDK
//		snetSDK := sdk.NewSDK(cfg)
//		defer snetSDK.Close()
//
//		// Create service client
//		service, err := snetSDK.NewServiceClient("snet", "example-service", "default_group")
//		if err != nil {
//			log.Fatal(err)
//		}
//		defer service.Close()
//
//		// Call service method
//		input := []byte(`{"image":"base64..."}`)
//		response, err := service.CallWithJSON("classify", input)
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Printf("Result: %s\n", string(response))
//	}
//
// # Architecture
//
// The SDK coordinates several subsystems:
//
//   - Blockchain: Ethereum client for registry, MPE (Multi-Party Escrow), and FET token
//   - Storage: IPFS and Lighthouse for fetching service metadata and proto files
//   - gRPC: Dynamic client creation from proto descriptors for service invocation
//   - Payment: Free-call, paid-call, and prepaid strategies for authentication
//
// # Core Components
//
// SnetSDK Interface:
//   - NewServiceClient: Create a client for a specific service
//   - NewOrganizationClient: Create a client for organization management
//   - Close: Release resources
//
// Service Interface (returned by NewServiceClient):
//   - CallWithJSON: Invoke methods with JSON input/output
//   - CallWithMap: Invoke methods with Go maps
//   - CallWithProto: Invoke methods with proto messages
//   - SetPaidPaymentStrategy: Use pay-per-call payment
//   - SetPrePaidPaymentStrategy: Use prepaid payment channels
//   - SetFreePaymentStrategy: Use free-call tokens
//   - GetFreeCallsAvailable: Check remaining free calls
//   - ProtoFiles: Access service API definitions (ProtoManager)
//   - Healthcheck: Check service availability
//   - Training: Access model training API
//   - Organization: Access parent organization
//   - UpdateServiceMetadata: Update service metadata
//   - DeleteService: Remove service from registry
//   - RawGrpc: Direct access to gRPC client
//
// Organization Interface (returned by NewOrganizationClient):
//   - CreateService: Register a new service
//   - ServiceClient: Get client for an existing service
//   - ListServices: List all services in the organization
//   - AddMembers: Add members to the organization
//   - RemoveMembers: Remove members from the organization
//   - UpdateOrgMetadataFull: Update organization metadata
//   - GetOrgMetadata: Get current organization metadata
//   - GetOrgID: Get organization identifier
//   - GetGroupName: Get current group name
//   - ChangeOwner: Transfer organization ownership
//   - DeleteOrganization: Remove organization from registry
//
// # Payment Strategies
//
// The SDK supports three payment models:
//
// 1. Free Calls (default for services supporting it):
//   - No payment required
//   - Service provides free-call tokens
//   - Limited usage per user/period
//
// 2. Paid Calls (MPE escrow):
//   - Pay per service invocation
//   - Uses Multi-Party Escrow contract
//   - Automatic channel management
//
// 3. Prepaid:
//   - Pre-fund a payment channel
//   - Lower per-call overhead
//   - Suitable for high-volume usage
//
// Switch strategies at runtime:
//
//	service.SetPaidPaymentStrategy()
//	response1, _ := service.CallWithJSON("method", input)
//
//	service.SetFreePaymentStrategy()
//	response2, _ := service.CallWithJSON("method", input)
//
// # Configuration
//
// Required configuration fields:
//   - RPCAddr: Ethereum RPC endpoint (wss:// for paid/prepaid, https:// for free)
//   - Network: Target network (config.Sepolia or config.Main)
//
// Optional fields:
//   - PrivateKey: Required for paid operations and write operations
//   - RegistryAddr: Custom registry contract address
//   - IpfsURL: Custom IPFS gateway
//   - LighthouseURL: Custom Lighthouse gateway
//   - Debug: Enable verbose logging
//   - Timeouts: Custom timeout configuration
//
// # Error Handling
//
// All methods return errors that should be checked. Common error scenarios:
//   - Configuration validation failures
//   - Network connectivity issues
//   - Service not found in registry
//   - Payment channel issues
//   - Service invocation failures
//
// Example with proper error handling:
//
//	service, err := snetSDK.NewServiceClient("org", "service", "group")
//	if err != nil {
//		return fmt.Errorf("failed to create service client: %w", err)
//	}
//	defer service.Close()
//
//	if err := service.SetPaidPaymentStrategy(); err != nil {
//		return fmt.Errorf("failed to set payment strategy: %w", err)
//	}
//
//	response, err := service.CallWithJSON("method", input)
//	if err != nil {
//		return fmt.Errorf("service call failed: %w", err)
//	}
//
// # Thread Safety
//
// The SDK Core and service clients are safe for concurrent use. You can share
// a single SDK instance across goroutines and make parallel service calls.
//
// # Resource Management
//
// Always call Close() on SDK and service instances to release network connections
// and other resources:
//
//	sdk := sdk.NewSDK(cfg)
//	defer sdk.Close()
//
//	service, _ := sdk.NewServiceClient("org", "service", "group")
//	defer service.Close()
//
// # Advanced Usage
//
// Access low-level blockchain client for custom operations:
//
//	evm := snetSDK.GetEvm()
//	// Use evm for direct contract calls, transactions, etc.
//
// # See Also
//
// For detailed examples, see the examples/ directory in the repository:
//   - examples/quick-start: Basic service call
//   - examples/paid-call: Using paid payment strategy
//   - examples/orgs-and-services: Organization and service management
//   - examples/training: Model training integration
package sdk
