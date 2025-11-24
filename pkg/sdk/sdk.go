// Package sdk exposes the high-level SingularityNET SDK entry points. It wires
// together blockchain access (registry/MPE), storage backends (IPFS/Lighthouse),
// dynamic gRPC invocation, and service/payment setup.
package sdk

import (
	"crypto/ecdsa"
	"os"

	"github.com/singnet/snet-sdk-go/pkg/blockchain"
	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/storage"
	"go.uber.org/zap"
)

// SnetSDK is the public interface for constructing per-service clients and
// releasing resources. Note: the Core implementation also offers a context-aware
// constructor (Core.NewServiceClient) that includes a context parameter.
type SnetSDK interface {
	// NewServiceClient creates a client bound to the given org/service/group.
	// Implementations may fetch metadata from on-chain registry/IPFS and
	// initialize a gRPC client to the service endpoint.
	NewServiceClient(serviceID, orgID, groupName string) (Service, error)

	// NewOrganizationClient creates an organization client for the specified organization and group
	NewOrganizationClient(orgID, groupName string) (Organization, error)

	// Close releases resources associated with the SDK instance.
	Close()
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
func NewSDK(config *config.Config) *Core {
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

// Close shuts down underlying network clients (e.g., Ethereum RPC).
func (c *Core) Close() {
	c.GetEvm().Close()
}
