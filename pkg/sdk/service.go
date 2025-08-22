package sdk

import (
	"archive/zip"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/singnet/snet-sdk-go/pkg/blockchain"
	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/grpc"
	"github.com/singnet/snet-sdk-go/pkg/model"
	"github.com/singnet/snet-sdk-go/pkg/payment"
	"github.com/singnet/snet-sdk-go/pkg/training"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// Service defines the high-level client API for invoking service methods and
// managing payment strategies. Implementations wrap a dynamic gRPC client and
// inject the appropriate payment metadata (free/escrow/prepaid) per request.
type Service interface {
	// CallWithMap calls a service method using a JSON-like map for the request
	// body. Parameters are marshaled to JSON and converted to the input protobuf
	// message based on parsed descriptors.
	CallWithMap(method string, params map[string]any) (map[string]any, error)
	// CallWithJSON calls a service method using raw JSON bytes as the request
	// body. The JSON is unmarshaled into the input protobuf message; the result
	// is returned as JSON bytes.
	CallWithJSON(method string, input []byte) ([]byte, error)
	// CallWithProto calls a service method using a concrete protobuf message
	// for the request and returns the protobuf response.
	CallWithProto(method string, input proto.Message) (proto.Message, error)
	// SetPaidPaymentStrategy configures the escrow (MPE) strategy. It ensures
	// there is a usable payment channel and prepares signatures for subsequent
	// calls. Requires a valid signer private key.
	SetPaidPaymentStrategy() error
	// SetPrePaidPaymentStrategy configures the prepaid strategy. It prepares an
	// allowance based on call count and obtains tokens on Refresh. Requires a
	// valid signer private key in the SDK config.
	SetPrePaidPaymentStrategy(count uint64) error
	// SetFreePaymentStrategy configures the free-call strategy and obtains a
	// short-lived free-call token on Refresh. Optional extendBlocks controls
	// token lifetime in blocks (daemon-dependent).
	SetFreePaymentStrategy(extendBlocks ...uint64) error
	// GetFreeCallsAvailable returns the remaining number of free calls for the
	// current user/token.
	GetFreeCallsAvailable() (uint64, error)
	// SaveProtoFiles writes parsed .proto sources to the given directory,
	// preserving relative paths contained in filenames.
	SaveProtoFiles(path string) error
	// SaveProtoFilesZip writes parsed .proto sources to a ZIP archive at the
	// given path, preserving relative paths contained in filenames.
	SaveProtoFilesZip(path string) error
	// ProtoFiles returns the in-memory .proto sources as a map of
	// filename -> file contents.
	ProtoFiles() (files map[string]string)
	// TrainingClient returns a training sub-client bound to this service.
	TrainingClient() training.Client
	// Heartbeat performs a simple health check against the service daemon
	// (HTTP GET "<endpoint>/heartbeat") and returns the decoded JSON payload.
	Heartbeat() (any, error)
	// Close releases resources (e.g., underlying gRPC connection).
	Close()
}

// ServiceClient is a concrete Service implementation. It holds blockchain
// context (EVM client), parsed metadata (org/service/groups), a dynamic gRPC
// client, and the active payment strategy.
type ServiceClient struct {
	*blockchain.EVMClient
	GRPC                *grpc.Client
	strategy            payment.Strategy
	config              *config.Config
	ServiceID           string
	OrgID               string
	CurrentServiceGroup *model.ServiceGroup
	OrgMetadata         *model.OrganizationMetaData
	ServiceMetadata     *model.ServiceMetadata
	CurrentOrgGroup     *model.OrganizationGroup
	SignerPrivateKey    *ecdsa.PrivateKey
	trainingClient      training.Client
}

// TrainingClient returns (and lazily initializes) a training client bound to
// this service, using the same signer and block-number provider.
func (sC *ServiceClient) TrainingClient() training.Client {
	if sC.trainingClient == nil {
		sC.trainingClient = training.NewTrainingClient(sC.GRPC, sC.SignerPrivateKey, sC.EVMClient.GetCurrentBlockNumber)
	}
	return sC.trainingClient
}

// withTimeout returns a derived context with the given timeout. A cancelable
// context is returned when d <= 0.
func (sC *ServiceClient) withTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, d)
}

// ensureTimeout selects a sensible timeout for strategy operations, preferring
// PaymentEnsure, then StrategyRefresh, and finally a 1-minute default.
func (sC *ServiceClient) ensureTimeout() time.Duration {
	if sC.config != nil {
		if sC.config.Timeouts.PaymentEnsure > 0 {
			return sC.config.Timeouts.PaymentEnsure
		}
		if sC.config.Timeouts.StrategyRefresh > 0 {
			return sC.config.Timeouts.StrategyRefresh
		}
	}
	return time.Minute
}

// SetPaidPaymentStrategy initializes the escrow (MPE) payment strategy and
// ensures a valid channel (funds/expiration). It does not perform a Refresh
// because escrow calls sign per-request.
func (sC *ServiceClient) SetPaidPaymentStrategy() error {
	ctx, cancel := sC.withTimeout(context.Background(), sC.ensureTimeout())
	defer cancel()

	strategy, err := payment.NewPaidStrategy(
		ctx,
		sC.EVMClient,
		sC.GRPC,
		sC.ServiceMetadata,
		sC.SignerPrivateKey,
		sC.CurrentServiceGroup,
		sC.CurrentOrgGroup,
	)
	if err != nil {
		return err
	}
	sC.strategy = strategy
	return nil
}

// SetPrePaidPaymentStrategy initializes the prepaid strategy and immediately
// refreshes the daemon-issued token. The count parameter indicates the number
// of calls to provision in the initial signed allowance.
func (sC *ServiceClient) SetPrePaidPaymentStrategy(count uint64) error {
	ctx, cancel := sC.withTimeout(context.Background(), sC.ensureTimeout())
	defer cancel()

	strategy, err := payment.NewPrePaidStrategy(ctx, sC.EVMClient, sC.GRPC, sC.ServiceMetadata.GetMpeAddr(), sC.CurrentServiceGroup, sC.CurrentOrgGroup, sC.config.PrivateKey, count)
	if err != nil {
		return err
	}
	sC.strategy = strategy

	return strategy.Refresh(ctx)
}

// SetFreePaymentStrategy initializes the free-call strategy and fetches a
// short-lived token. If extendBlocks is provided, it is forwarded to request a
// custom token lifetime (daemon may ignore or cap it).
func (sC *ServiceClient) SetFreePaymentStrategy(extendBlocks ...uint64) error {
	strategy, err := payment.NewFreeStrategy(sC.EVMClient, sC.GRPC, sC.OrgID, sC.ServiceID, sC.CurrentOrgGroup.ID, sC.SignerPrivateKey, optionalUint64(extendBlocks...))
	if err != nil {
		return err
	}
	sC.strategy = strategy
	ctx, cancel := sC.withTimeout(context.Background(), sC.config.Timeouts.StrategyRefresh)
	defer cancel()
	return strategy.Refresh(ctx)
}

// GetFreeCallsAvailable returns the number of remaining free calls for the
// current user/token. It requires the active strategy to be FreeStrategy.
func (sC *ServiceClient) GetFreeCallsAvailable() (uint64, error) {
	ctx, cancel := sC.withTimeout(context.Background(), sC.config.Timeouts.StrategyRefresh)
	defer cancel()

	freeStrat, ok := sC.strategy.(*payment.FreeStrategy)
	if !ok {
		return 0, errors.New("invalid strategy")
	}
	available, err := freeStrat.GetFreeCallsAvailable(ctx)
	if err != nil {
		return available, err
	}
	return available, nil
}

// CallWithMap invokes a method with a map-based request. Payment metadata is
// injected by the current strategy into the outgoing context.
func (sC *ServiceClient) CallWithMap(method string, params map[string]any) (resp map[string]any, err error) {
	ctx, cancel := sC.withTimeout(context.Background(), sC.config.Timeouts.GRPCUnary)
	defer cancel()
	resp, err = sC.GRPC.CallWithMap(sC.strategy.GRPCMetadata(ctx), method, params)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CallWithJSON invokes a method with raw JSON request bytes. The JSON is mapped
// to the protobuf input type using service descriptors.
func (sC *ServiceClient) CallWithJSON(method string, input []byte) (resp []byte, err error) {
	ctx, cancel := sC.withTimeout(context.Background(), sC.config.Timeouts.GRPCUnary)
	defer cancel()
	resp, err = sC.GRPC.CallWithJSON(sC.strategy.GRPCMetadata(ctx), method, input)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CallWithProto invokes a method with a concrete protobuf request message and
// returns the protobuf response message.
func (sC *ServiceClient) CallWithProto(method string, input proto.Message) (resp proto.Message, err error) {
	ctx, cancel := sC.withTimeout(context.Background(), sC.config.Timeouts.GRPCUnary)
	defer cancel()
	resp, err = sC.GRPC.CallWithProto(sC.strategy.GRPCMetadata(ctx), method, input)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// SaveProtoFilesZip writes all loaded .proto files from the ServiceClient
// into a ZIP archive at the specified path.
//
// Each proto file is added to the archive preserving its filename (including any subdirectories).
//
// Returns an error if the ZIP file cannot be created, or if any file cannot be added or written.
func (sC *ServiceClient) SaveProtoFilesZip(zipPath string) error {
	protos := sC.ServiceMetadata.ProtoFiles
	if len(protos) == 0 {
		return fmt.Errorf("no proto files to save")
	}

	outFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer func(outFile *os.File) {
		err = outFile.Close()
		if err != nil {
			zap.L().Error("failed to close zip file", zap.Error(err))
			return
		}
	}(outFile)

	zipWriter := zip.NewWriter(outFile)
	defer func(zipWriter *zip.Writer) {
		err = zipWriter.Close()
		if err != nil {
			zap.L().Error("failed to close zip writer", zap.Error(err))
		}
	}(zipWriter)

	for filename, content := range protos {
		// Create a file entry in the ZIP archive
		fw, err := zipWriter.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file %s in zip: %w", filename, err)
		}

		if _, err = fw.Write([]byte(content)); err != nil {
			return fmt.Errorf("failed to write content for %s: %w", filename, err)
		}
	}

	zap.L().Info("zip file created", zap.Int("files", len(protos)), zap.String("zip_path", zipPath))

	return nil
}

// SaveProtoFiles writes all loaded .proto files from the ServiceClient
// to the specified directory on disk. It preserves subdirectory structure
// based on the filenames.
//
// If the directory does not exist, it will be created along with any
// necessary subdirectories.
//
// Returns an error if any file cannot be written or if directory creation fails.
func (sC *ServiceClient) SaveProtoFiles(dirPath string) error {
	protos := sC.ServiceMetadata.ProtoFiles
	if len(protos) == 0 {
		return fmt.Errorf("no proto files to save")
	}

	// Ensure the target directory exists
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	for filename, content := range protos {
		fullPath := filepath.Join(dirPath, filename)

		// Create subdirectories if the filename includes folders
		if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create subdirectory for %s: %w", filename, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}

// ProtoFiles returns parsed .proto sources as a map where the key is the
// filename (possibly with subdirectories) and the value is the file content.
func (sC *ServiceClient) ProtoFiles() map[string]string {
	return sC.ServiceMetadata.ProtoFiles
}

// Close releases the underlying gRPC connection. It is safe to call multiple times.
func (sC *ServiceClient) Close() {
	_ = sC.GRPC.Close()
}

// Heartbeat performs a simple HTTP GET against "<first-endpoint>/heartbeat" and
// returns the decoded JSON payload on HTTP 200. A non-200 response yields an error.
func (sC *ServiceClient) Heartbeat() (any, error) {
	resp, err := http.Get(sC.CurrentServiceGroup.Endpoints[0] + "/heartbeat")
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
		return nil, errors.New("heartbeat failed")
	}
	var result any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode heartbeat response: %w", err)
	}

	return result, nil
}

// optionalUint64 returns a pointer to the first value if provided, or nil.
// Useful for optional parameters in strategy constructors.
func optionalUint64(v ...uint64) *uint64 {
	if len(v) > 0 {
		return &v[0]
	}
	return nil
}
