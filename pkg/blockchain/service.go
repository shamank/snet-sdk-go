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

// ServiceClient represents a blockchain client for service operations.
// It embeds service metadata and provides access to blockchain interactions.
type ServiceClient struct {
	ServiceID string
	*model.ServiceMetadata
	CurrentGroup *model.ServiceGroup
	org          *model.OrganizationMetaData
	*EVMClient
}

// NewServiceClient creates a new service client for the specified service and group.
// It fetches and parses service metadata including proto files from distributed storage.
func (orgClient *OrgClient) NewServiceClient(srvID, groupName string) (*ServiceClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	hash := orgClient.getServiceHash(srvID)

	rawMetadata, err := orgClient.ReadFile(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("can't read serviceMetadata file: %w", err)
	}

	var serviceMetadata model.ServiceMetadata
	if err := json.Unmarshal(rawMetadata, &serviceMetadata); err != nil {
		return nil, fmt.Errorf("can't parse serviceMetadata: %w", err)
	}

	var rawFile []byte

	// Backward compatibility: older metadata may use ModelIpfsHash.
	if serviceMetadata.ModelIpfsHash != "" {
		rawFile, err = orgClient.ReadFile(ctx, serviceMetadata.ModelIpfsHash)
	}
	if serviceMetadata.ServiceApiSource != "" {
		rawFile, err = orgClient.ReadFile(ctx, serviceMetadata.ServiceApiSource)
	}
	if err != nil {
		return nil, fmt.Errorf("can't read api source (proto) files: %w", err)
	}

	var currentSrvGroup *model.ServiceGroup
	for _, v := range serviceMetadata.Groups {
		if v.GroupName == groupName {
			currentSrvGroup = v
			break
		}
	}

	serviceMetadata.ProtoFiles, err = storage.ParseProtoFiles(rawFile)
	if err != nil {
		return nil, fmt.Errorf("can't parse proto files: %w", err)
	}

	if currentSrvGroup == nil {
		return nil, fmt.Errorf("group %s not found in service %s", groupName, srvID)
	}

	// TODO: endpoint selection strategy (currently takes the first endpoint).
	if len(currentSrvGroup.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints found in group %s", groupName)
	}

	return &ServiceClient{srvID, &serviceMetadata, currentSrvGroup, orgClient.OrganizationMetaData, orgClient.EVMClient}, nil
}

// updateMetadataUri updates the service metadata URI in the Registry contract.
// This is a low-level method that submits a transaction without authentication.
func (srvClient *ServiceClient) updateMetadataUri(uri string) common.Hash {
	resp, err := srvClient.Registry.UpdateServiceRegistration(nil, StringToBytes32(srvClient.org.OrgID), StringToBytes32(srvClient.ServiceID), []byte(uri))
	if err != nil || resp == nil {
		zap.L().Panic("Error updateMetadataUri",
			zap.String("OrganizationId", srvClient.org.OrgID),
			zap.String("ServiceId", srvClient.ServiceID),
			zap.String("uri", uri),
			zap.Error(err),
		)
	}
	return resp.Hash()
}

// Delete removes the service registration from the Registry contract.
// This is a low-level method that submits a transaction without authentication.
func (srvClient *ServiceClient) Delete() common.Hash {
	resp, err := srvClient.Registry.DeleteServiceRegistration(nil, StringToBytes32(srvClient.org.OrgID), StringToBytes32(srvClient.ServiceID))
	if err != nil || resp == nil {
		zap.L().Panic("Error DeleteServiceRegistration",
			zap.String("OrganizationId", srvClient.org.OrgID),
			zap.String("ServiceId", srvClient.ServiceID),
			zap.Error(err),
		)
	}
	return resp.Hash()
}

// Update updates the service registration in the Registry contract.
func (srvClient *ServiceClient) Update() common.Hash {
	resp, err := srvClient.Registry.DeleteServiceRegistration(nil, StringToBytes32(srvClient.org.OrgID), StringToBytes32(srvClient.ServiceID))
	if err != nil || resp == nil {
		zap.L().Panic("Error DeleteServiceRegistration",
			zap.String("OrganizationId", srvClient.org.OrgID),
			zap.String("ServiceId", srvClient.ServiceID),
			zap.Error(err),
		)
	}
	return resp.Hash()
}

// CreateServiceRegistration creates a new service registration in the Registry contract.
func (orgClient *OrgClient) CreateServiceRegistration(pk *ecdsa.PrivateKey, serviceID string, metadataURI []byte) (common.Hash, error) {
	opts, err := orgClient.GetTransactOpts(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transact opts: %w", err)
	}

	tx, err := orgClient.Registry.CreateServiceRegistration(opts, StringToBytes32(orgClient.OrgID), StringToBytes32(serviceID), metadataURI)
	if err != nil {
		zap.L().Error("Failed to create service registration",
			zap.String("orgId", orgClient.OrgID),
			zap.String("serviceId", serviceID),
			zap.Error(err))
		return common.Hash{}, fmt.Errorf("failed to create service registration: %w", err)
	}

	zap.L().Info("Service registration transaction sent",
		zap.String("orgId", orgClient.OrgID),
		zap.String("serviceId", serviceID),
		zap.String("txHash", tx.Hash().Hex()))
	return tx.Hash(), nil
}

// UpdateServiceMetadata updates the metadata URI for a service.
func (srvClient *ServiceClient) UpdateServiceMetadata(pk *ecdsa.PrivateKey, metadataURI []byte) (common.Hash, error) {
	opts, err := srvClient.GetTransactOpts(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transact opts: %w", err)
	}

	tx, err := srvClient.Registry.UpdateServiceRegistration(opts, StringToBytes32(srvClient.org.OrgID), StringToBytes32(srvClient.ServiceID), metadataURI)
	if err != nil {
		zap.L().Error("Failed to update service metadata",
			zap.String("orgId", srvClient.org.OrgID),
			zap.String("serviceId", srvClient.ServiceID),
			zap.Error(err))
		return common.Hash{}, fmt.Errorf("failed to update service metadata: %w", err)
	}

	zap.L().Info("Service metadata update transaction sent",
		zap.String("orgId", srvClient.org.OrgID),
		zap.String("serviceId", srvClient.ServiceID),
		zap.String("txHash", tx.Hash().Hex()))
	return tx.Hash(), nil
}

// DeleteServiceWithAuth deletes a service registration with authentication.
func (srvClient *ServiceClient) DeleteServiceWithAuth(pk *ecdsa.PrivateKey) (common.Hash, error) {
	opts, err := srvClient.GetTransactOpts(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create transact opts: %w", err)
	}

	tx, err := srvClient.Registry.DeleteServiceRegistration(opts, StringToBytes32(srvClient.org.OrgID), StringToBytes32(srvClient.ServiceID))
	if err != nil {
		zap.L().Error("Failed to delete service registration",
			zap.String("orgId", srvClient.org.OrgID),
			zap.String("serviceId", srvClient.ServiceID),
			zap.Error(err))
		return common.Hash{}, fmt.Errorf("failed to delete service registration: %w", err)
	}

	zap.L().Info("Service deletion transaction sent",
		zap.String("orgId", srvClient.org.OrgID),
		zap.String("serviceId", srvClient.ServiceID),
		zap.String("txHash", tx.Hash().Hex()))
	return tx.Hash(), nil
}
