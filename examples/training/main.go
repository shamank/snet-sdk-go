package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
	"github.com/singnet/snet-sdk-go/pkg/training"
)

func main() {
	// Create config
	c := config.Config{
		RPCAddr:    "wss://sepolia.infura.io/ws/v3/your_infura_project_id",
		PrivateKey: "your_private_key",
		Debug:      true,
		Network:    config.Sepolia,
	}

	// Create SDK instance
	snetSDK := sdk.NewSDK(&c)

	// Create service client
	service, err := snetSDK.NewServiceClient("your_org_id", "your_service_id", "default_group")
	if err != nil {
		log.Println(err)
		return
	}

	// Get training metadata
	metadata, err := service.Training().GetMetadata()
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nmetadata: %v", metadata)

	// Create new model
	m, err := service.Training().CreateModel(&training.ModelParams{
		Name:            "test model",
		Description:     "this is just test model",
		GrpcMethodName:  "test_method_name",
		GrpcServiceName: "example_service",
		IsPublic:        true,
	})
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nmetadata: %v", m)

	// Get all models
	models, err := service.Training().GetAllModels(training.GetAllModelsFilters{
		PageSize: 100,
		Page:     0,
	})
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nmodels: %v", models)

	model, err := service.Training().GetModel("1")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nmodel: %v", model)

	// Close connections
	service.Close()
	snetSDK.Close()
}
