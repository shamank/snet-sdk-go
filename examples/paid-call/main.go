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
		RPCAddr:    "https://sepolia.infura.io/v3/",
		PrivateKey: "",
		Debug:      true,
		Network:    config.Sepolia,
	}

	// creating new SDK core
	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("", "", "default_group")
	if err != nil {
		log.Println(err)
		return
	}

	err = service.SetPaidPaymentStrategy()
	if err != nil {
		log.Println("SetPaymentStrategy: ", err)
		return
	}

	resp, err := service.CallWithJSON("", []byte(`{"text"":"test"}`))
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nResponse from service: %v", string(resp))

	service.Close()
	snetSDK.Close()
}
