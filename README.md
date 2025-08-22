# snet-sdk-go

Golang SDK for the SingularityNet AI marketplace ecosystem

### Supported features

| Feature                     | Status           | Tests   | Notes  |
|-----------------------------|----------------- |-------- |--------|
| Smart contract bindings     | ✅ Done         | ⚪ TODO | ABIGEN |
| IPFS support                | ✅ Done         | ⚪ TODO |        |
| Lighthouse support          | ✅ Done         | ⚪ TODO |        |
| gRPC dynamic proto fetching | 🔄 In Progress  | ⚪ TODO |        |
| Payment strategy: free-call | ✅ Done         | ⚪ TODO |        |
| Payment strategy: paid-call | 🔄 In Progress  | ⚪ TODO |        |
| Payment strategy: pre-paid  | 🔄 In Progress  | ⚪ TODO |        |
| Training support            | 🔄 In Progress  | ⚪ TODO |        |
| Example projects & demos    | 🔄 In Progress  | ⚪ TODO |        |

---

## 📂 Project Structure

```plaintext
snet-sdk-go/
├── cmd/                          # Packages for the CLI application
│   ├── generate-smart-binds/     # Smart contract bindings generator
│   │   └── main.go               # Entry point for the generator
│   └── example/                  # Example of using the SDK as a CLI
│       └── main.go               # Demo main
│
├── pkg/                          # Public packages (for SDK users)
│   ├── config/                   # Configuration loading and validation
│   ├── blockchain/               # Smart contract calls (go-ethereum)
│   ├── storage/                  # IPFS & Lighthouse (Filecoin) clients
│   ├── grpc/                     # gRPC service generation and invocation
│   ├── payment/                  # Payment strategies (Strategy Pattern)
│   ├── model/                    # (optional) domain models
│   └── sdk/                      # High-level SDK facade
│       └── sdk.go
│
├── internal/                     # Internal utilities (not for public import)
│
├── scripts/                      # Scripts for generation, CI, pre-commit, etc.
│
├── go.mod
└── README.md
