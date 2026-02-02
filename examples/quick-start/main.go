package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
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
		log.Fatalln(err)
	}

	// Prepare input data
	inputJson := []byte(`{"a": 7, "b":2}`)

	// Call service with JSON input
	resp, err := service.CallWithJSON("your_method_name", inputJson)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("\nResponse from service: %v \n raw: %v\n", string(resp), resp)

	// Close connections
	service.Close()
	snetSDK.Close()
}
