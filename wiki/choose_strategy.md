## Choosing Payment Strategy for Service Calls

Payment strategies determine how you pay for AI service calls on the SingularityNET platform. The SDK supports three payment models: free calls, paid calls, and pre-paid channels. Choose the strategy that best fits your usage patterns and budget requirements.

## Payment Strategy Comparison

| Strategy | Description                               | Use Case | Requirements                                         |
|----------|-------------------------------------------|----------|------------------------------------------------------|
| **Free Call** | Uses free call tokens provided by service | Testing, demos, limited usage | Service with free-call support                       |
| **Paid** | Pay-per-call using FET tokens             | Occasional usage, unpredictable demand | ERC-20 wallet with FET tokens                        |
| **Pre-Paid** | Deposit tokens to a payment channel       | High volume, regular usage, lower fees | ERC-20 wallet with FET tokens, payment channel setup |

## Requirements for Each Strategy

### Free Call Strategy
- Service must support free calls
- No wallet funding required
- Limited number of calls per user/service

### Paid Strategy
- ERC-20 compatible wallet
- FET tokens for service payment
- Gas tokens (ETH) for transaction fees
- Higher per-call costs due to blockchain transactions

### Pre-Paid Strategy
- ERC-20 compatible wallet
- FET tokens deposited in payment channel
- Initial channel opening transaction
- Lower per-call costs (off-chain transactions)
- Best for sustained usage

## Prerequisites

* ERC-20 wallet
* FET tokens (for paid/pre-paid strategies)
* Service with free-call support (for free call strategy)

## Basic Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/shamank/snet-sdk-go/pkg/config"
	"github.com/shamank/snet-sdk-go/pkg/sdk"
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

	// Set payment strategy - choose one based on your needs
	// Option 1: Paid strategy (pay-per-call)
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

## Switching Payment Strategies at Runtime

You can dynamically change payment strategies during execution based on your needs. This is useful when you want to:
- Switch from testing (free calls) to production (paid/pre-paid)
- Optimize costs by using pre-paid channels for bulk operations
- Fall back to different payment methods if one fails

### Example: Switching from Paid to Pre-Paid

```go
// Start with paid strategy for initial calls
err = service.SetPaidPaymentStrategy()
if err != nil {
    log.Println("SetPaymentStrategy: ", err)
}

inputJson := []byte(`{"a": 7, "b":2}`)

resp, err := service.CallWithJSON("METHOD_NAME", inputJson)
if err != nil {
    log.Fatalln(err)
}

// Switch to pre-paid strategy for better rates on subsequent calls
// This is more cost-effective for multiple calls
err = service.SetPrePaidStrategy()
if err != nil {
   log.Println("SetPaymentStrategy: ", err)
}

// Continue making calls with the new strategy
resp2, err := service.CallWithJSON("METHOD_NAME", inputJson)
if err != nil {
    log.Fatalln(err)
}
```

### Example: Using Free Call Strategy

```go
// Use free calls for testing (if service supports it)
err = service.SetFreeCallPaymentStrategy()
if err != nil {
    log.Println("SetPaymentStrategy: ", err)
}

resp, err := service.CallWithJSON("METHOD_NAME", inputJson)
if err != nil {
    log.Fatalln(err)
}
```

## Billing Information

### Paid Strategy Billing
- Each call creates a blockchain transaction
- Costs include: service price + gas fees
- Payment processed immediately
- Higher total cost per call due to gas fees

### Pre-Paid Strategy Billing
- Initial deposit to payment channel (one-time gas fee)
- Subsequent calls are off-chain (no gas fees)
- Channel can be reused for multiple calls
- Much lower cost per call for high-volume usage
- Remaining funds can be reclaimed when closing channel

### Free Call Strategy Billing
- No FET tokens required
- Limited by service provider's free call allowance
- Ideal for development and testing
- May have rate limits or usage caps

## Best Practices

1. **Development & Testing**: Use free call strategy when available to minimize costs during development

2. **Low Volume Usage**: Use paid strategy for occasional service calls where setup overhead isn't justified

3. **High Volume Usage**: Use pre-paid strategy to minimize per-call costs through payment channels

4. **Production Deployments**: 
   - Start with pre-paid strategy for cost efficiency
   - Implement strategy fallback logic for reliability
   - Monitor channel balances and replenish proactively

5. **Error Handling**: Always check for errors when setting payment strategies, as some services may not support all methods

6. **Channel Management**: 
   - Monitor payment channel balances to avoid service interruptions
   - Close unused channels to reclaim deposited funds
   - Set appropriate channel expiration times

7. **Cost Optimization**:
   - Calculate break-even point between paid and pre-paid based on your usage
   - Typically pre-paid becomes cheaper after 3-5 calls due to gas savings
   - Batch operations when possible to maximize pre-paid channel efficiency

```