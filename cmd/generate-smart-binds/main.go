package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/abi/abigen"
	contracts "github.com/singnet/snet-ecosystem-contracts"
)

func main() {
	bindContent, err := abigen.Bind(
		[]string{"MultiPartyEscrow", "Registry", "FetchToken"},
		[]string{
			string(contracts.GetABIClean(contracts.MultiPartyEscrow)),
			string(contracts.GetABIClean(contracts.Registry)),
			string(contracts.GetABIClean(contracts.FetchToken))},
		[]string{
			string(contracts.GetBytecodeClean(contracts.MultiPartyEscrow)),
			string(contracts.GetBytecodeClean(contracts.Registry)),
			string(contracts.GetBytecodeClean(contracts.FetchToken))},
		nil, "blockchain", nil, nil)
	if err != nil {
		log.Fatalf("Failed to generate binding: %v", err)
	}

	root, err := moduleRoot()
	if err != nil {
		log.Fatalf("Failed to locate module root: %v", err)
	}

	outPath := filepath.Join(root, "pkg", "blockchain", "snet-contracts.go")
	if err := os.WriteFile(outPath, []byte(bindContent), 0o600); err != nil {
		log.Fatalf("Failed to write ABI binding: %v", err)
	}
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", fmt.Errorf("go.mod not found from %q", dir)
		}
		dir = next
	}
}
