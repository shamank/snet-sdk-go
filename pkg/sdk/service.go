package sdk

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shamank/snet-sdk-go/pkg/blockchain"
	"github.com/shamank/snet-sdk-go/pkg/config"
	"github.com/shamank/snet-sdk-go/pkg/grpc"
	"github.com/shamank/snet-sdk-go/pkg/model"
	"github.com/shamank/snet-sdk-go/pkg/payment"
	"github.com/shamank/snet-sdk-go/pkg/training"
	"google.golang.org/protobuf/proto"
)

// paymentStrategyFactory constructs payment strategies for the service client.
// Test doubles can implement this interface to intercept strategy creation.
type paymentStrategyFactory interface {
	Paid(ctx context.Context, evm *blockchain.EVMClient, grpcCli *grpc.Client, metadata *model.ServiceMetadata, key *ecdsa.PrivateKey, serviceGroup *model.ServiceGroup, orgGroup *model.OrganizationGroup) (payment.Strategy, error)
	PrePaid(ctx context.Context, evm *blockchain.EVMClient, grpcCli *grpc.Client, mpeAddr common.Address, serviceGroup *model.ServiceGroup, orgGroup *model.OrganizationGroup, privateKey string, count uint64) (payment.Strategy, error)
	Free(evm *blockchain.EVMClient, grpcCli *grpc.Client, orgID, serviceID, groupID string, key *ecdsa.PrivateKey, extend *uint64) (payment.Strategy, error)
}

// defaultStrategyFactory provides the production constructors for payment strategies.
type defaultStrategyFactory struct{}

func (defaultStrategyFactory) Paid(ctx context.Context, evm *blockchain.EVMClient, grpcCli *grpc.Client, metadata *model.ServiceMetadata, key *ecdsa.PrivateKey, serviceGroup *model.ServiceGroup, orgGroup *model.OrganizationGroup) (payment.Strategy, error) {
	return payment.NewPaidStrategy(ctx, evm, grpcCli, metadata, key, serviceGroup, orgGroup)
}

func (defaultStrategyFactory) PrePaid(ctx context.Context, evm *blockchain.EVMClient, grpcCli *grpc.Client, mpeAddr common.Address, serviceGroup *model.ServiceGroup, orgGroup *model.OrganizationGroup, privateKey string, count uint64) (payment.Strategy, error) {
	return payment.NewPrePaidStrategy(ctx, evm, grpcCli, mpeAddr, serviceGroup, orgGroup, privateKey, count)
}

func (defaultStrategyFactory) Free(evm *blockchain.EVMClient, grpcCli *grpc.Client, orgID, serviceID, groupID string, key *ecdsa.PrivateKey, extend *uint64) (payment.Strategy, error) {
	return payment.NewFreeStrategy(evm, grpcCli, orgID, serviceID, groupID, key, extend)
}

// Service defines the high-level client API for invoking service methods and
// managing payment strategies. Implementations wrap a dynamic gRPC client and
// inject the appropriate payment metadata (free/escrow/prepaid) per request.
type Service interface {
	// CallWithMap calls a service method using a JSON-like map for the request
	// body. Parameters are marshaled to JSON and converted to the input protobuf
	// message based on parsed descriptors.
	CallWithMap(method string, params map[string]any) (map[string]any, error)
	// CallWithJSON calls a service method using raw JSON bytes as the request
	// body. The JSON is unmarshaled into the input protobuf message; the result
	// is returned as JSON bytes.
	CallWithJSON(method string, input []byte) ([]byte, error)
	// CallWithProto calls a service method using a concrete protobuf message
	// for the request and returns the protobuf response.
	CallWithProto(method string, input proto.Message) (proto.Message, error)
	// SetPaidPaymentStrategy configures the escrow (MPE) strategy. It ensures
	// there is a usable payment channel and prepares signatures for subsequent
	// calls. Requires a valid signer private key.
	SetPaidPaymentStrategy() error
	// SetPrePaidPaymentStrategy configures the prepaid strategy. It prepares an
	// allowance based on call count and obtains tokens on Refresh. Requires a
	// valid signer private key in the SDK config.
	SetPrePaidPaymentStrategy(count uint64) error

	// SetFreePaymentStrategy configures the free-call strategy and obtains a
	// short-lived free-call token on Refresh. Optional extendBlocks controls
	// token lifetime in blocks (daemon-dependent).
	SetFreePaymentStrategy(extendBlocks ...uint64) error

	// GetFreeCallsAvailable returns the remaining number of free calls for the
	// current user/token.
	GetFreeCallsAvailable() (uint64, error)

	// ProtoFiles returns a proto file manager for this service
	ProtoFiles() grpc.ProtoManager

	// Training returns a training sub-client bound to this service.
	Training() training.Client

	// Organization returns the organization this service belongs to
	Organization() Organization

	// Healthcheck performs a simple health check against the service daemon
	Healthcheck() Healthcheck

	// GetCurrentOrgGroup returns the current organization group
	GetCurrentOrgGroup() *model.OrganizationGroup

	// GetCurrentServiceGroup returns the current service group
	GetCurrentServiceGroup() *model.ServiceGroup

	// GetServiceID returns the service identifier
	GetServiceID() string

	// GetServiceMetadata returns the full service metadata
	GetServiceMetadata() *model.ServiceMetadata

	// UpdateServiceMetadata updates the service metadata (uploads to IPFS and updates blockchain)
	UpdateServiceMetadata(metadata *model.ServiceMetadata) (common.Hash, error)

	// DeleteService deletes the service registration from blockchain
	DeleteService() (common.Hash, error)

	// RawGrpc returns direct access to the gRPC client (advanced usage)
	RawGrpc() *grpc.Client

	getBlockchainClient() *blockchain.ServiceClient

	// Close releases resources (e.g., underlying gRPC connection).
	Close()
}

// ServiceClient is a concrete Service implementation. It holds blockchain
// context (EVM client), parsed metadata (org/service/groups), a dynamic gRPC
// client, and the active payment strategy.
type ServiceClient struct {
	*blockchain.EVMClient
	GRPC                *grpc.Client
	strategy            payment.Strategy
	config              *config.Config
	org                 Organization
	orgClient           *blockchain.OrgClient
	srvClient           *blockchain.ServiceClient
	grpcClient          *grpc.Client
	ServiceID           string
	OrgID               string
	CurrentServiceGroup *model.ServiceGroup
	OrgMetadata         *model.OrganizationMetaData
	ServiceMetadata     *model.ServiceMetadata
	CurrentOrgGroup     *model.OrganizationGroup
	SignerPrivateKey    *ecdsa.PrivateKey
	trainingClient      training.Client
	strategies          paymentStrategyFactory
}

// newServiceClient wires together the runtime-facing ServiceClient wrapper using
// blockchain metadata, gRPC client and signer configuration. It keeps backward
// compatible field population for legacy call sites that rely on struct fields.
func newServiceClient(
	cfg *config.Config,
	org Organization,
	orgBC *blockchain.OrgClient,
	svcBC *blockchain.ServiceClient,
	grpcClient *grpc.Client,
	signer *ecdsa.PrivateKey,
) *ServiceClient {
	sc := &ServiceClient{
		EVMClient:           nil,
		GRPC:                grpcClient,
		strategy:            nil,
		config:              cfg,
		org:                 org,
		orgClient:           orgBC,
		srvClient:           svcBC,
		grpcClient:          grpcClient,
		ServiceID:           "",
		OrgID:               "",
		CurrentServiceGroup: nil,
		OrgMetadata:         nil,
		ServiceMetadata:     nil,
		CurrentOrgGroup:     nil,
		SignerPrivateKey:    signer,
		trainingClient:      nil,
	}

	if svcBC != nil {
		sc.EVMClient = svcBC.EVMClient
		sc.ServiceID = svcBC.ServiceID
		sc.ServiceMetadata = svcBC.ServiceMetadata
		sc.CurrentServiceGroup = svcBC.CurrentGroup
	}

	if orgBC != nil {
		sc.orgClient = orgBC
		sc.CurrentOrgGroup = orgBC.CurrentOrgGroup
		sc.OrgMetadata = orgBC.OrganizationMetaData
		if orgBC.OrganizationMetaData != nil {
			sc.OrgID = orgBC.OrganizationMetaData.OrgID
		}
	}

	return sc
}

// strategyFactory returns the strategy factory to use, defaulting to production constructors.
func (s *ServiceClient) strategyFactory() paymentStrategyFactory {
	if s.strategies == nil {
		s.strategies = defaultStrategyFactory{}
	}
	return s.strategies
}

// RawGrpc returns direct access to the gRPC client (advanced usage)
func (s *ServiceClient) RawGrpc() *grpc.Client {
	return s.GRPC
}

func (s *ServiceClient) getBlockchainClient() *blockchain.ServiceClient {
	return s.srvClient
}

// NewServiceClient builds a Service client from organization and blockchain service.
// It creates a gRPC connection to the service endpoint and prepares for RPC calls.
func (c *Core) NewServiceClient(orgID, serviceID, groupName string) (Service, error) {

	orgClient, err := c.NewOrganizationClient(orgID, groupName)
	if err != nil {
		return nil, err
	}

	serviceClient, err := orgClient.ServiceClient(serviceID, groupName)
	if err != nil {
		return nil, err
	}

	if serviceClient == nil {
		return nil, fmt.Errorf("blockchain service client is required")
	}

	if len(serviceClient.GetCurrentServiceGroup().Endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints available for service group %s",
			serviceClient.GetCurrentServiceGroup().GroupName)
	}

	// Create gRPC client to the first endpoint
	// TODO: endpoint selection strategy (currently takes the first endpoint)
	endpoint := serviceClient.GetCurrentServiceGroup().Endpoints[0]
	grpcClient := grpc.NewClient(endpoint, serviceClient.ProtoFiles().Get())

	return newServiceClient(
		c.Config,
		orgClient,
		orgClient.getBlockchainClient(),
		serviceClient.getBlockchainClient(),
		grpcClient,
		c.prvKey,
	), nil
}

// Organization returns the organization this service belongs to
func (s *ServiceClient) Organization() Organization {
	return s.org
}

// GetServiceID returns the service identifier
func (s *ServiceClient) GetServiceID() string {
	return s.srvClient.ServiceID
}

// GetServiceMetadata returns the full service metadata
func (s *ServiceClient) GetServiceMetadata() *model.ServiceMetadata {
	return s.srvClient.ServiceMetadata
}

// GetCurrentOrgGroup returns the current organization group
func (s *ServiceClient) GetCurrentOrgGroup() *model.OrganizationGroup {
	return s.orgClient.CurrentOrgGroup
}

// GetCurrentServiceGroup returns the current service group
func (s *ServiceClient) GetCurrentServiceGroup() *model.ServiceGroup {
	return s.srvClient.CurrentGroup
}

// ProtoFiles returns a proto file manager for this service
func (s *ServiceClient) ProtoFiles() grpc.ProtoManager {
	meta := s.ServiceMetadata
	if s.srvClient != nil && s.srvClient.ServiceMetadata != nil {
		meta = s.srvClient.ServiceMetadata
	}
	if meta == nil {
		meta = &model.ServiceMetadata{}
	}
	return grpc.NewProtoManager(meta)
}

// Healthcheck returns a healthcheck client for this service
func (s *ServiceClient) Healthcheck() Healthcheck {
	group := s.CurrentServiceGroup
	if s.srvClient != nil && s.srvClient.CurrentGroup != nil {
		group = s.srvClient.CurrentGroup
	}
	return newHealthcheckClient(
		s.grpcClient,
		group,
		s.config,
	)
}

// Training returns (and lazily initializes) a training client bound to
// this service, using the same signer and block-number provider.
func (s *ServiceClient) Training() training.Client {
	if s.trainingClient == nil {
		blockNumber := func() (*big.Int, error) {
			return nil, errors.New("block number provider not configured")
		}
		if s.srvClient != nil && s.srvClient.EVMClient != nil {
			blockNumber = s.srvClient.EVMClient.GetCurrentBlockNumber
		} else if s.EVMClient != nil {
			blockNumber = s.EVMClient.GetCurrentBlockNumber
		}

		var priv *ecdsa.PrivateKey
		if s.config != nil {
			priv = s.config.GetPrivateKey()
		}

		s.trainingClient = training.NewTrainingClient(
			s.OrgID,
			s.ServiceID,
			s.CurrentServiceGroup.GroupName,
			s.grpcClient,
			priv,
			s.config.Timeouts.GRPCUnary,
			s.config.Timeouts.GRPCStream,
			blockNumber,
			s.strategy,
		)
	}
	return s.trainingClient
}

// SetPaidPaymentStrategy initializes the escrow (MPE) payment strategy and
// ensures a valid channel (funds/expiration). It does not perform a Refresh
// because escrow calls sign per-request.
func (s *ServiceClient) SetPaidPaymentStrategy() error {
	if err := s.validateWebSocketRPC(); err != nil {
		return err
	}

	ctx, cancel := s.withTimeout(context.Background(), s.ensureTimeout())
	defer cancel()

	strategy, err := s.strategyFactory().Paid(
		ctx,
		s.EVMClient,
		s.GRPC,
		s.ServiceMetadata,
		s.config.GetPrivateKey(),
		s.CurrentServiceGroup,
		s.CurrentOrgGroup,
	)
	if err != nil {
		return fmt.Errorf("failed to create paid strategy: %w", err)
	}

	s.strategy = strategy
	return nil
}

// SetPrePaidPaymentStrategy initializes the prepaid strategy and immediately
// refreshes the daemon-issued token. The count parameter indicates the number
// of calls to provision in the initial signed allowance.
func (s *ServiceClient) SetPrePaidPaymentStrategy(count uint64) error {
	if err := s.validateWebSocketRPC(); err != nil {
		return err
	}

	ctx, cancel := s.withTimeout(context.Background(), s.ensureTimeout())
	defer cancel()

	strategy, err := s.strategyFactory().PrePaid(ctx, s.EVMClient, s.GRPC, s.ServiceMetadata.GetMpeAddr(), s.CurrentServiceGroup, s.CurrentOrgGroup, s.config.PrivateKey, count)
	if err != nil {
		return fmt.Errorf("failed to create prepaid strategy: %w", err)
	}

	s.strategy = strategy

	if err := s.strategy.Refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh prepaid strategy: %w", err)
	}

	return nil
}

// SetFreePaymentStrategy initializes the free-call strategy and fetches a
// short-lived token. If extendBlocks is provided, it is forwarded to request a
// custom token lifetime (daemon may ignore or cap it).
func (s *ServiceClient) SetFreePaymentStrategy(extendBlocks ...uint64) error {
	strategy, err := s.strategyFactory().Free(s.EVMClient, s.GRPC, s.OrgID, s.ServiceID, s.CurrentOrgGroup.ID, s.SignerPrivateKey, optionalUint64(extendBlocks...))
	if err != nil {
		return err
	}
	s.strategy = strategy
	ctx, cancel := s.withTimeout(context.Background(), s.config.Timeouts.StrategyRefresh)
	defer cancel()
	return strategy.Refresh(ctx)
}

// GetFreeCallsAvailable returns the number of remaining free calls for the
// current user/token. It requires the active strategy to be FreeStrategy.
func (s *ServiceClient) GetFreeCallsAvailable() (uint64, error) {
	freeStrat, ok := s.strategy.(*payment.FreeStrategy)
	if !ok {
		return 0, errors.New("current strategy is not FreeStrategy")
	}

	ctx, cancel := s.withTimeout(context.Background(), s.config.Timeouts.StrategyRefresh)
	defer cancel()

	available, err := freeStrat.GetFreeCallsAvailable(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get free calls available: %w", err)
	}

	return available, nil
}

// CallWithMap invokes a method with a map-based request. Payment metadata is
// injected by the current strategy into the outgoing context.
func (s *ServiceClient) CallWithMap(method string, params map[string]any) (map[string]any, error) {
	err := s.setDefaultStrategy()
	if err != nil {
		return nil, fmt.Errorf("can't auto set payment strategy; call SetPaidPaymentStrategy, SetPrePaidPaymentStrategy, or SetFreePaymentStrategy manually %v", err)
	}
	//if s.strategy == nil {
	//	return nil, errors.New("payment strategy not set; call SetPaidPaymentStrategy, SetPrePaidPaymentStrategy, or SetFreePaymentStrategy first")
	//}

	ctx, cancel := s.withTimeout(context.Background(), s.config.Timeouts.GRPCUnary)
	defer cancel()

	resp, err := s.grpcClient.CallWithMap(s.strategy.GRPCMetadata(ctx), method, params)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	return resp, nil
}

// CallWithJSON invokes a method with raw JSON request bytes. The JSON is mapped
// to the protobuf input type using service descriptors.
func (s *ServiceClient) CallWithJSON(method string, input []byte) ([]byte, error) {
	err := s.setDefaultStrategy()
	if err != nil {
		return nil, fmt.Errorf("can't auto set payment strategy; call SetPaidPaymentStrategy, SetPrePaidPaymentStrategy, or SetFreePaymentStrategy manually %v", err)
	}
	//if s.strategy == nil {
	//	return nil, errors.New("payment strategy not set; call SetPaidPaymentStrategy, SetPrePaidPaymentStrategy, or SetFreePaymentStrategy first")
	//}

	ctx, cancel := s.withTimeout(context.Background(), s.config.Timeouts.GRPCUnary)
	defer cancel()

	resp, err := s.grpcClient.CallWithJSON(s.strategy.GRPCMetadata(ctx), method, input)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	return resp, nil
}

func (s *ServiceClient) setDefaultStrategy() error {
	if s.strategy != nil {
		return nil
	}
	if s.CurrentServiceGroup.FreeCalls > 0 {
		if s.SetFreePaymentStrategy() == nil {
			return nil
		}
	}
	return s.SetPaidPaymentStrategy()
}

// CallWithProto invokes a method with a concrete protobuf request message and
// returns the protobuf response message.
func (s *ServiceClient) CallWithProto(method string, input proto.Message) (proto.Message, error) {
	err := s.setDefaultStrategy()
	if err != nil {
		return nil, fmt.Errorf("can't auto set payment strategy; call SetPaidPaymentStrategy, SetPrePaidPaymentStrategy, or SetFreePaymentStrategy manually %v", err)
	}
	//if s.strategy == nil {
	//	return nil, errors.New("payment strategy not set; call SetPaidPaymentStrategy, SetPrePaidPaymentStrategy, or SetFreePaymentStrategy first")
	//}

	ctx, cancel := s.withTimeout(context.Background(), s.config.Timeouts.GRPCUnary)
	defer cancel()

	resp, err := s.grpcClient.CallWithProto(s.strategy.GRPCMetadata(ctx), method, input)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	return resp, nil
}

// UpdateServiceMetadata updates the service metadata (uploads to IPFS and updates blockchain)
func (s *ServiceClient) UpdateServiceMetadata(metadata *model.ServiceMetadata) (common.Hash, error) {
	pk := s.config.GetPrivateKey()
	if pk == nil {
		return common.Hash{}, fmt.Errorf("private key not configured")
	}

	bcClient := s.getBlockchainClient()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Upload metadata to IPFS
	uri, err := bcClient.Storage.UploadJSON(ctx, metadata)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to upload metadata to IPFS: %w", err)
	}

	// Update service metadata in blockchain
	hash, err := bcClient.UpdateServiceMetadata(pk, []byte(uri))
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to update service metadata: %w", err)
	}

	return hash, nil
}

// DeleteService deletes the service registration from blockchain
func (s *ServiceClient) DeleteService() (common.Hash, error) {
	pk := s.config.GetPrivateKey()
	if pk == nil {
		return common.Hash{}, fmt.Errorf("private key not configured")
	}

	bcClient := s.getBlockchainClient()

	hash, err := bcClient.DeleteServiceWithAuth(pk)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to delete service: %w", err)
	}

	return hash, nil
}

// Close releases the underlying gRPC connection. It is safe to call multiple times.
func (s *ServiceClient) Close() {
	if s.grpcClient != nil {
		_ = s.grpcClient.Close()
	}
	if s.EVMClient != nil {
		s.EVMClient.Close()
	}
}

// withTimeout returns a derived context with the given timeout. A cancelable
// context is returned when d <= 0.
func (s *ServiceClient) withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, d)
}

// ensureTimeout selects a sensible timeout for strategy operations, preferring
// PaymentEnsure, then StrategyRefresh, and finally a 1-minute default.
func (s *ServiceClient) ensureTimeout() time.Duration {
	if s.config != nil && s.config.Timeouts.PaymentEnsure > 0 {
		return s.config.Timeouts.PaymentEnsure
	}
	if s.config != nil && s.config.Timeouts.StrategyRefresh > 0 {
		return s.config.Timeouts.StrategyRefresh
	}
	return time.Minute
}

// validateWebSocketRPC checks that the RPCAddr uses WebSocket protocol (wss:// or ws://).
// This validation is required for paid and prepaid strategies because they need
// to subscribe to blockchain events.
func (s *ServiceClient) validateWebSocketRPC() error {
	if s.config == nil || s.config.RPCAddr == "" {
		return fmt.Errorf("RPC address is required")
	}

	if !strings.HasPrefix(s.config.RPCAddr, "wss://") && !strings.HasPrefix(s.config.RPCAddr, "ws://") {
		return fmt.Errorf("RPC address must use WebSocket protocol (wss:// or ws://). HTTP endpoints are not supported for paid/prepaid strategies")
	}

	return nil
}

// optionalUint64 returns a pointer to the first value if provided, or nil.
// Useful for optional parameters in strategy constructors.
func optionalUint64(v ...uint64) *uint64 {
	if len(v) > 0 {
		return &v[0]
	}
	return nil
}
