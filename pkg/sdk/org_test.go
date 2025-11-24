package sdk

import (
"testing"

"github.com/ethereum/go-ethereum/common"
"github.com/singnet/snet-sdk-go/pkg/blockchain"
"github.com/singnet/snet-sdk-go/pkg/config"
"github.com/singnet/snet-sdk-go/pkg/model"
)

// Mock storage for testing
type mockStorage struct {
uploadJSONCalled bool
lastData         interface{}
shouldFail       bool
}

func (m *mockStorage) ReadFile(id string) ([]byte, error) {
return []byte("{}"), nil
}

func (m *mockStorage) UploadJSON(data interface{}) (string, error) {
m.uploadJSONCalled = true
m.lastData = data
if m.shouldFail {
return "", &testError{"upload failed"}
}
return "ipfs://QmTest123", nil
}

type testError struct {
msg string
}

func (e *testError) Error() string {
return e.msg
}

// Tests for CreateOrganization focusing on input validation

func TestCreateOrganization_PrivateKeyValidation(t *testing.T) {
cfg := &config.Config{PrivateKey: ""}
evm := &blockchain.EVMClient{Storage: &mockStorage{}}

metadata := &model.OrganizationMetaData{OrgName: "Test"}
_, err := CreateOrganization(evm, cfg, "test-org", metadata, nil)

if err == nil {
t.Fatal("expected error when private key not configured")
}
if !contains(err.Error(), "private key not configured") {
t.Errorf("expected 'private key not configured' error, got: %v", err)
}
}

func TestCreateOrganization_IPFSUploadFailure(t *testing.T) {
cfg := &config.Config{
PrivateKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
}
mockStorageFailure := &mockStorage{shouldFail: true}
evm := &blockchain.EVMClient{Storage: mockStorageFailure}

metadata := &model.OrganizationMetaData{OrgName: "Test"}
_, err := CreateOrganization(evm, cfg, "test-org", metadata, nil)

if err == nil {
t.Fatal("expected error when IPFS upload fails")
}
if !contains(err.Error(), "failed to upload metadata to IPFS") {
t.Errorf("expected 'failed to upload metadata to IPFS' error, got: %v", err)
}
}

// Tests for OrganizationClient methods - input validation only

func TestOrganizationClient_RemoveMembers_Validation(t *testing.T) {
cfg := &config.Config{PrivateKey: ""}
orgClient := &OrganizationClient{config: cfg}

// Test empty members list
_, err := orgClient.RemoveMembers([]common.Address{})
if err == nil || !contains(err.Error(), "no members to remove") {
t.Errorf("expected 'no members to remove' error, got: %v", err)
}

// Test without private key
_, err = orgClient.RemoveMembers([]common.Address{common.HexToAddress("0x1")})
if err == nil || !contains(err.Error(), "private key not configured") {
t.Errorf("expected 'private key not configured' error, got: %v", err)
}
}

func TestOrganizationClient_ChangeOwner_Validation(t *testing.T) {
cfg := &config.Config{PrivateKey: ""}
orgClient := &OrganizationClient{config: cfg}

_, err := orgClient.ChangeOwner(common.HexToAddress("0x123"))
if err == nil || !contains(err.Error(), "private key not configured") {
t.Errorf("expected 'private key not configured' error, got: %v", err)
}
}

func TestOrganizationClient_DeleteOrganization_Validation(t *testing.T) {
cfg := &config.Config{PrivateKey: ""}
orgClient := &OrganizationClient{config: cfg}

_, err := orgClient.DeleteOrganization()
if err == nil || !contains(err.Error(), "private key not configured") {
t.Errorf("expected 'private key not configured' error, got: %v", err)
}
}

func TestOrganizationClient_UpdateOrgMetadataFull_Validation(t *testing.T) {
cfg := &config.Config{PrivateKey: ""}
orgClient := &OrganizationClient{config: cfg}

metadata := &model.OrganizationMetaData{OrgName: "Test"}
_, err := orgClient.UpdateOrgMetadataFull(metadata)
if err == nil || !contains(err.Error(), "private key not configured") {
t.Errorf("expected 'private key not configured' error, got: %v", err)
}
}

func TestOrganizationClient_CreateService_Validation(t *testing.T) {
cfg := &config.Config{PrivateKey: ""}
orgClient := &OrganizationClient{config: cfg}

metadata := &model.ServiceMetadata{DisplayName: "Test"}
_, err := orgClient.CreateService("test-service", metadata)
if err == nil || !contains(err.Error(), "private key not configured") {
t.Errorf("expected 'private key not configured' error, got: %v", err)
}
}

// Tests for getter methods

func TestOrganizationClient_GetOrgMetadata(t *testing.T) {
expectedMetadata := &model.OrganizationMetaData{
OrgName: "Test Org",
OrgID:   "test-org",
}

orgClient := &OrganizationClient{
blockchainClient: &blockchain.OrgClient{
OrganizationMetaData: expectedMetadata,
},
}

got := orgClient.GetOrgMetadata()

if got != expectedMetadata {
t.Errorf("expected metadata %v, got %v", expectedMetadata, got)
}
}

func TestOrganizationClient_GetOrgID(t *testing.T) {
expectedID := "test-org-123"

orgClient := &OrganizationClient{
blockchainClient: &blockchain.OrgClient{
OrganizationMetaData: &model.OrganizationMetaData{
OrgID: expectedID,
},
},
}

got := orgClient.GetOrgID()

if got != expectedID {
t.Errorf("expected org ID %q, got %q", expectedID, got)
}
}

func TestOrganizationClient_GetCurrentGroup(t *testing.T) {
expectedGroup := &model.OrganizationGroup{
GroupName: "default_group",
ID:        "default_group",
}

orgClient := &OrganizationClient{
CurrentGroup: expectedGroup,
}

got := orgClient.GetCurrentGroup()

if got != expectedGroup {
t.Errorf("expected group %v, got %v", expectedGroup, got)
}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
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
