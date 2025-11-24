package payment

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"log"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-sdk-go/pkg/blockchain"
	"github.com/singnet/snet-sdk-go/pkg/grpc"
	"github.com/singnet/snet-sdk-go/pkg/model"
	"google.golang.org/grpc/metadata"
)

// FreeStrategy implements the "free-call" authentication flow against the daemon.
// It obtains a short-lived free-call token from the daemon's FreeCallState service,
// and attaches the required gRPC metadata (token, user address, signed message,
// current block) to each request.
type FreeStrategy struct {
	Token            []byte
	orgID            string
	groupID          string
	serviceID        string
	evmClient        *blockchain.EVMClient
	grpcClient       *grpc.Client
	serviceMetadata  *model.ServiceMetadata
	signerPrivateKey *ecdsa.PrivateKey
	signerAddress    common.Address
	stateClient      FreeCallStateServiceClient
	tokenLifetime    *uint64
	blockNumber      func(context.Context) (*big.Int, error)
}

// NewFreeStrategy constructs a FreeStrategy for the given org/service/group.
// The private key must be valid and corresponds to the user (caller) address.
// tokenLifetime controls the requested token lifetime in blocks (nil = daemon default).
func NewFreeStrategy(evm *blockchain.EVMClient, grpc *grpc.Client, orgID, serviceID, groupID string, privateKey *ecdsa.PrivateKey, tokenLifetime *uint64) (Strategy, error) {
	addr := blockchain.GetAddressFromPrivateKeyECDSA(privateKey)
	if addr == nil {
		return nil, errors.New("invalid private key")
	}

	return &FreeStrategy{
		evmClient:        evm,
		grpcClient:       grpc,
		serviceID:        serviceID,
		orgID:            orgID,
		groupID:          groupID,
		signerPrivateKey: privateKey,
		signerAddress:    *blockchain.GetAddressFromPrivateKeyECDSA(privateKey),
		stateClient:      NewFreeCallStateServiceClient(grpc.GRPC),
		tokenLifetime:    tokenLifetime,
		blockNumber: func(ctx context.Context) (*big.Int, error) {
			return evm.GetCurrentBlockNumberCtx(ctx)
		},
	}, nil
}

// Refresh requests (or refreshes) a free-call token from the daemon.
// It signs a message using the current block number for freshness and stores
// the received token in f.Token.
func (f *FreeStrategy) Refresh(ctx context.Context) error {
	if f.blockNumber == nil {
		return errors.New("block number provider not configured")
	}
	number, err := f.blockNumber(ctx)
	if err != nil {
		return err
	}
	msg := f.msgForNewFreeCallToken(number.Uint64())
	signedMsg := blockchain.GetSignature(msg, f.signerPrivateKey)
	token, err := f.stateClient.GetFreeCallToken(ctx, &GetFreeCallTokenRequest{
		Address:               f.signerAddress.Hex(),
		Signature:             signedMsg,
		CurrentBlock:          number.Uint64(),
		TokenLifetimeInBlocks: f.tokenLifetime,
	})
	if err != nil {
		log.Println(err)
		return err
	}
	f.Token = token.Token
	return nil
}

// GRPCMetadata injects the "free-call" authentication headers into the outgoing
// context, including the token, user address, signature and current block.
// Returns a derived context; if the current block cannot be read, it returns nil.
func (f *FreeStrategy) GRPCMetadata(ctx context.Context) context.Context {
	if f.blockNumber == nil {
		return nil
	}
	number, err := f.blockNumber(ctx)
	if err != nil {
		return nil
	}

	msg := f.msgForFreeCall(number.Uint64())
	signedMsg := blockchain.GetSignature(msg, f.signerPrivateKey)

	md := metadata.Pairs(
		PaymentTypeHeader, "free-call",
		FreeCallAuthTokenHeader, string(f.Token),
		FreeCallUserAddressHeader, f.signerAddress.Hex(),
		PaymentChannelSignatureHeader, string(signedMsg),
		CurrentBlockNumberHeader, strconv.FormatInt(int64(number.Uint64()), 10),
	)
	return metadata.NewOutgoingContext(ctx, md)
}

// GetFreeCallsAvailable queries the daemon for the remaining number of free
// calls for the current user/token pair. It signs a freshness-bound message
// using the current block.
func (f *FreeStrategy) GetFreeCallsAvailable(ctx context.Context) (uint64, error) {
	if f.blockNumber == nil {
		return 0, errors.New("block number provider not configured")
	}
	number, err := f.blockNumber(ctx)
	if err != nil {
		return 0, nil
	}
	msg := f.msgForFreeCall(number.Uint64())
	resp, err := f.stateClient.GetFreeCallsAvailable(ctx, &FreeCallStateRequest{
		Address:       f.signerAddress.Hex(),
		FreeCallToken: f.Token,
		Signature:     blockchain.GetSignature(msg, f.signerPrivateKey),
		CurrentBlock:  number.Uint64(),
	})
	if err != nil {
		return 0, err
	}
	return resp.FreeCallsAvailable, nil
}

// msgForNewFreeCallToken builds the message that authorizes issuance of a new
// free-call token. It includes prefix, user address, org/service/group IDs,
// and the current block number.
func (f *FreeStrategy) msgForNewFreeCallToken(currentBlockNumber uint64) []byte {
	return bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature), // prefix
		[]byte(f.signerAddress.Hex()),   // hex user address
		//[]byte(userID),                  // user id (web2 way)
		[]byte(f.orgID),
		[]byte(f.serviceID),
		[]byte(f.groupID),
		bigIntToBytes(big.NewInt(int64(currentBlockNumber))),
	}, nil)
}

// msgForFreeCall builds the message that authorizes a free-call with the current
// token and freshness block.
func (f *FreeStrategy) msgForFreeCall(currentBlockNumber uint64) []byte {
	return bytes.Join([][]byte{
		[]byte(FreeCallPrefixSignature), // prefix
		[]byte(f.signerAddress.Hex()),   // user address
		[]byte(f.orgID),
		[]byte(f.serviceID),
		[]byte(f.groupID),
		bigIntToBytes(big.NewInt(int64(currentBlockNumber))),
		f.Token,
	}, nil)
}
