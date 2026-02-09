## Working with Protocol Buffer (Proto) Files

Protocol Buffers (protobuf) are Google's language-neutral, platform-neutral mechanism for serializing structured data. In SingularityNET, proto files define the API interface for AI services, including available methods, input/output message structures, and data types.

Understanding proto files is essential for:
- Discovering available service methods
- Understanding expected input/output formats
- Generating client code
- Debugging service calls

## What is a Proto File?

A proto file (`.proto`) is a text file that defines:
- **Services**: RPC methods the service provides
- **Messages**: Data structures for requests and responses
- **Data Types**: Field types and constraints

### Example Proto File Structure

```protobuf
syntax = "proto3";

package example;

// Service definition
service Calculator {
    rpc Add(AddRequest) returns (AddResponse);
    rpc Multiply(MultiplyRequest) returns (MultiplyResponse);
}

// Request message
message AddRequest {
    int32 a = 1;
    int32 b = 2;
}

// Response message
message AddResponse {
    int32 result = 1;
}
```

## Retrieving Proto Files

You can inspect a service API by fetching its proto files using the SDK:

```go
package main

import (
	"fmt"
	"log"

	"github.com/singnet/snet-sdk-go/pkg/config"
	"github.com/singnet/snet-sdk-go/pkg/sdk"
)

func main() {
	c := config.Config{
		RPCAddr: "https://sepolia.infura.io/v3/YOUR_PROJECT_ID",
		Debug:   true,
		Network: config.Sepolia,
	}

	snetSDK := sdk.NewSDK(&c)

	service, err := snetSDK.NewServiceClient("orgID", "serviceID", "default_group")
	if err != nil {
		log.Fatalln(err)
	}

	// Get proto files as a map (filename -> content)
	protoFiles := service.ProtoFiles()
	
	// List all proto files
	fmt.Println("Available proto files:")
	for filename := range protoFiles {
		fmt.Printf("  - %s\n", filename)
	}

	// Read specific proto file content
	mainProto := protoFiles["main.proto"]
	fmt.Printf("\nContent of main.proto:\n%s\n", mainProto)

	// Save proto files to a directory
	err = service.ProtoFiles().Save("./proto_files/")
	if err != nil {
		log.Fatalln("Failed to save proto files:", err)
	}
	fmt.Println("Proto files saved to ./proto_files/")

	// Save proto files as a zip archive
	err = service.ProtoFiles().SaveAsZip("service_protos.zip")
	if err != nil {
		log.Fatalln("Failed to save as zip:", err)
	}
	fmt.Println("Proto files archived to service_protos.zip")
}
```

## Analyzing Proto Files

### Finding Available Methods

```go
import (
	"regexp"
	"strings"
)

// Extract RPC methods from proto file
func extractMethods(protoContent string) []string {
	var methods []string
	
	// Simple regex to find rpc definitions
	re := regexp.MustCompile(`rpc\s+(\w+)\s*\(`)
	matches := re.FindAllStringSubmatch(protoContent, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			methods = append(methods, match[1])
		}
	}
	
	return methods
}

// Usage
protoFiles := service.ProtoFiles()
mainProto := protoFiles["main.proto"]
methods := extractMethods(mainProto)

fmt.Println("Available methods:")
for _, method := range methods {
	fmt.Printf("  - %s\n", method)
}
```

### Identifying Message Structures

```go
// Extract message definitions from proto file
func extractMessages(protoContent string) []string {
	var messages []string
	
	re := regexp.MustCompile(`message\s+(\w+)\s*{`)
	matches := re.FindAllStringSubmatch(protoContent, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			messages = append(messages, match[1])
		}
	}
	
	return messages
}

// Usage
messages := extractMessages(mainProto)
fmt.Println("Defined messages:")
for _, msg := range messages {
	fmt.Printf("  - %s\n", msg)
}
```

### Understanding Service Package

```go
// Extract package name from proto file
func extractPackage(protoContent string) string {
	re := regexp.MustCompile(`package\s+([a-zA-Z0-9_.]+)\s*;`)
	matches := re.FindStringSubmatch(protoContent)
	
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Usage
packageName := extractPackage(mainProto)
fmt.Printf("Package: %s\n", packageName)
```

## Code Generation from Proto Files

Proto files can be used to generate strongly-typed client code for various languages.

### Generate Go Code

```bash
# Install protoc compiler
# macOS: brew install protobuf
# Linux: apt-get install protobuf-compiler

# Install Go proto plugin
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go code from proto files
protoc --go_out=. --go-grpc_out=. proto_files/*.proto
```

### Generate Python Code

```bash
# Install Python protobuf compiler
pip install grpcio-tools

# Generate Python code
python -m grpc_tools.protoc -I./proto_files --python_out=. --grpc_python_out=. proto_files/*.proto
```

## Practical Use Cases

### 1. Service Discovery

```go
func discoverServiceAPI(service *sdk.ServiceClient) {
	protoFiles := service.ProtoFiles()
	
	for filename, content := range protoFiles {
		fmt.Printf("\n=== %s ===\n", filename)
		
		// Extract and display methods
		methods := extractMethods(content)
		if len(methods) > 0 {
			fmt.Println("Methods:")
			for _, method := range methods {
				fmt.Printf("  - %s()\n", method)
			}
		}
		
		// Extract and display message types
		messages := extractMessages(content)
		if len(messages) > 0 {
			fmt.Println("\nMessage Types:")
			for _, msg := range messages {
				fmt.Printf("  - %s\n", msg)
			}
		}
	}
}
```

### 2. API Documentation Generation

```go
func generateAPIDocumentation(service *sdk.ServiceClient, outputPath string) error {
	protoFiles := service.ProtoFiles()
	
	var doc strings.Builder
	doc.WriteString("# Service API Documentation\n\n")
	
	for filename, content := range protoFiles {
		doc.WriteString(fmt.Sprintf("## %s\n\n", filename))
		doc.WriteString("```protobuf\n")
		doc.WriteString(content)
		doc.WriteString("\n```\n\n")
	}
	
	return os.WriteFile(outputPath, []byte(doc.String()), 0644)
}

// Usage
err := generateAPIDocumentation(service, "API_DOCS.md")
if err != nil {
	log.Fatalln("Failed to generate docs:", err)
}
```

### 3. Validating Service Compatibility

```go
func hasMethod(service *sdk.ServiceClient, methodName string) bool {
	protoFiles := service.ProtoFiles()
	
	for _, content := range protoFiles {
		methods := extractMethods(content)
		for _, method := range methods {
			if method == methodName {
				return true
			}
		}
	}
	
	return false
}

// Usage - check before calling
if !hasMethod(service, "Predict") {
	log.Fatalln("Service does not support Predict method")
}

// Safe to call
response, err := service.CallWithJSON("Predict", inputJson)
```

### 4. Backup and Version Control

```go
// Archive proto files with version info
func archiveProtoFiles(service *sdk.ServiceClient, version string) error {
	filename := fmt.Sprintf("protos_%s_%s.zip", 
		service.GetServiceID(), 
		version)
	
	return service.ProtoFiles().SaveAsZip(filename)
}

// Usage - backup before major changes
err := archiveProtoFiles(service, "v1.0.0")
if err != nil {
	log.Printf("Warning: Could not backup proto files: %v", err)
}
```

## Linking to Service Invocation

Once you understand the proto file structure, you can make informed service calls:

```go
// 1. Analyze proto to understand the API
protoFiles := service.ProtoFiles()
methods := extractMethods(protoFiles["main.proto"])
fmt.Printf("Available methods: %v\n", methods)

// 2. Construct request based on proto message definition
// For example, if proto defines:
//   message PredictRequest {
//     string text = 1;
//     int32 max_length = 2;
//   }

requestJSON := []byte(`{
	"text": "Hello world",
	"max_length": 100
}`)

// 3. Call the service method
response, err := service.CallWithJSON("Predict", requestJSON)
if err != nil {
	log.Fatalln("Service call failed:", err)
}

fmt.Printf("Response: %s\n", response)
```

For detailed information on making service calls, see [Service Invocation Guide](./quick_start.md).