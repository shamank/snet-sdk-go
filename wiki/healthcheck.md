## Service Healthcheck

Service healthchecks allow you to verify that an AI service is available and responsive before making actual service calls. The SDK provides three types of healthcheck protocols: HTTP, WebGRPC, and GRPC. Use healthchecks to ensure service availability, implement monitoring, and build resilient applications.

## Healthcheck Protocol Differences

| Protocol | Description | Use Case | Performance |
|----------|-------------|----------|-------------|
| **HTTP** | Standard HTTP health endpoint | Web-based services, simple checks | Fast, lightweight |
| **WebGRPC** | gRPC-Web protocol over HTTP | Browser-compatible gRPC services | Moderate overhead |
| **GRPC** | Native gRPC protocol | High-performance gRPC services | Fastest, most efficient |

### When to Use Each Protocol

- **HTTP**: Best for REST-based services or when you need simple, universal compatibility
- **WebGRPC**: Use when service runs gRPC-Web (common for browser-accessible services)
- **GRPC**: Preferred for native gRPC services with best performance and lowest latency

## Basic Healthcheck Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/shamank/snet-sdk-go/pkg/config"
	"github.com/shamank/snet-sdk-go/pkg/sdk"
)

func main() {
	c := config.Config{
		RPCAddr:      "https://sepolia.infura.io/v3/YOUR_PROJECT_ID",
		Debug:        true,
		RegistryAddr: "",
	}

	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("orgID", "serviceID", "default_group")
	if err != nil {
		log.Println(err)
		return
	}

	// HTTP healthcheck
	resp, err1 := service.Healthcheck().HTTP()
	if err1 != nil {
		log.Println("HTTP healthcheck failed:", err1)
	} else {
		fmt.Printf("\nHTTP Healthcheck: %v", resp)
	}

	// WebGRPC healthcheck
	respWebGrpc, err2 := service.Healthcheck().WebGRPC()
	if err2 != nil {
		log.Println("WebGRPC healthcheck failed:", err2)
	} else {
		fmt.Printf("\nWebGRPC Healthcheck: %v", respWebGrpc)
	}

	// GRPC healthcheck (native gRPC)
	respGrpc, err3 := service.Healthcheck().GRPC()
	if err3 != nil {
		log.Println("GRPC healthcheck failed:", err3)
	} else {
		fmt.Printf("\nGRPC Healthcheck: %v", respGrpc)
	}
}

```

## Using Healthcheck Results

### Service Availability Check

```go
func isServiceAvailable(service *sdk.ServiceClient) bool {
	// Try HTTP healthcheck first (most compatible)
	_, err := service.Healthcheck().HTTP()
	if err == nil {
		return true
	}

	// Fallback to GRPC if HTTP fails
	_, err = service.Healthcheck().GRPC()
	return err == nil
}

// Usage
if isServiceAvailable(service) {
	fmt.Println("Service is available")
	// Proceed with service calls
} else {
	fmt.Println("Service is unavailable")
	// Handle unavailability (retry, use backup service, etc.)
}
```

### Pre-Call Validation

```go
// Validate service health before making expensive calls
_, err := service.Healthcheck().GRPC()
if err != nil {
	log.Printf("Service unhealthy, skipping call: %v", err)
	return
}

// Service is healthy, proceed with actual call
resp, err := service.CallWithJSON("METHOD_NAME", inputJson)
if err != nil {
	log.Printf("Service call failed: %v", err)
}
```

## Result Interpretation Guide

### Successful Healthcheck
- Returns `true` (or success status)
- Service is reachable and responsive
- Safe to proceed with service calls

### Failed Healthcheck
Common failure reasons:
- **Network Error**: Service endpoint unreachable (check network/firewall)
- **Timeout**: Service is slow or overloaded (retry or wait)
- **Protocol Mismatch**: Wrong healthcheck protocol for service (try different protocol)
- **Service Down**: Service is not running (check service status, contact provider)

## Monitoring Examples

### Periodic Health Monitoring

```go
import "time"

func monitorServiceHealth(service *sdk.ServiceClient, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		healthy, err := service.Healthcheck().GRPC()
		if err != nil {
			log.Printf("[ALERT] Service unhealthy: %v", err)
			// Send alert notification
		} else {
			log.Printf("[OK] Service healthy: %v", healthy)
		}
	}
}

// Monitor every 30 seconds
go monitorServiceHealth(service, 30*time.Second)
```

### Multi-Protocol Health Check

```go
type HealthStatus struct {
	HTTP    bool
	WebGRPC bool
	GRPC    bool
}

func comprehensiveHealthCheck(service *sdk.ServiceClient) HealthStatus {
	status := HealthStatus{}

	_, err := service.Healthcheck().HTTP()
	status.HTTP = (err == nil)

	_, err = service.Healthcheck().WebGRPC()
	status.WebGRPC = (err == nil)

	_, err = service.Healthcheck().GRPC()
	status.GRPC = (err == nil)

	return status
}

// Usage
health := comprehensiveHealthCheck(service)
fmt.Printf("Service Health - HTTP: %v, WebGRPC: %v, GRPC: %v\n", 
	health.HTTP, health.WebGRPC, health.GRPC)
```

### Circuit Breaker Pattern

```go
type CircuitBreaker struct {
	maxFailures  int
	failures     int
	timeout      time.Duration
	lastAttempt  time.Time
	isOpen       bool
}

func (cb *CircuitBreaker) callWithHealthCheck(service *sdk.ServiceClient, fn func() error) error {
	// If circuit is open, check if timeout has passed
	if cb.isOpen {
		if time.Since(cb.lastAttempt) < cb.timeout {
			return fmt.Errorf("circuit breaker is open")
		}
		cb.isOpen = false
		cb.failures = 0
	}

	// Perform healthcheck
	_, err := service.Healthcheck().GRPC()
	cb.lastAttempt = time.Now()
	
	if err != nil {
		cb.failures++
		if cb.failures >= cb.maxFailures {
			cb.isOpen = true
			return fmt.Errorf("circuit breaker opened after %d failures", cb.failures)
		}
		return err
	}

	// Reset failures on success
	cb.failures = 0
	return fn()
}
```
