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
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	contracts "github.com/singnet/snet-ecosystem-contracts"
	"github.com/singnet/snet-sdk-go/pkg/storage"
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
	Storage    storage.Storage
}

type Evm interface {
	GetCurrentBlockNumber() (*big.Int, error)
	GetOrganizations() ([]string, error)
	NewOrgClient(orgID, groupName string) (*OrgClient, error)
	Close()
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
//   - registryAddress: optional registry contract address override. Provide an
//     empty string to resolve it from snet-ecosystem-contracts metadata.
//
// Returns a ready-to-use EVMClient or an error.
func InitEvm(network, endpoint, registryAddress string, storage storage.Storage) (*EVMClient, error) {
	registryNetworksRaw := contracts.GetNetworks(contracts.Registry)
	var rn networks
	err := json.Unmarshal(registryNetworksRaw, &rn)
	if err != nil {
		zap.L().Error("Failed to unmarshal", zap.Error(err))
		return nil, err
	}

	if registryAddress == "" {
		registryAddress = rn[network].Address
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var eth = new(EVMClient)
	eth.Client, err = ethclient.DialContext(ctx, endpoint)
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

	eth.Storage = storage

	return eth, err
}

// GetCurrentBlockNumber returns the latest block number using a non-cancellable
// background context. Prefer GetCurrentBlockNumberCtx if you need cancellation.
func (evm *EVMClient) GetCurrentBlockNumber() (*big.Int, error) {
	header, err := evm.Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		zap.L().Error("failed to get last block number", zap.Error(err))
		return nil, err
	}
	return header.Number, nil
}

// getOrgMetadataUri returns the organization metadata URI (as stored in Registry)
// for the given orgID. It panics if the organization cannot be retrieved or is not found.
//
// Note: the method name uses "Uri" for historical reasons; it returns a URI string.
func (evm *EVMClient) getOrgMetadataUri(orgID string) string {
	orgId := StringToBytes32(orgID)
	org, err := evm.Registry.GetOrganizationById(nil, orgId)
	if err != nil || &org == nil || !org.Found {
		zap.L().Panic("Error Retrieving contract details for the Given Organization and Service Ids ",
			zap.String("OrganizationId", orgID))
	}
	return string(org.OrgMetadataURI[:])
}

// GetOrganizations returns organization IDs from the on-chain Registry.
// On error, it logs and returns nil.
func (evm *EVMClient) GetOrganizations() ([]string, error) {
	organizations, err := evm.Registry.ListOrganizations(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	return Bytes32ArrayToStrings(organizations), nil
}

func (evm *EVMClient) Close() {
	evm.Client.Close()
}
