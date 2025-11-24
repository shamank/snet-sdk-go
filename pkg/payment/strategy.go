//go:generate protoc --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative --go_out=. --go-grpc_out=. --proto_path=. state_service.proto token_service.proto

// Package payment defines common interfaces and implementations for attaching
// payment/authentication metadata to daemon calls (escrow, prepaid, free-call).
// The go:generate directive above compiles the payment-related *.proto files
// in this directory to produce the corresponding Go types and gRPC stubs.
package payment

import (
	"context"
)

// Strategy abstracts a payment/authentication mechanism used by the SDK when
// invoking daemon methods over gRPC. Implementations include FreeStrategy
// (free-call tokens), PaidStrategy (MPE escrow), and PrepaidStrategy.
//
// Typical flow per request:
//  1. Call Refresh(ctx) to refresh tokens/signatures if needed.
//  2. Wrap the outbound context with GRPCMetadata(ctx) and pass it to the RPC.
type Strategy interface {
	// GRPCMetadata decorates the provided context with the required gRPC
	// headers (e.g., payment type, channel ID/nonce/amount, signatures, tokens).
	// It returns a derived context that should be used for the RPC invocation.
	GRPCMetadata(ctx context.Context) context.Context
	// Refresh updates internal state (e.g., token issuance/renewal or
	// recalculation of signatures) prior to making calls. Implementations
	// should be idempotent and cheap when no refresh is required.
	Refresh(ctx context.Context) error
}
