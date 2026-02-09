package sdk

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shamank/snet-sdk-go/pkg/blockchain"
	"github.com/shamank/snet-sdk-go/pkg/config"
	"github.com/shamank/snet-sdk-go/pkg/grpc"
	"github.com/shamank/snet-sdk-go/pkg/model"
)

func TestServiceClientSaveProtoFiles(t *testing.T) {
	temp := t.TempDir()
	content := "syntax = \"proto3\"; package svc; message Ping {}"

	client := &ServiceClient{
		ServiceMetadata: &model.ServiceMetadata{
			ProtoFiles: map[string]string{
				"api/test.proto": content,
			},
		},
	}

	if err := client.ProtoFiles().Save(temp); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	path := filepath.Join(temp, "api", "test.proto")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(got) != content {
		t.Fatalf("unexpected file content: %q", got)
	}
}

func TestServiceClientSaveProtoFilesNoContent(t *testing.T) {
	client := &ServiceClient{ServiceMetadata: &model.ServiceMetadata{ProtoFiles: nil}}
	if err := client.ProtoFiles().Save(t.TempDir()); err == nil {
		t.Fatal("expected error when no proto files are present")
	}
}

func TestServiceClientSaveProtoFilesZip(t *testing.T) {
	temp := t.TempDir()
	zipPath := filepath.Join(temp, "protos.zip")
	content := "syntax = \"proto3\"; package svc; message Pong {}"

	client := &ServiceClient{
		ServiceMetadata: &model.ServiceMetadata{
			ProtoFiles: map[string]string{
				"api/pong.proto": content,
			},
		},
	}

	if err := client.ProtoFiles().SaveAsZip(zipPath); err != nil {
		t.Fatalf("SaveAsZip returned error: %v", err)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("OpenReader error: %v", err)
	}
	defer r.Close()

	if len(r.File) != 1 {
		t.Fatalf("expected 1 file in archive, got %d", len(r.File))
	}
	f := r.File[0]
	rc, err := f.Open()
	if err != nil {
		t.Fatalf("zip file open error: %v", err)
	}
	defer rc.Close()
	bytes, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}
	if string(bytes) != content {
		t.Fatalf("unexpected zip content: %q", bytes)
	}
}

func TestServiceClientSaveProtoFilesZipNoContent(t *testing.T) {
	client := &ServiceClient{ServiceMetadata: &model.ServiceMetadata{ProtoFiles: nil}}
	if err := client.ProtoFiles().SaveAsZip(filepath.Join(t.TempDir(), "empty.zip")); err == nil {
		t.Fatal("expected error when no proto files are present for zip")
	}
}

func TestNewServiceClientPopulatesFields(t *testing.T) {
	orgGroup := &model.OrganizationGroup{ID: "org-group"}
	orgMeta := &model.OrganizationMetaData{OrgID: "org", Groups: []*model.OrganizationGroup{orgGroup}}
	evm := &blockchain.EVMClient{}
	orgBC := &blockchain.OrgClient{
		EVMClient:            evm,
		OrganizationMetaData: orgMeta,
		CurrentOrgGroup:      orgGroup,
	}

	svcGroup := &model.ServiceGroup{GroupName: "group", Endpoints: []string{"http://localhost:8080"}}
	svcMeta := &model.ServiceMetadata{ProtoFiles: map[string]string{"svc.proto": "syntax"}}
	svcBC := &blockchain.ServiceClient{
		ServiceID:       "svc",
		ServiceMetadata: svcMeta,
		CurrentGroup:    svcGroup,
		EVMClient:       evm,
	}

	grpcClient := &grpc.Client{}
	cfg := &config.Config{}

	sc := newServiceClient(cfg, nil, orgBC, svcBC, grpcClient, nil)

	if sc.GRPC != grpcClient {
		t.Fatal("expected GRPC client to be set")
	}
	if sc.EVMClient != evm {
		t.Fatal("expected EVMClient to be propagated from service client")
	}
	if sc.ServiceID != "svc" {
		t.Fatalf("expected ServiceID svc, got %q", sc.ServiceID)
	}
	if sc.OrgID != "org" {
		t.Fatalf("expected OrgID org, got %q", sc.OrgID)
	}
	if sc.CurrentOrgGroup != orgGroup {
		t.Fatal("expected CurrentOrgGroup to reference organization group")
	}
	if sc.CurrentServiceGroup != svcGroup {
		t.Fatal("expected CurrentServiceGroup to reference service group")
	}
	if sc.ServiceMetadata != svcMeta {
		t.Fatal("expected ServiceMetadata pointer to be preserved")
	}
	if sc.OrgMetadata != orgMeta {
		t.Fatal("expected OrgMetadata pointer to be preserved")
	}
	if sc.config != cfg {
		t.Fatal("expected config pointer to be preserved")
	}
}

func TestServiceClientTrainingClientCaching(t *testing.T) {
	evm := &blockchain.EVMClient{}
	svcGroup := &model.ServiceGroup{}
	svcMeta := &model.ServiceMetadata{ProtoFiles: map[string]string{}}
	grpcClient := &grpc.Client{}
	cfg := &config.Config{}

	sc := newServiceClient(cfg, nil, nil, &blockchain.ServiceClient{
		ServiceID:       "svc",
		ServiceMetadata: svcMeta,
		CurrentGroup:    svcGroup,
		EVMClient:       evm,
	}, grpcClient, nil)

	first := sc.Training()
	if first == nil {
		t.Fatal("expected training client instance")
	}
	second := sc.Training()
	if first != second {
		t.Fatal("expected cached training client")
	}
}

func TestServiceClientProtoFilesAccessor(t *testing.T) {
	files := map[string]string{"a.proto": "content"}
	sc := &ServiceClient{
		ServiceMetadata: &model.ServiceMetadata{
			ProtoFiles: files,
		},
	}
	returned := sc.ProtoFiles().Get()
	if returned["a.proto"] != "content" {
		t.Fatal("ProtoFiles should expose stored content")
	}
	returned["a.proto"] = "updated"
	if files["a.proto"] != "updated" {
		t.Fatal("ProtoFiles should share underlying map")
	}
}

func TestServiceClientHeartbeat(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/heartbeat" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	})
	server := startTestServer(t, handler)
	defer server.Close()

	sc := &ServiceClient{
		config: &config.Config{},
		CurrentServiceGroup: &model.ServiceGroup{
			Endpoints: []string{server.URL},
		},
		ServiceMetadata: &model.ServiceMetadata{},
	}

	resp, err := sc.Healthcheck().HTTP()
	if err != nil {
		t.Fatalf("Heartbeat returned error: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("unexpected heartbeat payload: %v", resp)
	}
}

func startTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if strings.Contains(msg, "operation not permitted") {
				t.Skip("network operations not permitted in sandbox")
			}
			panic(r)
		}
	}()
	return httptest.NewServer(handler)
}

func TestOptionalUint64(t *testing.T) {
	if optionalUint64() != nil {
		t.Fatal("expected nil when no values provided")
	}
	val := optionalUint64(42)
	if val == nil || *val != 42 {
		t.Fatalf("optionalUint64 returned %v", val)
	}
}

// Tests for new service management functionality - validation only

func TestServiceClient_UpdateServiceMetadata_Validation(t *testing.T) {
	cfg := &config.Config{PrivateKey: ""}
	srvClient := &ServiceClient{config: cfg}

	metadata := &model.ServiceMetadata{DisplayName: "Test"}
	_, err := srvClient.UpdateServiceMetadata(metadata)

	if err == nil || !containsSubstring(err.Error(), "private key not configured") {
		t.Errorf("expected 'private key not configured' error, got: %v", err)
	}
}

func TestServiceClient_DeleteService_Validation(t *testing.T) {
	cfg := &config.Config{PrivateKey: ""}
	srvClient := &ServiceClient{config: cfg}

	_, err := srvClient.DeleteService()

	if err == nil || !containsSubstring(err.Error(), "private key not configured") {
		t.Errorf("expected 'private key not configured' error, got: %v", err)
	}
}

func TestServiceClient_GetServiceID(t *testing.T) {
	expectedID := "test-service-123"
	srvClient := &ServiceClient{srvClient: &blockchain.ServiceClient{ServiceID: expectedID}}

	got := srvClient.GetServiceID()

	if got != expectedID {
		t.Errorf("expected service ID %q, got %q", expectedID, got)
	}
}

func TestServiceClient_GetServiceMetadata(t *testing.T) {
	expectedMetadata := &model.ServiceMetadata{
		DisplayName: "Test Service",
		Version:     1,
	}

	srvClient := &ServiceClient{
		srvClient: &blockchain.ServiceClient{
			ServiceMetadata: expectedMetadata,
		},
	}

	got := srvClient.GetServiceMetadata()

	if got != expectedMetadata {
		t.Errorf("expected metadata %v, got %v", expectedMetadata, got)
	}
}

// Helper function for string matching
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// TestServiceClient_SetPaidPaymentStrategy_RequiresWebSocket verifies that SetPaidPaymentStrategy
// validates that RPCAddr uses WebSocket protocol (wss:// or ws://).
func TestServiceClient_SetPaidPaymentStrategy_RequiresWebSocket(t *testing.T) {
	tests := []struct {
		name    string
		rpcAddr string
	}{
		{
			name:    "https protocol should fail",
			rpcAddr: "https://sepolia.infura.io",
		},
		{
			name:    "http protocol should fail",
			rpcAddr: "http://localhost:8545",
		},
		{
			name:    "empty RPCAddr should fail",
			rpcAddr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{RPCAddr: tt.rpcAddr}
			sc := &ServiceClient{config: cfg}

			err := sc.SetPaidPaymentStrategy()

			if err == nil {
				t.Fatal("expected WebSocket validation error")
			}
			if !containsSubstring(err.Error(), "WebSocket") && !containsSubstring(err.Error(), "RPC address") {
				t.Fatalf("expected WebSocket or RPC address error, got: %v", err)
			}
		})
	}

	// Test that wss/ws protocols pass WebSocket validation
	t.Run("wss protocol passes validation", func(t *testing.T) {
		cfg := &config.Config{RPCAddr: "wss://sepolia.infura.io/ws"}
		sc := &ServiceClient{config: cfg}
		err := sc.validateWebSocketRPC()
		if err != nil {
			t.Fatalf("wss protocol should pass validation: %v", err)
		}
	})

	t.Run("ws protocol passes validation", func(t *testing.T) {
		cfg := &config.Config{RPCAddr: "ws://localhost:8546"}
		sc := &ServiceClient{config: cfg}
		err := sc.validateWebSocketRPC()
		if err != nil {
			t.Fatalf("ws protocol should pass validation: %v", err)
		}
	})
}

// TestServiceClient_SetPrePaidPaymentStrategy_RequiresWebSocket verifies that SetPrePaidPaymentStrategy
// validates that RPCAddr uses WebSocket protocol (wss:// or ws://).
func TestServiceClient_SetPrePaidPaymentStrategy_RequiresWebSocket(t *testing.T) {
	tests := []struct {
		name    string
		rpcAddr string
	}{
		{
			name:    "https protocol should fail",
			rpcAddr: "https://sepolia.infura.io",
		},
		{
			name:    "http protocol should fail",
			rpcAddr: "http://localhost:8545",
		},
		{
			name:    "empty RPCAddr should fail",
			rpcAddr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{RPCAddr: tt.rpcAddr}
			sc := &ServiceClient{config: cfg}

			err := sc.SetPrePaidPaymentStrategy(10)

			if err == nil {
				t.Fatal("expected WebSocket validation error")
			}
			if !containsSubstring(err.Error(), "WebSocket") && !containsSubstring(err.Error(), "RPC address") {
				t.Fatalf("expected WebSocket or RPC address error, got: %v", err)
			}
		})
	}

	// Test that wss/ws protocols pass WebSocket validation
	t.Run("wss protocol passes validation", func(t *testing.T) {
		cfg := &config.Config{RPCAddr: "wss://sepolia.infura.io/ws"}
		sc := &ServiceClient{config: cfg}
		err := sc.validateWebSocketRPC()
		if err != nil {
			t.Fatalf("wss protocol should pass validation: %v", err)
		}
	})

	t.Run("ws protocol passes validation", func(t *testing.T) {
		cfg := &config.Config{RPCAddr: "ws://localhost:8546"}
		sc := &ServiceClient{config: cfg}
		err := sc.validateWebSocketRPC()
		if err != nil {
			t.Fatalf("ws protocol should pass validation: %v", err)
		}
	})
}
