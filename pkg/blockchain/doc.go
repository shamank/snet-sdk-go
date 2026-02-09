// Package blockchain provides low-level Ethereum blockchain interaction for SingularityNET.
//
// This package contains clients and utilities for interacting with:
//   - SingularityNET Registry contract (organizations and services)
//   - Multi-Party Escrow (MPE) contract for payment channels
//   - ERC-20 FET token contract
//
// # Architecture
//
// The package is organized around three main client types:
//
// EVMClient:
//   - Base Ethereum client with contract bindings
//   - Transaction signing and submission
//   - Event watching and filtering
//   - Contract call utilities
//
// OrgClient:
//   - Organization registration and management
//   - Member management (add/remove)
//   - Metadata updates
//   - Service creation within organization
//
// ServiceClient:
//   - Service registration and management
//   - Metadata updates
//   - Group configuration
//   - Service deletion
//
// # Smart Contracts
//
// The package interacts with three main contracts:
//
// 1. Registry Contract:
//   - Stores organization and service metadata URIs
//   - Manages organization membership
//   - Handles service lifecycle
//
// 2. MPE (Multi-Party Escrow) Contract:
//   - Manages payment channels
//   - Holds FET token deposits
//   - Validates payment claims
//
// 3. FET Token Contract:
//   - ERC-20 token for payments
//   - Approve/transfer operations
//   - Balance queries
//
// # Payment Channels
//
// The MPE integration (mpe.go) handles the complete lifecycle of payment channels:
//
// Opening Channels:
//
//	channelID, err := evm.OpenChannel(
//		recipientAddr,
//		depositAmount,
//		expirationBlock,
//	)
//
// Adding Funds:
//
//	err := evm.AddFundsToChannel(channelID, additionalAmount)
//
// Extending Expiration:
//
//	err := evm.ExtendChannel(channelID, newExpirationBlock)
//
// Watching Events:
//
//	events, err := evm.WatchChannelEvents(channelID)
//	for event := range events {
//		// Handle ChannelOpen, ChannelAddFunds, etc.
//	}
//
// # Organization Operations
//
// Creating an Organization:
//
//	import "github.com/shamank/snet-sdk-go/pkg/sdk"
//
//	evm := snetSDK.GetEvm()
//	txHash, err := sdk.CreateOrganization(
//		evm,
//		cfg,
//		"my-org",
//		organizationMetadata,
//		memberAddresses,
//	)
//
// Managing Members:
//
//	org, _ := snetSDK.NewOrganizationClient("my-org", "default_group")
//
//	// Add members
//	txHash, err := org.AddMembers([]common.Address{addr1, addr2})
//
//	// Remove members
//	txHash, err := org.RemoveMembers([]common.Address{addr1})
//
// Updating Metadata:
//
//	newMetadata := &model.OrganizationMetaData{...}
//	txHash, err := org.UpdateOrgMetadataFull(newMetadata)
//
// # Service Operations
//
// Creating a Service:
//
//	org, _ := snetSDK.NewOrganizationClient("my-org", "default_group")
//	serviceMetadata := &model.ServiceMetadata{...}
//	txHash, err := org.CreateService("my-service", serviceMetadata)
//
// Updating Service:
//
//	service, _ := org.ServiceClient("my-service", "default_group")
//	updatedMetadata := &model.ServiceMetadata{...}
//	txHash, err := service.UpdateServiceMetadata(updatedMetadata)
//
// Deleting Service:
//
//	txHash, err := service.DeleteService()
//
// # Transaction Management
//
// All write operations return transaction hashes:
//
//	txHash, err := org.AddMembers(members)
//	if err != nil {
//		log.Fatalf("Transaction failed: %v", err)
//	}
//	fmt.Printf("Transaction submitted: %s\n", txHash.Hex())
//
// The SDK waits for transaction confirmation based on Config.Timeouts.ReceiptWait.
//
// # Gas Management
//
// Gas is estimated automatically for all transactions. You can customize gas settings
// in the Config if needed.
//
// Ensure your wallet has enough ETH for gas:
//
//	balance, err := evm.GetBalance(address)
//	if balance.Cmp(minGas) < 0 {
//		log.Println("Warning: Low ETH balance for gas")
//	}
//
// # Error Handling
//
// Common blockchain errors:
//
//   - Insufficient gas: Wallet lacks ETH for transaction fees
//   - Insufficient FET: Cannot fund payment channels
//   - Transaction reverted: Contract rejected the operation
//   - Timeout: Transaction not mined within configured timeout
//   - Nonce too low/high: Transaction ordering issue
//
// Example error handling:
//
//	txHash, err := org.CreateService("service-id", metadata)
//	if err != nil {
//		if strings.Contains(err.Error(), "insufficient funds") {
//			return fmt.Errorf("need more ETH for gas")
//		}
//		if strings.Contains(err.Error(), "already exists") {
//			return fmt.Errorf("service ID already taken")
//		}
//		return err
//	}
//
// # Event Watching
//
// Subscribe to on-chain events for real-time updates:
//
//	// Watch for new services
//	events, err := evm.WatchServiceEvents(orgID)
//	for event := range events {
//		fmt.Printf("New service: %s\n", event.ServiceID)
//	}
//
// # Private Key Management
//
// The package uses ECDSA private keys for signing:
//
//	// Parsed automatically from Config.PrivateKey
//	address, key, err := blockchain.ParsePrivateKeyECDSA(hexKey)
//
// Replace with your actual private key:
//
//	cfg.PrivateKey = "YOUR_PRIVATE_KEY"
//
// # Thread Safety
//
// EVMClient and derived clients are safe for concurrent use. Internal state
// (transaction nonces, contract instances) is protected with appropriate locking.
//
// # Resource Management
//
// Close EVM client to release WebSocket connections:
//
//	evm, _ := blockchain.InitEvm(...)
//	defer evm.Close()
//
// # Best Practices
//
// 1. Use WebSocket endpoints (wss://) for event subscriptions
// 2. Replace YOUR_PROJECT_ID and YOUR_PRIVATE_KEY with actual values
// 3. Check wallet balances before expensive operations
// 4. Set appropriate timeouts for transaction confirmation
// 5. Handle transaction failures gracefully
// 6. Monitor gas prices on mainnet
// 7. Test on Sepolia before deploying to mainnet
//
// # Usage Example
//
//	evm, err := blockchain.NewEVMClient(ethereumEndpoint, cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer evm.Close()
//
//	orgClient, err := evm.NewOrgClient("snet", "default_group")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Perform operations
//	services, _ := orgClient.ListServices()
//	fmt.Printf("Found %d services\n", len(services))
//
// # See Also
//
//   - sdk package for high-level API
//   - model package for metadata structures
//   - examples/orgs-and-services for complete examples
//   - wiki/orgs_services.md for detailed guide
package blockchain
