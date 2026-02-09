## Model Training API Guide

The SingularityNET SDK provides a Training API that allows you to train, manage, and monitor machine learning models through AI services. This guide covers the complete workflow from creating models to monitoring training progress.

## Table of Contents

1. [Introduction to Training API](#introduction-to-training-api)
2. [Model Training Concepts](#model-training-concepts)
3. [Prerequisites](#prerequisites)
4. [Complete Workflow Example](#complete-workflow-example)
5. [Working with Training Data](#working-with-training-data)
6. [Model Monitoring](#model-monitoring)
7. [Daemon Requirements](#daemon-requirements)
8. [Best Practices](#best-practices)

---

## Introduction to Training API

The Training API enables you to:
- Create and manage custom machine learning models
- Submit training data to services
- Monitor training progress
- Retrieve trained models for inference
- Update and retrain existing models

### When to Use Training API

**Use Training API when:**
- You need custom models for your specific data
- The pre-trained models don't fit your use case
- You want to fine-tune models on your domain
- You need continuous model improvement

**Use Pre-trained Models when:**
- General-purpose models suit your needs
- You don't have training data
- You need immediate results
- You want lower costs

---

## Model Training Concepts

### Key Concepts

**Model**: A trained machine learning model that can make predictions

**Training Job**: The process of training a model on your data

**Training Data**: Dataset used to train the model (features + labels)

**Model Metadata**: Information about the model (name, description, status, accuracy)

**Training Parameters**: Hyperparameters that control the training process

**Model Version**: Different iterations of the same model

### Training Workflow

```
1. Create Model → 2. Upload Data → 3. Start Training → 4. Monitor Progress → 5. Use Model
```

---

## Prerequisites

Before using the Training API, ensure you have:

* **ERC-20 wallet** with funds for service payments
* **Service with training enabled** - Not all services support training
* **Latest daemon** with training capabilities
* **Training data** in the correct format
* **FET tokens** for training costs (usually higher than inference)

### Checking Service Training Support

```go
package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	cfg := config.Config{
		RPCAddr: "wss://sepolia.infura.io/ws/v3/YOUR_PROJECT_ID",
		PrivateKey: "", // Can check without private key
		Debug:   true,
		Network: config.Sepolia,
	}

	snetSDK := sdk.NewSDK(&cfg)
	defer snetSDK.Close()

	service, err := snetSDK.NewServiceClient("ORG_ID", "SERVICE_ID", "default_group")
	if err != nil {
		log.Fatalf("Failed to create service client: %v", err)
	}
	defer service.Close()

	// Check if training is supported
	metadata, err := service.Training().GetMetadata()
	if err != nil {
		log.Printf("Training not supported: %v", err)
		return
	}

	fmt.Printf("Training supported!\n")
	fmt.Printf("Metadata: %v\n", metadata)
}
```

---

## Complete Workflow Example

### Step 1: Initialize SDK and Service

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	// Configure SDK for training operations
	cfg := config.Config{
		RPCAddr: "wss://sepolia.infura.io/ws/v3/YOUR_PROJECT_ID",
		PrivateKey: "YOUR_PRIVATE_KEY", // Required for training
		Debug:      true,
		Network:    config.Sepolia,
		Timeouts: config.Timeouts{
			GRPCUnary: 120 * time.Second, // Training operations may take longer
		},
	}

	snetSDK := sdk.NewSDK(&cfg)
	defer snetSDK.Close()

	// Connect to service with training support
	service, err := snetSDK.NewServiceClient("TRAINING_ORG", "TRAINING_SERVICE", "default_group")
	if err != nil {
		log.Fatalf("Failed to create service client: %v", err)
	}
	defer service.Close()

	// Set payment strategy (training usually requires payment)
	if err := service.SetPrePaidStrategy(); err != nil {
		log.Fatalf("Failed to set payment strategy: %v", err)
	}

	fmt.Println("✓ SDK initialized for training")
```

### Step 2: Create a New Model

```go
	// Create a new model
	modelName := "my_custom_classifier"
	modelDescription := "Custom image classifier for product categories"

	fmt.Printf("Creating model: %s\n", modelName)
	
	model, err := service.Training().CreateModel(modelName, modelDescription)
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	fmt.Printf("✓ Model created successfully!\n")
	fmt.Printf("Model ID: %s\n", model.ID)
	fmt.Printf("Model Name: %s\n", model.Name)
	fmt.Printf("Status: %s\n", model.Status)
```

### Step 3: Upload Training Data

```go
	// Prepare training data
	trainingData := map[string]interface{}{
		"dataset": "product_images",
		"samples": []map[string]interface{}{
			{
				"image": "base64_encoded_image_1",
				"label": "electronics",
			},
			{
				"image": "base64_encoded_image_2",
				"label": "clothing",
			},
			// ... more samples
		},
		"validation_split": 0.2,
	}

	dataJSON, err := json.Marshal(trainingData)
	if err != nil {
		log.Fatalf("Failed to marshal training data: %v", err)
	}

	fmt.Println("Uploading training data...")
	
	// Upload data (implementation depends on service)
	// This is a placeholder - actual method may vary
	uploadResp, err := service.CallWithJSON("UploadTrainingData", dataJSON)
	if err != nil {
		log.Fatalf("Failed to upload training data: %v", err)
	}

	fmt.Printf("✓ Training data uploaded: %s\n", uploadResp)
```

### Step 4: Start Training

```go
	// Configure training parameters
	trainingParams := map[string]interface{}{
		"model_id":       model.ID,
		"epochs":         50,
		"batch_size":     32,
		"learning_rate":  0.001,
		"optimizer":      "adam",
	}

	paramsJSON, err := json.Marshal(trainingParams)
	if err != nil {
		log.Fatalf("Failed to marshal training params: %v", err)
	}

	fmt.Println("Starting training job...")
	
	trainResp, err := service.CallWithJSON("StartTraining", paramsJSON)
	if err != nil {
		log.Fatalf("Failed to start training: %v", err)
	}

	var jobInfo map[string]interface{}
	if err := json.Unmarshal(trainResp, &jobInfo); err != nil {
		log.Fatalf("Failed to parse training response: %v", err)
	}

	jobID := jobInfo["job_id"].(string)
	fmt.Printf("✓ Training started! Job ID: %s\n", jobID)
```

### Step 5: Monitor Training Progress

```go
	// Monitor training progress
	fmt.Println("\nMonitoring training progress...")
	
	for {
		time.Sleep(10 * time.Second)

		// Get model status
		updatedModel, err := service.Training().GetModel(model.ID)
		if err != nil {
			log.Printf("Failed to get model status: %v", err)
			continue
		}

		fmt.Printf("Status: %s, Progress: %v%%\n", 
			updatedModel.Status, 
			updatedModel.TrainingProgress)

		// Check if training completed
		if updatedModel.Status == "completed" {
			fmt.Printf("\n✓ Training completed!\n")
			fmt.Printf("Model Accuracy: %.2f%%\n", updatedModel.Accuracy)
			break
		}

		if updatedModel.Status == "failed" {
			log.Fatalf("Training failed: %s", updatedModel.ErrorMessage)
		}
	}
```

### Step 6: Use Trained Model

```go
	// Use the trained model for inference
	inferenceInput := map[string]interface{}{
		"model_id": model.ID,
		"image":    "base64_encoded_test_image",
	}

	inputJSON, err := json.Marshal(inferenceInput)
	if err != nil {
		log.Fatalf("Failed to marshal input: %v", err)
	}

	fmt.Println("Running inference with trained model...")
	
	prediction, err := service.CallWithJSON("Predict", inputJSON)
	if err != nil {
		log.Fatalf("Failed to run inference: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(prediction, &result); err != nil {
		log.Fatalf("Failed to parse prediction: %v", err)
	}

	fmt.Printf("\n✓ Prediction: %s (confidence: %.2f%%)\n",
		result["label"],
		result["confidence"])
}
```

---

## Working with Training Data

### Data Format Requirements

Training data format varies by service type:

**Image Classification:**
```go
trainingData := map[string]interface{}{
	"images": []map[string]string{
		{"url": "https://example.com/image1.jpg", "label": "cat"},
		{"url": "https://example.com/image2.jpg", "label": "dog"},
	},
}
```

**Text Classification:**
```go
trainingData := map[string]interface{}{
	"texts": []map[string]string{
		{"text": "This is positive", "label": "positive"},
		{"text": "This is negative", "label": "negative"},
	},
}
```

**Time Series Forecasting:**
```go
trainingData := map[string]interface{}{
	"series": []float64{1.0, 2.0, 3.0, 4.0, 5.0},
	"timestamps": []string{"2024-01-01", "2024-01-02", ...},
}
```

### Data Validation

```go
func validateTrainingData(data interface{}) error {
	// Check data size
	dataJSON, _ := json.Marshal(data)
	if len(dataJSON) > 100*1024*1024 { // 100 MB limit
		return fmt.Errorf("training data too large: %d bytes", len(dataJSON))
	}

	// Check data structure (example for image classification)
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data format")
	}

	samples, ok := dataMap["samples"].([]interface{})
	if !ok || len(samples) == 0 {
		return fmt.Errorf("no training samples provided")
	}

	if len(samples) < 10 {
		return fmt.Errorf("insufficient training data: need at least 10 samples, got %d", len(samples))
	}

	return nil
}

// Usage
if err := validateTrainingData(trainingData); err != nil {
	log.Fatalf("Data validation failed: %v", err)
}
```

### Batch Data Upload

```go
func uploadDataInBatches(service *sdk.ServiceClient, data []interface{}, batchSize int) error {
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]
		batchData := map[string]interface{}{
			"batch_id": i / batchSize,
			"samples":  batch,
		}

		dataJSON, err := json.Marshal(batchData)
		if err != nil {
			return fmt.Errorf("failed to marshal batch: %w", err)
		}

		_, err = service.CallWithJSON("UploadBatch", dataJSON)
		if err != nil {
			return fmt.Errorf("failed to upload batch %d: %w", i/batchSize, err)
		}

		log.Printf("Uploaded batch %d/%d", i/batchSize+1, (len(data)+batchSize-1)/batchSize)
	}

	return nil
}
```

---

## Model Monitoring

### Real-Time Training Metrics

```go
type TrainingMonitor struct {
	service   *sdk.ServiceClient
	modelID   string
	interval  time.Duration
	stopChan  chan bool
}

func NewTrainingMonitor(service *sdk.ServiceClient, modelID string) *TrainingMonitor {
	return &TrainingMonitor{
		service:  service,
		modelID:  modelID,
		interval: 5 * time.Second,
		stopChan: make(chan bool),
	}
}

func (tm *TrainingMonitor) Start() {
	ticker := time.NewTicker(tm.interval)
	defer ticker.Stop()

	fmt.Println("=== Training Monitor Started ===")

	for {
		select {
		case <-ticker.C:
			tm.checkProgress()
		case <-tm.stopChan:
			fmt.Println("=== Training Monitor Stopped ===")
			return
		}
	}
}

func (tm *TrainingMonitor) checkProgress() {
	model, err := tm.service.Training().GetModel(tm.modelID)
	if err != nil {
		log.Printf("Failed to get model: %v", err)
		return
	}

	fmt.Printf("[%s] Status: %s | Progress: %v%% | Accuracy: %.2f%% | Loss: %.4f\n",
		time.Now().Format("15:04:05"),
		model.Status,
		model.TrainingProgress,
		model.Accuracy,
		model.Loss)

	if model.Status == "completed" || model.Status == "failed" {
		tm.Stop()
	}
}

func (tm *TrainingMonitor) Stop() {
	tm.stopChan <- true
}

// Usage
monitor := NewTrainingMonitor(service, modelID)
go monitor.Start()

// Monitor runs in background...
// Will stop automatically when training completes
```

### Model Performance Metrics

```go
func getModelMetrics(service *sdk.ServiceClient, modelID string) {
	model, err := service.Training().GetModel(modelID)
	if err != nil {
		log.Fatalf("Failed to get model: %v", err)
	}

	fmt.Println("\n=== Model Performance Metrics ===")
	fmt.Printf("Model ID:       %s\n", model.ID)
	fmt.Printf("Name:           %s\n", model.Name)
	fmt.Printf("Status:         %s\n", model.Status)
	fmt.Printf("Accuracy:       %.2f%%\n", model.Accuracy)
	fmt.Printf("Precision:      %.2f%%\n", model.Precision)
	fmt.Printf("Recall:         %.2f%%\n", model.Recall)
	fmt.Printf("F1 Score:       %.2f\n", model.F1Score)
	fmt.Printf("Training Time:  %v\n", model.TrainingDuration)
	fmt.Printf("Created:        %s\n", model.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated:        %s\n", model.UpdatedAt.Format(time.RFC3339))
	fmt.Println("================================")
}
```

### Managing Multiple Models

```go
func listAndCompareModels(service *sdk.ServiceClient) {
	models, err := service.Training().GetAllModels()
	if err != nil {
		log.Fatalf("Failed to get models: %v", err)
	}

	fmt.Printf("\n=== All Models (%d total) ===\n", len(models))
	fmt.Printf("%-20s %-15s %-10s %-10s\n", "Name", "Status", "Accuracy", "Created")
	fmt.Println(strings.Repeat("-", 60))

	for _, model := range models {
		fmt.Printf("%-20s %-15s %-10.2f%% %s\n",
			truncateString(model.Name, 20),
			model.Status,
			model.Accuracy,
			model.CreatedAt.Format("2006-01-02"))
	}

	// Find best model
	bestModel := findBestModel(models)
	if bestModel != nil {
		fmt.Printf("\n✓ Best Model: %s (Accuracy: %.2f%%)\n", bestModel.Name, bestModel.Accuracy)
	}
}

func findBestModel(models []Model) *Model {
	var best *Model
	maxAccuracy := 0.0

	for i := range models {
		if models[i].Status == "completed" && models[i].Accuracy > maxAccuracy {
			maxAccuracy = models[i].Accuracy
			best = &models[i]
		}
	}

	return best
}

func truncateString(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
```

---

## Daemon Requirements

Services must be running with a training-enabled daemon to support the Training API.

### Daemon Configuration

The service daemon must be configured with:

```json
{
  "training": {
    "enabled": true,
    "max_concurrent_jobs": 5,
    "data_storage": "/path/to/training/data",
    "model_storage": "/path/to/models",
    "gpu_enabled": true,
    "timeout": 3600
  }
}
```

### Checking Daemon Capabilities

```go
func checkDaemonTrainingSupport(service *sdk.ServiceClient) error {
	// Try to get training metadata
	metadata, err := service.Training().GetMetadata()
	if err != nil {
		return fmt.Errorf("daemon does not support training: %w", err)
	}

	fmt.Println("✓ Daemon supports training")
	fmt.Printf("Max concurrent jobs: %d\n", metadata.MaxJobs)
	fmt.Printf("Supported model types: %v\n", metadata.SupportedTypes)
	fmt.Printf("GPU available: %v\n", metadata.GPUAvailable)

	return nil
}
```

### Daemon Resource Requirements

**Minimum Requirements:**
- CPU: 4+ cores
- RAM: 8+ GB
- Storage: 50+ GB for models and data
- GPU: Optional but recommended for deep learning

**Recommended:**
- CPU: 8+ cores
- RAM: 16+ GB
- Storage: 500+ GB SSD
- GPU: NVIDIA with 8+ GB VRAM

---

## Best Practices

### 1. Model Naming Convention

```go
// Use descriptive, versioned names
modelName := fmt.Sprintf("product_classifier_v%d_%s",
	version,
	time.Now().Format("20060102"))

// Examples:
// "product_classifier_v1_20240115"
// "sentiment_model_v2_20240115"
```

### 2. Save Training Configuration

```go
type TrainingConfig struct {
	ModelName     string
	Dataset       string
	Epochs        int
	BatchSize     int
	LearningRate  float64
	CreatedAt     time.Time
}

func saveTrainingConfig(config TrainingConfig, filename string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// Usage
config := TrainingConfig{
	ModelName:    "my_model",
	Dataset:      "products_v1",
	Epochs:       50,
	BatchSize:    32,
	LearningRate: 0.001,
	CreatedAt:    time.Now(),
}

saveTrainingConfig(config, "training_config.json")
```

### 3. Implement Checkpointing

```go
// Save model checkpoints during long training
type Checkpoint struct {
	ModelID  string
	Epoch    int
	Accuracy float64
	Saved    time.Time
}

func saveCheckpoint(modelID string, epoch int, accuracy float64) {
	checkpoint := Checkpoint{
		ModelID:  modelID,
		Epoch:    epoch,
		Accuracy: accuracy,
		Saved:    time.Now(),
	}

	filename := fmt.Sprintf("checkpoint_%s_epoch%d.json", modelID, epoch)
	data, _ := json.Marshal(checkpoint)
	os.WriteFile(filename, data, 0644)

	log.Printf("Checkpoint saved: Epoch %d, Accuracy %.2f%%", epoch, accuracy)
}
```

### 4. Validate Before Training

```go
func validateBeforeTraining(service *sdk.ServiceClient, data interface{}) error {
	// Check service health
	if _, err := service.Healthcheck().GRPC(); err != nil {
		return fmt.Errorf("service unhealthy: %w", err)
	}

	// Validate data
	if err := validateTrainingData(data); err != nil {
		return fmt.Errorf("invalid training data: %w", err)
	}

	// Check training support
	if _, err := service.Training().GetMetadata(); err != nil {
		return fmt.Errorf("training not supported: %w", err)
	}

	// Check payment channel balance
	// (implementation depends on SDK)

	return nil
}
```

### 5. Error Handling for Long Operations

```go
func trainWithErrorHandling(service *sdk.ServiceClient, modelName, description string) error {
	// Set longer timeout for training operations
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	// Create model with retry
	var model *Model
	var err error
	
	for attempt := 1; attempt <= 3; attempt++ {
		model, err = service.Training().CreateModel(modelName, description)
		if err == nil {
			break
		}
		
		log.Printf("Attempt %d failed: %v", attempt, err)
		time.Sleep(time.Duration(attempt) * 10 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to create model after 3 attempts: %w", err)
	}

	// Monitor with context cancellation
	done := make(chan error, 1)
	
	go func() {
		done <- monitorTraining(service, model.ID)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("training timed out: %w", ctx.Err())
	}
}
```

### 6. Cost Estimation

```go
func estimateTrainingCost(service *sdk.ServiceClient, samples int, epochs int) {
	// Get service pricing
	metadata := service.GetServiceMetadata()
	
	// Estimate based on samples and epochs
	// This is service-specific
	estimatedCalls := samples * epochs / 100 // Example calculation
	pricePerCall := 100 // cogs
	
	totalCogs := estimatedCalls * pricePerCall
	totalFET := float64(totalCogs) / 100000000 // Convert cogs to FET

	fmt.Printf("\n=== Cost Estimation ===\n")
	fmt.Printf("Samples:       %d\n", samples)
	fmt.Printf("Epochs:        %d\n", epochs)
	fmt.Printf("Est. Calls:    %d\n", estimatedCalls)
	fmt.Printf("Cost:          %.6f FET\n", totalFET)
	fmt.Printf("======================\n")
}
```

### 7. Model Versioning

```go
func createVersionedModel(service *sdk.ServiceClient, baseName string) (*Model, error) {
	// Get existing models
	models, err := service.Training().GetAllModels()
	if err != nil {
		return nil, err
	}

	// Find latest version
	version := 1
	for _, model := range models {
		if strings.HasPrefix(model.Name, baseName) {
			// Extract version number
			// Increment if found
			version++
		}
	}

	modelName := fmt.Sprintf("%s_v%d", baseName, version)
	description := fmt.Sprintf("Version %d of %s model", version, baseName)

	return service.Training().CreateModel(modelName, description)
}
```

### Best Practices Summary

1. **Use descriptive model names** with versions and dates
2. **Validate data** before starting training
3. **Monitor training progress** with regular status checks
4. **Save checkpoints** for long training jobs
5. **Estimate costs** before training
6. **Handle errors gracefully** with retries and timeouts
7. **Version your models** for easy comparison
8. **Document training parameters** for reproducibility
9. **Test on small datasets** before full training
10. **Clean up old models** to save storage and costs