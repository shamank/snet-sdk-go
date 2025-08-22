package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	c := config.Config{
		RPCAddr: "TODO",
	}

	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("", "", "default_group")
	if err != nil {
		log.Println(err)
		return
	}

	resp, err2 := service.Heartbeat()
	if err2 != nil {
		log.Println(err2)
		return
	}
	fmt.Printf("\nresp: %v", resp)
}
