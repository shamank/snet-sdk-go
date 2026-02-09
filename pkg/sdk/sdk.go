// Package sdk exposes the high-level SingularityNET SDK entry points. It wires
// together blockchain access (registry/MPE), storage backends (IPFS/Lighthouse),
// dynamic gRPC invocation, and service/payment setup.
package sdk

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shamank/snet-sdk-go/pkg/blockchain"
	"github.com/shamank/snet-sdk-go/pkg/config"
	"github.com/shamank/snet-sdk-go/pkg/model"
	"github.com/shamank/snet-sdk-go/pkg/storage"
	"go.uber.org/zap"
)

// SnetSDK is the public interface for constructing per-service clients and
// releasing resources. Note: the Core implementation also offers a context-aware
// constructor (Core.NewServiceClient) that includes a context parameter.
type SnetSDK interface {
	// NewServiceClient creates a client bound to the given org/service/group.
	// Implementations may fetch metadata from on-chain registry/IPFS and
	// initialize a gRPC client to the service endpoint.
	NewServiceClient(orgID, serviceID, groupName string) (Service, error)

	// NewOrganizationClient creates an organization client for the specified organization and group
	NewOrganizationClient(orgID, groupName string) (Organization, error)

	// CreateOrganization Create new organization
	CreateOrganization(orgID string, metadata *model.OrganizationMetaData, members []common.Address) (common.Hash, error)

	// GetOrganizations Get all organizations from registry smart contract as string array
	GetOrganizations() ([]string, error)

	// Close releases resources associated with the SDK instance.
	Close()
}

// init configures a default global zap logger for the SDK. Applications may
// replace it with zap.ReplaceGlobals(...) if they need custom logging.
func init() {
	c := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := c.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
}

// Core is the concrete SDK implementation. It embeds the initialized EVM
// client and runtime configuration.
type Core struct {
	evm *blockchain.EVMClient
	*config.Config
	prvKey *ecdsa.PrivateKey
}

// GetEvm returns the EVM client for advanced operations like organization creation.
// This is used internally by the SDK and can be accessed by users for custom blockchain operations.
func (c *Core) GetEvm() *blockchain.EVMClient {
	return c.evm
}

// NewSDK initializes the SDK Core with validated configuration and a connected
// EVM client. It applies default timeout values and aborts the process if the
// configuration is invalid or the Ethereum client cannot be initialized.
func NewSDK(config *config.Config) SnetSDK {
	err := config.Validate()
	if err != nil {
		zap.L().Fatal("Invalid config", zap.Error(err))
	}

	config.Timeouts = config.Timeouts.WithDefaults()

	storageClient := storage.NewStorage(config.IpfsURL, config.LighthouseURL)

	evmClient, err := blockchain.InitEvm(config.Network.ChainID, config.RPCAddr, config.RegistryAddr, storageClient)
	if err != nil {
		zap.L().Error("Init ethereum client failed", zap.Error(err))
		os.Exit(-1)
	}

	address, prvKey, err := blockchain.ParsePrivateKeyECDSA(config.PrivateKey)
	if err != nil {
		zap.L().Warn("some methods disabled: private key parsing failed", zap.Error(err))
	}

	if config.Debug {
		zap.L().Debug("signer address", zap.String("addr", address.Hex()))
	}

	return &Core{
		evmClient,
		config,
		prvKey,
	}
}

// NewOrganizationClient creates a new organization client for the specified organization and group.
func (c *Core) NewOrganizationClient(orgID, groupName string) (Organization, error) {

	client, err := c.GetEvm().NewOrgClient(orgID, groupName)
	if err != nil {
		return nil, err
	}

	return &OrganizationClient{
		config:           c.Config,
		blockchainClient: client,
		CurrentGroup:     client.CurrentOrgGroup,
	}, nil
}

// GetOrganizations  - get all organizations as array
func (c *Core) GetOrganizations() ([]string, error) {
	return c.GetEvm().GetOrganizations()
}

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
func (c *Core) CreateOrganization(orgID string, metadata *model.OrganizationMetaData, members []common.Address) (common.Hash, error) {
	pk := c.Config.GetPrivateKey()
	if pk == nil {
		return common.Hash{}, fmt.Errorf("private key not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Upload metadata to IPFS
	uri, err := c.evm.Storage.UploadJSON(ctx, metadata)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to upload metadata to IPFS: %w", err)
	}

	// Create organization in blockchain
	hash, err := c.evm.CreateOrganization(pk, orgID, []byte(uri), members)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create organization: %w", err)
	}

	return hash, nil
}

// Close shuts down underlying network clients (e.g., Ethereum RPC).
func (c *Core) Close() {
	c.GetEvm().Close()
}
