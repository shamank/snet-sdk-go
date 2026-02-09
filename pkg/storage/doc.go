// Package storage provides abstractions for retrieving service metadata and API sources
// from distributed storage systems (IPFS and Lighthouse/Filecoin).
//
// SingularityNET stores service metadata, proto files, and other resources on
// decentralized storage. This package provides unified access to these resources
// regardless of the underlying storage backend.
//
// # Supported Backends
//
// The package supports two storage backends:
//
// IPFS (InterPlanetary File System):
//   - Content-addressed storage
//   - Access via Kubo HTTP API
//   - Default: https://ipfs.singularitynet.io:443
//   - CID format: Qm... or bafybei...
//
// Lighthouse (Filecoin Gateway):
//   - Filecoin-based permanent storage
//   - Access via HTTP gateway
//   - Default: https://gateway.lighthouse.storage/ipfs/
//   - Compatible with IPFS CIDs
//
// # Content Types
//
// Common content retrieved from storage:
//   - Service metadata JSON
//   - Organization metadata JSON
//   - Proto files (tar.gz archives)
//   - Model files
//   - Documentation
//
// # Storage Client
//
// The Storage client provides unified access:
//
//	import "github.com/singnet/snet-sdk-go/pkg/storage"
//
//	client := storage.NewStorage(
//		"https://ipfs.singularitynet.io:443",
//		"https://gateway.lighthouse.storage/ipfs/",
//	)
//
//	// Fetch from IPFS
//	data, err := client.FetchFromIPFS(cid)
//
//	// Fetch from Lighthouse
//	data, err := client.FetchFromLighthouse(cid)
//
// The SDK automatically creates a storage client from Config.IpfsURL and
// Config.LighthouseURL.
//
// # IPFS Operations
//
// Direct IPFS access:
//
//	ipfsClient, err := storage.NewIPFSClient("http://localhost:5001")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Fetch file by CID
//	content, err := ipfsClient.GetFileFromIPFS("Qm...")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Upload file (requires local IPFS node)
//	cid, err := ipfsClient.AddFileToIPFS(data)
//
// # Lighthouse Operations
//
// Fetch from Lighthouse gateway:
//
//	content, err := storage.GetLighthouseFile(
//		"https://gateway.lighthouse.storage/ipfs/",
//		"QmYourCIDHere",
//	)
//
// Lighthouse provides permanent storage backed by Filecoin, ideal for:
//   - Production service metadata
//   - Long-term archival
//   - Guaranteed availability
//
// # Proto File Processing
//
// Service metadata often references compressed proto archives. Use ParseProtoFiles
// to extract .proto files from tar or tar.gz format:
//
//	// Fetch proto archive from storage
//	archiveData, err := client.FetchFromIPFS(protoArchiveCID)
//
//	// Extract proto files
//	protoFiles, err := storage.ParseProtoFiles(archiveData)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// protoFiles is map[filename]content
//	for filename, content := range protoFiles {
//		fmt.Printf("Proto file: %s (%d bytes)\n", filename, len(content))
//	}
//
// ParseProtoFiles handles both:
//   - Uncompressed tar archives
//   - Gzip-compressed tar.gz archives
//
// # CID Formats
//
// Both IPFS and Lighthouse use Content Identifiers (CIDs):
//
// CIDv0 (legacy):
//   - Starts with "Qm"
//   - 46 characters
//   - Example: QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG
//
// CIDv1 (modern):
//   - Starts with "bafybei" or similar
//   - Variable length
//   - Example: bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi
//
// Both formats are supported by the storage package.
//
// # Caching
//
// For production applications, consider caching:
//
//	var metadataCache = make(map[string][]byte)
//
//	func fetchWithCache(cid string) ([]byte, error) {
//		if cached, ok := metadataCache[cid]; ok {
//			return cached, nil
//		}
//		data, err := client.FetchFromIPFS(cid)
//		if err != nil {
//			return nil, err
//		}
//		metadataCache[cid] = data
//		return data, nil
//	}
//
// # Error Handling
//
// Common errors:
//   - CID not found: Content not available on network
//   - Gateway timeout: Network or gateway issues
//   - Invalid CID format: Malformed content identifier
//   - Decode error: Archive corruption or wrong format
//
// Example:
//
//	data, err := client.FetchFromIPFS(cid)
//	if err != nil {
//		if strings.Contains(err.Error(), "not found") {
//			return fmt.Errorf("content not available: %s", cid)
//		}
//		if strings.Contains(err.Error(), "timeout") {
//			// Retry with Lighthouse
//			data, err = client.FetchFromLighthouse(cid)
//		}
//		return err
//	}
//
// # Custom IPFS Node
//
// Use local IPFS node for development:
//
//	cfg := &config.Config{
//		IpfsURL: "http://localhost:5001",
//		// ...
//	}
//
// Benefits:
//   - Faster access
//   - No rate limiting
//   - Upload capability
//   - Offline development
//
// # Best Practices
//
// 1. Use Lighthouse for production metadata (higher availability)
// 2. Cache frequently accessed content
// 3. Handle CID not found gracefully
// 4. Validate fetched content before use
// 5. Use compression for proto archives
// 6. Pin important content on multiple nodes
// 7. Monitor gateway availability
//
// # See Also
//
//   - config package for storage configuration
//   - model package for metadata structures
//   - sdk package for automatic storage integration
//   - examples/proto-files for proto handling example
package storage
