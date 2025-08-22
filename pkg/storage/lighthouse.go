package storage

import (
	"io"
	"net/http"

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
	zap.L().Debug("Getting lighthouse file", zap.String("cid", cID))
	resp, err := http.Get(lighthouseEndpoint + cID)
	if err != nil {
		return nil, err
	}
	file, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return file, nil
}
