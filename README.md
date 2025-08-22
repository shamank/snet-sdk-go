# snet-sdk-go

Golang SDK for the SingularityNet AI marketplace ecosystem

Go 1.24+

### Supported features

| Feature                     | Status         | Tests  | Notes |
|-----------------------------|----------------|--------|-------|
| Smart contract bindings     | ✅ Done         | ⚪ TODO |       |
| IPFS support                | ✅ Done         | ⚪ TODO |       |
| Lighthouse support          | ✅ Done         | ⚪ TODO |       |
| gRPC dynamic proto fetching | ✅ Done         | ⚪ TODO |       |
| Payment strategy: free-call | ✅ Done         | ⚪ TODO |       |
| Payment strategy: paid-call | 🔄 Testing     | ⚪ TODO |       |
| Payment strategy: pre-paid  | 🔄 In Progress | ⚪ TODO |       |
| Training support            | 🔄 In Progress | ⚪ TODO |       |
| Examples                    | 🔄 In Progress | -      |       |

---

## 📂 Project Structure

```plaintext
snet-sdk-go/
├── cmd/                          
│   ├── generate-smart-binds/     # Smart contract bindings generator
│   └────  main.go                # Entry point for the generator
├── examples/                     # Examples of using the SDK
│     
│
├── pkg/                          # Public packages (for SDK users)
│   ├── config/                   # Configuration loading and validation
│   ├── blockchain/               # Smart contract calls
│   ├── storage/                  # IPFS & Lighthouse support
│   ├── grpc/                     # gRPC service generation and invocation
│   ├── payment/                  # Payment strategies
│   ├── model/                    # Common structures
│   └── sdk/                      # High-level SDK facade
│   └── training/                 # Training support
│       
├── go.mod
└── README.md
