package blockchain

import (
	"encoding/json"
	"testing"

	"github.com/shamank/snet-sdk-go/pkg/model"
)

func TestOrgMetadataJSONParsing(t *testing.T) {
	jsonData := `{
		"org_name": "SingularityNET",
		"org_id": "snet",
		"groups": [
			{
				"group_id": "default_group",
				"group_name": "default",
				"payment": {
					"payment_address": "0x1234567890123456789012345678901234567890",
					"payment_expiration_threshold": 100,
					"payment_channel_storage_type": "etcd",
					"payment_channel_storage_client": {
						"connection_timeout": "5s",
						"request_timeout": "3s",
						"endpoints": ["http://127.0.0.1:2379"]
					}
				}
			}
		]
	}`

	var metadata model.OrganizationMetaData
	err := json.Unmarshal([]byte(jsonData), &metadata)
	if err != nil {
		t.Fatalf("failed to parse org metadata: %v", err)
	}

	if metadata.OrgName != "SingularityNET" {
		t.Fatalf("expected OrgName=SingularityNET, got %s", metadata.OrgName)
	}

	if len(metadata.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(metadata.Groups))
	}

	if metadata.Groups[0].GroupName != "default" {
		t.Fatalf("expected group name=default, got %s", metadata.Groups[0].GroupName)
	}
}
