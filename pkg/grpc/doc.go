// Package grpc provides dynamic gRPC client functionality for SingularityNET services.
//
// This package enables runtime gRPC invocation without generated stubs by compiling
// proto files on-the-fly and using protocol buffer reflection for method resolution.
//
// # Features
//
//   - Dynamic gRPC client creation from proto descriptors
//   - Runtime protocol buffer compilation via protocompile
//   - Multiple invocation styles: Proto messages, JSON, or Go maps
//   - Automatic transport security detection (TLS/insecure)
//   - Connection lifecycle management
//
// # Client Creation
//
// Create a dynamic gRPC client with proto files:
//
//	protoFiles := map[string]string{
//		"service.proto": "syntax = \"proto3\"; ...",
//	}
//
//	client := grpc.NewClient("https://service.endpoint:443", protoFiles)
//	if client == nil {
//		log.Fatal("Failed to create gRPC client")
//	}
//	defer client.Close()
//
// # Invocation Methods
//
// Call with JSON (most common for SingularityNET):
//
//	ctx := context.Background()
//	input := []byte(`{"key": "value"}`)
//	output, err := client.CallWithJSON(ctx, "MethodName", input)
//
// Call with Go map:
//
//	params := map[string]any{"key": "value"}
//	result, err := client.CallWithMap(ctx, "MethodName", params)
//
// Call with proto message (advanced):
//
//	request := &MyRequest{Field: "value"}
//	response, err := client.CallWithProto(ctx, "MethodName", request)
//
// # Proto File Management
//
// Access compiled proto descriptors:
//
//	protoManager := client.ProtoFiles()
//	// Use for introspection, validation, or custom operations
//
// Proto files are compiled using protocompile (no external protoc required):
//   - Files provided as map[filename]content
//   - Compiled to linker.Files descriptors
//   - Used for runtime method resolution
//   - Support for imports and dependencies
//
// # Transport Security
//
// Transport is determined by endpoint scheme:
//
//	"https://host:443"  → TLS with system certificates
//	"http://host:8080"  → Insecure plaintext
//	"host:8080"         → Insecure plaintext (no scheme)
//
// # Method Resolution
//
// Methods are resolved automatically from proto descriptors:
//
//  1. Search all services for matching method name
//  2. Build fully-qualified path: /<package>.<Service>/<Method>
//  3. Resolve input/output message types
//  4. Marshal request and invoke via gRPC
//
// # Error Handling
//
// Common errors:
//   - Proto compilation failure: Invalid proto syntax
//   - Method not found: Method doesn't exist in proto files
//   - Connection error: Service unreachable
//   - Marshal/unmarshal error: Invalid input/output format
//
// Example:
//
//	output, err := client.CallWithJSON(ctx, "Process", input)
//	if err != nil {
//		if strings.Contains(err.Error(), "not found") {
//			return fmt.Errorf("method doesn't exist in service")
//		}
//		if strings.Contains(err.Error(), "connection") {
//			return fmt.Errorf("service is unavailable")
//		}
//		return err
//	}
//
// # Thread Safety
//
// Client instances are safe for concurrent use. Multiple goroutines can
// make parallel calls through the same client.
//
// # Resource Management
//
// Always close clients to release connections:
//
//	client := grpc.NewClient(endpoint, protoFiles)
//	defer client.Close()
//
// # See Also
//
//   - sdk.Service for high-level service invocation
//   - storage package for fetching proto files from IPFS
//   - examples/quick-start for complete usage example
package grpc
