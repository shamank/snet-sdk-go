## Working with Organizations and Services

This guide covers how to manage organizations and services on the SingularityNET platform. Organizations are entities that publish and manage AI services. Understanding these concepts is essential for both service providers and consumers.

### Core Concepts

- **Organization**: A registered entity that can publish and manage multiple AI services
- **Service**: An AI service published by an organization, accessible via the platform
- **Members**: Ethereum addresses authorized to manage the organization
- **Groups**: Logical groupings within an organization for payment and endpoint management
- **Metadata**: On-chain and off-chain information describing organizations and services

## Table of Contents

1. [Organization Operations](#organization-operations)
   - [Creating an Organization](#creating-an-organization)
   - [Working with Existing Organizations](#working-with-existing-organizations)
   - [Updating Organization Metadata](#updating-organization-metadata)
   - [Deleting an Organization](#deleting-an-organization)

2. [Service Operations](#service-operations)
   - [Creating a Service](#creating-a-service)
   - [Working with Existing Services](#working-with-existing-services)
   - [Updating Service Metadata](#updating-service-metadata)
   - [Deleting a Service](#deleting-a-service)

3. [Member Management](#member-management)
   - [Adding Members](#adding-members)
   - [Removing Members](#removing-members)
   - [Changing Organization Owner](#changing-organization-owner)

4. [Metadata Management](#metadata-management)
   - [Organization Metadata Structure](#organization-metadata-structure)
   - [Service Metadata Structure](#service-metadata-structure)

---

## Organization Operations

Organizations are the top-level entities that publish services. Each organization has members, metadata, and payment groups.

### Organization-Service Relationship

```
Organization (e.g., "ai-lab")
├── Members (Ethereum addresses)
├── Groups (payment & endpoints)
│   ├── default_group
│   └── premium_group
└── Services
    ├── Service A
    ├── Service B
    └── Service C
```

### Creating an Organization
```go
package main

import (
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shamank/snet-sdk-go/pkg/config"
	"github.com/shamank/snet-sdk-go/pkg/model"
	"github.com/shamank/snet-sdk-go/pkg/sdk"
)

func main() {
	cfg := config.Config{
		RPCAddr:    "wss://sepolia.infura.io/ws/v3/YOUR_INFURA_KEY",
		PrivateKey: "YOUR_PRIVATE_KEY", // Required for write operations
		Debug:      true,
		Network:    config.Sepolia,
	}

	core := sdk.NewSDK(&cfg)
	defer core.Close()

	// Define organization metadata
	orgMetadata := &model.OrganizationMetaData{
		OrgName: "My AI Lab",
		OrgID:   "my-ai-lab",
		Groups: []*model.OrganizationGroup{
			{
				ID:        "default_group",
				GroupName: "default_group",
				PaymentDetails: model.Payment{
					PaymentAddress: "0x1234567890123456789012345678901234567890",
				},
			},
		},
	}

	// Define initial members (Ethereum addresses)
	members := []common.Address{
		common.HexToAddress("0xYOUR_MEMBER_ADDRESS"),
	}
	
	// Create the organization on-chain
	txHash, err := core.CreateOrganization("my-ai-lab", orgMetadata, members)
	if err != nil {
		log.Fatalf("Failed to create organization: %v", err)
	}
	
	fmt.Printf("✓ Organization created successfully!\n")
	fmt.Printf("Transaction hash: %s\n", txHash.Hex())
	fmt.Printf("View on Etherscan: https://sepolia.etherscan.io/tx/%s\n", txHash.Hex())
}
```

**Note**: Creating an organization requires gas fees. Ensure your wallet has sufficient ETH for the transaction.

### Working with Existing Organizations

```go
// Connect to an existing organization
org, err := core.NewOrganizationClient("example-org", "default_group")
if err != nil {
	log.Fatalf("Failed to create organization client: %v", err)
}

// Get organization metadata
orgData := org.GetOrgMetadata()
fmt.Printf("Organization Name: %s\n", orgData.OrgName)
fmt.Printf("Organization ID: %s\n", org.GetOrgID())

// List all services in the organization
services, err := org.ListServices()
if err != nil {
	log.Printf("Failed to list services: %v", err)
} else {
	fmt.Printf("Services in organization:\n")
	for _, service := range services {
		fmt.Printf("  - %s\n", service)
	}
}
```

### Updating Organization Metadata

```go
// Update organization information
updatedMetadata := &model.OrganizationMetaData{
	OrgName: "Updated Organization Name",
	OrgID:   "example-org",
	Groups: []*model.OrganizationGroup{
		{
			ID:        "default_group",
			GroupName: "default_group",
			PaymentDetails: model.Payment{
				PaymentAddress: "0x1234567890123456789012345678901234567890",
			},
		},
		{
			ID:        "premium_group",
			GroupName: "premium_group",
			PaymentDetails: model.Payment{
				PaymentAddress: "0x0987654321098765432109876543210987654321",
			},
		},
	},
}

txHash, err := org.UpdateOrgMetadataFull(updatedMetadata)
if err != nil {
	log.Fatalf("Failed to update metadata: %v", err)
}

fmt.Printf("✓ Metadata updated! Transaction: %s\n", txHash.Hex())
```

### Deleting an Organization

```go
// Delete an organization (removes from registry)
txHash, err := org.DeleteOrganization()
if err != nil {
	log.Fatalf("Failed to delete organization: %v", err)
}

fmt.Printf("✓ Organization deleted! Transaction: %s\n", txHash.Hex())
```

**Warning**: Deleting an organization is irreversible and will affect all associated services.

---

## Service Operations

Services are AI models or applications published by organizations. Each service has endpoints, pricing, and metadata.

### Creating a Service

```go
// Define service metadata
serviceMetadata := &model.ServiceMetadata{
	Version:     1,
	DisplayName: "Image Classification Service",
	Description: "Deep learning model for image classification",
	Groups: []*model.ServiceGroup{
		{
			GroupName: "default_group",
			Endpoints: []string{"https://api.example.com:8080"},
			Pricing: &model.PricingInfo{
				PriceModel:  "fixed_price",
				PriceInCogs: 100, // Price per call in FET cogs
			},
		},
	},
}

// Create service under organization
txHash, err := org.CreateService("image-classifier", serviceMetadata)
if err != nil {
	log.Fatalf("Failed to create service: %v", err)
}

fmt.Printf("✓ Service created! Transaction: %s\n", txHash.Hex())
```

### Working with Existing Services

```go
// Get a service client from the organization
service, err := org.ServiceClient("image-classifier", "default_group")
if err != nil {
	log.Fatalf("Failed to create service client: %v", err)
}
defer service.Close()

// Get service information
serviceData := service.GetServiceMetadata()
fmt.Printf("Service Display Name: %s\n", serviceData.DisplayName)
fmt.Printf("Service ID: %s\n", service.GetServiceID())
fmt.Printf("Organization ID: %s\n", service.GetOrgID())

// Get service endpoints
for _, group := range serviceData.Groups {
	fmt.Printf("Group: %s\n", group.GroupName)
	for _, endpoint := range group.Endpoints {
		fmt.Printf("  Endpoint: %s\n", endpoint)
	}
}
```

### Updating Service Metadata

```go
// Update service information
updatedServiceMetadata := &model.ServiceMetadata{
	Version:     2, // Increment version
	DisplayName: "Advanced Image Classifier",
	Description: "Updated model with better accuracy",
	Groups: []*model.ServiceGroup{
		{
			GroupName: "default_group",
			Endpoints: []string{
				"https://api.example.com:8080",
				"https://backup.example.com:8080", // Add backup endpoint
			},
			Pricing: &model.PricingInfo{
				PriceModel:  "fixed_price",
				PriceInCogs: 150, // Updated pricing
			},
		},
	},
}

txHash, err := service.UpdateServiceMetadata(updatedServiceMetadata)
if err != nil {
	log.Fatalf("Failed to update service metadata: %v", err)
}

fmt.Printf("✓ Service updated! Transaction: %s\n", txHash.Hex())
```

### Deleting a Service

```go
// Remove service from the registry
txHash, err := service.DeleteService()
if err != nil {
	log.Fatalf("Failed to delete service: %v", err)
}

fmt.Printf("✓ Service deleted! Transaction: %s\n", txHash.Hex())
```

---

## Member Management

Organization members are Ethereum addresses with permissions to manage the organization and its services.

### Permission Levels

- **Owner**: Full control over organization (can add/remove members, delete org)
- **Member**: Can manage services and update metadata
- **Non-member**: No management permissions (read-only via registry)

### Adding Members

```go
// Add new members to the organization
newMembers := []common.Address{
	common.HexToAddress("0xNEW_MEMBER_ADDRESS_1"),
	common.HexToAddress("0xNEW_MEMBER_ADDRESS_2"),
}

txHash, err := org.AddMembers(newMembers)
if err != nil {
	log.Fatalf("Failed to add members: %v", err)
}

fmt.Printf("✓ Members added! Transaction: %s\n", txHash.Hex())
```

### Removing Members

```go
// Remove members from the organization
membersToRemove := []common.Address{
	common.HexToAddress("0xMEMBER_TO_REMOVE"),
}

txHash, err := org.RemoveMembers(membersToRemove)
if err != nil {
	log.Fatalf("Failed to remove members: %v", err)
}

fmt.Printf("✓ Members removed! Transaction: %s\n", txHash.Hex())
```

**Note**: The organization owner cannot be removed as a member.

### Changing Organization Owner

```go
// Transfer organization ownership to a new address
newOwner := common.HexToAddress("0xNEW_OWNER_ADDRESS")

txHash, err := org.ChangeOwner(newOwner)
if err != nil {
	log.Fatalf("Failed to change owner: %v", err)
}

fmt.Printf("✓ Owner changed! Transaction: %s\n", txHash.Hex())
```

**Warning**: This transfers all ownership rights. The current owner will lose admin privileges.

---

## Metadata Management

Metadata describes organizations and services both on-chain and off-chain.

### Organization Metadata Structure

```go
type OrganizationMetaData struct {
	OrgName     string                // Display name
	OrgID       string                // Unique identifier
	Groups      []*OrganizationGroup  // Payment groups
	Description string                // Optional description
	// Additional fields...
}

type OrganizationGroup struct {
	ID             string   // Group identifier
	GroupName      string   // Group display name
	PaymentDetails Payment  // Payment configuration
}

type Payment struct {
	PaymentAddress string  // Ethereum address for payments
	// Additional payment config...
}
```

### Service Metadata Structure

```go
type ServiceMetadata struct {
	Version      int             // Metadata version
	DisplayName  string          // Service display name
	Description  string          // Service description
	Groups       []*ServiceGroup // Service groups with endpoints
	// Additional fields...
}

type ServiceGroup struct {
	GroupName string       // Must match org group
	Endpoints []string     // Service endpoint URLs
	Pricing   *PricingInfo // Pricing details
}

type PricingInfo struct {
	PriceModel  string  // "fixed_price" or other models
	PriceInCogs int64   // Price per call in FET cogs (1 FET = 10^8 cogs)
}
```