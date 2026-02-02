## Healthcheck


```go

package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	c := config.Config{
		RPCAddr:      "https://sepolia.infura.io/v3/",
		Debug:        true,
		RegistryAddr: "",
	}

	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("orgID", "serviceID", "default_group")
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

```

