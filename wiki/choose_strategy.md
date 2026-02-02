## Choosing strategy for service calls

Step 1. Prerequisites

* ERC-20 wallet
* Service with free-call support

### Write some code

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
		RPCAddr:    "https://sepolia.infura.io/v3/{PROJECT_ID}",
		PrivateKey: "",
		Debug:      true,
		Network:    config.Sepolia,
	}

	// creating a new SDK core
	snetSDK := sdk.NewSDK(&c)

	// creating service client
	service, err := snetSDK.NewServiceClient("ORG", "SERVICE", "default_group")
	if err != nil {
		log.Fatalln(err)
	}

	// choosing pay strategy
	err = service.SetPaidPaymentStrategy()
	if err != nil {
		log.Println("SetPaymentStrategy: ", err)
	}

	inputJson := []byte(`{"a": 7, "b":2}`)

	resp, err := service.CallWithJSON("METHOD_NAME", inputJson)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("\nResponse from service: %v \n raw: %v\n", string(resp), resp)
}
```

Also you can change strategy in runtime:
```go
// choosing pay strategy
err = service.SetPaidPaymentStrategy()
if err != nil {
    log.Println("SetPaymentStrategy: ", err)
}

inputJson := []byte(`{"a": 7, "b":2}`)

resp, err := service.CallWithJSON("METHOD_NAME", inputJson)
if err != nil {
    log.Fatalln(err)
}

// changing strategy again to pre paid:
err = service.SetPrePaidStrategy()
if err != nil {
   log.Println("SetPaymentStrategy: ", err)
}

```