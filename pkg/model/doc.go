// Package model defines data structures representing SingularityNET organizations,
// services, and their metadata.
//
// This package contains the core data models that represent:
//   - Organization metadata (groups, payment details)
//   - Service metadata (endpoints, pricing, API definitions)
//   - Payment configurations
//   - Licensing information
//   - Protocol buffer descriptors
//
// These structures are populated from on-chain registry data and IPFS/Lighthouse
// storage, providing a Go-native representation of the SingularityNET ecosystem.
//
// # Organization Metadata
//
// OrganizationMetaData represents an organization in the SingularityNET network:
//
//	type OrganizationMetaData struct {
//		OrgName string                  // Human-readable name
//		OrgID   string                  // Unique identifier
//		Groups  []*OrganizationGroup    // Payment groups
//	}
//
// Organizations contain one or more groups, each with its own payment configuration.
// This allows different pricing and payment setups within a single organization.
//
// # Organization Groups
//
// OrganizationGroup defines a payment group within an organization:
//
//	type OrganizationGroup struct {
//		ID             string      // Group identifier
//		GroupName      string      // Human-readable name
//		PaymentDetails Payment     // Payment configuration
//		Licenses       Licenses    // Optional licensing info
//	}
//
// Groups enable organizations to:
//   - Offer different pricing tiers
//   - Accept payments to different addresses
//   - Apply different licensing terms
//   - Segment customers or usage patterns
//
// # Service Metadata
//
// ServiceMetadata describes a deployed AI service:
//
//	type ServiceMetadata struct {
//		Version          int              // Metadata version
//		DisplayName      string           // Human-readable name
//		Encoding         string           // Data encoding (usually "proto")
//		ServiceType      string           // Service classification
//		Groups           []*ServiceGroup  // Deployment groups
//		ModelIpfsHash    string           // Model storage reference
//		ServiceApiSource string           // Proto file location
//		MPEAddress       string           // Payment escrow contract
//		ProtoDescriptors []FileDescriptor // Compiled proto definitions
//		ProtoFiles       map[string]string // Proto source files
//	}
//
// The SDK fetches this metadata from IPFS/Lighthouse using the hash stored
// in the on-chain registry.
//
// # Service Groups
//
// ServiceGroup represents a deployment instance of a service:
//
//	type ServiceGroup struct {
//		GroupName      string      // Group identifier
//		Endpoints      []string    // gRPC service endpoints
//		Pricing        []Pricing   // Price models
//		FreeCalls      int         // Free calls available
//		FreeCallSigner string      // Free call token signer
//	}
//
// A service may have multiple groups for:
//   - Geographic distribution (different endpoints)
//   - Load balancing
//   - Different pricing models
//   - Testing vs production environments
//
// # Pricing Models
//
// Pricing defines how a service charges for calls:
//
//	type Pricing struct {
//		PriceModel     string            // "fixed_price", "subscription", etc.
//		PriceInCogs    *big.Int          // Price in FET cogs (1 FET = 10^8 cogs)
//		PackageName    string            // Optional package identifier
//		Default        bool              // Is this the default price
//		PricingDetails []PricingDetails  // Extended pricing information
//	}
//
// Common price models:
//   - fixed_price: Pay per call with PriceInCogs
//   - subscription: Recurring payment for unlimited calls
//   - dynamic: Price varies based on input/output
//
// # Payment Configuration
//
// Payment defines how payments are collected and managed:
//
//	type Payment struct {
//		PaymentAddress              string  // Recipient Ethereum address
//		PaymentExpirationThreshold  *big.Int // Min blocks before channel expiry
//		PaymentChannelStorageType   string   // "etcd", "redis", etc.
//		PaymentChannelStorageClient PaymentChannelStorageClient
//	}
//
// This configuration is used by both:
//   - SDK: To open payment channels to the correct address
//   - Daemon: To validate payments and track channel state
//
// # Proto Descriptors
//
// ServiceMetadata includes proto file information:
//
//	ProtoDescriptors []protoreflect.FileDescriptor  // Compiled descriptors
//	ProtoFiles       map[string]string              // filename -> source
//
// The SDK:
//  1. Fetches proto files from IPFS using ServiceApiSource
//  2. Compiles them to FileDescriptors
//  3. Uses descriptors for dynamic gRPC method invocation
//  4. Exposes ProtoFiles for inspection or code generation
//
// # Usage Examples
//
// Access organization metadata:
//
//	org, _ := sdk.NewOrganizationClient("snet", "default_group")
//	metadata := org.GetOrgMetadata()
//	fmt.Printf("Organization: %s\n", metadata.OrgName)
//	for _, group := range metadata.Groups {
//		fmt.Printf("  Group: %s, Payment: %s\n",
//			group.GroupName, group.PaymentDetails.PaymentAddress)
//	}
//
// Access service metadata:
//
//	service, _ := sdk.NewServiceClient("snet", "example-service", "default_group")
//	metadata := service.GetServiceMetadata()
//	fmt.Printf("Service: %s (v%d)\n", metadata.DisplayName, metadata.Version)
//	fmt.Printf("Endpoints: %v\n", metadata.Groups[0].Endpoints)
//
// Inspect pricing:
//
//	for _, group := range metadata.Groups {
//		for _, pricing := range group.Pricing {
//			if pricing.Default {
//				cogs := pricing.PriceInCogs
//				fet := new(big.Float).Quo(
//					new(big.Float).SetInt(cogs),
//					big.NewFloat(1e8))
//				fmt.Printf("Price: %v FET per call\n", fet)
//			}
//		}
//	}
//
// # Metadata Updates
//
// Metadata is typically cached by the SDK. To force refresh:
//   - Re-create the service/organization client
//   - Restart your application
//
// Organizations and service owners can update metadata by:
//  1. Creating new metadata JSON
//  2. Uploading to IPFS/Lighthouse
//  3. Calling updateOrgMetadata or updateServiceMetadata on registry contract
//
// # JSON Serialization
//
// All types have JSON tags and can be marshaled/unmarshaled:
//
//	metadata, _ := json.Marshal(serviceMetadata)
//	ioutil.WriteFile("service-metadata.json", metadata, 0644)
//
// This is useful for:
//   - Debugging and inspection
//   - Caching metadata locally
//   - Building service discovery UIs
//
// # Thread Safety
//
// Model instances are typically created once during service client initialization
// and then used read-only. If you modify metadata, ensure proper synchronization
// for concurrent access.
//
// # See Also
//
//   - sdk.Organization.GetOrgMetadata() for accessing organization metadata
//   - sdk.Service.GetServiceMetadata() for accessing service metadata
//   - storage package for fetching metadata from IPFS/Lighthouse
//   - blockchain package for on-chain registry interactions
//   - examples/orgs-and-services for metadata usage examples
package model
