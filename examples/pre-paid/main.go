package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	// this is an example how to use SDK

	// new config
	c := config.Config{
		RPCAddr:    "wss://sepolia.infura.io/ws/v3/",
		PrivateKey: "TODO",
		Debug:      true,
	}

	// creating new SDK core
	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("", "", "default_group")
	if err != nil {
		log.Println(err)
		return
	}

	err = service.SetPrePaidPaymentStrategy(111)
	if err != nil {
		log.Println("SetFreePaymentStrategy: ", err)
		return
	}

	resp, err := service.CallWithMap("", map[string]any{"text": "1"})
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nResponse from service: %v", resp)

	service.Close()
	snetSDK.Close()
}
