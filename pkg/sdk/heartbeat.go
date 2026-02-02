package sdk

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/grpc"
	"github.com/singnet/snet-sdk-go/pkg/model"
	"go.uber.org/zap"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
)

// Healthcheck defines the interface for performing health checks against a service daemon
// using different protocols (gRPC, gRPC-Web, HTTP).
type Healthcheck interface {
	// GRPC performs a standard gRPC health check
	GRPC() (*grpc_health_v1.HealthCheckResponse, error)
	// WebGRPC performs a gRPC-Web health check
	WebGRPC() (*grpc_health_v1.HealthCheckResponse, error)
	// HTTP performs an HTTP health check
	HTTP() (map[string]any, error)
}

// healthcheckClient is a concrete implementation of Healthcheck interface.
// It provides methods to check service daemon health using different protocols.
type healthcheckClient struct {
	grpcClient   *grpc.Client        // gRPC client for health check operations
	serviceGroup *model.ServiceGroup // Service group containing endpoints
	config       *config.Config      // SDK configuration for debug mode and settings
}

// newHealthcheckClient creates a new health check client for a service.
//
// Parameters:
//   - grpcClient: connected gRPC client to the service daemon
//   - serviceGroup: service group containing endpoint configuration
//   - cfg: SDK configuration (used for debug output and timeouts)
//
// Returns a Healthcheck interface implementation.
func newHealthcheckClient(grpcClient *grpc.Client, serviceGroup *model.ServiceGroup, cfg *config.Config) Healthcheck {
	return &healthcheckClient{
		grpcClient:   grpcClient,
		serviceGroup: serviceGroup,
		config:       cfg,
	}
}

// GRPC performs a standard gRPC health check against the connected service.
func (hc *healthcheckClient) GRPC() (*grpc_health_v1.HealthCheckResponse, error) {
	client := grpc_health_v1.NewHealthClient(hc.grpcClient.GRPC) // Conn — *grpc.ClientConn
	resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return nil, fmt.Errorf("grpc heartbeat failed: %w", err)
	}
	return resp, nil
}

// WebGRPC performs a gRPC-Web health check using the gRPC Health protocol
// over an HTTP/1.1 transport.
func (hc *healthcheckClient) WebGRPC() (*grpc_health_v1.HealthCheckResponse, error) {

	healthResp := &grpc_health_v1.HealthCheckResponse{}

	reqBody, err := proto.Marshal(&grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return nil, err
	}

	frame := make([]byte, 5+len(reqBody))
	frame[0] = 0x0 // flags: message frame
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(reqBody)))
	copy(frame[5:], reqBody)

	req, err := http.NewRequest("POST", hc.serviceGroup.Endpoints[0]+"/grpc.health.v1.Health/Check", bytes.NewReader(frame))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/grpc-web+proto")
	req.Header.Set("X-Grpc-Web", "1")
	req.Header.Set("X-User-Agent", "grpc-go/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read response body", err)
		return nil, err
	}
	if hc.config.Debug {
		log.Println("HTTP status:", resp.Status)
	}

	i := 0
	for i < len(body) {
		flags := body[i]
		length := binary.BigEndian.Uint32(body[i+1 : i+5])
		i += 5

		if i+int(length) > len(body) {
			log.Println("Frame length exceeds body size")
			break
		}

		payload := body[i : i+int(length)]
		i += int(length)

		if flags&0x80 != 0 {
			// trailers frame, usually can skip
			continue
		}

		if err := proto.Unmarshal(payload, healthResp); err != nil {
			return nil, err
		}
		if hc.config.Debug {
			log.Println("Health status:", healthResp.Status.String())
		}
	}
	return healthResp, err
}

// HTTP performs a simple HTTP GET request to "<endpoint>/heartbeat"
// and returns the decoded JSON response payload.
func (hc *healthcheckClient) HTTP() (map[string]any, error) {
	resp, err := http.Get(hc.serviceGroup.Endpoints[0] + "/heartbeat")
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			zap.L().Error("failed to close heartbeat", zap.Error(err))
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("heartbeat failed with: %v", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode heartbeat response: %w", err)
	}

	if hc.config.Debug {
		log.Println("Protocol used:", resp.Proto) // e.g. "HTTP/2.0" or "HTTP/1.1"
	}

	return result, nil
}
