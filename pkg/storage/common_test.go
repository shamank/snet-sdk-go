package storage

import (
	"context"
	"fmt"
	"testing"
)

func TestFormatHash_SanitizesPrefixes(t *testing.T) {
	input := "ipfs://Qm-AbC=123!?#"
	if got := formatHash(input); got != "QmAbC=123" {
		t.Fatalf("formatHash returned %q, want %q", got, "QmAbC=123")
	}

	input = "filecoin://bafy-BeEf==/metadata"
	if got := formatHash(input); got != "bafyBeEf==metadata" {
		t.Fatalf("formatHash returned %q, want %q", got, "bafyBeEf==metadata")
	}
}

func TestRemoveSpecialCharacters(t *testing.T) {
	input := "Qm-._$Hello=World"
	if got := removeSpecialCharacters(input); got != "QmHello=World" {
		t.Fatalf("removeSpecialCharacters returned %q, want %q", got, "QmHello=World")
	}
}

func TestReadFileSelectsLighthouse(t *testing.T) {
	called := false
	fetcher := lighthouseFetcherFunc(func(endpoint, cid string) ([]byte, error) {
		called = true
		if endpoint != "https://gw/" {
			t.Fatalf("unexpected endpoint: %s", endpoint)
		}
		if cid != "CID123" {
			t.Fatalf("unexpected cid: %s", cid)
		}
		return []byte("ok"), nil
	})

	s := &Client{
		LighthouseUrl:     "https://gw/",
		lighthouseFetcher: fetcher,
	}
	data, err := s.ReadFile(context.Background(), "filecoin://CID123")
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(data) != "ok" {
		t.Fatalf("unexpected data: %q", data)
	}
	if !called {
		t.Fatal("expected lighthouse fetch to be used")
	}
}

func TestReadFileIPFSError(t *testing.T) {
	s := &Client{
		ipfsFetcher: ipfsFetcherFunc(func(context.Context, string) ([]byte, error) {
			return nil, fmt.Errorf("ipfs failure")
		}),
	}
	if _, err := s.ReadFile(context.Background(), "QmHash"); err == nil {
		t.Fatal("expected error from IPFS read")
	}
}

type lighthouseFetcherFunc func(string, string) ([]byte, error)

func (f lighthouseFetcherFunc) Fetch(endpoint, cid string) ([]byte, error) {
	return f(endpoint, cid)
}

type ipfsFetcherFunc func(context.Context, string) ([]byte, error)

func (f ipfsFetcherFunc) Fetch(ctx context.Context, hash string) ([]byte, error) {
	return f(ctx, hash)
}
