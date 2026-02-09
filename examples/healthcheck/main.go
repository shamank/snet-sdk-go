package main

import (
	"fmt"
	"log"

	"github.com/shamank/snet-sdk-go/pkg/config"
	"github.com/shamank/snet-sdk-go/pkg/sdk"
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

	// Do http healthcheck
	resp, err1 := service.Healthcheck().HTTP()
	if err1 != nil {
		log.Println(err1)
	}
	fmt.Printf("\nHeartbeatHTTP: %v", resp)

	// Do WebGRPC healthcheck
	respWebGrpc, err2 := service.Healthcheck().WebGRPC()
	if err2 != nil {
		log.Println(err2)
	}
	fmt.Printf("\nHeartbeatWebGRPC: %v", respWebGrpc)

	// Do GRPC healthcheck
	respGrpc, err3 := service.Healthcheck().GRPC()
	if err3 != nil {
		log.Println(err3)
	}
	fmt.Printf("\nHeartbeatGRPC: %v", respGrpc)
}
