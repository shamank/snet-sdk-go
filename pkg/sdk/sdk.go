// Package sdk exposes the high-level SingularityNET SDK entry points. It wires
// together blockchain access (registry/MPE), storage backends (IPFS/Lighthouse),
// dynamic gRPC invocation, and service/payment setup.
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/singnet/snet-sdk-go/pkg/blockchain"
	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/grpc"
	"github.com/singnet/snet-sdk-go/pkg/model"
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
	// Close releases resources associated with the SDK instance.
	Close()
}

// Core is the concrete SDK implementation. It embeds the initialized EVM
// client and runtime configuration.
type Core struct {
	*blockchain.EVMClient
	*config.Config
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
func (c Core) Close() {
	c.EVMClient.Client.Close()
}

// withOpTimeout returns a derived context that is canceled after d, or a cancel
// context if d <= 0. Helper for per-operation timeouts.
func withOpTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, d)
}

// readFileWithTimeout runs read() in a goroutine and returns its result or an
// error if the provided timeout elapses. Useful for IPFS/Filecoin fetches.
func readFileWithTimeout(d time.Duration, read func() ([]byte, error)) ([]byte, error) {
	type res struct {
		b   []byte
		err error
	}
	ch := make(chan res, 1)
	go func() { b, err := read(); ch <- res{b, err} }()
	tctx, cancel := withOpTimeout(context.Background(), d)
	defer cancel()
	select {
	case <-tctx.Done():
		return nil, tctx.Err()
	case r := <-ch:
		return r.b, r.err
	}
}

// NewServiceClient builds a Service client for (orgID, serviceID, groupName).
// It performs the following steps:
//  1. Resolve service/org metadata URIs from on-chain Registry.
//  2. Fetch and parse metadata JSON from IPFS/Lighthouse.
//  3. Fetch and parse the service API sources (.proto, tar/tar.gz).
//  4. Select the requested group and construct a dynamic gRPC client.
//  5. Parse the caller’s private key (if provided) for payment strategies.
//
// The returned Service instance can perform RPC calls and configure payment
// strategies (free, prepaid, escrow).
func (c Core) NewServiceClient(serviceID, orgID, groupName string) (Service, error) {
	c.Config.Timeouts = c.Config.Timeouts.WithDefaults()

	hash := c.EVMClient.GetServiceMetadataHashRegistry(orgID, serviceID) // TODO: use ctx in chain queries
	orgHash := c.EVMClient.GetOrgMetadataUri(orgID)                      // TODO: use ctx in chain queries

	storageClient := storage.NewStorage(c.Config.IpfsURL, c.Config.LighthouseURL)

	rawOrgMetadata, err := readFileWithTimeout(c.Config.Timeouts.ChainRead, func() ([]byte, error) {
		return storageClient.ReadFile(orgHash)
	})
	if err != nil {
		return nil, fmt.Errorf("can't read orgMetadata file: %w", err)
	}

	rawMetadata, err := readFileWithTimeout(c.Config.Timeouts.ChainRead, func() ([]byte, error) {
		return storageClient.ReadFile(hash)
	})
	if err != nil {
		return nil, fmt.Errorf("can't read serviceMetadata file: %w", err)
	}

	var serviceMetadata model.ServiceMetadata
	if err := json.Unmarshal(rawMetadata, &serviceMetadata); err != nil {
		return nil, fmt.Errorf("can't parse serviceMetadata: %w", err)
	}

	var orgMetadata model.OrganizationMetaData
	if err := json.Unmarshal(rawOrgMetadata, &orgMetadata); err != nil {
		return nil, fmt.Errorf("can't parse orgMetadata: %w", err)
	}

	var rawFile []byte

	// Backward compatibility: older metadata may use ModelIpfsHash.
	if serviceMetadata.ModelIpfsHash != "" {
		rawFile, err = readFileWithTimeout(c.Config.Timeouts.ChainRead, func() ([]byte, error) {
			return storageClient.ReadFile(serviceMetadata.ModelIpfsHash)
		})
	}
	if serviceMetadata.ServiceApiSource != "" {
		rawFile, err = readFileWithTimeout(c.Config.Timeouts.ChainRead, func() ([]byte, error) {
			return storageClient.ReadFile(serviceMetadata.ServiceApiSource)
		})
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

	var currentOrgGroup *model.OrganizationGroup
	for _, v := range orgMetadata.Groups {
		if v.GroupName == groupName {
			currentOrgGroup = v
			break
		}
	}

	if currentSrvGroup == nil || currentOrgGroup == nil {
		return nil, fmt.Errorf("can't find group with name %s", groupName)
	}

	serviceMetadata.ProtoFiles, err = storage.ParseProtoFiles(rawFile)
	if err != nil {
		return nil, fmt.Errorf("can't parse proto files: %w", err)
	}

	// TODO: endpoint selection strategy (currently takes the first endpoint).
	if len(currentSrvGroup.Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints found in group %s", groupName)
	}
	gClient := grpc.NewClient(currentSrvGroup.Endpoints[0], serviceMetadata.ProtoFiles)

	address, prvKey, err := blockchain.ParsePrivateKeyECDSA(c.PrivateKey)
	if err != nil {
		zap.L().Warn("service calls disabled: private key parsing failed", zap.Error(err))
	}

	if c.Config.Debug {
		zap.L().Debug("signer address", zap.String("addr", address.Hex()))
	}

	s := &ServiceClient{
		EVMClient:           c.EVMClient,
		ServiceMetadata:     &serviceMetadata,
		OrgMetadata:         &orgMetadata,
		GRPC:                gClient,
		config:              c.Config,
		ServiceID:           serviceID,
		OrgID:               orgID,
		CurrentServiceGroup: currentSrvGroup,
		CurrentOrgGroup:     currentOrgGroup,
		SignerPrivateKey:    prvKey,
	}

	return s, nil
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

	evmClient, err := blockchain.InitEvm(config.Network.ChainID, config.RPCAddr) // TODO: wrap with withOpTimeout(ctx, config.Timeouts.Dial)
	if err != nil {
		zap.L().Error("Init ethereum client failed", zap.Error(err))
		os.Exit(-1)
	}

	return &Core{
		evmClient,
		config,
	}
}
