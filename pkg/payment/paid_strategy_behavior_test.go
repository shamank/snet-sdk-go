package payment

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shamank/snet-sdk-go/pkg/blockchain"
	sggrpc "github.com/shamank/snet-sdk-go/pkg/grpc"
	"github.com/shamank/snet-sdk-go/pkg/model"
	ogrpc "google.golang.org/grpc"
)

func TestNewPaidStrategySuccess(t *testing.T) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	chainStub := &stubChainOps{
		currentBlock: big.NewInt(100),
		networkID:    big.NewInt(1),
		filterResult: &blockchain.MultiPartyEscrowChannelOpen{
			ChannelId:  big.NewInt(111),
			Signer:     common.HexToAddress("0x00000000000000000000000000000000000000aa"),
			Sender:     common.HexToAddress("0x00000000000000000000000000000000000000aa"),
			Recipient:  common.HexToAddress("0x00000000000000000000000000000000000000bb"),
			GroupId:    [32]byte{},
			Amount:     big.NewInt(90),
			Expiration: big.NewInt(100),
		},
		ensureResult: big.NewInt(999),
	}
	channelStateStub := stubChannelState{
		reply: &ChannelStateReply{
			CurrentSignedAmount: big.NewInt(90).Bytes(),
			CurrentNonce:        big.NewInt(2).Bytes(),
		},
	}

	serviceMeta := &model.ServiceMetadata{
		MPEAddress: "0x00000000000000000000000000000000000000aa",
	}
	price := big.NewInt(10)
	serviceMeta.Groups = []*model.ServiceGroup{
		{
			GroupName: "default",
			Endpoints: []string{"127.0.0.1:0"},
			Pricing: []model.Pricing{
				{
					PriceModel:  "fixed",
					PriceInCogs: price,
				},
			},
		},
	}

	groupID := make([]byte, 32)
	copy(groupID, []byte("group-identifier-000000000000000000"))
	orgGroup := &model.OrganizationGroup{
		ID:        base64.StdEncoding.EncodeToString(groupID),
		GroupName: "default",
		PaymentDetails: model.Payment{
			PaymentAddress:             "0x00000000000000000000000000000000000000bb",
			PaymentExpirationThreshold: big.NewInt(50),
		},
	}

	ps, err := NewPaidStrategy(
		context.Background(),
		&blockchain.EVMClient{},
		&sggrpc.Client{},
		serviceMeta,
		priv,
		serviceMeta.Groups[0],
		orgGroup,
		WithPaidStrategyDependencies(PaidStrategyDependencies{
			Chain:        chainStub,
			ChannelState: channelStateStub,
		}),
	)
	if err != nil {
		t.Fatalf("NewPaidStrategy error: %v", err)
	}

	strategy, ok := ps.(*PaidStrategy)
	if !ok {
		t.Fatalf("expected PaidStrategy, got %T", ps)
	}

	if !chainStub.filterCalled {
		t.Fatal("expected filterChannelsFn to be called")
	}

	if strategy.channelID.Cmp(big.NewInt(999)) != 0 {
		t.Fatalf("unexpected channel ID: %s", strategy.channelID)
	}
	if strategy.signedAmount.Cmp(big.NewInt(90)) != 0 {
		t.Fatalf("unexpected signed amount: %s want 90", strategy.signedAmount)
	}
	if strategy.priceInCogs.Cmp(price) != 0 {
		t.Fatalf("unexpected price: %s want %s", strategy.priceInCogs, price)
	}
	if strategy.nonce.Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("unexpected nonce: %s", strategy.nonce)
	}
}

type stubChainOps struct {
	currentBlock *big.Int
	networkID    *big.Int
	filterResult *blockchain.MultiPartyEscrowChannelOpen
	ensureResult *big.Int
	filterCalled bool
}

func (s *stubChainOps) CurrentBlock(context.Context) (*big.Int, error) {
	return s.currentBlock, nil
}

func (s *stubChainOps) NetworkID(context.Context) (*big.Int, error) {
	return s.networkID, nil
}

func (s *stubChainOps) BuildBindOpts(common.Address, *big.Int, *big.Int, *ecdsa.PrivateKey, context.Context) (*blockchain.BindOpts, error) {
	return &blockchain.BindOpts{
		Call:     &bind.CallOpts{},
		Transact: &bind.TransactOpts{},
		Watch:    &bind.WatchOpts{},
		Filter:   &bind.FilterOpts{},
	}, nil
}

func (s *stubChainOps) FilterChannels([]common.Address, []common.Address, [][32]byte, *blockchain.BindOpts) (*blockchain.MultiPartyEscrowChannelOpen, error) {
	s.filterCalled = true
	return s.filterResult, nil
}

func (s *stubChainOps) EnsurePaymentChannel(common.Address, *blockchain.MultiPartyEscrowChannelOpen, *big.Int, *big.Int, *big.Int, *blockchain.BindOpts, *blockchain.ChansToWatch, []common.Address, []common.Address, [][32]byte) (*big.Int, error) {
	return s.ensureResult, nil
}

type stubChannelState struct {
	reply *ChannelStateReply
}

func (s stubChannelState) ChannelState(*ogrpc.ClientConn, context.Context, common.Address, *big.Int, *big.Int, *ecdsa.PrivateKey) (*ChannelStateReply, error) {
	return s.reply, nil
}
