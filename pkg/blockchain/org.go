package blockchain

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shamank/snet-sdk-go/pkg/model"
	"github.com/shamank/snet-sdk-go/pkg/storage"
	"go.uber.org/zap"
)

// OrgClient represents a blockchain client for organization operations.
// It embeds EVMClient and organization metadata for blockchain interactions.
type OrgClient struct {
	*EVMClient
	*model.OrganizationMetaData
	storage.Storage
	CurrentOrgGroup *model.OrganizationGroup
}

// NewOrgClient creates a new organization client for the specified organization and group.
// It fetches and parses the organization metadata from distributed storage.
func (evm *EVMClient) NewOrgClient(orgID, groupName string) (*OrgClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	orgHash := evm.getOrgMetadataUri(orgID)

	rawOrgMetadata, err := evm.Storage.ReadFile(ctx, orgHash)
	if err != nil {
		return nil, fmt.Errorf("can't read orgMetadata file: %w", err)
	}

	var orgMetadata model.OrganizationMetaData
	if err := json.Unmarshal(rawOrgMetadata, &orgMetadata); err != nil {
		return nil, fmt.Errorf("can't parse orgMetadata: %w", err)
	}

	var currentOrgGroup *model.OrganizationGroup
	for _, v := range orgMetadata.Groups {
		if v.GroupName == groupName {
			currentOrgGroup = v
			break
		}
	}

	return &OrgClient{evm, &orgMetadata, evm.Storage, currentOrgGroup}, nil
}

// getServiceHash retrieves the service metadata URI (hash) from the Registry contract.
// It queries the Registry for the service registration using the organization and service IDs.
func (orgClient *OrgClient) getServiceHash(srvID string) string {
	orgId := StringToBytes32(orgClient.OrgID)
	serviceId := StringToBytes32(srvID)
	serviceRegistration, err := orgClient.Registry.GetServiceRegistrationById(nil, orgId, serviceId)
	if err != nil || &serviceRegistration == nil || !serviceRegistration.Found {
		zap.L().Panic("Error Retrieving contract details for the Given Organization and Service Ids ",
			zap.String("OrganizationId", orgClient.OrgID),
			zap.String("ServiceId", srvID))
	}

	return string(serviceRegistration.MetadataURI[:])
}

// GetServices returns service IDs for the given organization ID.
// If the organization is not found or a read error occurs, it logs and returns nil.
func (orgClient *OrgClient) GetServices() []string {
	organizations, err := orgClient.Registry.ListServicesForOrganization(nil, StringToBytes32(orgClient.OrgID))
	if err != nil {
		zap.L().Error("Failed to list organizations", zap.Error(err))
		return nil
	}
	if !organizations.Found {
		zap.L().Error("Organization not found", zap.String("OrganizationID", orgClient.OrgID))
		return nil
	}
	return Bytes32ArrayToStrings(organizations.ServiceIds)
}

// UpdateOrgMetadata updates the organization metadata URI in the Registry contract.
// Returns the transaction hash.
func (orgClient *OrgClient) UpdateOrgMetadata(uri string) common.Hash {
	resp, err := orgClient.Registry.ChangeOrganizationMetadataURI(nil, StringToBytes32(orgClient.OrgID), []byte(uri))
	if err != nil || &resp == nil {
		zap.L().Panic("Error UpdateOrgMetadata ",
			zap.String("OrganizationId", orgClient.OrgID),
			zap.String("uri", uri))
	}

	return resp.Hash()
}

// AddMember adds new members to the organization.
// Returns the transaction hash.
func (orgClient *OrgClient) AddMember(newMembers []common.Address) common.Hash {
	resp, err := orgClient.Registry.AddOrganizationMembers(nil, StringToBytes32(orgClient.OrgID), newMembers)
	if err != nil || &resp == nil {
		zap.L().Panic("Error UpdateOrgMetadata ",
			zap.String("OrganizationId", orgClient.OrgID))
	}

	return resp.Hash()
}

// CreateOrganization creates a new organization in the Registry contract.
func (evm *EVMClient) CreateOrganization(pk *ecdsa.PrivateKey, orgID string, orgMetadataURI []byte, members []common.Address) (common.Hash, error) {
	opts, err := evm.GetTransactOpts(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transact opts: %w", err)
	}

	tx, err := evm.Registry.CreateOrganization(opts, StringToBytes32(orgID), orgMetadataURI, members)
	if err != nil {
		zap.L().Error("Failed to create organization",
			zap.String("orgId", orgID),
			zap.Error(err))
		return common.Hash{}, fmt.Errorf("failed to create organization: %w", err)
	}

	zap.L().Info("Organization creation transaction sent",
		zap.String("orgId", orgID),
		zap.String("txHash", tx.Hash().Hex()))
	return tx.Hash(), nil
}

// DeleteOrganization deletes an organization from the Registry contract.
func (orgClient *OrgClient) DeleteOrganization(pk *ecdsa.PrivateKey) (common.Hash, error) {
	opts, err := orgClient.GetTransactOpts(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transact opts: %w", err)
	}

	tx, err := orgClient.Registry.DeleteOrganization(opts, StringToBytes32(orgClient.OrgID))
	if err != nil {
		zap.L().Error("Failed to delete organization",
			zap.String("orgId", orgClient.OrgID),
			zap.Error(err))
		return common.Hash{}, fmt.Errorf("failed to delete organization: %w", err)
	}

	zap.L().Info("Organization deletion transaction sent",
		zap.String("orgId", orgClient.OrgID),
		zap.String("txHash", tx.Hash().Hex()))
	return tx.Hash(), nil
}

// ChangeOrganizationOwner changes the owner of the organization.
func (orgClient *OrgClient) ChangeOrganizationOwner(pk *ecdsa.PrivateKey, newOwner common.Address) (common.Hash, error) {
	opts, err := orgClient.GetTransactOpts(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transact opts: %w", err)
	}

	tx, err := orgClient.Registry.ChangeOrganizationOwner(opts, StringToBytes32(orgClient.OrgID), newOwner)
	if err != nil {
		zap.L().Error("Failed to change organization owner",
			zap.String("orgId", orgClient.OrgID),
			zap.String("newOwner", newOwner.Hex()),
			zap.Error(err))
		return common.Hash{}, fmt.Errorf("failed to change organization owner: %w", err)
	}

	zap.L().Info("Organization owner change transaction sent",
		zap.String("orgId", orgClient.OrgID),
		zap.String("newOwner", newOwner.Hex()),
		zap.String("txHash", tx.Hash().Hex()))
	return tx.Hash(), nil
}

// RemoveOrganizationMembers removes members from the organization.
func (orgClient *OrgClient) RemoveOrganizationMembers(pk *ecdsa.PrivateKey, members []common.Address) (common.Hash, error) {
	opts, err := orgClient.GetTransactOpts(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transact opts: %w", err)
	}

	tx, err := orgClient.Registry.RemoveOrganizationMembers(opts, StringToBytes32(orgClient.OrgID), members)
	if err != nil {
		zap.L().Error("Failed to remove organization members",
			zap.String("orgId", orgClient.OrgID),
			zap.Error(err))
		return common.Hash{}, fmt.Errorf("failed to remove organization members: %w", err)
	}

	zap.L().Info("Organization members removal transaction sent",
		zap.String("orgId", orgClient.OrgID),
		zap.String("txHash", tx.Hash().Hex()))
	return tx.Hash(), nil
}

// UpdateOrgMetadataWithAuth updates organization metadata URI with authentication.
func (orgClient *OrgClient) UpdateOrgMetadataWithAuth(pk *ecdsa.PrivateKey, uri string) (common.Hash, error) {
	opts, err := orgClient.GetTransactOpts(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transact opts: %w", err)
	}

	tx, err := orgClient.Registry.ChangeOrganizationMetadataURI(opts, StringToBytes32(orgClient.OrgID), []byte(uri))
	if err != nil {
		zap.L().Error("Failed to update organization metadata",
			zap.String("orgId", orgClient.OrgID),
			zap.String("uri", uri),
			zap.Error(err))
		return common.Hash{}, fmt.Errorf("failed to update organization metadata: %w", err)
	}

	zap.L().Info("Organization metadata update transaction sent",
		zap.String("orgId", orgClient.OrgID),
		zap.String("uri", uri),
		zap.String("txHash", tx.Hash().Hex()))
	return tx.Hash(), nil
}

// AddOrganizationMembersWithAuth adds members to the organization with authentication.
func (orgClient *OrgClient) AddOrganizationMembersWithAuth(pk *ecdsa.PrivateKey, members []common.Address) (common.Hash, error) {
	opts, err := orgClient.GetTransactOpts(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transact opts: %w", err)
	}

	tx, err := orgClient.Registry.AddOrganizationMembers(opts, StringToBytes32(orgClient.OrgID), members)
	if err != nil {
		zap.L().Error("Failed to add organization members",
			zap.String("orgId", orgClient.OrgID),
			zap.Error(err))
		return common.Hash{}, fmt.Errorf("failed to add organization members: %w", err)
	}

	zap.L().Info("Organization members addition transaction sent",
		zap.String("orgId", orgClient.OrgID),
		zap.String("txHash", tx.Hash().Hex()))
	return tx.Hash(), nil
}
