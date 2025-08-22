package payment

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-sdk-go/pkg/blockchain"
	"github.com/singnet/snet-sdk-go/pkg/grpc"
	"github.com/singnet/snet-sdk-go/pkg/model"
	"google.golang.org/grpc/metadata"
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
) (Strategy, error) {

	if ctx == nil {
		ctx = context.TODO()
	}

	mpeAddress := common.HexToAddress(serviceMetadata.MPEAddress)
	priceInCogs := serviceGroup.Pricing[0].PriceInCogs

	groupID, err := blockchain.DecodePaymentGroupID(orgGroup.ID)
	if err != nil {
		return nil, err
	}

	recipient := common.HexToAddress(orgGroup.PaymentDetails.PaymentAddress)
	fromAddress := blockchain.GetAddressFromPrivateKeyECDSA(privateKeyECDSA)

	currentBlockNumber, err := evm.GetCurrentBlockNumberCtx(ctx)
	if err != nil {
		return nil, err
	}

	chainID, err := evm.Client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	transactOpts, err := blockchain.GetTransactOpts(chainID, privateKeyECDSA)
	if err != nil {
		return nil, err
	}
	opts := &blockchain.BindOpts{
		Call:     blockchain.GetCallOpts(*fromAddress, currentBlockNumber, ctx),
		Transact: transactOpts,
		Watch:    blockchain.GetWatchOpts(currentBlockNumber, ctx),
		Filter:   blockchain.GetFilterOpts(currentBlockNumber, ctx),
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

	filteredChannel, err := evm.FilterChannels(senders, recipients, groupIDs, opts.Filter)
	if err != nil {
		return nil, err
	}

	newExpiration := blockchain.GetNewExpiration(currentBlockNumber, orgGroup.PaymentDetails.PaymentExpirationThreshold)

	currentSignedAmount := big.NewInt(0)
	currentNonce := big.NewInt(0)

	if filteredChannel != nil {
		state, err := GetChannelStateFromDaemon(grpcCli.GRPC, ctx, mpeAddress, filteredChannel.ChannelId, currentBlockNumber, privateKeyECDSA)
		if err != nil {
			return nil, err
		}
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

	channelID, err := evm.EnsurePaymentChannel(mpeAddress, filteredChannel, currentSignedAmount, priceInCogs, newExpiration, opts, chans, senders, recipients, groupIDs)
	if err != nil {
		return nil, err
	}

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
