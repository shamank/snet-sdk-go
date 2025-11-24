package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	// new config
	c := config.Config{
		RPCAddr: "https://sepolia.infura.io/v3/",
		// You can unfill the private key if you do not want to call the services
		PrivateKey: "",
		Debug:      true,
		Network:    config.Sepolia,
	}

	// creating a new SDK core
	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("", "", "default_group")
	if err != nil {
		log.Println(err)
		return
	}

	err = service.SetFreePaymentStrategy()
	if err != nil {
		log.Println("SetFreePaymentStrategy: ", err)
		return
	}

	available, err := service.GetFreeCallsAvailable()
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nFree calls available: %v", available)

	resp, err := service.CallWithMap("", map[string]any{"text": "test"})
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nResponse from service: %v", resp)

	service.Close()
	snetSDK.Close()
}
