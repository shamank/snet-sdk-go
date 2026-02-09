package payment

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/shamank/snet-sdk-go/pkg/blockchain"
	"github.com/shamank/snet-sdk-go/pkg/grpc"
	"github.com/shamank/snet-sdk-go/pkg/model"
	"google.golang.org/grpc/metadata"
)

// PrepaidStrategy implements the "prepaid-call" flow. The client signs a claim
// for (channelID, nonce, signedAmount) and obtains a short-lived token from the
// daemon. Subsequent RPCs present this token (plus channel identifiers) as
// metadata; the daemon validates freshness (by block number) and the claim
// signature before serving the request.
type PrepaidStrategy struct {
	// Token is the opaque auth token returned by the daemon.
	Token string
	// tokenClient is a gRPC client for the token issuance service.
	tokenClient TokenServiceClient
	// evmClient is the on-chain client used to read the current block number, etc.
	evmClient *blockchain.EVMClient
	// grpcClient is the underlying dynamic gRPC client to the daemon.
	grpcClient *grpc.Client
	// mpeAddr is the MPE contract address.
	mpeAddr common.Address
	// channelID is the active payment channel identifier.
	channelID *big.Int
	// nonce is the channel nonce at which the signed amount applies.
	nonce *big.Int
	// signedAmount is the total authorized amount (in cogs) for the current claim.
	signedAmount *big.Int
	// privateKeyECDSA is the caller’s signing key.
	privateKeyECDSA *ecdsa.PrivateKey
}

// getSignature signs the provided MPE claim signature together with the current
// block number, producing a freshness-bound signature required by the daemon.
// The block number is encoded as 32-byte big-endian (math.U256Bytes).
func (p *PrepaidStrategy) getSignature(mpeSignature []byte, currentBlockNumber *big.Int) []byte {
	// always 32 bytes big‑endian
	blockBytes := math.U256Bytes(currentBlockNumber)
	message := bytes.Join([][]byte{mpeSignature, blockBytes}, nil)
	return blockchain.GetSignature(message, p.privateKeyECDSA)
}

// getClaimSignature builds and signs the canonical MPE claim message:
// concat(PrefixInSignature, MPEAddress, ChannelID, Nonce, SignedAmount).
// The resulting signature is later wrapped with the current block (see getSignature).
func (p *PrepaidStrategy) getClaimSignature() []byte {
	message := bytes.Join([][]byte{
		[]byte(PrefixInSignature),
		p.mpeAddr.Bytes(),
		bigIntToBytes(p.channelID),
		bigIntToBytes(p.nonce),
		bigIntToBytes(p.signedAmount),
	}, nil)
	return blockchain.GetSignature(message, p.privateKeyECDSA)
}

// GRPCMetadata returns a child context augmented with prepaid-call headers:
// payment type, channel ID, nonce, and the daemon-issued auth token.
// The token MUST be kept up-to-date by calling Refresh.
func (p *PrepaidStrategy) GRPCMetadata(ctx context.Context) context.Context {
	md := metadata.Pairs(
		PaymentTypeHeader, "prepaid-call",
		PaymentChannelIDHeader, p.channelID.String(),
		PaymentChannelNonceHeader, p.nonce.String(),
		PrePaidAuthTokenHeader, p.Token,
	)
	return metadata.NewOutgoingContext(ctx, md)
}

// Refresh obtains or renews the prepaid auth token from the daemon. It signs the
// MPE claim (channelID, nonce, signedAmount) and then signs again with the
// current block number to prove freshness. On success, the token is stored in
// p.Token.
func (p *PrepaidStrategy) Refresh(ctx context.Context) error {
	currentBlockNumber, err := p.evmClient.GetCurrentBlockNumberCtx(ctx)
	if err != nil {
		return err
	}

	claimSignature := p.getClaimSignature()
	signature := p.getSignature(claimSignature, currentBlockNumber)
	request := TokenRequest{
		ChannelId:      p.channelID.Uint64(),
		CurrentNonce:   p.nonce.Uint64(),
		SignedAmount:   p.signedAmount.Uint64(), // usedAmount + (priceInCogs * callCount)
		Signature:      signature,
		CurrentBlock:   currentBlockNumber.Uint64(),
		ClaimSignature: claimSignature,
	}

	tokenReply, err := p.tokenClient.GetToken(ctx, &request)
	if err != nil {
		return err
	}
	p.Token = tokenReply.GetToken()
	return nil
}

// NewPrePaidStrategy constructs a PrepaidStrategy for an existing MPE channel,
// ensuring the channel has sufficient funds/expiration and preparing the initial
// signed amount used to request a token.
//
// Flow:
//  1. Resolve groupID/recipient; parse signer private key.
//  2. Read chain tip/chainID; build bind opts (call/watch/filter/transact).
//  3. Locate the sender’s channel for (recipient, groupID).
//  4. Query daemon for current (nonce, signedAmount).
//  5. Ensure/extend/add-funds on the channel as needed.
//  6. Compute signedAmount = currentSigned + priceInCogs * callCount.
//  7. Create strategy with token client; caller should invoke Refresh(ctx)
//     before issuing RPC calls to obtain the token.
//
// Note: ctx is used for on-chain and daemon calls. Caller should provide
// a context with appropriate timeout (e.g., 30-60 seconds for daemon calls).
func NewPrePaidStrategy(ctx context.Context, evm *blockchain.EVMClient, grpc *grpc.Client, mpeAddress common.Address, srvGroup *model.ServiceGroup, orgGroup *model.OrganizationGroup, privateKey string, callCount uint64) (Strategy, error) {
	priceInCogs := srvGroup.Pricing[0].PriceInCogs

	groupID, err := blockchain.DecodePaymentGroupID(orgGroup.ID)
	if err != nil {
		return nil, err
	}

	recipient := common.HexToAddress(orgGroup.PaymentDetails.PaymentAddress)

	fromAddress, privateKeyECDSA, err := blockchain.ParsePrivateKeyECDSA(privateKey)
	if err != nil {
		return nil, err
	}

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
		Call:     blockchain.GetCallOpts(fromAddress, currentBlockNumber, ctx),
		Transact: transactOpts,
		Watch:    blockchain.GetWatchOpts(currentBlockNumber, ctx),
		Filter:   blockchain.GetFilterOpts(currentBlockNumber, ctx),
	}

	chans := &blockchain.ChansToWatch{ // TODO discuss to optimize this
		ChannelOpens:    make(chan *blockchain.MultiPartyEscrowChannelOpen),
		ChannelExtends:  make(chan *blockchain.MultiPartyEscrowChannelExtend),
		ChannelAddFunds: make(chan *blockchain.MultiPartyEscrowChannelAddFunds),
		DepositFunds:    make(chan *blockchain.MultiPartyEscrowDepositFunds),
		Err:             make(chan error),
	}

	senders := []common.Address{fromAddress}
	recipients := []common.Address{recipient}
	groupIDs := [][32]byte{groupID}

	filteredChannel, err := evm.FilterChannels(senders, recipients, groupIDs, opts.Filter)
	if err != nil {
		return nil, err
	}

	filteredChannelState, err := GetChannelStateFromDaemon(grpc.GRPC, ctx, mpeAddress, filteredChannel.ChannelId, currentBlockNumber, privateKeyECDSA)
	if err != nil {
		return nil, err
	}

	currentSignedAmount := new(big.Int).SetBytes(filteredChannelState.GetCurrentSignedAmount())

	newExpiration := blockchain.GetNewExpiration(currentBlockNumber, orgGroup.PaymentDetails.PaymentExpirationThreshold) // TODO move this to selectPaymentChannel func

	channelID, err := evm.EnsurePaymentChannel(mpeAddress, filteredChannel, currentSignedAmount, priceInCogs, newExpiration, opts, chans, senders, recipients, groupIDs)
	if err != nil {
		return nil, err
	}

	channelState, err := GetChannelStateFromDaemon(grpc.GRPC, ctx, mpeAddress, channelID, currentBlockNumber, privateKeyECDSA)
	if err != nil {
		return nil, err
	}

	nonce := new(big.Int).SetBytes(channelState.GetCurrentNonce())
	if nonce == nil {
		return nil, errors.New("error while getting current nonce")
	}

	currentSignedAmount = new(big.Int).SetBytes(channelState.GetCurrentSignedAmount())
	if currentSignedAmount == nil {
		return nil, errors.New("error while getting signed amount")
	}

	increment := new(big.Int).Mul(priceInCogs, big.NewInt(int64(callCount)))
	signedAmount := new(big.Int).Add(currentSignedAmount, increment)

	return &PrepaidStrategy{
		tokenClient:     NewTokenServiceClient(grpc.GRPC),
		evmClient:       evm,
		grpcClient:      grpc,
		mpeAddr:         mpeAddress,
		privateKeyECDSA: privateKeyECDSA,
		signedAmount:    signedAmount,
		channelID:       channelID,
		nonce:           nonce,
	}, nil
}
