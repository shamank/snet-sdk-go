Step 1. Prerequisites

* erc20 wallet
* Latest Daemon with training enabled

```go

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
	}

	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("", "", "default_group")
	if err != nil {
		log.Println(err)
		return
	}

	metadata, err := service.Training().GetMetadata()
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nmetadata: %v", metadata)

	m, err := service.Training().CreateModel("test", "test")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("\nmetadata: %v", m)

	models, err := service.Training().GetAllModels()
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

	service.Close()
	snetSDK.Close()
}

```