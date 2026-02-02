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
		log.Println(err)
		return
	}

	// Set pre-paid payment strategy
	err = service.SetPrePaidPaymentStrategy(100)
	if err != nil {
		log.Println(err)
		return
	}

	// Call service with map input
	resp, err := service.CallWithMap("your_method_name", map[string]any{"text": "test"})
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nResponse from service: %v", resp)

	// Close connections
	service.Close()
	snetSDK.Close()
}
