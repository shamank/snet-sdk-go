//go:generate protoc --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative --go_out=. --go-grpc_out=. --proto_path=. *.proto
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

type Client interface {
	GetMetadata() (*TrainingMetadata, error)
	GetAllModels() (*ModelsResponse, error)
	GetModel(modelId string) (*ModelResponse, error)
	CreateModel(name, desc string) (*ModelResponse, error)
}

type TrainingClient struct {
	DaemonClient
	privateKey         *ecdsa.PrivateKey
	currentBlockNumber func() (*big.Int, error)
}

func NewTrainingClient(client *grpc.Client, priv *ecdsa.PrivateKey, currentBlockNumber func() (*big.Int, error)) *TrainingClient {
	return &TrainingClient{
		DaemonClient:       NewDaemonClient(client.GRPC),
		privateKey:         priv,
		currentBlockNumber: currentBlockNumber,
	}
}

func (c *TrainingClient) GetMetadata() (*TrainingMetadata, error) {
	metadata, err := c.GetTrainingMetadata(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

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

func newAuth(addr, msg string, blockNumber uint64, signature []byte) *AuthorizationDetails {
	return &AuthorizationDetails{
		CurrentBlock:  blockNumber,
		Message:       msg,
		Signature:     signature,
		SignerAddress: addr,
	}
}

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
