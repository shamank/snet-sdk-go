// Package training provides utilities for authenticating and managing model training
// requests to SingularityNET services.
//
// Some SingularityNET services support model training, allowing users to train custom
// AI models on their data. This package implements the authentication mechanism required
// for training API calls and provides a client for training operations.
//
// # Training API Overview
//
// The training API enables:
//   - Creating custom models
//   - Listing available models
//   - Retrieving model details
//   - Updating model configurations
//   - Deleting models
//   - Uploading and validating training data
//   - Validating model pricing
//   - Training models
//   - Querying training methods and metadata
//   - Monitoring training status
//
// # Authentication Mechanism
//
// Training methods require special authentication to prevent unauthorized access and
// ensure billing integrity. The package implements Ethereum-style message signing:
//
//  1. Construct message: concat(methodName, signerAddress, blockNumber)
//  2. Hash with keccak256
//  3. Sign with Ethereum personal_sign format (EIP-191)
//  4. Attach signature to gRPC metadata
//
// The daemon validates:
//   - Signature matches the message
//   - Signer address is authorized
//   - Block number is recent (prevents replay attacks)
//
// # Usage Example
//
//	import (
//		"github.com/singnet/snet-sdk-go/pkg/sdk"
//	)
//
//	// Create service client
//	service, err := snetSDK.NewServiceClient("org", "service", "group")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Access training API
//	trainingClient := service.Training()
//
//	// Get training metadata
//	metadata, err := trainingClient.GetMetadata()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Available training methods: %v\n", metadata.TrainingMethods)
//
// # Creating Models
//
// Create a new model for training:
//
//	modelParams := &training.ModelParams{
//		Name:        "my-model",
//		Description: "Custom image classifier",
//	}
//	model, err := trainingClient.CreateModel(modelParams)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Model created with ID: %s\n", model.ID)
//
// # Listing Models
//
// Retrieve all models with optional filters:
//
//	filters := training.GetAllModelsFilters{
//		// Add filters if needed
//	}
//	modelsResponse, err := trainingClient.GetAllModels(filters)
//	if err != nil {
//		log.Fatal(err)
//	}
//	for _, model := range modelsResponse.Models {
//		fmt.Printf("Model %s: %s\n", model.ID, model.Name)
//	}
//
// # Getting Model Details
//
// Retrieve specific model information:
//
//	model, err := trainingClient.GetModel("model-id-123")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Model: %s, Status: %s\n", model.Name, model.Status)
//
// # Training Workflow
//
// Complete workflow for training a model:
//
//	// 1. Create model
//	modelParams := &training.ModelParams{
//		Name:        "image-classifier",
//		Description: "Custom classifier",
//	}
//	model, err := trainingClient.CreateModel(modelParams)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 2. Upload and validate training data
//	uploadReq := &training.UploadValidateRequest{
//		ModelID:         model.ID,
//		TrainingDataLink: "ipfs://...",
//	}
//	err = trainingClient.UploadAndValidate(uploadReq)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 3. Validate model price
//	price, err := trainingClient.ValidateModelPrice(model.ID, "ipfs://...")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Training will cost: %d cogs\n", price)
//
//	// 4. Start training
//	status, err := trainingClient.TrainModel(model.ID)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Training started: %v\n", status)
//
//	// 5. Monitor training progress
//	ticker := time.NewTicker(10 * time.Second)
//	for range ticker.C {
//		model, _ := trainingClient.GetModel(model.ID)
//		fmt.Printf("Training status: %s\n", model.Status)
//		if model.Status == "completed" {
//			break
//		}
//	}
//
// # Security Considerations
//
// Block Number Freshness:
//
// The blockNumber parameter provides freshness guarantees. The daemon can reject
// signatures that reference blocks too far in the past, preventing replay attacks:
//
//	// Get current block
//	currentBlock, err := ethClient.BlockNumber(ctx)
//
//	// Create auth with recent block
//	auth, err := training.CreateAuth(methodName, currentBlock, privateKey)
//
// Signature Format:
//
// Uses Ethereum personal_sign format with "\x19Ethereum Signed Message:\n" prefix,
// compatible with standard Ethereum signing tools and wallets.
//
// # Daemon Requirements
//
// Services must enable training support in their daemon configuration:
//
//	{
//		"training": {
//			"enabled": true,
//			"methods": ["create_model", "get_model", "list_models"],
//			"authorized_addresses": ["0x..."]
//		}
//	}
//
// Not all services support training. Check service metadata:
//
//	metadata := service.GetServiceMetadata()
//	if len(metadata.TrainingMethods) > 0 {
//		fmt.Println("Training supported")
//	} else {
//		fmt.Println("Training not available")
//	}
//
// # Error Handling
//
// Common training errors:
//   - Training not enabled: Service doesn't support training
//   - Unauthorized: Signer address not in whitelist
//   - Invalid signature: Authentication failed
//   - Stale block: Block number too old
//   - Model not found: Invalid model ID
//
// Example:
//
//	model, err := trainingClient.CreateModel(modelParams)
//	if err != nil {
//		if strings.Contains(err.Error(), "not enabled") {
//			return fmt.Errorf("service doesn't support training")
//		}
//		if strings.Contains(err.Error(), "unauthorized") {
//			return fmt.Errorf("address not authorized for training")
//		}
//		return err
//	}
//
// # Best Practices
//
// 1. Verify training is supported before attempting operations
// 2. Use recent block numbers for authentication
// 3. Handle training-specific errors gracefully
// 4. Monitor model status asynchronously
// 5. Cache model metadata to reduce API calls
// 6. Implement proper cleanup for failed training jobs
// 7. Consider cost implications of training operations
//
// # Thread Safety
//
// The training client is safe for concurrent use. Multiple goroutines can
// make training API calls simultaneously.
//
// # See Also
//
//   - sdk.Service.Training() to access training client
//   - examples/training for complete training example
//   - wiki/training.md for detailed training guide
package training
