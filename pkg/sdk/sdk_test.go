package sdk

import (
	"context"
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/singnet/snet-sdk-go/pkg/blockchain"
	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/grpc"
	"github.com/singnet/snet-sdk-go/pkg/model"
	"github.com/singnet/snet-sdk-go/pkg/payment"
)

func TestCoreNewServiceClient(t *testing.T) {
	orgGroup := &model.OrganizationGroup{ID: "org-group", GroupName: "default"}
	orgMeta := &model.OrganizationMetaData{OrgID: "org", Groups: []*model.OrganizationGroup{orgGroup}}
	evm := &blockchain.EVMClient{}
	orgBC := &blockchain.OrgClient{
		EVMClient:            evm,
		OrganizationMetaData: orgMeta,
		CurrentOrgGroup:      orgGroup,
	}

	svcGroup := &model.ServiceGroup{GroupName: "default", Endpoints: []string{"http://localhost:8080"}}
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

type mockStrategyFactory struct {
	paidFn    func(context.Context, *blockchain.EVMClient, *grpc.Client, *model.ServiceMetadata, *ecdsa.PrivateKey, *model.ServiceGroup, *model.OrganizationGroup) (payment.Strategy, error)
	prePaidFn func(context.Context, *blockchain.EVMClient, *grpc.Client, common.Address, *model.ServiceGroup, *model.OrganizationGroup, string, uint64) (payment.Strategy, error)
	freeFn    func(*blockchain.EVMClient, *grpc.Client, string, string, string, *ecdsa.PrivateKey, *uint64) (payment.Strategy, error)
}

func (m *mockStrategyFactory) Paid(ctx context.Context, evm *blockchain.EVMClient, grpcCli *grpc.Client, metadata *model.ServiceMetadata, key *ecdsa.PrivateKey, serviceGroup *model.ServiceGroup, orgGroup *model.OrganizationGroup) (payment.Strategy, error) {
	return m.paidFn(ctx, evm, grpcCli, metadata, key, serviceGroup, orgGroup)
}

func (m *mockStrategyFactory) PrePaid(ctx context.Context, evm *blockchain.EVMClient, grpcCli *grpc.Client, mpeAddr common.Address, serviceGroup *model.ServiceGroup, orgGroup *model.OrganizationGroup, privateKey string, count uint64) (payment.Strategy, error) {
	return m.prePaidFn(ctx, evm, grpcCli, mpeAddr, serviceGroup, orgGroup, privateKey, count)
}

func (m *mockStrategyFactory) Free(evm *blockchain.EVMClient, grpcCli *grpc.Client, orgID, serviceID, groupID string, key *ecdsa.PrivateKey, extend *uint64) (payment.Strategy, error) {
	return m.freeFn(evm, grpcCli, orgID, serviceID, groupID, key, extend)
}

type stubStrategy struct {
	refreshed bool
}

func (s *stubStrategy) GRPCMetadata(ctx context.Context) context.Context { return ctx }

func (s *stubStrategy) Refresh(ctx context.Context) error {
	s.refreshed = true
	return nil
}

func TestServiceClientEnsureTimeout(t *testing.T) {
	sc := &ServiceClient{config: &config.Config{Timeouts: config.Timeouts{PaymentEnsure: 42 * time.Second}}}
	if d := sc.ensureTimeout(); d != 42*time.Second {
		t.Fatalf("unexpected timeout: %v", d)
	}

	sc = &ServiceClient{config: &config.Config{Timeouts: config.Timeouts{StrategyRefresh: 5 * time.Second}}}
	if d := sc.ensureTimeout(); d != 5*time.Second {
		t.Fatalf("expected StrategyRefresh fallback, got %v", d)
	}
}

func TestServiceClientSetPaidStrategyUsesFactory(t *testing.T) {
	stub := &stubStrategy{}
	called := false
	factory := &mockStrategyFactory{
		paidFn: func(context.Context, *blockchain.EVMClient, *grpc.Client, *model.ServiceMetadata, *ecdsa.PrivateKey, *model.ServiceGroup, *model.OrganizationGroup) (payment.Strategy, error) {
			called = true
			return stub, nil
		},
	}

	sc := &ServiceClient{
		EVMClient:           &blockchain.EVMClient{},
		GRPC:                &grpc.Client{},
		strategies:          factory,
		ServiceMetadata:     &model.ServiceMetadata{},
		CurrentServiceGroup: &model.ServiceGroup{},
		CurrentOrgGroup:     &model.OrganizationGroup{},
		SignerPrivateKey:    nil,
		config:              &config.Config{Timeouts: config.Timeouts{PaymentEnsure: time.Second}},
	}

	if err := sc.SetPaidPaymentStrategy(); err != nil {
		t.Fatalf("SetPaidPaymentStrategy error: %v", err)
	}
	if !called {
		t.Fatal("expected Paid to be called")
	}
	if sc.strategy != stub {
		t.Fatal("strategy not assigned")
	}
}

func TestServiceClientSetPrepaidStrategy(t *testing.T) {
	stub := &stubStrategy{}
	called := false
	factory := &mockStrategyFactory{
		prePaidFn: func(context.Context, *blockchain.EVMClient, *grpc.Client, common.Address, *model.ServiceGroup, *model.OrganizationGroup, string, uint64) (payment.Strategy, error) {
			called = true
			return stub, nil
		},
	}

	sc := &ServiceClient{
		EVMClient:           &blockchain.EVMClient{},
		GRPC:                &grpc.Client{},
		strategies:          factory,
		ServiceMetadata:     &model.ServiceMetadata{},
		CurrentServiceGroup: &model.ServiceGroup{},
		CurrentOrgGroup:     &model.OrganizationGroup{},
		config:              &config.Config{Timeouts: config.Timeouts{StrategyRefresh: time.Second}},
	}

	if err := sc.SetPrePaidPaymentStrategy(3); err != nil {
		t.Fatalf("SetPrePaidPaymentStrategy error: %v", err)
	}
	if !called {
		t.Fatal("expected PrePaid to be called")
	}
	if !stub.refreshed {
		t.Fatal("expected strategy Refresh to be invoked")
	}
}

func TestServiceClientSetFreeStrategy(t *testing.T) {
	stub := &stubStrategy{}
	called := false
	factory := &mockStrategyFactory{
		freeFn: func(*blockchain.EVMClient, *grpc.Client, string, string, string, *ecdsa.PrivateKey, *uint64) (payment.Strategy, error) {
			called = true
			return stub, nil
		},
	}

	sc := &ServiceClient{
		EVMClient:        &blockchain.EVMClient{},
		GRPC:             &grpc.Client{},
		strategies:       factory,
		ServiceMetadata:  &model.ServiceMetadata{},
		CurrentOrgGroup:  &model.OrganizationGroup{},
		OrgID:            "org",
		ServiceID:        "svc",
		config:           &config.Config{Timeouts: config.Timeouts{StrategyRefresh: time.Second}},
		SignerPrivateKey: nil,
	}

	if err := sc.SetFreePaymentStrategy(); err != nil {
		t.Fatalf("SetFreePaymentStrategy error: %v", err)
	}
	if !called {
		t.Fatal("expected Free to be called")
	}
	if !stub.refreshed {
		t.Fatal("expected Refresh to be executed")
	}
}
