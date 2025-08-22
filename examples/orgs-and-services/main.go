package main

import (
	"fmt"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {

	// new config
	c := config.Config{
		RPCAddr: "wss://sepolia.infura.io/ws/v3/",
		// You can unfill the private key if you do not want to call the services
		PrivateKey: "",
		Debug:      true,
	}

	// creating a new SDK core
	snetSDK := sdk.NewSDK(&c)

	//how to get orgs list:
	fmt.Println(snetSDK.GetOrganizations())

	//how to get services by orgID
	fmt.Println(snetSDK.GetServices(""))

	snetSDK.Close()

	//service.CallWithJSON("test", []byte(`"test": "1"`))
	//service.CallWithProto("test", nil)
}
