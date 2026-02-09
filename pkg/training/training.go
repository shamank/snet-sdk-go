//go:generate protoc --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative --go_out=. --go-grpc_out=. --proto_path=. training.proto training_daemon.proto

// Package training provides a client for interacting with SingularityNET training daemons.
// It allows users to manage AI models, retrieve training metadata, and perform model-related operations.
package training

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shamank/snet-sdk-go/pkg/blockchain"
	"github.com/shamank/snet-sdk-go/pkg/grpc"
	"github.com/shamank/snet-sdk-go/pkg/payment"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

// Client defines the interface for training-related operations.
type Client interface {
	// GetMetadata retrieves training daemon metadata
	GetMetadata() (*TrainingMetadata, error)
	// GetMethodMetadata retrieves method metadata
	GetMethodMetadata(request *MethodMetadataRequest) (*MethodMetadata, error)
	// GetAllModels retrieves all available models
	GetAllModels(r GetAllModelsFilters) (*ModelsResponse, error)
	// GetModel retrieves a specific model by ID
	GetModel(modelId string) (*ModelResponse, error)
	// CreateModel creates a new model with the given name and description
	CreateModel(model *ModelParams) (*ModelResponse, error)

	DeleteModel(modelID string) (Status, error)

	UpdateModel(r *UpdateModelRequest) (*ModelResponse, error)
	UploadAndValidate(r *UploadValidateRequest) error

	ValidateModelPrice(modelID, TrainingDataLink string) (price uint64, err error)
	ValidateModel(modelID, TrainingDataLink string) (Status, error)

	TrainModelPrice(modelID string) (uint64, error)
	TrainModel(modelID string) (Status, error)
}

// TrainingClient is the concrete implementation of the Client interface
// for training operations.
type TrainingClient struct {
	DaemonClient
	timeout            time.Duration
	streamTimeout      time.Duration
	privateKey         *ecdsa.PrivateKey
	currentBlockNumber func() (*big.Int, error)
	OrgID              string
	SrvID              string
	GroupID            string
	strat              payment.Strategy
}

// NewTrainingClient creates a new training client with the provided gRPC client,
// private key for signing requests, and a function to retrieve the current block number.
func NewTrainingClient(orgID, srvID, groupID string, client *grpc.Client, priv *ecdsa.PrivateKey, timeout, streamTimeout time.Duration, currentBlockNumber func() (*big.Int, error), strat payment.Strategy) *TrainingClient {
	return &TrainingClient{
		DaemonClient:       NewDaemonClient(client.GRPC),
		timeout:            timeout,
		streamTimeout:      streamTimeout,
		privateKey:         priv,
		currentBlockNumber: currentBlockNumber,
		OrgID:              orgID,
		SrvID:              srvID,
		GroupID:            groupID,
		strat:              strat,
	}
}

func (c *TrainingClient) GetMethodMetadata(request *MethodMetadataRequest) (*MethodMetadata, error) {
	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	metadata, err := c.DaemonClient.GetMethodMetadata(ctx, request)
	if err != nil {
		return nil, err
	}
	return metadata, err
}

type UploadValidateRequest struct {
	ModelID     string
	ZipPath     string
	PriceInCogs *big.Int
	BatchSize   uint64
}

func (c *TrainingClient) UploadAndValidate(r *UploadValidateRequest) error {
	if r == nil {
		return errors.New("nil request")
	}
	if r.ModelID == "" {
		return errors.New("ModelID is required")
	}
	if r.ZipPath == "" {
		return errors.New("ZipPath is required")
	}

	// stream timeout
	ctx, cancel := c.withTimeout(context.Background(), c.streamTimeout)
	defer cancel()

	err := c.strat.Refresh(ctx)
	if err != nil {
		return err
	}

	ctx = c.strat.GRPCMetadata(ctx)
	ctx = metadata.AppendToOutgoingContext(ctx, "snet-payment-type", "train-call", "snet-train-model-id", r.ModelID)

	block, err := c.currentBlockNumber()
	if err != nil {
		return err
	}
	sig, err := getSignature("upload_and_validate", block, c.privateKey)
	if err != nil {
		return err
	}
	auth := newAuth(
		blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(),
		"upload_and_validate",
		block.Uint64(),
		sig,
	)

	f, err := os.Open(r.ZipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat zip: %w", err)
	}
	fileSize := uint64(st.Size())
	if fileSize == 0 {
		return errors.New("zip file is empty")
	}
	fileName := filepath.Base(r.ZipPath)

	batchSize := r.BatchSize
	if batchSize == 0 {
		batchSize = 1024 * 1024
	}

	batchCount := fileSize / batchSize
	if fileSize%batchSize != 0 {
		batchCount++
	}

	stream, err := c.DaemonClient.UploadAndValidate(ctx)
	if err != nil {
		return fmt.Errorf("open upload stream: %w", err)
	}

	buf := make([]byte, batchSize)
	var batchNumber uint64 = 1

	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			input := &UploadInput{
				ModelId:     r.ModelID,
				Data:        buf[:n],
				FileName:    fileName,
				FileSize:    fileSize,
				BatchSize:   batchSize,
				BatchNumber: batchNumber,
				BatchCount:  batchCount,
			}

			msg := &UploadAndValidateRequest{
				Authorization: auth,
				UploadInput:   input,
			}

			if err := stream.Send(msg); err != nil {
				_ = stream.CloseSend()
				return fmt.Errorf("send batch %d/%d: %w", batchNumber, batchCount, err)
			}

			batchNumber++
		}

		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			_ = stream.CloseSend()
			return fmt.Errorf("read zip: %w", readErr)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("close and recv: %w", err)
	}
	if resp == nil {
		return errors.New("nil response")
	}
	if resp.Status == Status_ERRORED {
		return fmt.Errorf("upload_and_validate failed: status=%v", resp.Status)
	}

	return nil
}

func (c *TrainingClient) ValidateModelPrice(modelID, TrainingDataLink string) (price uint64, err error) {
	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	block, err := c.currentBlockNumber()
	if err != nil {
		return 0, err
	}

	signature, err := getSignature("validate_model_price", block, c.privateKey)
	if err != nil {
		return 0, err
	}

	r := &AuthValidateRequest{
		Authorization:    newAuth(blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(), "validate_model_price", block.Uint64(), signature),
		ModelId:          modelID,
		TrainingDataLink: TrainingDataLink,
	}

	resp, err := c.DaemonClient.ValidateModelPrice(ctx, r)
	if err != nil {
		return price, err
	}
	return resp.Price, nil
}

func (c *TrainingClient) ValidateModel(modelID, TrainingDataLink string) (Status, error) {
	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	block, err := c.currentBlockNumber()
	if err != nil {
		return Status_ERRORED, err
	}

	signature, err := getSignature("validate_model", block, c.privateKey)
	if err != nil {
		return Status_ERRORED, err
	}

	r := &AuthValidateRequest{
		Authorization:    newAuth(blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(), "validate_model", block.Uint64(), signature),
		ModelId:          modelID,
		TrainingDataLink: TrainingDataLink,
	}

	status, err := c.DaemonClient.ValidateModel(ctx, r)
	if err != nil {
		return Status_ERRORED, err
	}
	return status.Status, err
}

func (c *TrainingClient) TrainModelPrice(modelID string) (uint64, error) {
	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	block, err := c.currentBlockNumber()
	if err != nil {
		return 0, err
	}

	signature, err := getSignature("train_model_price", block, c.privateKey)
	if err != nil {
		return 0, err
	}

	r := &CommonRequest{
		Authorization: newAuth(blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(), "train_model_price", block.Uint64(), signature),
		ModelId:       modelID,
	}

	resp, err := c.DaemonClient.TrainModelPrice(ctx, r)
	if err != nil {
		return 0, err
	}
	return resp.Price, err
}

func (c *TrainingClient) TrainModel(modelID string) (Status, error) {
	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	block, err := c.currentBlockNumber()
	if err != nil {
		return Status_ERRORED, err
	}

	signature, err := getSignature("train_model", block, c.privateKey)
	if err != nil {
		return Status_ERRORED, err
	}

	r := &CommonRequest{
		Authorization: newAuth(blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(), "train_model", block.Uint64(), signature),
		ModelId:       modelID,
	}

	status, err := c.DaemonClient.TrainModel(ctx, r)
	if err != nil {
		return Status_ERRORED, err
	}
	return status.Status, err
}

func (c *TrainingClient) DeleteModel(modelID string) (Status, error) {
	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	block, err := c.currentBlockNumber()
	if err != nil {
		return Status_ERRORED, err
	}

	signature, err := getSignature("delete_model", block, c.privateKey)
	if err != nil {
		return Status_ERRORED, err
	}

	r := &CommonRequest{
		Authorization: newAuth(blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(), "delete_model", block.Uint64(), signature),
		ModelId:       modelID,
	}

	status, err := c.DaemonClient.DeleteModel(ctx, r)
	if err != nil {
		return Status_ERRORED, err
	}
	return status.Status, err
}

func (c *TrainingClient) UpdateModel(request *UpdateModelRequest) (*ModelResponse, error) {

	block, err := c.currentBlockNumber()
	if err != nil {
		return nil, err
	}
	signature, err := getSignature("update_model", block, c.privateKey)
	if err != nil {
		return nil, err
	}

	var req = &UpdateModelRequest{
		Authorization: newAuth(blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(), "update_model", block.Uint64(), signature),
		ModelId:       request.ModelId,
		ModelName:     request.ModelName,
		Description:   request.Description,
		AddressList:   request.AddressList,
	}

	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.DaemonClient.UpdateModel(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetMetadata retrieves training daemon metadata including supported operations.
func (c *TrainingClient) GetMetadata() (*TrainingMetadata, error) {
	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	metadata, err := c.GetTrainingMetadata(ctx, nil)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

type GetAllModelsFilters struct {
	Statuses         []Status
	IsPublic         *bool
	GrpcMethodName   string
	GrpcServiceName  string
	Name             string
	CreatedByAddress string
	PageSize         uint64
	Page             uint64
}

// GetAllModels retrieves all available models from the training daemon.
// The request is authenticated with the client's private key.
func (c *TrainingClient) GetAllModels(r GetAllModelsFilters) (*ModelsResponse, error) {

	block, err := c.currentBlockNumber()
	if err != nil {
		return nil, err
	}
	signature, err := getSignature("get_all_models", block, c.privateKey)
	if err != nil {
		return nil, err
	}

	if r.PageSize == 0 {
		r.PageSize = 100
	}

	var allModelsReq = &AllModelsRequest{
		Authorization:    newAuth(blockchain.GetAddressFromPrivateKeyECDSA(c.privateKey).Hex(), "get_all_models", block.Uint64(), signature),
		Statuses:         r.Statuses,
		IsPublic:         r.IsPublic,
		GrpcMethodName:   r.GrpcMethodName,
		GrpcServiceName:  r.GrpcServiceName,
		Name:             r.Name,
		CreatedByAddress: r.CreatedByAddress,
		PageSize:         r.PageSize,
		Page:             r.Page,
	}

	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.DaemonClient.GetAllModels(ctx, allModelsReq)
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

	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.DaemonClient.GetModel(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type ModelParams struct {
	Name            string
	Description     string
	GrpcMethodName  string
	GrpcServiceName string
	// List of all addresses that will have access to this model
	AddressList []string
	// Set this to true if you want your model to be accessible by other AI consumers
	IsPublic bool
}

// CreateModel creates a new model with the specified name and description.
// The request is authenticated with the client's private key.
func (c *TrainingClient) CreateModel(r *ModelParams) (*ModelResponse, error) {

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
			Name:            r.Name,
			Description:     r.Description,
			GrpcMethodName:  r.GrpcMethodName,
			GrpcServiceName: r.GrpcServiceName,
			AddressList:     r.AddressList,
			IsPublic:        r.IsPublic,
			OrganizationId:  c.OrgID,
			ServiceId:       c.SrvID,
			GroupId:         c.GroupID,
		},
	}

	ctx, cancel := c.withTimeout(context.Background(), c.timeout)
	defer cancel()

	resp, err := c.DaemonClient.CreateModel(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// newAuth creates authorization details for a training daemon request.
//
// Parameters:
//   - addr: signer's Ethereum address in hex format
//   - msg: method name being called (used in signature message)
//   - blockNumber: current block number for freshness verification
//   - signature: computed signature bytes
//
// Returns authorization details to include in the training API request.
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
//
// Message structure:
//
//	concat(methodName, signerAddress, blockNumber)
//
// The message is then hashed and signed using Ethereum personal sign format:
//
//	keccak256("\x19Ethereum Signed Message:\n32" || keccak256(message))
//
// Parameters:
//   - methodName: name of the training method being called
//   - blockNumber: current block number for freshness verification
//   - privateKey: ECDSA private key for signing
//
// Returns:
//   - signature: 65-byte signature (R||S||V)
//   - error: if signing fails
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

func (c *TrainingClient) withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return ctx, func() {}
	}
	// if ctx already has a deadline, do not expand it
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
