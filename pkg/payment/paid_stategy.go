package payment

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-sdk-go/pkg/blockchain"
	"github.com/singnet/snet-sdk-go/pkg/grpc"
	"github.com/singnet/snet-sdk-go/pkg/model"
	grpcconn "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// PaidStrategy implements the "escrow" payment flow backed by the
// MultiPartyEscrow (MPE) contract. It maintains the channel ID, current
// nonce, and a running signed amount to authorize server withdrawals.
type PaidStrategy struct {
	evmClient       *blockchain.EVMClient
	grpcClient      *grpc.Client
	serviceMetadata *model.ServiceMetadata
	channelID       *big.Int
	nonce           *big.Int
	signedAmount    *big.Int
	privateKeyECDSA *ecdsa.PrivateKey
}

// ChainOperations captures blockchain interactions required by the paid
// strategy. Different implementations can be injected for testing.
type ChainOperations interface {
	CurrentBlock(ctx context.Context) (*big.Int, error)
	NetworkID(ctx context.Context) (*big.Int, error)
	BuildBindOpts(from common.Address, currentBlockNumber, chainID *big.Int, key *ecdsa.PrivateKey, ctx context.Context) (*blockchain.BindOpts, error)
	FilterChannels(senders, recipients []common.Address, groupIDs [][32]byte, opts *blockchain.BindOpts) (*blockchain.MultiPartyEscrowChannelOpen, error)
	EnsurePaymentChannel(mpe common.Address, filtered *blockchain.MultiPartyEscrowChannelOpen, currentSigned, price, desiredExpiration *big.Int, opts *blockchain.BindOpts, chans *blockchain.ChansToWatch, senders, recipients []common.Address, groupIDs [][32]byte) (*big.Int, error)
}

// ChannelStateClient reads the current daemon channel state.
type ChannelStateClient interface {
	ChannelState(conn *grpcconn.ClientConn, ctx context.Context, mpe common.Address, channelID, currentBlock *big.Int, key *ecdsa.PrivateKey) (*ChannelStateReply, error)
}

// PaidStrategyDependencies groups optional overrides for blockchain and daemon access.
// Used for dependency injection in tests to isolate PaidStrategy logic.
type PaidStrategyDependencies struct {
	Chain        ChainOperations
	ChannelState ChannelStateClient
}

// PaidStrategyOption configures construction of a PaidStrategy using the functional options pattern.
type PaidStrategyOption func(*paidStrategyConfig)

// paidStrategyConfig holds the resolved configuration for PaidStrategy creation.
type paidStrategyConfig struct {
	chain        ChainOperations    // Blockchain operations implementation
	channelState ChannelStateClient // Channel state client implementation
}

// WithPaidStrategyDependencies overrides default dependencies used by NewPaidStrategy.
// This option is useful for testing to inject mocks or custom implementations.
//
// Example:
//
//	deps := PaidStrategyDependencies{
//		Chain: mockChainOps,
//		ChannelState: mockChannelState,
//	}
//	strategy, err := NewPaidStrategy(ctx, evm, grpc, meta, key, srvGroup, orgGroup,
//		WithPaidStrategyDependencies(deps))
func WithPaidStrategyDependencies(deps PaidStrategyDependencies) PaidStrategyOption {
	return func(cfg *paidStrategyConfig) {
		if deps.Chain != nil {
			cfg.chain = deps.Chain
		}
		if deps.ChannelState != nil {
			cfg.channelState = deps.ChannelState
		}
	}
}

// newPaidStrategyConfig creates a configuration with default dependencies.
// It applies the provided options to customize the configuration.
//
// Parameters:
//   - evm: EVM client used for default blockchain operations
//   - opts: optional configuration overrides
//
// Returns a fully configured paidStrategyConfig.
func newPaidStrategyConfig(evm *blockchain.EVMClient, opts []PaidStrategyOption) paidStrategyConfig {
	cfg := paidStrategyConfig{
		chain:        defaultChainOperations{evm: evm},
		channelState: defaultChannelStateClient{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// defaultChainOperations implements ChainOperations using a real EVMClient.
// This is the production implementation used when no custom dependencies are provided.
type defaultChainOperations struct {
	evm *blockchain.EVMClient
}

func (d defaultChainOperations) CurrentBlock(ctx context.Context) (*big.Int, error) {
	return d.evm.GetCurrentBlockNumberCtx(ctx)
}

func (d defaultChainOperations) NetworkID(ctx context.Context) (*big.Int, error) {
	return d.evm.Client.NetworkID(ctx)
}

func (d defaultChainOperations) BuildBindOpts(from common.Address, currentBlockNumber, chainID *big.Int, key *ecdsa.PrivateKey, ctx context.Context) (*blockchain.BindOpts, error) {
	transactOpts, err := blockchain.GetTransactOpts(chainID, key)
	if err != nil {
		return nil, err
	}
	return &blockchain.BindOpts{
		Call:     blockchain.GetCallOpts(from, currentBlockNumber, ctx),
		Transact: transactOpts,
		Watch:    blockchain.GetWatchOpts(currentBlockNumber, ctx),
		Filter:   blockchain.GetFilterOpts(currentBlockNumber, ctx),
	}, nil
}

func (d defaultChainOperations) FilterChannels(senders, recipients []common.Address, groupIDs [][32]byte, opts *blockchain.BindOpts) (*blockchain.MultiPartyEscrowChannelOpen, error) {
	return d.evm.FilterChannels(senders, recipients, groupIDs, opts.Filter)
}

func (d defaultChainOperations) EnsurePaymentChannel(mpe common.Address, filtered *blockchain.MultiPartyEscrowChannelOpen, currentSigned, price, desiredExpiration *big.Int, opts *blockchain.BindOpts, chans *blockchain.ChansToWatch, senders, recipients []common.Address, groupIDs [][32]byte) (*big.Int, error) {
	return d.evm.EnsurePaymentChannel(mpe, filtered, currentSigned, price, desiredExpiration, opts, chans, senders, recipients, groupIDs)
}

// defaultChannelStateClient implements ChannelStateClient using the real gRPC daemon client.
// This is the production implementation used when no custom dependencies are provided.
type defaultChannelStateClient struct{}

func (defaultChannelStateClient) ChannelState(conn *grpcconn.ClientConn, ctx context.Context, mpe common.Address, channelID, currentBlock *big.Int, key *ecdsa.PrivateKey) (*ChannelStateReply, error) {
	return GetChannelStateFromDaemon(conn, ctx, mpe, channelID, currentBlock, key)
}

// NewPaidStrategy builds a PaidStrategy, ensuring there is a usable payment
// channel (sufficient funds and expiration) for the given org/service group.
// Steps performed:
//
//  1. Resolve payment group ID and recipient address from metadata.
//  2. Read current block, chain ID and prepare bind opts.
//  3. Look up an existing channel (sender, recipient, groupID).
//  4. If found, query the daemon for current nonce/signed amount.
//  5. Ensure a valid channel (open/extend/add-funds as needed).
//  6. Initialize signedAmount = currentSigned + single-call price.
//
// Parameters:
//   - ctx: context for on-chain lookups and tx submission.
//   - evm: initialized EVM client.
//   - grpcCli: connected daemon gRPC client.
//   - serviceMetadata: service metadata (for MPE address).
//   - privateKeyECDSA: caller's signing key.
//   - serviceGroup: selected service group (for price & endpoints).
//   - orgGroup: selected org group (for payment settings).
//
// Returns:
//   - Strategy: the configured PaidStrategy.
//   - error: if any chain/daemon operation fails.
func NewPaidStrategy(
	ctx context.Context,
	evm *blockchain.EVMClient,
	grpcCli *grpc.Client,
	serviceMetadata *model.ServiceMetadata,
	privateKeyECDSA *ecdsa.PrivateKey,
	serviceGroup *model.ServiceGroup,
	orgGroup *model.OrganizationGroup,
	opts ...PaidStrategyOption,
) (Strategy, error) {

	if ctx == nil {
		ctx = context.TODO()
	}

	cfg := newPaidStrategyConfig(evm, opts)

	mpeAddress := common.HexToAddress(serviceMetadata.MPEAddress)
	priceInCogs := serviceGroup.Pricing[0].PriceInCogs

	groupID, err := blockchain.DecodePaymentGroupID(orgGroup.ID)
	if err != nil {
		return nil, err
	}

	recipient := common.HexToAddress(orgGroup.PaymentDetails.PaymentAddress)
	fromAddress := blockchain.GetAddressFromPrivateKeyECDSA(privateKeyECDSA)

	currentBlockNumber, err := cfg.chain.CurrentBlock(ctx)
	if err != nil {
		return nil, err
	}

	chainID, err := cfg.chain.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	bindOpts, err := cfg.chain.BuildBindOpts(*fromAddress, currentBlockNumber, chainID, privateKeyECDSA, ctx)
	if err != nil {
		return nil, err
	}

	chans := &blockchain.ChansToWatch{
		ChannelOpens:    make(chan *blockchain.MultiPartyEscrowChannelOpen),
		ChannelExtends:  make(chan *blockchain.MultiPartyEscrowChannelExtend),
		ChannelAddFunds: make(chan *blockchain.MultiPartyEscrowChannelAddFunds),
		DepositFunds:    make(chan *blockchain.MultiPartyEscrowDepositFunds),
		Err:             make(chan error),
	}

	senders := []common.Address{*fromAddress}
	recipients := []common.Address{recipient}
	groupIDs := [][32]byte{groupID}

	filteredChannel, err := cfg.chain.FilterChannels(senders, recipients, groupIDs, bindOpts)
	if err != nil {
		return nil, err
	}

	newExpiration := blockchain.GetNewExpiration(currentBlockNumber, orgGroup.PaymentDetails.PaymentExpirationThreshold)

	currentSignedAmount := big.NewInt(0)
	currentNonce := big.NewInt(0)

	if filteredChannel != nil {
		state, err := cfg.channelState.ChannelState(grpcCli.GRPC, ctx, mpeAddress, filteredChannel.ChannelId, currentBlockNumber, privateKeyECDSA)
		if err != nil {
			st, ok := status.FromError(err)
			switch {
			case ok && (st.Code() == codes.NotFound || st.Code() == codes.Unknown) && strings.Contains(st.Message(), "channel is not found"):
				log.Printf("paid strategy: channel %s not found in daemon, will create a new one", filteredChannel.ChannelId.String())
				filteredChannel = nil
			case strings.Contains(err.Error(), "channel is not found"):
				log.Printf("paid strategy: channel %s not found in daemon, will create a new one", filteredChannel.ChannelId.String())
				filteredChannel = nil
			default:
				log.Println("daemon err")
				return nil, err
			}
		}
		if filteredChannel != nil && state != nil {
			if b := state.GetCurrentSignedAmount(); len(b) > 0 {
				currentSignedAmount = new(big.Int).SetBytes(b)
			}
			if b := state.GetCurrentNonce(); len(b) > 0 {
				currentNonce = new(big.Int).SetBytes(b)
				if currentNonce == nil {
					return nil, errors.New("error while getting current nonce")
				}
			}
		}
	}

	channelID, err := cfg.chain.EnsurePaymentChannel(mpeAddress, filteredChannel, currentSignedAmount, priceInCogs, newExpiration, bindOpts, chans, senders, recipients, groupIDs)
	if err != nil {
		return nil, err
	}

	log.Println("channel id EnsurePaymentChannel", channelID)

	signedAmount := new(big.Int).Add(currentSignedAmount, priceInCogs)

	return &PaidStrategy{
		evmClient:       evm,
		grpcClient:      grpcCli,
		serviceMetadata: serviceMetadata,
		privateKeyECDSA: privateKeyECDSA,
		signedAmount:    signedAmount,
		channelID:       channelID,
		nonce:           currentNonce,
	}, nil
}

// Refresh updates internal state if needed before making subsequent calls.
// TODO: fetch latest nonce/signedAmount from daemon to reflect concurrent usage.
func (p *PaidStrategy) Refresh(ctx context.Context) error {
	return nil
}

// GRPCMetadata returns a child context carrying escrow payment headers required
// by the daemon (channel ID, nonce, total signed amount, and the claim signature).
func (p *PaidStrategy) GRPCMetadata(ctx context.Context) context.Context {
	md := metadata.Pairs(
		PaymentTypeHeader, "escrow",
		PaymentChannelIDHeader, p.channelID.String(),
		PaymentChannelNonceHeader, p.nonce.String(),
		PaymentChannelAmountHeader, p.signedAmount.String(),
		PaymentChannelSignatureHeader, string(p.signMessage()),
	)
	return metadata.NewOutgoingContext(ctx, md)
}

// signMessage builds and signs the MPE claim message used to authorize server
// withdrawal up to p.signedAmount on (channelID, nonce). The message layout is:
//
//	concat(PrefixInSignature, MPEAddress, ChannelID, Nonce, SignedAmount)
//
// The resulting hash is signed using an Ethereum personal-sign style signature.
func (p *PaidStrategy) signMessage() []byte {
	message := bytes.Join([][]byte{
		[]byte(PrefixInSignature),
		common.HexToAddress(p.serviceMetadata.MPEAddress).Bytes(),
		bigIntToBytes(p.channelID),
		bigIntToBytes(p.nonce),
		bigIntToBytes(p.signedAmount),
	}, nil)
	return blockchain.GetSignature(message, p.privateKeyECDSA)
}

// NextSignedAmount increments the locally tracked signed amount by price and
// returns the new total. Use this after a successful call priced at 'price'.
func (p *PaidStrategy) NextSignedAmount(price *big.Int) *big.Int {
	if p.signedAmount == nil {
		p.signedAmount = big.NewInt(0)
	}
	p.signedAmount = new(big.Int).Add(p.signedAmount, price)
	return p.signedAmount
}
