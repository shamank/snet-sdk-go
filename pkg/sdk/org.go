package sdk

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-sdk-go/pkg/blockchain"
	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/grpc"
	"github.com/singnet/snet-sdk-go/pkg/model"
)

// CreateOrganization creates a new organization in the Registry with the given metadata.
// It uploads the metadata to IPFS and registers the organization in the blockchain.
// This is a package-level function that operates at the same level as service creation.
//
// Parameters:
//   - evm: EVMClient for blockchain interaction
//   - cfg: Configuration containing private key
//   - orgID: Unique identifier for the organization
//   - metadata: Organization metadata to upload to IPFS
//   - members: List of member addresses for the organization
//
// Returns transaction hash and error if any.
func CreateOrganization(evm *blockchain.EVMClient, cfg *config.Config, orgID string, metadata *model.OrganizationMetaData, members []common.Address) (common.Hash, error) {
	pk := cfg.GetPrivateKey()
	if pk == nil {
		return common.Hash{}, fmt.Errorf("private key not configured")
	}

	// Upload metadata to IPFS
	uri, err := evm.Storage.UploadJSON(metadata)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to upload metadata to IPFS: %w", err)
	}

	// Create organization in blockchain
	hash, err := evm.CreateOrganization(pk, orgID, []byte(uri), members)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create organization: %w", err)
	}

	return hash, nil
}

// Organization represents a high-level interface for working with an organization
// in SingularityNET. This is the main API for SDK users.
type Organization interface {
	// ServiceClient creates a service client for the specified service and group
	ServiceClient(serviceID, groupName string) (Service, error)

	// ListServices returns a list of all services in the organization
	ListServices() ([]string, error)

	// GetOrgMetadata returns the organization metadata
	GetOrgMetadata() *model.OrganizationMetaData

	// GetCurrentGroup returns the current organization group
	GetCurrentGroup() *model.OrganizationGroup

	// GetOrgID returns the organization identifier
	GetOrgID() string

	// UpdateMetadata updates the organization metadata URI in the blockchain
	UpdateMetadata(uri string) (common.Hash, error)

	// AddMembers adds new members to the organization
	AddMembers(members []common.Address) (common.Hash, error)

	// RemoveMembers removes members from the organization
	RemoveMembers(members []common.Address) (common.Hash, error)

	// ChangeOwner changes the organization owner
	ChangeOwner(newOwner common.Address) (common.Hash, error)

	// DeleteOrganization deletes the organization
	DeleteOrganization() (common.Hash, error)

	// UpdateOrgMetadataFull updates organization metadata (uploads to IPFS and updates blockchain)
	UpdateOrgMetadataFull(metadata *model.OrganizationMetaData) (common.Hash, error)

	// CreateService creates a new service in the organization
	CreateService(serviceID string, metadata *model.ServiceMetadata) (common.Hash, error)

	// getBlockchainClient returns access to low-level blockchain operations
	// (optional, if direct access is needed)
	getBlockchainClient() *blockchain.OrgClient
}

// OrganizationClient is the concrete implementation of the Organization interface.
type OrganizationClient struct {
	config           *config.Config
	blockchainClient *blockchain.OrgClient
	CurrentGroup     *model.OrganizationGroup
}

// NewOrganizationClient creates a new organization client for the specified organization and group.
func (c *Core) NewOrganizationClient(orgID, groupName string) (Organization, error) {

	client, err := c.GetEvm().NewOrgClient(orgID, groupName)
	if err != nil {
		return nil, err
	}

	//if orgID == nil {
	//	return nil, fmt.Errorf("blockchain organization client is required")
	//}

	return &OrganizationClient{
		config:           c.Config,
		blockchainClient: client,
		CurrentGroup:     client.CurrentOrgGroup,
	}, nil
}

// ServiceClient creates a service client for the specified service and group within this organization.
func (o *OrganizationClient) ServiceClient(serviceID, groupName string) (Service, error) {

	serviceClient, err := o.blockchainClient.NewServiceClient(serviceID, groupName)
	if err != nil {
		return nil, fmt.Errorf("failed to create blockchain service client: %w", err)
	}

	if serviceClient == nil {
		return nil, fmt.Errorf("blockchain service client is required")
	}

	if serviceClient.CurrentGroup == nil || len(serviceClient.CurrentGroup.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints available for service group %s", groupName)
	}

	endpoint := serviceClient.CurrentGroup.Endpoints[0]
	grpcClient := grpc.NewClient(endpoint, serviceClient.ServiceMetadata.ProtoFiles)

	return newServiceClient(
		o.config,
		o,
		o.blockchainClient,
		serviceClient,
		grpcClient,
		o.config.GetPrivateKey(),
	), nil
}

// ListServices returns a list of IDs of all services in the organization.
func (o *OrganizationClient) ListServices() ([]string, error) {
	services := o.blockchainClient.GetServices()
	return services, nil
}

// GetOrgMetadata returns the organization metadata.
func (o *OrganizationClient) GetOrgMetadata() *model.OrganizationMetaData {
	return o.blockchainClient.OrganizationMetaData
}

// GetCurrentGroup returns the current organization group.
func (o *OrganizationClient) GetCurrentGroup() *model.OrganizationGroup {
	return o.CurrentGroup
}

// GetOrgID returns the organization identifier.
func (o *OrganizationClient) GetOrgID() string {
	return o.blockchainClient.OrgID
}

// UpdateMetadata updates the organization metadata URI in the blockchain.
func (o *OrganizationClient) UpdateMetadata(uri string) (common.Hash, error) {
	hash := o.blockchainClient.UpdateOrgMetadata(uri)
	return hash, nil
}

// AddMembers adds new members to the organization.
func (o *OrganizationClient) AddMembers(members []common.Address) (common.Hash, error) {
	if len(members) == 0 {
		return common.Hash{}, fmt.Errorf("no members to add")
	}
	hash := o.blockchainClient.AddMember(members)
	return hash, nil
}

// getBlockchainClient returns direct access to the blockchain client.
// Use with caution - for advanced use cases only.
func (o *OrganizationClient) getBlockchainClient() *blockchain.OrgClient {
	return o.blockchainClient
}

// RemoveMembers removes members from the organization.
func (o *OrganizationClient) RemoveMembers(members []common.Address) (common.Hash, error) {
	if len(members) == 0 {
		return common.Hash{}, fmt.Errorf("no members to remove")
	}

	pk := o.config.GetPrivateKey()
	if pk == nil {
		return common.Hash{}, fmt.Errorf("private key not configured")
	}

	hash, err := o.blockchainClient.RemoveOrganizationMembers(pk, members)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to remove members: %w", err)
	}

	return hash, nil
}

// ChangeOwner changes the organization owner.
func (o *OrganizationClient) ChangeOwner(newOwner common.Address) (common.Hash, error) {
	pk := o.config.GetPrivateKey()
	if pk == nil {
		return common.Hash{}, fmt.Errorf("private key not configured")
	}

	hash, err := o.blockchainClient.ChangeOrganizationOwner(pk, newOwner)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to change owner: %w", err)
	}

	return hash, nil
}

// DeleteOrganization deletes the organization.
func (o *OrganizationClient) DeleteOrganization() (common.Hash, error) {
	pk := o.config.GetPrivateKey()
	if pk == nil {
		return common.Hash{}, fmt.Errorf("private key not configured")
	}

	hash, err := o.blockchainClient.DeleteOrganization(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to delete organization: %w", err)
	}

	return hash, nil
}

// UpdateOrgMetadataFull updates organization metadata (uploads to IPFS and updates blockchain).
func (o *OrganizationClient) UpdateOrgMetadataFull(metadata *model.OrganizationMetaData) (common.Hash, error) {
	pk := o.config.GetPrivateKey()
	if pk == nil {
		return common.Hash{}, fmt.Errorf("private key not configured")
	}

	// Upload metadata to IPFS
	uri, err := o.blockchainClient.Storage.UploadJSON(metadata)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to upload metadata to IPFS: %w", err)
	}

	// Update URI in blockchain
	hash, err := o.blockchainClient.UpdateOrgMetadataWithAuth(pk, uri)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to update metadata in blockchain: %w", err)
	}

	return hash, nil
}

// CreateService creates a new service in the organization.
func (o *OrganizationClient) CreateService(serviceID string, metadata *model.ServiceMetadata) (common.Hash, error) {
	pk := o.config.GetPrivateKey()
	if pk == nil {
		return common.Hash{}, fmt.Errorf("private key not configured")
	}

	// Upload service metadata to IPFS
	uri, err := o.blockchainClient.Storage.UploadJSON(metadata)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to upload service metadata to IPFS: %w", err)
	}

	// Create service in blockchain
	hash, err := o.blockchainClient.CreateServiceRegistration(pk, serviceID, []byte(uri))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create service registration: %w", err)
	}

	return hash, nil
}
