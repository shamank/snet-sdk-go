### How to deal with services and orgs

```go

package main

import (
    "fmt"
	"log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/model"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
    cfg := config.Config{
        RPCAddr:    "wss://sepolia.infura.io/ws/v3/YOUR_INFURA_KEY",
        PrivateKey: "YOUR_PRIVATE_KEY", // Required for write operations
        Debug:      true,
	}

	core := sdk.NewSDK(&cfg)
	defer core.Close()

	// Example 1: Creating a new organization
	orgMetadata := &model.OrganizationMetaData{
		OrgName: "My Test Organization",
		OrgID:   "my-test-org",
		Groups: []*model.OrganizationGroup{
			{
				ID:        "default_group",
				GroupName: "default_group",
				PaymentDetails: model.Payment{
					PaymentAddress: "0x0000000000000000000000000000000000000000",
				},
			},
		},
	}

	members := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
	}

	evm := core.GetEvm()
	txHash, err := sdk.CreateOrganization(evm, &cfg, "my-test-org", orgMetadata, members)
	if err != nil {
		log.Printf("Failed to create organization: %v", err)
	} else {
		fmt.Printf("Organization created! Transaction hash: %s\n", txHash.Hex())
	}

	// Example 2: Working with existing organization
	org, err := core.NewOrganizationClient("example-org", "default_group")
	if err != nil {
		log.Fatalf("Failed to create organization client: %v", err)
	}

	// Get organization metadata
	orgData := org.GetOrgMetadata()
	fmt.Printf("Organization Name: %s\n", orgData.OrgName)
	fmt.Printf("Organization ID: %s\n", org.GetOrgID())

	// List services
	services, err := org.ListServices()
	if err != nil {
		log.Printf("Failed to list services: %v", err)
	} else {
		fmt.Printf("Services in organization: %v\n", services)
	}

	// Example 3: Adding members to organization
	newMembers := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}

	txHash, err = org.AddMembers(newMembers)
	if err != nil {
		log.Printf("Failed to add members: %v", err)
	} else {
		fmt.Printf("Members added! Transaction hash: %s\n", txHash.Hex())
	}

	// Example 4: Updating organization metadata
	updatedMetadata := &model.OrganizationMetaData{
		OrgName: "Updated Organization Name",
		OrgID:   "example-org",
		Groups: []*model.OrganizationGroup{
			{
				ID:        "default_group",
				GroupName: "default_group",
				PaymentDetails: model.Payment{
					PaymentAddress: "0x0000000000000000000000000000000000000000",
				},
			},
			{
				ID:        "premium_group",
				GroupName: "premium_group",
				PaymentDetails: model.Payment{
					PaymentAddress: "0x0000000000000000000000000000000000000001",
				},
			},
		},
	}

	txHash, err = org.UpdateOrgMetadataFull(updatedMetadata)
	if err != nil {
		log.Printf("Failed to update metadata: %v", err)
	} else {
		fmt.Printf("Metadata updated! Transaction hash: %s\n", txHash.Hex())
	}

	// Example 5: Creating a new service
	serviceMetadata := &model.ServiceMetadata{
		Version:     1,
		DisplayName: "My AI Service",
		Groups: []*model.ServiceGroup{
			{
				GroupName: "default_group",
				Endpoints: []string{"https://my-service-endpoint.com"},
			},
		},
	}

	txHash, err = org.CreateService("my-ai-service", serviceMetadata)
	if err != nil {
		log.Printf("Failed to create service: %v", err)
	} else {
		fmt.Printf("Service created! Transaction hash: %s\n", txHash.Hex())
	}

	// Example 6: Working with existing service
	service, err := org.ServiceClient("existing-service", "default_group")
	if err != nil {
		log.Printf("Failed to create service client: %v", err)
		return
	}
	defer service.Close()

	serviceData := service.GetServiceMetadata()
	fmt.Printf("Service Display Name: %s\n", serviceData.DisplayName)
	fmt.Printf("Service ID: %s\n", service.GetServiceID())

	// Update service metadata
	updatedServiceMetadata := &model.ServiceMetadata{
		Version:     1,
		DisplayName: "Updated AI Service",
		Groups: []*model.ServiceGroup{
			{
				GroupName: "default_group",
				Endpoints: []string{"https://updated-endpoint.com"},
			},
		},
	}

	txHash, err = service.UpdateServiceMetadata(updatedServiceMetadata)
	if err != nil {
		log.Printf("Failed to update service metadata: %v", err)
	} else {
		fmt.Printf("Service metadata updated! Transaction hash: %s\n", txHash.Hex())
	}

	// Example 7: Removing members from organization
	membersToRemove := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}

	txHash, err = org.RemoveMembers(membersToRemove)
	if err != nil {
		log.Printf("Failed to remove members: %v", err)
	} else {
		fmt.Printf("Members removed! Transaction hash: %s\n", txHash.Hex())
	}

	// Example 8: Changing organization owner
	newOwner := common.HexToAddress("0x0000000000000000000000000000000000000004")
	txHash, err = org.ChangeOwner(newOwner)
	if err != nil {
		log.Printf("Failed to change owner: %v", err)
	} else {
		fmt.Printf("Owner changed! Transaction hash: %s\n", txHash.Hex())
	}

	// Example 9: Deleting service
	txHash, err = service.DeleteService()
	if err != nil {
		log.Printf("Failed to delete service: %v", err)
	} else {
		fmt.Printf("Service deleted! Transaction hash: %s\n", txHash.Hex())
	}

	// Example 10: Deleting organization
	txHash, err = org.DeleteOrganization()
	if err != nil {
		log.Printf("Failed to delete organization: %v", err)
	} else {
		fmt.Printf("Organization deleted! Transaction hash: %s\n", txHash.Hex())
	}
}
```
