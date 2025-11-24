//go:generate protoc --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative --go_out=. --go-grpc_out=. --proto_path=. training.proto training_daemon.proto

// Package training provides a client for interacting with SingularityNET training daemons.
// It allows users to manage AI models, retrieve training metadata, and perform model-related operations.
package training

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	_ "embed"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-sdk-go/pkg/blockchain"
	"github.com/singnet/snet-sdk-go/pkg/grpc"
	"go.uber.org/zap"
)

// Client defines the interface for training-related operations.
type Client interface {
	// GetMetadata retrieves training daemon metadata
	GetMetadata() (*TrainingMetadata, error)
	// GetAllModels retrieves all available models
	GetAllModels() (*ModelsResponse, error)
	// GetModel retrieves a specific model by ID
	GetModel(modelId string) (*ModelResponse, error)
	// CreateModel creates a new model with the given name and description
	CreateModel(name, desc string) (*ModelResponse, error)
}

// TrainingClient is the concrete implementation of the Client interface
// for training operations.
type TrainingClient struct {
	DaemonClient
	privateKey         *ecdsa.PrivateKey
	currentBlockNumber func() (*big.Int, error)
}

// NewTrainingClient creates a new training client with the provided gRPC client,
// private key for signing requests, and a function to retrieve the current block number.
func NewTrainingClient(client *grpc.Client, priv *ecdsa.PrivateKey, currentBlockNumber func() (*big.Int, error)) *TrainingClient {
	return &TrainingClient{
		DaemonClient:       NewDaemonClient(client.GRPC),
		privateKey:         priv,
		currentBlockNumber: currentBlockNumber,
	}
}

// GetMetadata retrieves training daemon metadata including supported operations.
func (c *TrainingClient) GetMetadata() (*TrainingMetadata, error) {
	metadata, err := c.GetTrainingMetadata(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

// GetAllModels retrieves all available models from the training daemon.
// The request is authenticated with the client's private key.
func (c *TrainingClient) GetAllModels() (*ModelsResponse, error) {

	block, err := c.currentBlockNumber()
	if err != nil {
		return nil, err
	}
	signature, err := getSignature("get_all_models", block, c.privateKey)
	if err != nil {
		return nil, err
	}

	var allModelsReq = &AllModelsRequest{
		Authorization:    newAuth(blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(), "get_all_models", block.Uint64(), signature),
		Statuses:         nil,
		IsPublic:         nil,
		GrpcMethodName:   "",
		GrpcServiceName:  "",
		Name:             "",
		CreatedByAddress: "",
		PageSize:         100,
		Page:             0,
	}

	resp, err := c.DaemonClient.GetAllModels(context.Background(), allModelsReq)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetModel retrieves a specific model by its ID from the training daemon.
// The request is authenticated with the client's private key.
func (c *TrainingClient) GetModel(modelId string) (*ModelResponse, error) {

	block, err := c.currentBlockNumber()
	if err != nil {
		return nil, err
	}
	signature, err := getSignature("get_model", block, c.privateKey)
	if err != nil {
		return nil, err
	}

	var req = &CommonRequest{
		Authorization: newAuth(blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(), "get_model", block.Uint64(), signature),
		ModelId:       modelId,
	}

	resp, err := c.DaemonClient.GetModel(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateModel creates a new model with the specified name and description.
// The request is authenticated with the client's private key.
func (c *TrainingClient) CreateModel(name, desc string) (*ModelResponse, error) {

	block, err := c.currentBlockNumber()
	if err != nil {
		return nil, err
	}
	signature, err := getSignature("create_model", block, c.privateKey)
	if err != nil {
		return nil, err
	}

	var req = &NewModelRequest{
		Authorization: newAuth(blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(), "create_model", block.Uint64(), signature),
		Model: &NewModel{
			Name:            name,
			Description:     desc,
			GrpcMethodName:  "stt",
			GrpcServiceName: "Example",
			AddressList:     nil,
			IsPublic:        true,
			OrganizationId:  "",
			ServiceId:       "",
			GroupId:         "default_group",
		},
	}

	resp, err := c.DaemonClient.CreateModel(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// newAuth creates authorization details for a training daemon request.
func newAuth(addr, msg string, blockNumber uint64, signature []byte) *AuthorizationDetails {
	return &AuthorizationDetails{
		CurrentBlock:  blockNumber,
		Message:       msg,
		Signature:     signature,
		SignerAddress: addr,
	}
}

// getSignature generates an Ethereum-style signature for a training method call.
// The signature is computed over a message containing the method name, signer address, and block number.
func getSignature(methodName string, blockNumber *big.Int, privateKey *ecdsa.PrivateKey) (signature []byte, err error) {
	message := bytes.Join([][]byte{
		[]byte(methodName),
		crypto.PubkeyToAddress(privateKey.PublicKey).Bytes(),
		math.U256Bytes(blockNumber),
	}, nil)
	hash := crypto.Keccak256(
		blockchain.HashPrefix32Bytes,
		crypto.Keccak256(message),
	)
	signature, err = crypto.Sign(hash, privateKey)
	if err != nil {
		return nil, fmt.Errorf("can't sign message: %w", err)
	}
	zap.L().Debug("signed with", zap.String("addr", common.BytesToAddress(crypto.PubkeyToAddress(privateKey.PublicKey).Bytes()).Hex()))

	return signature, nil
}
