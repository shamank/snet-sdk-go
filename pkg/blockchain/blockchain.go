// Package blockchain provides Go bindings and helpers to interact with
// SingularityNET contracts on EVM chains. It initializes an Ethereum client,
// wires typed bindings for Registry, MultiPartyEscrow (MPE) and FetchToken
// contracts, exposes lightweight read helpers for on-chain metadata, and
// includes utilities for bytes32 conversions and Ethereum-compatible message
// signatures.
//
//go:generate go run ../../cmd/generate-smart-binds/main.go
package blockchain

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	contracts "github.com/singnet/snet-ecosystem-contracts"
	"go.uber.org/zap"
)

var (
	// HashPrefix32Bytes is the standard Ethereum personal-sign prefix for 32-byte
	// messages: "\x19Ethereum Signed Message:\n32".
	// See Geth reference:
	// https://github.com/ethereum/go-ethereum/blob/bf468a81ec261745b25206b2a596eb0ee0a24a74/internal/ethapi/api.go#L361
	HashPrefix32Bytes = []byte("\x19Ethereum Signed Message:\n32")
)

// EVMClient holds a connected ethclient.Client and typed bindings for the core
// SingularityNET contracts: Registry, MultiPartyEscrow (MPE) and FetchToken.
type EVMClient struct {
	Client     *ethclient.Client
	Registry   *Registry
	MPE        *MultiPartyEscrow
	FetchToken *FetchToken
}

// networks is a helper type that mirrors the JSON payload produced by
// snet-ecosystem-contracts (network name → contract address).
type networks map[string]struct {
	Address string `json:"address"`
}

// InitEvm dials an Ethereum endpoint and initializes typed bindings for
// Registry and MultiPartyEscrow using addresses resolved from
// snet-ecosystem-contracts for the given network. It also discovers the
// FetchToken address via MPE and binds it.
//
// Parameters:
//   - network: chain/network key as used by snet-ecosystem-contracts (e.g. "11155111").
//   - endpoint: RPC/WS endpoint URL to dial.
//
// Returns a ready-to-use EVMClient or an error.
func InitEvm(network, endpoint string) (*EVMClient, error) {
	registryNetworksRaw := contracts.GetNetworks(contracts.Registry)
	var rn networks
	err := json.Unmarshal(registryNetworksRaw, &rn)
	if err != nil {
		zap.L().Error("Failed to unmarshal", zap.Error(err))
		return nil, err
	}
	registryAddress := rn[network].Address

	var eth = new(EVMClient)

	eth.Client, err = ethclient.Dial(endpoint)
	if err != nil {
		zap.L().Error("Failed to ethdial", zap.Error(err))
		return nil, err
	}

	eth.Registry, err = NewRegistry(common.HexToAddress(registryAddress), eth.Client)
	if err != nil {
		return eth, err
	}

	MPENetworksRaw := contracts.GetNetworks(contracts.MultiPartyEscrow)
	var mpen networks
	err = json.Unmarshal(MPENetworksRaw, &mpen)
	if err != nil {
		zap.L().Error("Failed to unmarshal", zap.Error(err))
		return nil, err
	}
	MPEAddress := mpen[network].Address
	eth.MPE, err = NewMultiPartyEscrow(common.HexToAddress(MPEAddress), eth.Client)

	callOpts := &bind.CallOpts{}
	tokenAddr, err := eth.MPE.Token(callOpts)
	if err != nil {
		zap.L().Error("Failed to get token address", zap.Error(err))
		return nil, err
	}

	eth.FetchToken, err = NewFetchToken(tokenAddr, eth.Client)
	if err != nil {
		zap.L().Error("Failed to get FetchToken", zap.Error(err))
		return nil, err
	}

	return eth, err
}

// GetOrganizations returns organization IDs from the on-chain Registry.
// On error, it logs and returns nil.
func (eth *EVMClient) GetOrganizations() []string {
	organizations, err := eth.Registry.ListOrganizations(nil)
	if err != nil {
		zap.L().Error("Failed to list organizations", zap.Error(err))
		return nil
	}
	return Bytes32ArrayToStrings(organizations)
}

// GetServices returns service IDs for the given organization ID.
// If the organization is not found or a read error occurs, it logs and returns nil.
func (eth *EVMClient) GetServices(orgID string) []string {
	organizations, err := eth.Registry.ListServicesForOrganization(nil, StringToBytes32(orgID))
	if err != nil {
		zap.L().Error("Failed to list organizations", zap.Error(err))
		return nil
	}
	if !organizations.Found {
		zap.L().Error("Organization not found", zap.String("OrganizationID", orgID))
		return nil
	}
	return Bytes32ArrayToStrings(organizations.ServiceIds)
}

// Bytes32ArrayToStrings converts an array of [32]byte values into a slice of strings,
// trimming trailing NUL bytes on the right of each element.
func Bytes32ArrayToStrings(arr [][32]byte) []string {
	result := make([]string, len(arr))
	for i, b := range arr {
		// b[:] is the 32-byte slice; trim trailing '\x00'.
		clean := bytes.TrimRight(b[:], "\x00")
		result[i] = string(clean)
	}
	return result
}

// GetCurrentBlockNumber returns the latest block number using a non-cancellable
// background context. Prefer GetCurrentBlockNumberCtx if you need cancellation.
func (eth *EVMClient) GetCurrentBlockNumber() (*big.Int, error) {
	header, err := eth.Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		zap.L().Error("failed to get last block number", zap.Error(err))
		return nil, err
	}
	return header.Number, nil
}

// GetOrgMetadataUri returns the organization metadata URI (as stored in Registry)
// for the given orgID. It panics if the organization cannot be retrieved or is not found.
//
// Note: the method name uses "Uri" for historical reasons; it returns a URI string.
func (eth *EVMClient) GetOrgMetadataUri(orgID string) string {
	orgId := StringToBytes32(orgID)
	org, err := eth.Registry.GetOrganizationById(nil, orgId)
	if err != nil || &org == nil || !org.Found {
		zap.L().Panic("Error Retrieving contract details for the Given Organization and Service Ids ",
			zap.String("OrganizationId", orgID))
	}

	return string(org.OrgMetadataURI[:])
}

// GetServiceMetadataHashRegistry returns the service metadata URI (hash) recorded
// in the Registry for the given (orgID, srvID). It panics if the entry cannot be
// retrieved or is not found.
func (eth *EVMClient) GetServiceMetadataHashRegistry(orgID, srvID string) string {
	orgId := StringToBytes32(orgID)
	serviceId := StringToBytes32(srvID)
	serviceRegistration, err := eth.Registry.GetServiceRegistrationById(nil, orgId, serviceId)
	if err != nil || &serviceRegistration == nil || !serviceRegistration.Found {
		zap.L().Panic("Error Retrieving contract details for the Given Organization and Service Ids ",
			zap.String("OrganizationId", orgID),
			zap.String("ServiceId", srvID))
	}

	return string(serviceRegistration.MetadataURI[:])
}

// StringToBytes32 returns a right-padded [32]byte containing at most the first
// 32 bytes of the provided string.
func StringToBytes32(str string) [32]byte {
	var byte32 [32]byte
	copy(byte32[:], str)
	return byte32
}

// GetSignature produces an Ethereum-compatible personal-sign (EIP-191 style)
// signature over the given message. It hashes the payload as
// keccak256("\x19Ethereum Signed Message:\n32" || keccak256(message)) and
// signs with the provided ECDSA private key.
//
// Returns the 65-byte signature (R||S||V). On signing error it logs and returns nil.
func GetSignature(message []byte, privateKeyECDSA *ecdsa.PrivateKey) []byte {
	hash := crypto.Keccak256(
		HashPrefix32Bytes,
		crypto.Keccak256(message),
	)

	signature, err := crypto.Sign(hash, privateKeyECDSA)
	if err != nil {
		zap.L().Error("Failed to sign message", zap.Error(err))
	}

	return signature
}
