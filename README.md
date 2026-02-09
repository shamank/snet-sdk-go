# snet-sdk-go

Golang SDK for the SingularityNet AI marketplace ecosystem

Go 1.24+

### Supported features

| Feature                     | Status |
|-----------------------------|--------|
| Smart contract bindings     | ✅ Done |  
| IPFS & Lighthouse support   | ✅ Done | 
| Services & orgs funcs       | ✅ Done | 
| Payment strategy: free-call | ✅ Done | 
| Payment strategy: paid-call | ✅ Done |
| Payment strategy: pre-paid  | ✅ Done |
| Training support            | ✅ Done |
| Examples & tutorials        | ✅ Done |

---

# Tutorials

* [Wiki](wiki)
* [Quick Start](wiki/quick_start.md)

## 📂 Project Structure

```plaintext
snet-sdk-go/
├── cmd/                          
│   ├── generate-smart-binds/     # Smart contract bindings generator
│   └────  main.go                # Entry point for the generator
├── examples/                     # Examples of using the SDK
├── wiki/                         # Tutorials of using the SDK
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
```

## License

This SDK is released under the MIT License.
