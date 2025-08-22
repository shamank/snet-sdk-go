// Package payment contains payment-related helpers and strategies used by the
// SDK to interact with the SingularityNET daemon (payment channel state,
// free/paid/prepaid strategies, and associated headers and requests).
package payment

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/singnet/snet-sdk-go/pkg/blockchain"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// PrefixGetChannelState is the fixed message prefix used when requesting the
// current channel state from the daemon.
const PrefixGetChannelState = "__get_channel_state"

// GetChannelStateFromDaemon queries the daemon for the latest signed state of a
// given MPE payment channel.
//
// It builds a message:
//
//	concat("__get_channel_state", MPEAddress, channelID, currentBlockNumber)
//
// hashes/signs it with an Ethereum personal-sign style signature (see
// blockchain.GetSignature) using privateKeyECDSA, and sends a ChannelStateRequest
// via the PaymentChannelStateService. The daemon verifies the signature and
// returns the current signed amount/nonce for the channel.
//
// Parameters:
//   - grpcConn: connected gRPC ClientConn to the daemon.
//   - ctx:      call context (cancellation/deadline).
//   - MPEAddress: address of the MPE contract.
//   - channelID:  channel identifier.
//   - currentBlockNumber: chain tip (or recent) block number used for freshness.
//   - privateKeyECDSA: signer key corresponding to the user address.
//
// Returns a non-nil ChannelStateReply on success or an error.
func GetChannelStateFromDaemon(grpcConn *grpc.ClientConn, ctx context.Context, MPEAddress common.Address, channelID, currentBlockNumber *big.Int, privateKeyECDSA *ecdsa.PrivateKey) (*ChannelStateReply, error) {

	client := NewPaymentChannelStateServiceClient(grpcConn)

	message := bytes.Join([][]byte{
		[]byte(PrefixGetChannelState),
		MPEAddress.Bytes(),
		bigIntToBytes(channelID),
		math.U256Bytes(currentBlockNumber),
	}, nil)

	request := &ChannelStateRequest{
		ChannelId:    channelID.Bytes(),
		Signature:    blockchain.GetSignature(message, privateKeyECDSA),
		CurrentBlock: currentBlockNumber.Uint64(),
	}

	reply, err := client.GetChannelState(ctx, request)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, errors.New("channel state reply is nil")
	}
	zap.L().Debug("Channel state reply", zap.Any("reply", reply))
	return reply, nil
}

// bigIntToBytes encodes a big.Int as a 32-byte big-endian slice, matching
// Ethereum's common.BigToHash formatting.
func bigIntToBytes(value *big.Int) []byte {
	return common.BigToHash(value).Bytes()
}
