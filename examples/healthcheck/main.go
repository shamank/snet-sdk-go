package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	c := config.Config{
		RPCAddr:    "wss://sepolia.infura.io/ws/v3/",
		PrivateKey: "",
		Debug:      true,
	}

	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("", "", "default_group")
	if err != nil {
		log.Println(err)
		return
	}

	resp, err1 := service.Healthcheck().HTTP()
	if err1 != nil {
		log.Println(err1)
	}
	fmt.Printf("\nHeartbeatHTTP: %v", resp)
	respWebGrpc, err2 := service.Healthcheck().WebGRPC()
	if err2 != nil {
		log.Println(err2)
	}
	fmt.Printf("\nHeartbeatWebGRPC: %v", respWebGrpc)
	respGrpc, err3 := service.Healthcheck().WebGRPC()
	if err3 != nil {
		log.Println(err3)
	}
	fmt.Printf("\nHeartbeatGRPC: %v", respGrpc)
}
