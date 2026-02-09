// Package payment provides payment strategy implementations for SingularityNET service calls.
//
// The package implements three payment models that control how service invocations are
// authenticated and paid for:
//
//   - Free Call Strategy: Uses free-call tokens for services that support free usage
//   - Paid Call Strategy: Pay-per-call using Multi-Party Escrow (MPE) channels
//   - Prepaid Strategy: Pre-funded payment channels for lower per-call overhead
//
// # Strategy Interface
//
// All payment strategies implement the Strategy interface:
//
//	type Strategy interface {
//		GRPCMetadata(ctx context.Context) context.Context
//		Refresh(ctx context.Context) error
//	}
//
// The SDK automatically manages strategy lifecycle:
//  1. Call Refresh() to update tokens/signatures if needed
//  2. Wrap the gRPC context with GRPCMetadata() to attach payment headers
//  3. Invoke the service method with the wrapped context
//
// # Free Call Strategy
//
// Services may offer free calls with usage limits. The free call strategy:
//   - Fetches free-call tokens from the service daemon
//   - Manages token lifecycle and renewal
//   - No payment required, no blockchain transactions
//   - Limited calls per user/time period
//
// Usage:
//
//	service, _ := sdk.NewServiceClient("org", "service", "group")
//	err := service.SetFreeCallPaymentStrategy()
//	if err != nil {
//		// Service doesn't support free calls or limit reached
//		log.Printf("Free calls not available: %v", err)
//	}
//	response, _ := service.CallWithJSON("method", input)
//
// Free call tokens are automatically refreshed when needed.
//
// # Paid Call Strategy
//
// Pay-per-call using MPE escrow channels. Each call:
//  1. Opens or reuses a payment channel with escrow
//  2. Increments the call amount in the channel
//  3. Signs the payment authorization
//  4. Attaches signed payment to gRPC metadata
//
// The daemon validates the signature and amount before processing the call.
//
// Usage:
//
//	service, _ := sdk.NewServiceClient("org", "service", "group")
//	err := service.SetPaidPaymentStrategy()
//	if err != nil {
//		log.Fatalf("Failed to set paid strategy: %v", err)
//	}
//	// Each call increments the escrow amount
//	response, _ := service.CallWithJSON("method", input)
//
// Requirements:
//   - WebSocket RPC endpoint (wss:// or ws://)
//   - FET token balance for escrow
//   - Gas (ETH) for channel operations
//   - Private key configured
//
// # Prepaid Strategy
//
// Uses pre-funded payment channels. Similar to paid strategy but:
//   - Channel is opened and funded upfront
//   - Lower per-call overhead (no channel management per call)
//   - Better for high-volume usage
//   - Requires manual channel funding
//
// Usage:
//
//	service, _ := sdk.NewServiceClient("org", "service", "group")
//	err := service.SetPrePaidStrategy()
//	if err != nil {
//		log.Fatalf("Failed to set prepaid strategy: %v", err)
//	}
//	// Calls deduct from pre-funded channel
//	response, _ := service.CallWithJSON("method", input)
//
// # Strategy Comparison
//
//	┌──────────────┬─────────────┬────────────┬──────────────┐
//	│ Feature      │ Free Call   │ Paid Call  │ Prepaid      │
//	├──────────────┼─────────────┼────────────┼──────────────┤
//	│ Payment      │ None        │ Per call   │ Pre-funded   │
//	│ Setup Cost   │ None        │ Gas        │ Gas + FET    │
//	│ Per-call Cost│ None        │ Gas + FET  │ FET only     │
//	│ RPC Type     │ HTTP/WSS    │ WSS only   │ WSS only     │
//	│ Private Key  │ Not needed  │ Required   │ Required     │
//	│ Use Case     │ Testing     │ Low volume │ High volume  │
//	└──────────────┴─────────────┴────────────┴──────────────┘
//
// # Switching Strategies
//
// Strategies can be changed at runtime:
//
//	// Start with free calls for testing
//	service.SetFreeCallPaymentStrategy()
//	testResponse, _ := service.CallWithJSON("test", testInput)
//
//	// Switch to paid calls for production
//	service.SetPaidPaymentStrategy()
//	prodResponse, _ := service.CallWithJSON("process", realInput)
//
// # Payment Channel Lifecycle
//
// For paid and prepaid strategies, the SDK manages payment channels:
//
// 1. Check if suitable channel exists
// 2. If not, open new channel with escrow deposit
// 3. For each call, increment amount and sign
// 4. Daemon validates signature and processes call
// 5. Channel can be extended or closed later
//
// Channel state is tracked on-chain via MPE contract events.
//
// # Error Handling
//
// Common payment errors:
//
//   - Insufficient FET balance: Cannot open/fund channels
//   - Insufficient gas: Cannot submit channel transactions
//   - No WebSocket RPC: Paid/prepaid require WSS for events
//   - No private key: Paid/prepaid require signing
//   - Free call limit reached: Service quota exceeded
//   - Invalid signature: Payment authorization rejected
//
// Example error handling:
//
//	err := service.SetPaidPaymentStrategy()
//	if err != nil {
//		if strings.Contains(err.Error(), "insufficient balance") {
//			return fmt.Errorf("please fund your wallet with FET tokens")
//		}
//		if strings.Contains(err.Error(), "websocket") {
//			return fmt.Errorf("paid calls require WebSocket RPC endpoint")
//		}
//		return err
//	}
//
// # Metadata Format
//
// Payment metadata attached to gRPC calls varies by strategy:
//
// Free Call:
//   - snet-free-call-auth-token-bin: Authentication token
//   - snet-payment-type: "free-call"
//
// Paid/Prepaid:
//   - snet-payment-type: "escrow"
//   - snet-payment-channel-id: Channel identifier
//   - snet-payment-channel-nonce: Channel nonce
//   - snet-payment-channel-amount: Cumulative amount
//   - snet-payment-channel-signature-bin: Payment signature
//
// # Thread Safety
//
// Strategy instances are safe for concurrent use. Internal state (tokens, channel
// nonces) is protected with appropriate synchronization.
//
// # Best Practices
//
// 1. Use free calls for development and testing
// 2. Use paid calls for low-volume production (simplest setup)
// 3. Use prepaid for high-volume production (best performance)
// 4. Always check strategy.Refresh() errors before critical calls
// 5. Handle payment errors gracefully with fallbacks
// 6. Monitor channel balances and expiration
// 7. Use WebSocket RPC for paid/prepaid strategies
//
// # See Also
//
//   - sdk.Service.SetFreeCallPaymentStrategy()
//   - sdk.Service.SetPaidPaymentStrategy()
//   - sdk.Service.SetPrePaidStrategy()
//   - examples/free-calls for free call example
//   - examples/paid-call for paid call example
//   - examples/pre-paid for prepaid example
//   - wiki/choose_strategy.md for strategy selection guide
package payment
