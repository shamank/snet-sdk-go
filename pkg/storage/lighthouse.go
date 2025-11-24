package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// GetLighthouseFile fetches a blob from a Lighthouse HTTP gateway.
//
// It performs a simple HTTP GET to {lighthouseEndpoint}{cID} and returns the
// response body as bytes. The function logs the CID being requested for traceability.
//
// Parameters:
//   - lighthouseEndpoint: Base URL of the Lighthouse gateway (e.g.,
//     "https://gateway.lighthouse.storage/ipfs/"). The CID is concatenated
//     directly to this string; ensure the trailing slash if required by the gateway.
//   - cID: The content identifier to fetch.
//
// Returns:
//   - []byte: The raw file content on success.
//   - error: Any error encountered during the request or read.
//
// Note: This helper does not validate HTTP status codes; callers may want to
// verify content semantics (e.g., 404 handling) as needed.
func GetLighthouseFile(lighthouseEndpoint, cID string) ([]byte, error) {
	// Backward-compatible helper. Use context-aware variant with defaults.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return GetLighthouseFileCtx(ctx, lighthouseEndpoint, cID, 10*time.Second)
}

// GetLighthouseFileCtx is like GetLighthouseFile but allows passing a context
// and a per-request timeout. If timeout <= 0, no explicit client timeout is set.
func GetLighthouseFileCtx(ctx context.Context, lighthouseEndpoint, cID string, timeout time.Duration) ([]byte, error) {
	zap.L().Debug("Getting lighthouse file", zap.String("cid", cID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lighthouseEndpoint+cID, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	if timeout > 0 {
		client.Timeout = timeout
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("lighthouse GET %s: status %d: %s", req.URL, resp.StatusCode, string(b))
	}
	return io.ReadAll(resp.Body)
}
