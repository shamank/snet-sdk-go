// Package payment defines constants for gRPC metadata headers used in
// SingularityNET daemon payment and authentication flows.
package payment

const (
	// PrefixInSignature is the message prefix used for MPE claim signatures.
	PrefixInSignature = "__MPE_claim_message"
	// FreeCallPrefixSignature is the agreed constant value for free trial signatures.
	FreeCallPrefixSignature = "__prefix_free_trial"
	// PaymentTypeHeader is the gRPC metadata key for the type of payment used for an RPC call.
	// Supported types are: "escrow".
	// Note: "job" PaymentDetails type is deprecated.
	PaymentTypeHeader = "snet-payment-type"
	// ClientTypeHeader identifies the client making the call (e.g., "snet-cli", "snet-dapp", "snet-sdk").
	ClientTypeHeader = "snet-client-type"
	// UserInfoHeader contains the user's Ethereum address (e.g., "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207").
	UserInfoHeader = "snet-user-info"
	// UserAgentHeader contains user agent details set in the server stream info.
	UserAgentHeader = "user-agent"
	// PaymentChannelIDHeader is a MultiPartyEscrow contract payment channel
	// id. Value is a string containing a decimal number.
	PaymentChannelIDHeader = "snet-payment-channel-id"
	// PaymentChannelNonceHeader is a payment channel nonce value. Value is a
	// string containing a decimal number.
	PaymentChannelNonceHeader = "snet-payment-channel-nonce"
	// PaymentChannelAmountHeader is an amount of payment channel value
	// which server is authorized to withdraw after handling the RPC call.
	// Value is a string containing a decimal number.
	PaymentChannelAmountHeader = "snet-payment-channel-amount"
	// PaymentChannelSignatureHeader is a signature of the client to confirm
	// amount withdrawing authorization. Value is an array of bytes.
	PaymentChannelSignatureHeader = "snet-payment-channel-signature-bin"
	// PaymentMultiPartyEscrowAddressHeader contains the MPE contract address.
	// This is useful when the daemon is running in blockchain-disabled mode,
	// allowing the client to remain oblivious to the daemon's mode while
	// standardizing signatures. Value is a string containing a decimal number.
	PaymentMultiPartyEscrowAddressHeader = "snet-payment-mpe-address"

	// Free call support headers

	// FreeCallUserIdHeader contains the user ID of the person making the free call.
	FreeCallUserIdHeader = "snet-free-call-user-id"
	// FreeCallUserAddressHeader contains the user's Ethereum address for free calls.
	FreeCallUserAddressHeader = "snet-free-call-user-address"

	// CurrentBlockNumberHeader is used to verify if the signature is still valid.
	CurrentBlockNumberHeader = "snet-current-block-number"

	// FreeCallAuthTokenHeader contains the free call authentication token issued by the daemon.
	FreeCallAuthTokenHeader = "snet-free-call-auth-token-bin"
	// FreeCallAuthTokenExpiryBlockNumberHeader contains the block number when the token was issued,
	// used to track token expiry (typically ~1 month).
	FreeCallAuthTokenExpiryBlockNumberHeader = "snet-free-call-token-expiry-block"

	// PrePaidAuthTokenHeader contains the prepaid authentication token.
	// Users may sign upfront and make calls using this token for the amount signed.
	PrePaidAuthTokenHeader = "snet-prepaid-auth-token-bin"

	// DynamicPriceDerived contains the derived dynamic price cost.
	DynamicPriceDerived = "snet-derived-dynamic-price-cost"

	// TrainingModelId contains the training model identifier.
	TrainingModelId = "snet-train-model-id"
)
