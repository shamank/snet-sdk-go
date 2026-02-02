## Simple service call

Step 1. Prerequisites

* erc20 wallet
This guide assumes you've got a erc-20 wallet (you will need to fill private key in code or use environment param).

* OrgID & ServiceID for calls 
You can find test services on the marketplace: https://testnet.marketplace.singularitynet.io/

* Golang 1.24+

### Install sdk in your project
```go get -u github.com/shamank/snet-sdk-go```


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

	inputJson := []byte(`{"a": 7, "b":2}`)
	// calling with JSON input
	resp, err := service.CallWithJSON("METHOD_NAME", inputJson)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("\nResponse from service: %v \n raw: %v\n", string(resp), resp)

	service.Close()
	snetSDK.Close()
}

```


**Notes**

If your call fails with authorization/payment errors, pick a suitable payment strategy (see choose_strategy.md).
Prefer env vars for secrets. Avoid committing private keys.

After executing this code, you should have the following result in console:

**Response** from service: 14 \
Note: You can also use other values and methods to call in that service. Moreover, you can change id of the organization and service to call other services and chain id or RPC endpoint to call services on mainnet.



Also you can call with map:
```go
respMap, err := service.CallWithMap("METHOD_NAME", map[string]any{"input_a":"1234"})
if err != nil {
	log.Fatalln(err)
}

fmt.Println(respMap["output_in_map"])
```