package blockchain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"go.uber.org/zap"
)

// GetTransactOpts creates a transactor bound to the given chainID and ECDSA key.
func GetTransactOpts(chainID *big.Int, pk *ecdsa.PrivateKey) (*bind.TransactOpts, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		zap.L().Error("failed to create transactor", zap.Error(err))
		return nil, err
	}
	return opts, nil
}

// GetTransactOptsFromEVM creates a transactor from the EVM client context.
func (evm *EVMClient) GetTransactOpts(pk *ecdsa.PrivateKey) (*bind.TransactOpts, error) {
	if pk == nil {
		return nil, fmt.Errorf("private key is required for transactions")
	}

	chainID, err := evm.Client.ChainID(context.Background())
	if err != nil {
		zap.L().Error("failed to get chain ID", zap.Error(err))
		return nil, err
	}

	return GetTransactOpts(chainID, pk)
}
