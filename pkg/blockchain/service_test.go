package blockchain

import (
	"encoding/json"
	"testing"

	"github.com/singnet/snet-sdk-go/pkg/model"
)

func TestServiceMetadataJSONParsing(t *testing.T) {
	jsonData := `{
		"version": 1,
		"display_name": "Example Service",
		"encoding": "proto",
		"service_type": "grpc",
		"model_ipfs_hash": "",
		"service_api_source": "QmProtoHash",
		"mpe_address": "0x1234567890123456789012345678901234567890",
		"groups": [
			{
				"group_name": "default",
				"endpoints": ["https://service.example.com:8080"],
				"pricing": [
					{
						"price_model": "fixed_price",
						"price_in_cogs": 1000
					}
				],
				"free_calls": 10,
				"free_call_signer_address": "0x0987654321098765432109876543210987654321"
			}
		]
	}`

	var metadata model.ServiceMetadata
	err := json.Unmarshal([]byte(jsonData), &metadata)
	if err != nil {
		t.Fatalf("failed to parse service metadata: %v", err)
	}

	if metadata.DisplayName != "Example Service" {
		t.Fatalf("expected DisplayName=Example Service, got %s", metadata.DisplayName)
	}

	if len(metadata.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(metadata.Groups))
	}

	if metadata.Groups[0].GroupName != "default" {
		t.Fatalf("expected group name=default, got %s", metadata.Groups[0].GroupName)
	}

	if len(metadata.Groups[0].Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(metadata.Groups[0].Endpoints))
	}

	if metadata.Groups[0].FreeCalls != 10 {
		t.Fatalf("expected FreeCalls=10, got %d", metadata.Groups[0].FreeCalls)
	}
}

func TestServiceMetadataBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name             string
		modelIpfsHash    string
		serviceApiSource string
		wantHash         string
	}{
		{
			name:             "ServiceApiSource preferred",
			modelIpfsHash:    "QmOldHash",
			serviceApiSource: "QmNewHash",
			wantHash:         "QmNewHash",
		},
		{
			name:             "ModelIpfsHash fallback",
			modelIpfsHash:    "QmOldHash",
			serviceApiSource: "",
			wantHash:         "QmOldHash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := &model.ServiceMetadata{
				ModelIpfsHash:    tt.modelIpfsHash,
				ServiceApiSource: tt.serviceApiSource,
			}

			var hash string
			if metadata.ServiceApiSource != "" {
				hash = metadata.ServiceApiSource
			} else if metadata.ModelIpfsHash != "" {
				hash = metadata.ModelIpfsHash
			}

			if hash != tt.wantHash {
				t.Fatalf("expected hash=%s, got %s", tt.wantHash, hash)
			}
		})
	}
}
