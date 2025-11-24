package blockchain

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"
)

func TestGetNewExpiration(t *testing.T) {
	current := big.NewInt(100)
	threshold := big.NewInt(50)
	want := big.NewInt(100 + 50 + 240)
	if got := GetNewExpiration(current, threshold); got.Cmp(want) != 0 {
		t.Fatalf("GetNewExpiration = %s, want %s", got, want)
	}
}

func TestWaitOpenIDSuccess(t *testing.T) {
	ch := make(chan *MultiPartyEscrowChannelOpen, 1)
	errc := make(chan error, 1)
	expected := big.NewInt(42)
	ch <- &MultiPartyEscrowChannelOpen{ChannelId: expected}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got, err := waitOpenID(ctx, ch, errc, time.Second)
	if err != nil {
		t.Fatalf("waitOpenID error: %v", err)
	}
	if got.Cmp(expected) != 0 {
		t.Fatalf("waitOpenID returned %s, want %s", got, expected)
	}
}

func TestWaitOpenIDTimeout(t *testing.T) {
	ch := make(chan *MultiPartyEscrowChannelOpen)
	errc := make(chan error)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if _, err := waitOpenID(ctx, ch, errc, 10*time.Millisecond); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestWaitDepositPropagatesError(t *testing.T) {
	ch := make(chan *MultiPartyEscrowDepositFunds)
	errc := make(chan error, 1)
	want := errors.New("watcher error")
	errc <- want

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := waitDeposit(ctx, ch, errc, time.Second); !errors.Is(err, want) {
		t.Fatalf("expected error %v, got %v", want, err)
	}
}
