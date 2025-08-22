package payment

const (
	PrefixInSignature = "__MPE_claim_message"
	// Agreed constant value.
	FreeCallPrefixSignature = "__prefix_free_trial"
	// PaymentTypeHeader is a type of payment used to pay for a RPC call.
	// Supported types are: "escrow".
	// Note: "job" PaymentDetails type is deprecated
	PaymentTypeHeader = "snet-payment-type"
	// Client that calls the Daemon ( example can be "snet-cli","snet-dapp","snet-sdk")
	ClientTypeHeader = "snet-client-type"
	// Value is a user address , example "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207"
	UserInfoHeader = "snet-user-info"
	// User Agent details set in on the server stream info
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
	// This is useful information in the header sent in by the client
	// All clients will have this information and they need this to Sign anyways
	// When Daemon is running in the block chain disabled mode , it would use this
	// header to get the MPE address. The goal here is to keep the client oblivious to the
	// Daemon block chain enabled or disabled mode and also standardize the signatures.
	// id. Value is a string containing a decimal number.
	PaymentMultiPartyEscrowAddressHeader = "snet-payment-mpe-address"

	// Added for free call support in Daemon

	// The user Id of the person making the call
	FreeCallUserIdHeader      = "snet-free-call-user-id"
	FreeCallUserAddressHeader = "snet-free-call-user-address"

	// Will be used to check if the Signature is still valid
	CurrentBlockNumberHeader = "snet-current-block-number"

	// Place holder to set the free call Auth FetchToken issued
	FreeCallAuthTokenHeader = "snet-free-call-auth-token-bin"
	// Block number on when the FetchToken was issued , to track the expiry of the token , which is ~ 1 Month
	FreeCallAuthTokenExpiryBlockNumberHeader = "snet-free-call-token-expiry-block"

	// Users may decide to sign upfront and make calls .Daemon generates and Auth FetchToken
	// Users/Clients will need to use this token to make calls for the amount signed upfront.
	PrePaidAuthTokenHeader = "snet-prepaid-auth-token-bin"

	DynamicPriceDerived = "snet-derived-dynamic-price-cost"

	TrainingModelId = "snet-train-model-id"
)
