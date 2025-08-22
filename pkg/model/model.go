// Package model defines data structures for organization and service metadata
// used by the SDK: organizations, groups, pricing, licensing, and service API
// descriptors (.proto). These structs mirror the JSON documents stored on-chain
// (via URIs in Registry) and/or in IPFS/Filecoin.
package model

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// OrganizationMetaData describes an organization and its groups as found in
// organization metadata. Note: the type name retains the original spelling
// from existing metadata ("MetaData") for compatibility.
type OrganizationMetaData struct {
	OrgName string               `json:"org_name"`
	OrgID   string               `json:"org_id"`
	Groups  []*OrganizationGroup `json:"groups"`
	// daemonGroup/daemonGroupID/recipientPaymentAddress are internal fields used
	// by the daemon to select and reference the active group/payment details.
	daemonGroup             *OrganizationGroup
	daemonGroupID           [32]byte
	recipientPaymentAddress common.Address
}

// ServiceMetadata describes a service, its groups and pricing, API source, and
// auxiliary capabilities (e.g., training). Proto descriptors/files are filled
// at runtime after fetching and parsing the API sources.
type ServiceMetadata struct {
	Version                   int                           `json:"version"`
	DisplayName               string                        `json:"display_name"`
	Encoding                  string                        `json:"encoding"`
	ServiceType               string                        `json:"service_type"`
	Groups                    []*ServiceGroup               `json:"groups"`
	ModelIpfsHash             string                        `json:"model_ipfs_hash"`
	ServiceApiSource          string                        `json:"service_api_source"`
	MPEAddress                string                        `json:"mpe_address"`
	DynamicPriceMethodMapping map[string]string             `json:"dynamicpricing"`
	TrainingMethods           []string                      `json:"training_methods"`
	ProtoDescriptors          []protoreflect.FileDescriptor `json:"-"`
	ProtoFiles                map[string]string             `json:"-"`
}

// GetMpeAddr returns the MPE contract address parsed from ServiceMetadata.MPEAddress.
func (s *ServiceMetadata) GetMpeAddr() common.Address {
	return common.HexToAddress(s.MPEAddress)
}

// OrganizationGroup represents a logical group within an organization, including
// payment configuration and optional licensing information.
type OrganizationGroup struct {
	ID             string   `json:"group_id"`
	GroupName      string   `json:"group_name"`
	PaymentDetails Payment  `json:"payment"`
	Licenses       Licenses `json:"licenses,omitempty"`
}

// Pricing defines a price model for a service or package. When PriceModel refers
// to fixed pricing, PriceInCogs is commonly used; more complex models can be
// encoded via PricingDetails.
type Pricing struct {
	PriceModel     string           `json:"price_model"`
	PriceInCogs    *big.Int         `json:"price_in_cogs,omitempty"`
	PackageName    string           `json:"package_name,omitempty"`
	Default        bool             `json:"default,omitempty"`
	PricingDetails []PricingDetails `json:"details,omitempty"`
}

// ServiceGroup contains endpoint(s), pricing, and free-call configuration for a
// concrete deployment of a service.
type ServiceGroup struct {
	Pricing        []Pricing `json:"pricing"`
	GroupName      string    `json:"group_name"`
	Endpoints      []string  `json:"endpoints"`
	FreeCalls      int       `json:"free_calls"`
	FreeCallSigner string    `json:"free_call_signer_address"`
}

// Payment captures payment parameters for a group, including the recipient
// address, the expiration threshold for channels, and storage settings for
// payment channel state.
type Payment struct {
	PaymentAddress              string                      `json:"payment_address"`                // Payment address.
	PaymentExpirationThreshold  *big.Int                    `json:"payment_expiration_threshold"`   // Payment expiration threshold.
	PaymentChannelStorageType   string                      `json:"payment_channel_storage_type"`   // Payment channel storage type.
	PaymentChannelStorageClient PaymentChannelStorageClient `json:"payment_channel_storage_client"` // Payment channel storage client.
}

// PaymentChannelStorageClient configures the client used to persist channel
// state externally (timeouts and endpoints).
type PaymentChannelStorageClient struct {
	ConnectionTimeout string   `json:"connection_timeout" mapstructure:"connection_timeout"` // Connection timeout.
	RequestTimeout    string   `json:"request_timeout" mapstructure:"request_timeout"`       // Request timeout.
	Endpoints         []string `json:"endpoints"`                                            // List of endpoints.
}

// Subscription describes a subscription-based licensing plan.
type Subscription struct {
	PeriodInDays         int     `json:"periodInDays"`
	DiscountInPercentage float64 `json:"discountInPercentage"`
	PlanName             string  `json:"planName"`
	LicenseCost          big.Int `json:"licenseCost"`
	GrpcServiceName      string  `json:"grpcServiceName,omitempty"`
	GrpcMethodName       string  `json:"grpcMethodName,omitempty"`
}

// Subscriptions aggregates subscription metadata and its status.
type Subscriptions struct {
	Type         string         `json:"type"`
	DetailsURL   string         `json:"detailsUrl"`
	IsActive     string         `json:"isActive"`
	Subscription []Subscription `json:"subscription"`
}

// Tier describes a tiered licensing plan with ranges and optional per-RPC scoping.
type Tier struct {
	Type            string      `json:"type"`
	PlanName        string      `json:"planName"`
	GrpcServiceName string      `json:"grpcServiceName,omitempty"`
	GrpcMethodName  string      `json:"grpcMethodName,omitempty"`
	Range           []TierRange `json:"range"`
	DetailsURL      string      `json:"detailsUrl"`
	IsActive        string      `json:"isActive"`
}

// PricingDetails lists per-method pricing for a given service name.
type PricingDetails struct {
	ServiceName   string          `json:"service_name"`
	MethodPricing []MethodPricing `json:"method_pricing"`
}

// MethodPricing is a single method-to-price mapping (price in cogs).
type MethodPricing struct {
	MethodName  string   `json:"method_name"`
	PriceInCogs *big.Int `json:"price_in_cogs"`
}

// Tiers wraps a single Tier definition (historical schema quirk).
type Tiers struct {
	Tiers Tier `json:"tier"`
}

// AddOns describes an optional add-on with a discount and a fixed cost.
// Note: JSON tag uses "AddOnCostInAGIX" for backward compatibility with older naming.
type AddOns struct {
	DiscountInPercentage float64 `json:"discountInPercentage"`
	AddOnCostInASI       int     `json:"addOnCostInAGIX"`
	Name                 string  `json:"name"`
}

// TierRange defines a numeric upper bound and an associated discount.
type TierRange struct {
	High                 int     `json:"high"`
	DiscountInPercentage float64 `json:"DiscountInPercentage"`
}

// Licenses aggregates subscription and tier-based licensing information.
type Licenses struct {
	Subscriptions Subscriptions `json:"subscriptions,omitempty"`
	Tiers         []Tier        `json:"tiers"`
}
